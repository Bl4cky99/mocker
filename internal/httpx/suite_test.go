// SPDX-License-Identifier: MIT
// Copyright (c) 2025-2026 Jason Giese (Bl4cky99)

package httpx

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHttpx(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Httpx Suite")
}
