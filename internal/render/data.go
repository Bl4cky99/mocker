// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package render

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Data struct {
	Path       map[string]string
	Query      map[string]string
	Header     map[string]string
	Body       any
	NowRFC3339 string
}

func BuildData(r *http.Request, now string) Data {
	path := make(map[string]string)
	query := make(map[string]string)
	header := make(map[string]string)

	if rc := chi.RouteContext(r.Context()); rc != nil {
		for i := range rc.URLParams.Keys {
			key := rc.URLParams.Keys[i]
			val := rc.URLParams.Values[i]
			if key != "" {
				path[key] = val
			}
		}
	}

	for k, vals := range r.URL.Query() {
		if len(vals) > 0 {
			query[k] = vals[0]
		}
	}

	for k, vals := range r.Header {
		if len(vals) > 0 {
			header[http.CanonicalHeaderKey(k)] = vals[0]
		}
	}

	return Data{
		Path:       path,
		Query:      query,
		Header:     header,
		NowRFC3339: now,
	}
}
