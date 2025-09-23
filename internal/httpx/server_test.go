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
	"testing"
	"time"

	"github.com/Bl4cky99/mocker/internal/auth"
	"github.com/Bl4cky99/mocker/internal/config"
	"github.com/Bl4cky99/mocker/internal/validate"
)

func TestServer(t *testing.T) {
	cfg := mustLoad(t, filepath.Join("testdata", "ok.basic.yaml"))
	s, _ := New(context.Background(), cfg, WithLogger(discardLogger()))
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	resp := httptest.NewRecorder()

	s.Handler().ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("wrong status: %d, expected '200'", resp.Code)
	}
	if rid := resp.Header().Get("X-Request-ID"); rid == "" {
		t.Fatal("missing request id header")
	}
}

func mustLoad(t *testing.T, path string) *config.Config {
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("could not load test config: %v", path)
	}

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

func TestNewWithOptions(t *testing.T) {
	tmp := t.TempDir()
	schemaPath := filepath.Join(tmp, "schema.json")
	if err := os.WriteFile(schemaPath, []byte(`{"type":"object"}`), 0o600); err != nil {
		t.Fatalf("write schema: %v", err)
	}

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
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if srv.log != logger {
		t.Fatalf("logger option not applied")
	}
	if srv.authMode != "token" || srv.authProv != prov {
		t.Fatalf("auth option not applied")
	}
	if srv.handler == nil {
		t.Fatalf("handler not initialised")
	}
	if srv.httpSrv == nil {
		t.Fatalf("http server not initialised")
	}
	if srv.httpSrv.Addr != cfg.Server.Addr {
		t.Fatalf("http server addr mismatch: %s", srv.httpSrv.Addr)
	}
	if got := srv.httpSrv.BaseContext(nil); got != ctx {
		t.Fatalf("base context mismatch")
	}

	abs, _ := filepath.Abs(schemaPath)
	if v := srv.validators[abs]; v == nil {
		t.Fatalf("validator for schema not loaded")
	}
}

func TestEndpointHandlerResponds(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	h.ServeHTTP(resp, req)

	if got := resp.Code; got != http.StatusOK {
		t.Fatalf("status mismatch: %d", got)
	}
	if body := strings.TrimSpace(resp.Body.String()); body != "{\"ok\":true}" {
		t.Fatalf("body mismatch: %q", body)
	}
	if want := "override"; resp.Header().Get("X-App") != want {
		t.Fatalf("default header overwrote existing value")
	}
	if ct := resp.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("missing response header")
	}
}

func TestEndpointHandlerBodyFileError(t *testing.T) {
	srv := &Server{cfg: &config.Config{Server: config.ServerConfig{}}, log: discardLogger()}
	ep := config.Endpoint{Responses: []config.ResponseVariant{{BodyFile: "missing.json", Status: http.StatusCreated}}}

	h := endpointHandler(srv, ep)
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	h.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.Code)
	}
}

func TestEndpointHandlerDelayCancelledContext(t *testing.T) {
	srv := &Server{cfg: &config.Config{Server: config.ServerConfig{}}, log: discardLogger()}
	ep := config.Endpoint{Responses: []config.ResponseVariant{{DelayMs: 50}}}

	h := endpointHandler(srv, ep)
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx, cancel := context.WithCancel(req.Context())
	cancel()

	start := time.Now()
	h.ServeHTTP(resp, req.WithContext(ctx))

	if elapsed := time.Since(start); elapsed > 30*time.Millisecond {
		t.Fatalf("handler did not cancel early: %v", elapsed)
	}
	if resp.Body.Len() != 0 {
		t.Fatalf("expected no body when context cancelled")
	}
}

func TestValidateBody(t *testing.T) {
	tmp := t.TempDir()
	schemaPath := filepath.Join(tmp, "schema.json")
	if err := os.WriteFile(schemaPath, []byte(`{"type":"object","required":["name"]}`), 0o600); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	validator, err := validate.CompileSchema(schemaPath, validate.JSONSchemaValidatorOptions{})
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}

	nextCalled := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled++
		w.WriteHeader(http.StatusCreated)
	})

	ctHandler := validateBody("application/json", validator)(next)

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"mocker"}`))
	req.Header.Set("Content-Type", "application/json")
	ctHandler.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated || nextCalled != 1 {
		t.Fatalf("expected handler to proceed, status %d", resp.Code)
	}

	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"foo":1}`))
	req.Header.Set("Content-Type", "application/json")
	ctHandler.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected schema error, got %d", resp.Code)
	}

	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "text/plain")
	ctHandler.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d", resp.Code)
	}

	noop := validateBody("", nil)
	if reflect.ValueOf(noop(next)).Pointer() != reflect.ValueOf(next).Pointer() {
		t.Fatalf("expected noop middleware to return original handler")
	}
}

func TestRequireAuth(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := principalFrom(r.Context()); !ok {
			t.Fatalf("principal missing in context")
		}
		w.WriteHeader(http.StatusAccepted)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	requireAuth(stubProvider{principal: auth.Principal{Name: "ok"}, ok: true}, "token")(next).ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected success, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	requireAuth(stubProvider{err: errors.New("boom")}, "token")(next).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected error to return 401")
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	requireAuth(stubProvider{ok: false}, "basic")(next).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized when not ok")
	}
	if got := rec.Header().Get("WWW-Authenticate"); !strings.Contains(got, "Basic") {
		t.Fatalf("missing basic auth challenge header")
	}
}

func TestSkipAuthForOPTIONS(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	final.ServeHTTP(rec, req)
	if called == 0 || rec.Code != http.StatusNoContent {
		t.Fatalf("expected OPTIONS request to reach handler")
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	final.ServeHTTP(rec, req)
	if called <= 1 || rec.Code != http.StatusNoContent {
		t.Fatalf("expected handler to be invoked for GET")
	}
}

func TestRecoverMiddleware(t *testing.T) {
	buf := new(bytes.Buffer)
	log := slog.New(slog.NewTextHandler(buf, nil))
	panicHandler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	recoverMW(log)(panicHandler).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("panic should return 500, got %d", rec.Code)
	}
	if buf.Len() == 0 {
		t.Fatalf("expected panic to be logged")
	}
}

func TestRequestIDAndLoggingMiddleware(t *testing.T) {
	buf := new(bytes.Buffer)
	log := slog.New(slog.NewTextHandler(buf, nil))

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Context().Value(ctxKeyReqID{}).(string); !ok {
			t.Fatalf("request id not in context")
		}
		w.WriteHeader(http.StatusCreated)
	})

	wrapped := loggingMW(log)(requestIDMW()(next))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/hello", nil)
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if rid := rec.Header().Get("X-Request-ID"); rid == "" {
		t.Fatalf("missing request id header")
	}
	if buf.Len() == 0 {
		t.Fatalf("expected log output")
	}
}
