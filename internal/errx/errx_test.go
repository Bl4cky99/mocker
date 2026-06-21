// SPDX-License-Identifier: MIT
// Copyright (c) 2025-2026 Jason Giese (Bl4cky99)

package errx

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	errFoo = errors.New("foo")
	errBar = errors.New("bar")
)

var _ = Describe("Collector", func() {
	It("returns nil when empty", func() {
		Expect(New().Err()).To(BeNil())
	})

	It("aggregates errors with correct sentinels and substrings", func() {
		c := New()
		c.Add(nil)
		c.Add(errors.New("first"))
		c.Wrap(errFoo, "wrapped")
		c.Wrapf(errBar, "value %d", 42)
		c.If(true, errFoo, "conditional %s", "hit")
		c.If(false, errBar, "should not add")

		err := c.Err()
		Expect(err).To(HaveOccurred())
		Expect(ErrIsAll(err, errFoo, errBar)).To(BeTrue())
		Expect(ErrContainsAll(err, "wrapped", "value 42", "conditional")).To(BeTrue())
	})

	It("prefixes errors with scope name", func() {
		c := New()
		s := c.Scope("endpoint[0]")
		s.Wrapf(errFoo, "bad %s", "value")
		s.If(true, errBar, "missing %s", "field")

		err := c.Err()
		Expect(ErrIsAll(err, errFoo, errBar)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring("endpoint[0]"))
	})
})

var _ = Describe("ErrContainsAll", func() {
	It("returns false for nil error", func() {
		Expect(ErrContainsAll(nil, "anything")).To(BeFalse())
	})
})
