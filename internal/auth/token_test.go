// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package auth

import (
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Bl4cky99/mocker/internal/errx"
)

var _ = Describe("TokenAuth", func() {
	DescribeTable("Authenticate",
		func(header string, build func() (*TokenAuth, string), wantOK, wantErr bool, wantSub []string) {
			prov, headerName := build()
			req := httptest.NewRequest("GET", "http://example.com/x", nil)
			if header != "" {
				req.Header.Set(headerName, header)
			}
			_, ok, err := prov.Authenticate(req)

			if wantErr {
				Expect(err).To(HaveOccurred())
				if len(wantSub) > 0 {
					Expect(errx.ErrContainsAll(err, wantSub...)).To(BeTrue())
				}
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
			Expect(ok).To(Equal(wantOK))
		},
		Entry("missing header => not authenticated",
			"",
			func() (*TokenAuth, string) {
				return NewTokenAuth("Authorization", "Bearer ", []string{"devtoken123", "secret2"}), "Authorization"
			},
			false, false, []string(nil),
		),
		Entry("wrong prefix",
			"Token devtoken123",
			func() (*TokenAuth, string) {
				return NewTokenAuth("Authorization", "Bearer ", []string{"devtoken123"}), "Authorization"
			},
			false, false, []string(nil),
		),
		Entry("wrong token with right prefix",
			"Bearer nope",
			func() (*TokenAuth, string) {
				return NewTokenAuth("Authorization", "Bearer ", []string{"devtoken123"}), "Authorization"
			},
			false, false, []string(nil),
		),
		Entry("correct token",
			"Bearer devtoken123",
			func() (*TokenAuth, string) {
				return NewTokenAuth("Authorization", "Bearer ", []string{"devtoken123"}), "Authorization"
			},
			true, false, []string(nil),
		),
		Entry("extra spaces after prefix are trimmed",
			"Bearer   devtoken123   ",
			func() (*TokenAuth, string) {
				return NewTokenAuth("Authorization", "Bearer ", []string{"devtoken123"}), "Authorization"
			},
			true, false, []string(nil),
		),
		Entry("custom header without prefix",
			"t1",
			func() (*TokenAuth, string) {
				return NewTokenAuth("X-Token", "", []string{"t1"}), "X-Token"
			},
			true, false, []string(nil),
		),
	)
})
