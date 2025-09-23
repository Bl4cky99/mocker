// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package validate

import "errors"

var (
	ErrEmptySchema      = errors.New("empty schema path")
	ErrAbsPath          = errors.New("absolute path")
	ErrSchemaCompile    = errors.New("schema compile")
	ErrUnmarshalJSON    = errors.New("unmarshal instance")
	ErrSchemaValidation = errors.New("schema validation")
)
