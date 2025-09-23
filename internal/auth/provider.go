// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package auth

import "net/http"

type Principal struct {
	Name string
}

type Provider interface {
	Authenticate(*http.Request) (Principal, bool, error)
}
