// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package errx

import (
	"errors"
	"strings"
)

func ErrIsAll(err error, sentinels ...error) bool {
	for _, s := range sentinels {
		if !errors.Is(err, s) {
			return false
		}
	}
	return true
}

func ErrContainsAll(err error, subs ...string) bool {
	if err == nil {
		return false
	}

	msg := err.Error()
	for _, s := range subs {
		if !strings.Contains(msg, s) {
			return false
		}
	}

	return true
}
