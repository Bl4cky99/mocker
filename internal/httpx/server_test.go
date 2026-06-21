// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package httpx

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Bl4cky99/mocker/internal/auth"
	"github.com/Bl4cky99/mocker/internal/config"
	"github.com/Bl4cky99/mocker/internal/validate"
)

func mustLoad(path string) *config.Config {
	cfg, err := config.Load(path)
	Expect(err).NotTo(HaveOccurred(), "could not load test config: %v", path)
	return cfg
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

type stubProvider struct {
	principal auth.Principal
	ok        bool
	err       error
}

func (s stubProvider) Authenticate(*http.Request) (auth.Principal, bool, error) {
	return s.principal, s.ok, s.err
}

var _ = Describe("Server", func() {
	It("responds 200 with a request-id header for a registered route", func() {
		cfg := mustLoad(filepath.Join("testdata", "ok.basic.yaml"))
		s, _ := New(context.Background(), cfg, WithLogger(discardLogger()))
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		resp := httptest.NewRecorder()

		s.Handler().ServeHTTP(resp, req)

		Expect(resp.Code).To(Equal(http.StatusOK))
		Expect(resp.Header().Get("X-Request-ID")).NotTo(BeEmpty())
	})

	Describe("New with options", func() {
		It("applies logger, auth, and schema options", func() {
			tmp := GinkgoT().TempDir()
			schemaPath := filepath.Join(tmp, "schema.json")
			Expect(os.WriteFile(schemaPath, []byte(`{"type":"object"}`), 0o600)).To(Succeed())

			cfg := &config.Config{
				Server: config.ServerConfig{Addr: ":0", BasePath: "/api"},
				Endpoints: []config.Endpoint{{
					Method:   "POST",
					Path:     "/foo",
					Validate: &config.ValidateSpec{SchemaFile: schemaPath},
					Responses: []config.ResponseVariant{{
						Status: 201,
						Body:   "{}",
					}},
				}},
			}

			buf := new(bytes.Buffer)
			logger := slog.New(slog.NewTextHandler(buf, nil))
			prov := stubProvider{principal: auth.Principal{Name: "tester"}, ok: true}

			ctx := context.Background()
			srv, err := New(ctx, cfg, WithLogger(logger), WithAuth(prov, "token"))
			Expect(err).NotTo(HaveOccurred())

			Expect(srv.log).To(Equal(logger))
			Expect(srv.authMode).To(Equal("token"))
			Expect(srv.authProv).To(Equal(prov))
			Expect(srv.handler).NotTo(BeNil())
			Expect(srv.httpSrv).NotTo(BeNil())
			Expect(srv.httpSrv.Addr).To(Equal(cfg.Server.Addr))
			Expect(srv.httpSrv.BaseContext(nil)).To(Equal(ctx))

			abs, _ := filepath.Abs(schemaPath)
			Expect(srv.validators[abs]).NotTo(BeNil())
		})
	})

	Describe("endpointHandler", func() {
		It("writes status, body, and response headers", func() {
			srv := &Server{
				cfg: &config.Config{
					Server: config.ServerConfig{DefaultHeaders: map[string]string{"X-App": "mocker"}},
				},
				log: discardLogger(),
			}
			ep := config.Endpoint{Responses: []config.ResponseVariant{{
				Status:  http.StatusOK,
				Headers: map[string]string{"Content-Type": "application/json"},
				Body:    "{\"ok\":true}",
			}}}

			h := endpointHandler(srv, ep)
			resp := httptest.NewRecorder()
			resp.Header().Set("X-App", "override")
			h.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/", nil))

			Expect(resp.Code).To(Equal(http.StatusOK))
			Expect(strings.TrimSpace(resp.Body.String())).To(Equal("{\"ok\":true}"))
			Expect(resp.Header().Get("X-App")).To(Equal("override"), "default header must not overwrite existing value")
			Expect(resp.Header().Get("Content-Type")).To(Equal("application/json"))
		})

		It("returns 500 when the body file is missing", func() {
			srv := &Server{cfg: &config.Config{Server: config.ServerConfig{}}, log: discardLogger()}
			ep := config.Endpoint{Responses: []config.ResponseVariant{{BodyFile: "missing.json", Status: http.StatusCreated}}}

			h := endpointHandler(srv, ep)
			resp := httptest.NewRecorder()
			h.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/", nil))

			Expect(resp.Code).To(Equal(http.StatusInternalServerError))
		})

		It("exits early when context is cancelled during delay", func() {
			srv := &Server{cfg: &config.Config{Server: config.ServerConfig{}}, log: discardLogger()}
			ep := config.Endpoint{Responses: []config.ResponseVariant{{DelayMs: 50}}}

			h := endpointHandler(srv, ep)
			resp := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			ctx, cancel := context.WithCancel(req.Context())
			cancel()

			start := time.Now()
			h.ServeHTTP(resp, req.WithContext(ctx))

			Expect(time.Since(start)).To(BeNumerically("<", 30*time.Millisecond))
			Expect(resp.Body.Len()).To(Equal(0))
		})
	})

	Describe("validateBody middleware", func() {
		var ctHandler http.Handler

		BeforeEach(func() {
			tmp := GinkgoT().TempDir()
			schemaPath := filepath.Join(tmp, "schema.json")
			Expect(os.WriteFile(schemaPath, []byte(`{"type":"object","required":["name"]}`), 0o600)).To(Succeed())

			validator, err := validate.CompileSchema(schemaPath, validate.JSONSchemaValidatorOptions{})
			Expect(err).NotTo(HaveOccurred())

			nextCalled := 0
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled++
				w.WriteHeader(http.StatusCreated)
			})
			_ = nextCalled
			ctHandler = validateBody("application/json", validator)(next)
		})

		It("passes a valid JSON body through", func() {
			resp := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"mocker"}`))
			req.Header.Set("Content-Type", "application/json")
			ctHandler.ServeHTTP(resp, req)
			Expect(resp.Code).To(Equal(http.StatusCreated))
		})

		It("returns 400 for a schema-violating body", func() {
			resp := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"foo":1}`))
			req.Header.Set("Content-Type", "application/json")
			ctHandler.ServeHTTP(resp, req)
			Expect(resp.Code).To(Equal(http.StatusBadRequest))
		})

		It("returns 415 for the wrong content-type", func() {
			resp := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
			req.Header.Set("Content-Type", "text/plain")
			ctHandler.ServeHTTP(resp, req)
			Expect(resp.Code).To(Equal(http.StatusUnsupportedMediaType))
		})

		It("returns the original handler unchanged when schema is nil", func() {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
			noop := validateBody("", nil)
			Expect(reflect.ValueOf(noop(next)).Pointer()).To(Equal(reflect.ValueOf(next).Pointer()))
		})
	})

	Describe("requireAuth middleware", func() {
		It("passes the principal into context on success", func() {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, ok := principalFrom(r.Context())
				Expect(ok).To(BeTrue(), "principal missing in context")
				w.WriteHeader(http.StatusAccepted)
			})
			rec := httptest.NewRecorder()
			requireAuth(stubProvider{principal: auth.Principal{Name: "ok"}, ok: true}, "token")(next).
				ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
			Expect(rec.Code).To(Equal(http.StatusAccepted))
		})

		It("returns 401 when auth provider returns an error", func() {
			rec := httptest.NewRecorder()
			requireAuth(stubProvider{err: errors.New("boom")}, "token")(
				http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
			).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
			Expect(rec.Code).To(Equal(http.StatusUnauthorized))
		})

		It("returns 401 with WWW-Authenticate header for basic auth failure", func() {
			rec := httptest.NewRecorder()
			requireAuth(stubProvider{ok: false}, "basic")(
				http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
			).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
			Expect(rec.Code).To(Equal(http.StatusUnauthorized))
			Expect(rec.Header().Get("WWW-Authenticate")).To(ContainSubstring("Basic"))
		})
	})

	Describe("skipAuthForOPTIONS middleware", func() {
		It("skips the inner middleware for OPTIONS but runs it for other methods", func() {
			called := 0
			mw := func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					called++
					next.ServeHTTP(w, r)
				})
			}
			final := skipAuthForOPTIONS(mw)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))

			rec := httptest.NewRecorder()
			final.ServeHTTP(rec, httptest.NewRequest(http.MethodOptions, "/", nil))
			Expect(rec.Code).To(Equal(http.StatusNoContent))
			optionsCalled := called

			rec = httptest.NewRecorder()
			final.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
			Expect(rec.Code).To(Equal(http.StatusNoContent))
			Expect(called).To(BeNumerically(">", optionsCalled))
		})
	})

	Describe("recoverMW middleware", func() {
		It("returns 500 and logs when a handler panics", func() {
			buf := new(bytes.Buffer)
			log := slog.New(slog.NewTextHandler(buf, nil))
			panicHandler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") })

			rec := httptest.NewRecorder()
			recoverMW(log)(panicHandler).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

			Expect(rec.Code).To(Equal(http.StatusInternalServerError))
			Expect(buf.Len()).To(BeNumerically(">", 0))
		})
	})

	Describe("requestIDMW and loggingMW middleware", func() {
		It("injects request-id into context and logs the request", func() {
			buf := new(bytes.Buffer)
			log := slog.New(slog.NewTextHandler(buf, nil))

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, ok := r.Context().Value(ctxKeyReqID{}).(string)
				Expect(ok).To(BeTrue(), "request id not in context")
				w.WriteHeader(http.StatusCreated)
			})

			wrapped := loggingMW(log)(requestIDMW()(next))
			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/hello", nil))

			Expect(rec.Code).To(Equal(http.StatusCreated))
			Expect(rec.Header().Get("X-Request-ID")).NotTo(BeEmpty())
			Expect(buf.Len()).To(BeNumerically(">", 0))
		})
	})
})
