// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package auth

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
)

type BasicAuth struct {
	users map[string]string
	Realm string
}

func NewBasicAuth(users map[string]string, realm string) *BasicAuth {
	cp := make(map[string]string, len(users))
	for u, p := range users {
		cp[u] = p
	}

	return &BasicAuth{users: cp, Realm: realm}
}

func (a *BasicAuth) Authenticate(r *http.Request) (Principal, bool, error) {
	const prefix = "Basic "
	h := r.Header.Get("Authorization")
	if h == "" {
		return Principal{}, false, nil
	}

	if !strings.HasPrefix(h, prefix) {
		return Principal{}, false, nil
	}

	raw := strings.TrimPrefix(h, prefix)
	dec, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return Principal{}, false, err
	}

	parts := strings.SplitN(string(dec), ":", 2)
	if len(parts) != 2 {
		return Principal{}, false, errors.New("invalid basic token")
	}
	user, pass := parts[0], parts[1]

	want, ok := a.users[user]
	if !ok {
		return Principal{}, false, nil
	}

	if subtle.ConstantTimeCompare([]byte(want), []byte(pass)) == 1 {
		return Principal{Name: user}, true, nil
	}
	return Principal{}, false, nil
}
