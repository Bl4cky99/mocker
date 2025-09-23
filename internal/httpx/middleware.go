// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package httpx

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Bl4cky99/mocker/internal/auth"
	"github.com/Bl4cky99/mocker/internal/validate"
)

func recoverMW(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic", "err", rec)
					http.Error(w, "internal error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

type ctxKeyReqID struct{}

func requestIDMW() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := strconv.FormatInt(time.Now().UnixNano(), 36)
			ctx := context.WithValue(r.Context(), ctxKeyReqID{}, id)
			w.Header().Set("X-Request-ID", id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (lw *loggingResponseWriter) WriteHeader(code int) {
	lw.status = code
	lw.ResponseWriter.WriteHeader(code)
}

func loggingMW(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			lrw := &loggingResponseWriter{ResponseWriter: w, status: 200}
			next.ServeHTTP(lrw, r)
			rid, _ := r.Context().Value(ctxKeyReqID{}).(string)
			log.Info("http",
				"method", r.Method, "path", r.URL.Path,
				"status", lrw.status, "dur_ms", time.Since(start).Milliseconds(),
				"rid", rid)
		})
	}
}

type ctxKeyPrincipal struct{}

func principalFrom(ctx context.Context) (auth.Principal, bool) {
	p, ok := ctx.Value(ctxKeyPrincipal{}).(auth.Principal)
	return p, ok
}

func requireAuth(p auth.Provider, mode string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if p == nil || mode == "none" {
			return next
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			pr, ok, err := p.Authenticate(r)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if !ok {
				if mode == "basic" {
					w.Header().Set("WWW-Authenticate", `Basic realm="mocker"`)
				}

				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), ctxKeyPrincipal{}, pr)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func skipAuthForOPTIONS(mw func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		}))
	}
}

func validateBody(wantCT string, v *validate.JSONSchemaValidator) func(http.Handler) http.Handler {
	if strings.TrimSpace(wantCT) == "" && v == nil {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ct := strings.TrimSpace(wantCT); ct != "" {
				got, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
				if err != nil || !strings.EqualFold(got, ct) {
					http.Error(w, "invalid content-type", http.StatusUnsupportedMediaType)
					return
				}
			}

			if v == nil {
				next.ServeHTTP(w, r)
				return
			}

			const maxBody = 1 << 20
			var buf bytes.Buffer
			limited := http.MaxBytesReader(w, r.Body, maxBody)
			if _, err := io.Copy(&buf, limited); err != nil && err != io.EOF {
				http.Error(w, "failed to read body", http.StatusBadRequest)
				return
			}
			body := buf.Bytes()
			r.Body = io.NopCloser(bytes.NewReader(body))

			if err := v.Validate(body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
