// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package config

import (
	"errors"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Bl4cky99/mocker/internal/errx"
)

func writeTemp(name, content string) string {
	return writeTempAt(GinkgoT().TempDir(), name, content)
}

func writeTempAt(dir, name, content string) string {
	path := filepath.Join(dir, name)
	Expect(os.WriteFile(path, []byte(content), 0o600)).To(Succeed())
	return path
}

func cloneConfig(cfg Config) Config {
	dup := cfg
	if cfg.Endpoints != nil {
		dup.Endpoints = make([]Endpoint, len(cfg.Endpoints))
		for i := range cfg.Endpoints {
			dup.Endpoints[i] = cloneEndpoint(cfg.Endpoints[i])
		}
	}
	return dup
}

func cloneEndpoint(ep Endpoint) Endpoint {
	copyEp := ep
	if ep.Validate != nil {
		val := *ep.Validate
		copyEp.Validate = &val
	}
	if ep.Responses != nil {
		copyEp.Responses = make([]ResponseVariant, len(ep.Responses))
		for i := range ep.Responses {
			copyEp.Responses[i] = cloneResponse(ep.Responses[i])
		}
	}
	return copyEp
}

func cloneResponse(rv ResponseVariant) ResponseVariant {
	copyRv := rv
	if rv.Headers != nil {
		copyRv.Headers = map[string]string{}
		for k, v := range rv.Headers {
			copyRv.Headers[k] = v
		}
	}
	if rv.When != nil {
		wc := *rv.When
		if rv.When.Query != nil {
			wc.Query = map[string]string{}
			for k, v := range rv.When.Query {
				wc.Query[k] = v
			}
		}
		if rv.When.Header != nil {
			wc.Header = map[string]string{}
			for k, v := range rv.When.Header {
				wc.Header[k] = v
			}
		}
		copyRv.When = &wc
	}
	return copyRv
}

var _ = Describe("Load", func() {
	DescribeTable("loads config files",
		func(file string, wantErr bool, wantIs []error, wantSub []string, check func(*Config)) {
			cfg, err := Load(filepath.Join("testdata", file))
			if wantErr {
				Expect(err).To(HaveOccurred())
				for _, sentinel := range wantIs {
					Expect(errors.Is(err, sentinel)).To(BeTrue(),
						"error %q does not wrap sentinel %v", err, sentinel)
				}
				Expect(errx.ErrContainsAll(err, wantSub...)).To(BeTrue(),
					"error %q missing substrings %v", err, wantSub)
			} else {
				Expect(err).NotTo(HaveOccurred())
				if check != nil {
					check(cfg)
				}
			}
		},
		Entry("ok basic yaml", "ok.basic.yaml", false, nil, nil,
			func(c *Config) {
				Expect(c.Server.Addr).To(Equal(":8080"))
				Expect(c.Server.BasePath).To(Equal("/"))
				Expect(c.Server.DefaultHeaders).NotTo(BeNil())
				Expect(c.Auth.Type).NotTo(BeEmpty())
				Expect(c.Endpoints).To(HaveLen(1))
			},
		),
		Entry("ok json", "ok.json", false, nil, nil,
			func(c *Config) {
				Expect(c.Server.Addr).To(Equal(":9090"))
				Expect(c.Server.BasePath).To(Equal("/api"))
				Expect(c.Auth.Type).To(Equal("none"))
				Expect(c.Endpoints).To(HaveLen(1))
				Expect(c.Endpoints[0].Path).To(Equal("/status"))
				Expect(c.Endpoints[0].Method).To(Equal("GET"))
			},
		),
		Entry("bad token auth missing tokens",
			"bad.auth.token.missing.yaml", true,
			[]error{ErrAuthConfig},
			[]string{"auth.token.tokens", "must not be empty"},
			nil,
		),
		Entry("bad endpoint status out of range",
			"bad.endpoint.status.yaml", true,
			[]error{ErrEndpointConfig},
			[]string{"responses[0].status", "out of range"},
			nil,
		),
		Entry("bad endpoint body and bodyFile both set",
			"bad.endpoint.body_both.yaml", true,
			[]error{ErrEndpointConfig},
			[]string(nil),
			nil,
		),
		Entry("bad schema file missing",
			"bad.schema.missing.yaml", true,
			[]error{ErrSchemaRef},
			[]string{"validate.schemaFile", "not found"},
			nil,
		),
		Entry("bad unknown extension",
			"bad.unknown_ext.txt", true,
			[]error{ErrUnsupportedExt},
			[]string(nil),
			nil,
		),
	)

	Context("error cases", func() {
		It("returns error for empty path", func() {
			_, err := Load("")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty config path"))
		})

		It("returns error for missing file", func() {
			_, err := Load(filepath.Join("testdata", "does-not-exist.yaml"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("read"))
		})

		It("returns error for yaml with unknown field", func() {
			p := writeTemp("bad.yaml", "server:\n  invalid: true\n")
			_, err := Load(p)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("yaml decode"))
		})

		It("returns error for json with unknown field", func() {
			p := writeTemp("bad.json", `{"server":{"invalid":true}}`)
			_, err := Load(p)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("json decode"))
		})
	})
})

var _ = Describe("Config.ApplyDefaults", func() {
	It("fills in all default values", func() {
		cfg := Config{
			Endpoints: []Endpoint{{
				Method: "GET",
				Path:   "/",
				Responses: []ResponseVariant{{
					Status: 200,
					Body:   "ok",
				}},
			}},
			Server: ServerConfig{
				CORS: &CORSConfig{Enabled: true},
			},
		}

		cfg.ApplyDefaults()

		Expect(cfg.Server.Addr).To(Equal(":8080"))
		Expect(cfg.Server.BasePath).To(Equal("/"))
		Expect(cfg.Server.DefaultHeaders).NotTo(BeNil())
		Expect(cfg.Server.CORS.AllowMethods).NotTo(BeEmpty())
		Expect(cfg.Server.CORS.AllowHeaders).NotTo(BeNil())
		Expect(cfg.Server.CORS.AllowOrigins).NotTo(BeEmpty())
		Expect(cfg.Auth.Type).To(Equal("none"))
	})
})

var _ = Describe("Config.Validate", func() {
	var (
		shared     string
		bodyFile   string
		schemaFile string
		valid      Config
	)

	BeforeEach(func() {
		shared = GinkgoT().TempDir()
		bodyFile = writeTempAt(shared, "body.json", "{}")
		schemaFile = writeTempAt(shared, "schema.json", "{}")
		valid = Config{
			Server: ServerConfig{BasePath: "/"},
			Auth:   AuthConfig{Type: "none"},
			Endpoints: []Endpoint{{
				Method: "GET",
				Path:   "/ok",
				Validate: &ValidateSpec{
					ContentType: "application/json",
					SchemaFile:  schemaFile,
				},
				Responses: []ResponseVariant{{
					Status:   200,
					BodyFile: bodyFile,
					When:     &WhenClause{Query: map[string]string{"foo": "bar"}},
				}},
			}},
		}
	})

	It("accepts a fully valid config", func() {
		Expect(valid.Validate()).To(Succeed())
	})

	DescribeTable("rejects invalid configs",
		func(makeCfg func() Config, wantSubs []string) {
			cfg := makeCfg()
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
			for _, sub := range wantSubs {
				Expect(err.Error()).To(ContainSubstring(sub))
			}
		},
		Entry("invalid auth type",
			func() Config { c := cloneConfig(valid); c.Auth.Type = "oauth"; return c },
			[]string{"auth.type"},
		),
		Entry("token header empty",
			func() Config {
				c := cloneConfig(valid)
				c.Auth.Type = "token"
				c.Auth.Token = &TokenAuthConfig{Tokens: []string{"a"}}
				return c
			},
			[]string{"auth.token.header"},
		),
		Entry("basic config missing",
			func() Config {
				c := cloneConfig(valid)
				c.Auth.Type = "basic"
				c.Auth.Token = nil
				c.Auth.Basic = nil
				return c
			},
			[]string{"auth.type=basic"},
		),
		Entry("basic user empty",
			func() Config {
				c := cloneConfig(valid)
				c.Auth.Type = "basic"
				c.Auth.Basic = &BasicAuthConfig{Users: []BasicUser{{Username: "", Password: ""}}}
				return c
			},
			[]string{"auth.basic.users[0]"},
		),
		Entry("base path missing leading slash",
			func() Config { c := cloneConfig(valid); c.Server.BasePath = "api"; return c },
			[]string{"server.basePath"},
		),
		Entry("no endpoints",
			func() Config { c := cloneConfig(valid); c.Endpoints = nil; return c },
			[]string{"at least one endpoint"},
		),
		Entry("duplicate endpoint",
			func() Config {
				c := cloneConfig(valid)
				c.Endpoints[0].Responses[0].When = nil
				c.Endpoints = append(c.Endpoints, cloneEndpoint(c.Endpoints[0]))
				return c
			},
			[]string{"duplicate endpoint"},
		),
		Entry("invalid method",
			func() Config { c := cloneConfig(valid); c.Endpoints[0].Method = "TRACE"; return c },
			[]string{"method"},
		),
		Entry("path missing leading slash",
			func() Config { c := cloneConfig(valid); c.Endpoints[0].Path = "status"; return c },
			[]string{"path must"},
		),
		Entry("no responses",
			func() Config { c := cloneConfig(valid); c.Endpoints[0].Responses = nil; return c },
			[]string{"must have at least one response"},
		),
		Entry("body and bodyFile both missing",
			func() Config {
				c := cloneConfig(valid)
				c.Endpoints[0].Responses[0].Body = ""
				c.Endpoints[0].Responses[0].BodyFile = ""
				return c
			},
			[]string{"set exactly one of body or bodyFile"},
		),
		Entry("body file does not exist",
			func() Config {
				c := cloneConfig(valid)
				c.Endpoints[0].Responses[0].BodyFile = filepath.Join(shared, "nope.json")
				return c
			},
			[]string{"bodyFile"},
		),
		Entry("invalid content type",
			func() Config {
				c := cloneConfig(valid)
				c.Endpoints[0].Validate.ContentType = "not/a type"
				return c
			},
			[]string{"contentType"},
		),
	)
})

var _ = Describe("epHasNoWhen", func() {
	It("returns false when at least one response has a when clause", func() {
		ep := Endpoint{Responses: []ResponseVariant{{When: &WhenClause{Header: map[string]string{"x": "1"}}}}}
		Expect(epHasNoWhen(ep)).To(BeFalse())
	})

	It("returns true when no responses have a when clause", func() {
		ep := Endpoint{Responses: []ResponseVariant{{When: nil}}}
		Expect(epHasNoWhen(ep)).To(BeTrue())
	})
})
