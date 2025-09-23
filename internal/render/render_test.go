// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package render

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRenderFileCachesAndReloads(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.tmpl")

	if err := os.WriteFile(path, []byte("hello {{.Name}}"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}

	r := New()

	out, err := r.RenderFile(path, map[string]string{"Name": "world"})
	if err != nil {
		t.Fatalf("first render: %v", err)
	}
	if string(out) != "hello world" {
		t.Fatalf("unexpected output: %s", out)
	}

	time.Sleep(10 * time.Millisecond)

	if err := os.WriteFile(path, []byte("hi {{.Name}}"), 0o600); err != nil {
		t.Fatalf("rewrite template: %v", err)
	}

	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	out, err = r.RenderFile(path, map[string]string{"Name": "again"})
	if err != nil {
		t.Fatalf("second render: %v", err)
	}
	if string(out) != "hi again" {
		t.Fatalf("expected updated template, got: %s", out)
	}
}

func TestRenderString(t *testing.T) {
	r := New()

	out, err := r.RenderString("value={{.V}}", map[string]int{"V": 42})
	if err != nil {
		t.Fatalf("render string: %v", err)
	}
	if string(out) != "value=42" {
		t.Fatalf("unexpected inline render: %s", out)
	}
}

func TestRenderStringMissingKeyDefaults(t *testing.T) {
	r := New()

	out, err := r.RenderString("{{.Query.page}}", map[string]map[string]string{"Query": {}})
	if err != nil {
		t.Fatalf("render string missing key: %v", err)
	}
	if string(out) != "" {
		t.Fatalf("expected empty string for missing key, got %q", out)
	}
}
