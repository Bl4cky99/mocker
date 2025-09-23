// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package render

import (
	"encoding/json"
	"html/template"
)

func Funcs() template.FuncMap {
	return template.FuncMap{
		"json": func(v any) (string, error) {
			b, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			return string(b), nil
		},
	}
}
