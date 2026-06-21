// SPDX-License-Identifier: MIT
// Copyright (c) 2025-2026 Jason Giese (Bl4cky99)

package cli

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCli(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CLI Suite")
}
