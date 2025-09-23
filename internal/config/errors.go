// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package config

import "errors"

var (
	ErrUnsupportedExt = errors.New("unsupported extension")
	ErrDecode         = errors.New("decode error")
	ErrServerConfig   = errors.New("invalid server config")
	ErrAuthConfig     = errors.New("invalid auth config")
	ErrEndpointConfig = errors.New("invalid endpoint config")
	ErrSchemaRef      = errors.New("invalid schema reference")
)
