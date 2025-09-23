// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package httpx

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Bl4cky99/mocker/internal/auth"
	"github.com/Bl4cky99/mocker/internal/config"
	"github.com/Bl4cky99/mocker/internal/render"
	"github.com/Bl4cky99/mocker/internal/validate"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

type Server struct {
	cfg        *config.Config
	log        *slog.Logger
	authMode   string
	authProv   auth.Provider
	handler    http.Handler
	httpSrv    *http.Server
	validators map[string]*validate.JSONSchemaValidator
	renderer   *render.Renderer
}

type Option func(*Server)

func WithLogger(l *slog.Logger) Option {
	return func(s *Server) {
		s.log = l
	}
}

func WithAuth(p auth.Provider, mode string) Option {
	return func(s *Server) {
		s.authProv = p
		s.authMode = mode
	}
}

func WithRenderer(r *render.Renderer) Option {
	return func(s *Server) {
		s.renderer = r
	}
}

func New(ctx context.Context, cfg *config.Config, opts ...Option) (*Server, error) {
	s := &Server{cfg: cfg, log: slog.New(slog.NewTextHandler(os.Stdout, nil))}
	for _, o := range opts {
		o(s)
	}

	s.validators = make(map[string]*validate.JSONSchemaValidator)
	for _, ep := range cfg.Endpoints {
		if ep.Validate == nil || ep.Validate.SchemaFile == "" {
			continue
		}

		abs, _ := filepath.Abs(ep.Validate.SchemaFile)
		if s.validators[abs] != nil {
			continue
		}

		v, err := validate.CompileSchema(abs, validate.JSONSchemaValidatorOptions{
			AssertFormat:  true,
			AssertContent: false,
			DefaultDraft:  jsonschema.Draft2020,
		})
		if err != nil {
			return nil, err
		}

		s.validators[abs] = v
	}

	s.handler = buildRouter(s)
	s.httpSrv = &http.Server{
		Addr:        cfg.Server.Addr,
		Handler:     s.handler,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}

	return s, nil
}

func (s *Server) Handler() http.Handler {
	return s.handler
}

func (s *Server) ListenAndServe() error {
	s.log.Info("mocker running", "addr", s.cfg.Server.Addr, "basePath", s.cfg.Server.BasePath)
	return s.httpSrv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}
