// SPDX-License-Identifier: MIT
// Copyright (c) 2025-2026 Jason Giese (Bl4cky99)

package auth

import (
	"encoding/base64"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Bl4cky99/mocker/internal/errx"
)

func b64(user, pass string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))
}

var _ = Describe("BasicAuth", func() {
	DescribeTable("Authenticate",
		func(header string, build func() *BasicAuth, wantOK, wantErr bool, wantSub []string) {
			prov := build()
			req := httptest.NewRequest("GET", "http://example.com/x", nil)
			if header != "" {
				req.Header.Set("Authorization", header)
			}
			p, ok, err := prov.Authenticate(req)

			if wantErr {
				Expect(err).To(HaveOccurred())
				if len(wantSub) > 0 {
					Expect(errx.ErrContainsAll(err, wantSub...)).To(BeTrue(),
						"error %q does not contain %v", err, wantSub)
				}
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
			Expect(ok).To(Equal(wantOK), "principal=%q", p.Name)
		},
		Entry("missing header => not authenticated",
			"",
			func() *BasicAuth { return NewBasicAuth(map[string]string{"alice": "secret"}, "mocker") },
			false, false, []string(nil),
		),
		Entry("wrong scheme",
			"Bearer something",
			func() *BasicAuth { return NewBasicAuth(map[string]string{"alice": "secret"}, "mocker") },
			false, false, []string(nil),
		),
		Entry("invalid base64 => error",
			"Basic !!!notbase64!!!",
			func() *BasicAuth { return NewBasicAuth(map[string]string{"alice": "secret"}, "mocker") },
			false, true, []string(nil),
		),
		Entry("decoded without colon => error",
			"Basic "+base64.StdEncoding.EncodeToString([]byte("justuser")),
			func() *BasicAuth { return NewBasicAuth(map[string]string{"alice": "secret"}, "mocker") },
			false, true, []string{"invalid basic token"},
		),
		Entry("unknown user",
			b64("mallory", "secret"),
			func() *BasicAuth { return NewBasicAuth(map[string]string{"alice": "secret"}, "mocker") },
			false, false, []string(nil),
		),
		Entry("wrong password",
			b64("alice", "nope"),
			func() *BasicAuth { return NewBasicAuth(map[string]string{"alice": "secret"}, "mocker") },
			false, false, []string(nil),
		),
		Entry("correct credentials",
			b64("alice", "secret"),
			func() *BasicAuth { return NewBasicAuth(map[string]string{"alice": "secret"}, "mocker") },
			true, false, []string(nil),
		),
		Entry("password contains colons",
			b64("bob", "p:a:ss"),
			func() *BasicAuth { return NewBasicAuth(map[string]string{"bob": "p:a:ss"}, "mocker") },
			true, false, []string(nil),
		),
	)
})
