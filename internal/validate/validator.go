// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package validate

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

type JSONSchemaValidator struct {
	schema *jsonschema.Schema
}

type JSONSchemaValidatorOptions struct {
	AssertFormat  bool
	AssertContent bool
	DefaultDraft  *jsonschema.Draft
}

func CompileSchema(schemaPath string, opt JSONSchemaValidatorOptions) (*JSONSchemaValidator, error) {
	if schemaPath == "" {
		return nil, ErrEmptySchema
	}

	abs, err := filepath.Abs(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAbsPath, err)
	}

	fileURL := "file://" + filepath.ToSlash(abs)

	c := jsonschema.NewCompiler()
	if opt.DefaultDraft != nil {
		c.DefaultDraft(opt.DefaultDraft)
	}
	if opt.AssertFormat {
		c.AssertFormat()
	}
	if opt.AssertContent {
		c.AssertContent()
	}

	sch, err := c.Compile(fileURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrSchemaCompile, fileURL, err)
	}

	return &JSONSchemaValidator{schema: sch}, nil
}

func (v *JSONSchemaValidator) Validate(body []byte) error {
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnmarshalJSON, err)
	}
	if err := v.schema.Validate(inst); err != nil {
		return fmt.Errorf("%w: %w", ErrSchemaValidation, err)
	}
	return nil
}
