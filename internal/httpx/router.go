// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package httpx

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/Bl4cky99/mocker/internal/validate"
	"github.com/go-chi/chi/v5"
)

func buildRouter(s *Server) http.Handler {
	r := chi.NewRouter()
	r.Use(recoverMW(s.log), requestIDMW(), loggingMW(s.log))

	if s.authMode != "" && s.authMode != "none" && s.authProv != nil {
		if s.cfg.Server.CORS != nil {
			r.Use(skipAuthForOPTIONS(requireAuth(s.authProv, s.authMode)))
		} else {
			r.Use(requireAuth(s.authProv, s.authMode))
		}

	}

	base := strings.TrimRight(s.cfg.Server.BasePath, "/")
	if base == "" {
		base = "/"
	}
	r.Route(base, func(sr chi.Router) {
		for _, ep := range s.cfg.Endpoints {
			h := endpointHandler(s, ep)

			if ep.Validate != nil && (ep.Validate.ContentType != "" || ep.Validate.SchemaFile != "") {
				var sch *validate.JSONSchemaValidator
				if ep.Validate.SchemaFile != "" {
					abs, _ := filepath.Abs(ep.Validate.SchemaFile)
					sch = s.validators[abs]
				}

				sr.With(validateBody(ep.Validate.ContentType, sch)).Method(ep.Method, ep.Path, h)
				continue
			}

			sr.Method(ep.Method, ep.Path, h)
		}
	})

	return r
}
