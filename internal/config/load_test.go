// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Bl4cky99/mocker/internal/errx"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		wantErr bool
		wantIs  []error
		wantSub []string
		check   func(t *testing.T, cfg *Config)
	}{
		{
			"ok basic yaml",
			"ok.basic.yaml",
			false,
			nil,
			nil,
			func(t *testing.T, c *Config) {
				if c.Server.Addr != ":8080" {
					t.Fatalf("default addr: %v, expected ':8080'", c.Server.Addr)
				}
				if c.Server.BasePath != "/" {
					t.Fatalf("default base-path: %v, expected '/'", c.Server.BasePath)
				}
				if c.Server.DefaultHeaders == nil {
					t.Fatalf("default headers: %v, expected empty map", c.Server.DefaultHeaders)
				}
				if c.Auth.Type == "" {
					t.Fatalf("auth type default not applied")
				}
				if len(c.Endpoints) != 1 {
					t.Fatalf("expected 1 endpoint, got %d", len(c.Endpoints))
				}
			},
		},
		{
			"ok json",
			"ok.json",
			false,
			nil,
			nil,
			func(t *testing.T, c *Config) {
				if c.Server.Addr != ":9090" {
					t.Fatalf("wrong addr: %v, expected ':9090'", c.Server.Addr)
				}
				if c.Server.BasePath != "/api" {
					t.Fatalf("wrong basePath: %v, expected '/api'", c.Server.BasePath)
				}
				if c.Auth.Type != "none" {
					t.Fatalf("wrong auth: %v, expected 'none'", c.Auth.Type)
				}
				if c.Endpoints == nil || len(c.Endpoints) != 1 {
					t.Fatalf("endpoints not parsed correctly")
				}
				ep := c.Endpoints[0]
				if ep.Path != "/status" {
					t.Fatalf("wrong path: %v, expected '/status'", ep.Path)
				}
				if ep.Method != "GET" {
					t.Fatalf("wrong method for %v, expected 'GET'", ep.Path)
				}
			},
		},
		{
			name:    "bad token auth missing tokens",
			file:    "bad.auth.token.missing.yaml",
			wantErr: true,
			wantIs:  []error{ErrAuthConfig},
			wantSub: []string{"auth.token.tokens", "must not be empty"},
		},
		{
			name:    "bad endpoint status out of range",
			file:    "bad.endpoint.status.yaml",
			wantErr: true,
			wantIs:  []error{ErrEndpointConfig},
			wantSub: []string{"responses[0].status", "out of range"},
		},
		{
			name:    "bad endpoint body and bodyFile both set",
			file:    "bad.endpoint.body_both.yaml",
			wantErr: true,
			wantIs:  []error{ErrEndpointConfig},
		},
		{
			name:    "bad schema file missing",
			file:    "bad.schema.missing.yaml",
			wantErr: true,
			wantIs:  []error{ErrSchemaRef},
			wantSub: []string{"validate.schemaFile", "not found"},
		},
		{
			name:    "bad unknown extension",
			file:    "bad.unknown_ext.txt",
			wantErr: true,
			wantIs:  []error{ErrUnsupportedExt},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := Load(filepath.Join("testdata", tc.file))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				if len(tc.wantIs) > 0 && !errx.ErrIsAll(err, tc.wantIs...) {
					t.Fatalf("error %q does not match sentinels %v", err, tc.wantIs)
				}
				if len(tc.wantSub) > 0 && !errx.ErrContainsAll(err, tc.wantSub...) {
					t.Fatalf("error %q does not contain %v", err, tc.wantSub)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tc.check != nil {
					tc.check(t, cfg)
				}
			}
		})
	}
}

func TestLoadErrors(t *testing.T) {
	t.Run("empty path", func(t *testing.T) {
		if _, err := Load(""); err == nil || !strings.Contains(err.Error(), "empty config path") {
			t.Fatalf("expected empty path error, got %v", err)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		if _, err := Load(filepath.Join("testdata", "does-not-exist.yaml")); err == nil || !strings.Contains(err.Error(), "read") {
			t.Fatalf("expected read error, got %v", err)
		}
	})

	t.Run("yaml decode unknown field", func(t *testing.T) {
		p := writeTemp(t, "bad.yaml", "server:\n  invalid: true\n")
		if _, err := Load(p); err == nil || !strings.Contains(err.Error(), "yaml decode") {
			t.Fatalf("expected yaml decode error, got %v", err)
		}
	})

	t.Run("json decode unknown field", func(t *testing.T) {
		p := writeTemp(t, "bad.json", "{\"server\":{\"invalid\":true}}")
		if _, err := Load(p); err == nil || !strings.Contains(err.Error(), "json decode") {
			t.Fatalf("expected json decode error, got %v", err)
		}
	})
}

func TestConfigApplyDefaults(t *testing.T) {
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

	if cfg.Server.Addr != ":8080" {
		t.Fatalf("addr default missing: %s", cfg.Server.Addr)
	}
	if cfg.Server.BasePath != "/" {
		t.Fatalf("base path default missing: %s", cfg.Server.BasePath)
	}
	if cfg.Server.DefaultHeaders == nil {
		t.Fatalf("default headers not initialised")
	}
	if cfg.Server.CORS.AllowMethods == nil || len(cfg.Server.CORS.AllowMethods) == 0 {
		t.Fatalf("cors methods default missing")
	}
	if cfg.Server.CORS.AllowHeaders == nil {
		t.Fatalf("cors headers default missing")
	}
	if cfg.Server.CORS.AllowOrigins == nil || len(cfg.Server.CORS.AllowOrigins) == 0 {
		t.Fatalf("cors origins default missing")
	}
	if cfg.Auth.Type != "none" {
		t.Fatalf("auth type default missing: %s", cfg.Auth.Type)
	}
}

func TestConfigValidate(t *testing.T) {
	shared := t.TempDir()
	bodyFile := writeTempAt(t, shared, "body.json", "{}")
	schemaFile := writeTempAt(t, shared, "schema.json", "{}")

	valid := Config{
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

	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}

	tests := []struct {
		name string
		cfg  Config
		want []string
	}{
		{
			name: "invalid auth type",
			cfg: func() Config {
				c := cloneConfig(valid)
				c.Auth.Type = "oauth"
				return c
			}(),
			want: []string{"auth.type"},
		},
		{
			name: "token header empty",
			cfg: func() Config {
				c := cloneConfig(valid)
				c.Auth.Type = "token"
				c.Auth.Token = &TokenAuthConfig{Tokens: []string{"a"}}
				return c
			}(),
			want: []string{"auth.token.header"},
		},
		{
			name: "basic config missing",
			cfg: func() Config {
				c := cloneConfig(valid)
				c.Auth.Type = "basic"
				c.Auth.Token = nil
				c.Auth.Basic = nil
				return c
			}(),
			want: []string{"auth.type=basic"},
		},
		{
			name: "basic user empty",
			cfg: func() Config {
				c := cloneConfig(valid)
				c.Auth.Type = "basic"
				c.Auth.Basic = &BasicAuthConfig{Users: []BasicUser{{Username: "", Password: ""}}}
				return c
			}(),
			want: []string{"auth.basic.users[0]"},
		},
		{
			name: "base path missing slash",
			cfg: func() Config {
				c := cloneConfig(valid)
				c.Server.BasePath = "api"
				return c
			}(),
			want: []string{"server.basePath"},
		},
		{
			name: "no endpoints",
			cfg: func() Config {
				c := cloneConfig(valid)
				c.Endpoints = nil
				return c
			}(),
			want: []string{"at least one endpoint"},
		},
		{
			name: "duplicate endpoint",
			cfg: func() Config {
				c := cloneConfig(valid)
				c.Endpoints[0].Responses[0].When = nil
				c.Endpoints = append(c.Endpoints, cloneEndpoint(c.Endpoints[0]))
				return c
			}(),
			want: []string{"duplicate endpoint"},
		},
		{
			name: "invalid method",
			cfg: func() Config {
				c := cloneConfig(valid)
				c.Endpoints[0].Method = "TRACE"
				return c
			}(),
			want: []string{"method"},
		},
		{
			name: "path missing slash",
			cfg: func() Config {
				c := cloneConfig(valid)
				c.Endpoints[0].Path = "status"
				return c
			}(),
			want: []string{"path must"},
		},
		{
			name: "no responses",
			cfg: func() Config {
				c := cloneConfig(valid)
				c.Endpoints[0].Responses = nil
				return c
			}(),
			want: []string{"must have at least one response"},
		},
		{
			name: "body missing both",
			cfg: func() Config {
				c := cloneConfig(valid)
				r := &c.Endpoints[0].Responses[0]
				r.Body = ""
				r.BodyFile = ""
				return c
			}(),
			want: []string{"set exactly one of body or bodyFile"},
		},
		{
			name: "body file missing",
			cfg: func() Config {
				c := cloneConfig(valid)
				c.Endpoints[0].Responses[0].BodyFile = filepath.Join(shared, "nope.json")
				return c
			}(),
			want: []string{"bodyFile"},
		},
		{
			name: "invalid content type",
			cfg: func() Config {
				c := cloneConfig(valid)
				c.Endpoints[0].Validate.ContentType = "not/a type"
				return c
			}(),
			want: []string{"contentType"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.cfg.Validate(); err == nil {
				t.Fatalf("expected validation error")
			} else {
				for _, want := range tc.want {
					if !strings.Contains(err.Error(), want) {
						t.Fatalf("error %q missing substring %q", err, want)
					}
				}
			}
		})
	}
}

func TestEpHasNoWhen(t *testing.T) {
	ep := Endpoint{Responses: []ResponseVariant{{When: &WhenClause{Header: map[string]string{"x": "1"}}}}}
	if epHasNoWhen(ep) {
		t.Fatalf("expected epHasNoWhen to be false when when clauses exist")
	}
	ep.Responses[0].When = nil
	if !epHasNoWhen(ep) {
		t.Fatalf("expected epHasNoWhen to be true without when clauses")
	}
}

func writeTemp(t *testing.T, name, content string) string {
	return writeTempAt(t, t.TempDir(), name, content)
}

func writeTempAt(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
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
