// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package httpx

import (
	"net/http"
	"os"
	"time"

	"github.com/Bl4cky99/mocker/internal/config"
	"github.com/Bl4cky99/mocker/internal/render"
)

func endpointHandler(s *Server, ep config.Endpoint) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v := pickVariant(ep, r)

		for k, val := range s.cfg.Server.DefaultHeaders {
			if w.Header().Get(k) == "" {
				w.Header().Set(k, val)
			}
		}

		for k, val := range v.Headers {
			w.Header().Set(k, val)
		}

		if v.DelayMs > 0 {
			d := time.Duration(v.DelayMs) * time.Millisecond
			timer := time.NewTimer(d)
			defer timer.Stop()

			select {
			case <-timer.C:
			case <-r.Context().Done():
				return
			}
		}

		now := time.Now().UTC().Format(time.RFC3339)
		data := render.BuildData(r, now)

		var body []byte
		var err error
		switch {
		case v.Body != "":
			if s.renderer != nil {
				body, err = s.renderer.RenderString(v.Body, data)
				if err != nil {
					s.log.Error("template render (inline) failed", "err", err)
					http.Error(w, "template error", http.StatusInternalServerError)
					return
				}
			} else {
				body = []byte(v.Body)
			}
		case v.BodyFile != "":
			if s.renderer != nil {
				body, err = s.renderer.RenderFile(v.BodyFile, data)
				if err != nil {
					s.log.Error("template render (file) failed", "file", v.BodyFile, "err", err)
					http.Error(w, "template error", http.StatusInternalServerError)
					return
				}
			} else {
				body, err = os.ReadFile(v.BodyFile)
				if err != nil {
					s.log.Error(ErrBodyFileNotFound.Error())
					http.Error(w, "internal error", http.StatusInternalServerError)
					return
				}
			}
		}

		if v.Status == 0 {
			v.Status = 200
		}

		w.WriteHeader(v.Status)
		if len(body) > 0 {
			_, _ = w.Write(body)
		}
	}
}
