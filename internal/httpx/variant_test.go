// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package httpx

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Bl4cky99/mocker/internal/config"
)

var _ = Describe("pickVariant", func() {
	It("selects the variant whose query params match", func() {
		ep := config.Endpoint{Responses: []config.ResponseVariant{
			{Status: 200, Body: "default"},
			{Status: 201, Body: "query", When: &config.WhenClause{Query: map[string]string{"foo": "bar"}}},
			{Status: 202, Body: "other"},
		}}
		req := httptest.NewRequest(http.MethodGet, "/test?foo=bar", nil)

		v := pickVariant(ep, req)
		Expect(v.Status).To(Equal(201))
		Expect(v.Body).To(Equal("query"))
	})

	It("selects the variant whose headers match", func() {
		ep := config.Endpoint{Responses: []config.ResponseVariant{
			{Status: 200, Body: "default"},
			{Status: 202, Body: "header", When: &config.WhenClause{Header: map[string]string{"X-Test": "yes"}}},
		}}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Test", "yes")

		v := pickVariant(ep, req)
		Expect(v.Status).To(Equal(202))
		Expect(v.Body).To(Equal("header"))
	})

	It("falls back to the first variant without a when clause", func() {
		ep := config.Endpoint{Responses: []config.ResponseVariant{
			{Status: 200, Body: "first", When: nil},
			{Status: 201, Body: "second", When: &config.WhenClause{Query: map[string]string{"foo": "no"}}},
		}}
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		v := pickVariant(ep, req)
		Expect(v.Status).To(Equal(200))
		Expect(v.Body).To(Equal("first"))
	})

	It("returns the first variant when nothing matches and there is no fallback", func() {
		ep := config.Endpoint{Responses: []config.ResponseVariant{
			{Status: 200, Body: "first", When: &config.WhenClause{Query: map[string]string{"a": "1"}}},
			{Status: 201, Body: "second", When: &config.WhenClause{Header: map[string]string{"X": "y"}}},
		}}
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		v := pickVariant(ep, req)
		Expect(v.Status).To(Equal(200))
		Expect(v.Body).To(Equal("first"))
	})
})

var _ = Describe("whenMatches", func() {
	It("returns true when both query and header conditions match", func() {
		req := httptest.NewRequest(http.MethodGet, "/?foo=bar", nil)
		req.Header.Set("X-Test", "yes")
		clause := &config.WhenClause{
			Query:  map[string]string{"foo": "bar"},
			Header: map[string]string{"X-Test": "yes"},
		}
		Expect(whenMatches(req, clause)).To(BeTrue())
	})

	It("returns false when query value does not match", func() {
		req := httptest.NewRequest(http.MethodGet, "/?foo=bar", nil)
		req.Header.Set("X-Test", "yes")
		clause := &config.WhenClause{
			Query:  map[string]string{"foo": "nope"},
			Header: map[string]string{"X-Test": "yes"},
		}
		Expect(whenMatches(req, clause)).To(BeFalse())
	})

	It("returns false when header value does not match", func() {
		req := httptest.NewRequest(http.MethodGet, "/?foo=bar", nil)
		req.Header.Set("X-Test", "yes")
		clause := &config.WhenClause{
			Query:  map[string]string{"foo": "bar"},
			Header: map[string]string{"X-Test": "nope"},
		}
		Expect(whenMatches(req, clause)).To(BeFalse())
	})
})
