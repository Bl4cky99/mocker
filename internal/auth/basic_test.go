// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package auth

import (
	"encoding/base64"
	"net/http/httptest"
	"testing"

	"github.com/Bl4cky99/mocker/internal/errx"
)

func b64(user, pass string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))
}

func TestBasicAuth(t *testing.T) {
	type tc struct {
		name    string
		header  string
		build   func() *BasicAuth
		wantOK  bool
		wantErr bool
		wantSub []string
	}

	tests := []tc{
		{
			name:   "missing header => not authenticated",
			header: "",
			build: func() *BasicAuth {
				return NewBasicAuth(map[string]string{"alice": "secret"}, "mocker")
			},
			wantOK:  false,
			wantErr: false,
		},
		{
			name:   "wrong scheme",
			header: "Bearer something",
			build: func() *BasicAuth {
				return NewBasicAuth(map[string]string{"alice": "secret"}, "mocker")
			},
			wantOK:  false,
			wantErr: false,
		},
		{
			name:   "invalid base64 => error",
			header: "Basic !!!notbase64!!!",
			build: func() *BasicAuth {
				return NewBasicAuth(map[string]string{"alice": "secret"}, "mocker")
			},
			wantOK:  false,
			wantErr: true,
			// base64 Fehlermeldungen variieren; kein Substring nötig
		},
		{
			name:   "decoded without colon => error",
			header: "Basic " + base64.StdEncoding.EncodeToString([]byte("justuser")),
			build: func() *BasicAuth {
				return NewBasicAuth(map[string]string{"alice": "secret"}, "mocker")
			},
			wantOK:  false,
			wantErr: true,
			wantSub: []string{"invalid basic token"},
		},
		{
			name:   "unknown user",
			header: b64("mallory", "secret"),
			build: func() *BasicAuth {
				return NewBasicAuth(map[string]string{"alice": "secret"}, "mocker")
			},
			wantOK:  false,
			wantErr: false,
		},
		{
			name:   "wrong password",
			header: b64("alice", "nope"),
			build: func() *BasicAuth {
				return NewBasicAuth(map[string]string{"alice": "secret"}, "mocker")
			},
			wantOK:  false,
			wantErr: false,
		},
		{
			name:   "correct credentials",
			header: b64("alice", "secret"),
			build: func() *BasicAuth {
				return NewBasicAuth(map[string]string{"alice": "secret"}, "mocker")
			},
			wantOK:  true,
			wantErr: false,
		},
		{
			name:   "password contains colons",
			header: b64("bob", "p:a:ss"),
			build: func() *BasicAuth {
				return NewBasicAuth(map[string]string{"bob": "p:a:ss"}, "mocker")
			},
			wantOK:  true,
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prov := tc.build()
			req := httptest.NewRequest("GET", "http://example.com/x", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			p, ok, err := prov.Authenticate(req)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				if len(tc.wantSub) > 0 && !errx.ErrContainsAll(err, tc.wantSub...) {
					t.Fatalf("error %q does not contain %v", err, tc.wantSub)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
			if ok != tc.wantOK {
				t.Fatalf("auth ok=%v, want %v (principal=%q)", ok, tc.wantOK, p.Name)
			}
		})
	}
}
