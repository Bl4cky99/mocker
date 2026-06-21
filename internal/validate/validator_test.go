// SPDX-License-Identifier: MIT
// Copyright (c) 2025-2026 Jason Giese (Bl4cky99)

package validate

import (
	"errors"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func writeFile(dir, name, contents string) string {
	path := filepath.Join(dir, name)
	Expect(os.WriteFile(path, []byte(contents), 0o600)).To(Succeed())
	return path
}

var _ = Describe("CompileSchema", func() {
	It("returns ErrEmptySchema for an empty path", func() {
		_, err := CompileSchema("", JSONSchemaValidatorOptions{})
		Expect(errors.Is(err, ErrEmptySchema)).To(BeTrue())
	})

	It("returns ErrSchemaCompile for invalid JSON", func() {
		dir := GinkgoT().TempDir()
		bad := writeFile(dir, "bad.json", "{not json}")
		_, err := CompileSchema(bad, JSONSchemaValidatorOptions{})
		Expect(errors.Is(err, ErrSchemaCompile)).To(BeTrue())
	})

	Context("with a valid schema", func() {
		var v *JSONSchemaValidator

		BeforeEach(func() {
			dir := GinkgoT().TempDir()
			schema := writeFile(dir, "schema.json", `{"type":"object","required":["name"],"properties":{"name":{"type":"string"}}}`)
			var err error
			v, err = CompileSchema(schema, JSONSchemaValidatorOptions{AssertFormat: true, DefaultDraft: jsonschema.Draft2020})
			Expect(err).NotTo(HaveOccurred())
		})

		It("accepts a valid instance", func() {
			Expect(v.Validate([]byte(`{"name":"ok"}`))).To(Succeed())
		})

		It("returns ErrUnmarshalJSON for non-JSON input", func() {
			Expect(errors.Is(v.Validate([]byte("not json")), ErrUnmarshalJSON)).To(BeTrue())
		})

		It("returns ErrSchemaValidation for a schema-violating instance", func() {
			Expect(errors.Is(v.Validate([]byte(`{"id":1}`)), ErrSchemaValidation)).To(BeTrue())
		})
	})
})
