// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package errx

import (
	"errors"
	"strings"
	"testing"
)

var (
	errFoo = errors.New("foo")
	errBar = errors.New("bar")
)

func TestCollectorErr(t *testing.T) {
	c := New()
	if err := c.Err(); err != nil {
		t.Fatalf("expected nil when empty")
	}

	c.Add(nil)
	c.Add(errors.New("first"))
	c.Wrap(errFoo, "wrapped")
	c.Wrapf(errBar, "value %d", 42)
	c.If(true, errFoo, "conditional %s", "hit")
	c.If(false, errBar, "should not add")

	err := c.Err()
	if err == nil {
		t.Fatalf("collector should aggregate errors")
	}
	if !ErrIsAll(err, errFoo, errBar) {
		t.Fatalf("ErrIsAll failed: %v", err)
	}
	if !ErrContainsAll(err, "wrapped", "value 42", "conditional") {
		t.Fatalf("ErrContainsAll missing text: %v", err)
	}
}

func TestScope(t *testing.T) {
	c := New()
	s := c.Scope("endpoint[0]")
	s.Wrapf(errFoo, "bad %s", "value")
	s.If(true, errBar, "missing %s", "field")

	err := c.Err()
	if !ErrIsAll(err, errFoo, errBar) {
		t.Fatalf("scoped errors missing sentinels: %v", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "endpoint[0]") {
		t.Fatalf("expected scope prefix in %q", msg)
	}
}

func TestErrContainsAllNil(t *testing.T) {
	if ErrContainsAll(nil, "anything") {
		t.Fatalf("nil err should not satisfy contains")
	}
}
