// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package errx

import (
	"errors"
	"fmt"
)

type Collector struct {
	errs []error
}

func New() *Collector {
	return &Collector{errs: nil}
}

func (c *Collector) Add(err error) {
	if err != nil {
		c.errs = append(c.errs, err)
	}
}

func (c *Collector) Wrap(sentinel error, msg string) {
	c.errs = append(c.errs, fmt.Errorf("%w, %s", sentinel, msg))
}

func (c *Collector) Wrapf(sentinel error, format string, args ...any) {
	c.errs = append(c.errs, fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...))
}

func (c *Collector) If(cond bool, sentinel error, format string, args ...any) {
	if cond {
		c.Wrapf(sentinel, format, args...)
	}
}

func (c *Collector) Err() error {
	if len(c.errs) == 0 {
		return nil
	}

	return errors.Join(c.errs...)
}

type Scope struct {
	c      *Collector
	prefix string
}

func (c *Collector) Scope(prefix string) *Scope {
	return &Scope{c: c, prefix: prefix}
}

func (s *Scope) Wrapf(sentinel error, format string, args ...any) {
	s.c.Wrapf(sentinel, s.prefix+": "+format, args...)
}

func (s *Scope) If(cond bool, sentinel error, format string, args ...any) {
	if cond {
		s.Wrapf(sentinel, format, args...)
	}
}
