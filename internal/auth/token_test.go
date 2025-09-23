// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package auth

import (
	"net/http/httptest"
	"testing"

	"github.com/Bl4cky99/mocker/internal/errx"
)

func TestTokenAuth(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		build   func() (*TokenAuth, string)
		wantOK  bool
		wantErr bool
		wantSub []string
	}{
		{
			name:   "missing header => not authenticated",
			header: "",
			build: func() (*TokenAuth, string) {
				return NewTokenAuth("Authorization", "Bearer ", []string{"devtoken123", "secret2"}), "Authorization"
			},
			wantOK:  false,
			wantErr: false,
		},
		{
			name:   "wrong prefix",
			header: "Token devtoken123",
			build: func() (*TokenAuth, string) {
				return NewTokenAuth("Authorization", "Bearer ", []string{"devtoken123"}), "Authorization"
			},
			wantOK:  false,
			wantErr: false,
		},
		{
			name:   "wrong token with right prefix",
			header: "Bearer nope",
			build: func() (*TokenAuth, string) {
				return NewTokenAuth("Authorization", "Bearer ", []string{"devtoken123"}), "Authorization"
			},
			wantOK:  false,
			wantErr: false,
		},
		{
			name:   "correct token",
			header: "Bearer devtoken123",
			build: func() (*TokenAuth, string) {
				return NewTokenAuth("Authorization", "Bearer ", []string{"devtoken123"}), "Authorization"
			},
			wantOK:  true,
			wantErr: false,
		},
		{
			name:   "extra spaces after prefix are trimmed",
			header: "Bearer   devtoken123   ",
			build: func() (*TokenAuth, string) {
				return NewTokenAuth("Authorization", "Bearer ", []string{"devtoken123"}), "Authorization"
			},
			wantOK:  true,
			wantErr: false,
		},
		{
			name:   "custom header without prefix",
			header: "t1",
			build: func() (*TokenAuth, string) {
				return NewTokenAuth("X-Token", "", []string{"t1"}), "X-Token"
			},
			wantOK:  true,
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prov, headerName := tc.build()
			req := httptest.NewRequest("GET", "http://example.com/x", nil)
			if tc.header != "" {
				req.Header.Set(headerName, tc.header)
			}
			_, ok, err := prov.Authenticate(req)

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
				t.Fatalf("auth ok=%v, want %v", ok, tc.wantOK)
			}
		})
	}
}
