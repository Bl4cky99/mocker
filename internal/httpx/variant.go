// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package httpx

import (
	"net/http"

	"github.com/Bl4cky99/mocker/internal/config"
)

func pickVariant(ep config.Endpoint, r *http.Request) config.ResponseVariant {
	var fallback *config.ResponseVariant

	for i := range ep.Responses {
		v := &ep.Responses[i]
		if v.When == nil || (len(v.When.Query) == 0 && len(v.When.Header) == 0) {
			if fallback == nil {
				fallback = v
			}
			continue
		}
		if whenMatches(r, v.When) {
			return *v
		}
	}

	if fallback != nil {
		return *fallback
	}

	return ep.Responses[0]
}

func whenMatches(r *http.Request, w *config.WhenClause) bool {
	if len(w.Query) > 0 {
		q := r.URL.Query()
		for k, want := range w.Query {
			if q.Get(k) != want {
				return false
			}
		}
	}

	if len(w.Header) > 0 {
		for k, want := range w.Header {
			if r.Header.Get(k) != want {
				return false
			}
		}
	}

	return true
}
