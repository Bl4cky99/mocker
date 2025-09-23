// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package validate

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func writeFile(t *testing.T, dir, name, contents string) string {
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	return path
}

func TestCompileSchemaErrors(t *testing.T) {
	if _, err := CompileSchema("", JSONSchemaValidatorOptions{}); !errors.Is(err, ErrEmptySchema) {
		t.Fatalf("expected empty schema error, got %v", err)
	}

	dir := t.TempDir()
	bad := writeFile(t, dir, "bad.json", "{not json}")

	if _, err := CompileSchema(bad, JSONSchemaValidatorOptions{}); !errors.Is(err, ErrSchemaCompile) {
		t.Fatalf("expected compile error, got %v", err)
	}
}

func TestCompileSchemaSuccessAndValidate(t *testing.T) {
	dir := t.TempDir()
	schema := writeFile(t, dir, "schema.json", `{"type":"object","required":["name"],"properties":{"name":{"type":"string"}}}`)

	v, err := CompileSchema(schema, JSONSchemaValidatorOptions{AssertFormat: true, DefaultDraft: jsonschema.Draft2020})
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}

	if err := v.Validate([]byte(`{"name":"ok"}`)); err != nil {
		t.Fatalf("expected valid instance, got %v", err)
	}

	if err := v.Validate([]byte("not json")); !errors.Is(err, ErrUnmarshalJSON) {
		t.Fatalf("expected unmarshal error, got %v", err)
	}

	if err := v.Validate([]byte(`{"id":1}`)); !errors.Is(err, ErrSchemaValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
