// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package auth

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

type TokenAuth struct {
	Header  string
	Prefix  string
	allowed map[string]struct{}
}

func NewTokenAuth(header, prefix string, tokens []string) *TokenAuth {
	m := make(map[string]struct{}, len(tokens))
	for _, t := range tokens {
		if t == "" {
			continue
		}
		m[t] = struct{}{}
	}
	return &TokenAuth{Header: header, Prefix: prefix, allowed: m}
}

func (a *TokenAuth) Authenticate(r *http.Request) (Principal, bool, error) {
	hv := r.Header.Get(a.Header)
	if hv == "" {
		return Principal{}, false, nil
	}

	token := hv
	if a.Prefix != "" {
		if !strings.HasPrefix(hv, a.Prefix) {
			return Principal{}, false, nil
		}
		token = strings.TrimPrefix(hv, a.Prefix)
		token = strings.TrimSpace(token)
	}

	for t := range a.allowed {
		if subtle.ConstantTimeCompare([]byte(t), []byte(token)) == 1 {
			return Principal{Name: "token"}, true, nil
		}
	}

	return Principal{}, false, nil
}
