// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package render

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Renderer", func() {
	Describe("RenderFile", func() {
		It("caches templates and reloads on file modification", func() {
			dir := GinkgoT().TempDir()
			path := filepath.Join(dir, "test.tmpl")

			Expect(os.WriteFile(path, []byte("hello {{.Name}}"), 0o600)).To(Succeed())

			r := New()

			out, err := r.RenderFile(path, map[string]string{"Name": "world"})
			Expect(err).NotTo(HaveOccurred())
			Expect(string(out)).To(Equal("hello world"))

			time.Sleep(10 * time.Millisecond)

			Expect(os.WriteFile(path, []byte("hi {{.Name}}"), 0o600)).To(Succeed())
			future := time.Now().Add(2 * time.Second)
			Expect(os.Chtimes(path, future, future)).To(Succeed())

			out, err = r.RenderFile(path, map[string]string{"Name": "again"})
			Expect(err).NotTo(HaveOccurred())
			Expect(string(out)).To(Equal("hi again"))
		})
	})

	Describe("RenderString", func() {
		It("renders a template string with data", func() {
			r := New()
			out, err := r.RenderString("value={{.V}}", map[string]int{"V": 42})
			Expect(err).NotTo(HaveOccurred())
			Expect(string(out)).To(Equal("value=42"))
		})

		It("defaults missing map keys to empty string", func() {
			r := New()
			out, err := r.RenderString("{{.Query.page}}", map[string]map[string]string{"Query": {}})
			Expect(err).NotTo(HaveOccurred())
			Expect(string(out)).To(Equal(""))
		})
	})
})
