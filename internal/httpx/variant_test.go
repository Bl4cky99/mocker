// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Bl4cky99/mocker/internal/config"
)

func TestPickVariantMatchesQuery(t *testing.T) {
	ep := config.Endpoint{Responses: []config.ResponseVariant{
		{Status: 200, Body: "default"},
		{Status: 201, Body: "query", When: &config.WhenClause{Query: map[string]string{"foo": "bar"}}},
		{Status: 202, Body: "other"},
	}}

	req := httptest.NewRequest(http.MethodGet, "/test?foo=bar", nil)

	v := pickVariant(ep, req)
	if v.Status != 201 || v.Body != "query" {
		t.Fatalf("expected query variant, got status=%d body=%q", v.Status, v.Body)
	}
}

func TestPickVariantMatchesHeader(t *testing.T) {
	ep := config.Endpoint{Responses: []config.ResponseVariant{
		{Status: 200, Body: "default"},
		{Status: 202, Body: "header", When: &config.WhenClause{Header: map[string]string{"X-Test": "yes"}}},
	}}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Test", "yes")

	v := pickVariant(ep, req)
	if v.Status != 202 || v.Body != "header" {
		t.Fatalf("expected header variant, got status=%d body=%q", v.Status, v.Body)
	}
}

func TestPickVariantFallsBack(t *testing.T) {
	ep := config.Endpoint{Responses: []config.ResponseVariant{
		{Status: 200, Body: "first", When: nil},
		{Status: 201, Body: "second", When: &config.WhenClause{Query: map[string]string{"foo": "no"}}},
	}}

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	v := pickVariant(ep, req)
	if v.Status != 200 || v.Body != "first" {
		t.Fatalf("expected fallback variant, got status=%d body=%q", v.Status, v.Body)
	}
}

func TestPickVariantNoFallbackReturnsFirst(t *testing.T) {
	ep := config.Endpoint{Responses: []config.ResponseVariant{
		{Status: 200, Body: "first", When: &config.WhenClause{Query: map[string]string{"a": "1"}}},
		{Status: 201, Body: "second", When: &config.WhenClause{Header: map[string]string{"X": "y"}}},
	}}

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	v := pickVariant(ep, req)
	if v.Status != 200 || v.Body != "first" {
		t.Fatalf("expected first response when no match, got status=%d body=%q", v.Status, v.Body)
	}
}

func TestWhenMatches(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?foo=bar", nil)
	req.Header.Set("X-Test", "yes")

	clause := &config.WhenClause{Query: map[string]string{"foo": "bar"}, Header: map[string]string{"X-Test": "yes"}}
	if !whenMatches(req, clause) {
		t.Fatalf("expected when clause to match")
	}

	clause.Query["foo"] = "nope"
	if whenMatches(req, clause) {
		t.Fatalf("expected query mismatch to fail")
	}

	clause.Query["foo"] = "bar"
	clause.Header["X-Test"] = "nope"
	if whenMatches(req, clause) {
		t.Fatalf("expected header mismatch to fail")
	}
}
