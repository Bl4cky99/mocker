// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/Bl4cky99/mocker/internal/errx"
	"gopkg.in/yaml.v3"
)

func Load(path string) (*Config, error) {
	if path == "" {
		return nil, fmt.Errorf("empty config path")
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", path, err)
	}

	var cfg Config
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".yml", ".yaml":
		dec := yaml.NewDecoder(bytes.NewReader(b))
		dec.KnownFields(true)
		if err := dec.Decode(&cfg); err != nil {
			return nil, fmt.Errorf("%w: yaml decode %q: %v", ErrDecode, path, err)
		}
	case ".json":
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&cfg); err != nil {
			return nil, fmt.Errorf("%w: json decode %q: %v", ErrDecode, path, err)
		}
	default:
		return nil, fmt.Errorf("%w: %q (use .yaml, .yml or .json)", ErrUnsupportedExt, ext)
	}

	cfg.ApplyDefaults()

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config %q: %w", path, err)
	}

	return &cfg, nil
}

func (c *Config) ApplyDefaults() {
	if c.Server.Addr == "" {
		c.Server.Addr = ":8080"
	}

	if c.Server.BasePath == "" {
		c.Server.BasePath = "/"
	}

	if c.Server.DefaultHeaders == nil {
		c.Server.DefaultHeaders = map[string]string{}
	}

	if c.Server.CORS != nil && c.Server.CORS.Enabled {
		if c.Server.CORS.AllowMethods == nil || len(c.Server.CORS.AllowMethods) == 0 {
			c.Server.CORS.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
		}

		if c.Server.CORS.AllowHeaders == nil {
			c.Server.CORS.AllowHeaders = []string{}
		}

		if c.Server.CORS.AllowOrigins == nil {
			c.Server.CORS.AllowOrigins = []string{"127.0.0.1", "localhost"}
		}
	}

	if c.Auth.Type == "" {
		c.Auth.Type = "none"
	}
}

func (c *Config) Validate() error {
	e := errx.New()

	switch c.Auth.Type {
	case "none":
	case "token":
		if c.Auth.Token == nil {
			e.Wrap(ErrAuthConfig, "auth.type=token but token config missing")
		} else {
			e.If(strings.TrimSpace(c.Auth.Token.Header) == "", ErrAuthConfig, "auth.token.header must not be empty")
			e.If(len(c.Auth.Token.Tokens) == 0, ErrAuthConfig, "auth.token.tokens must not be empty")
		}
	case "basic":
		if c.Auth.Basic == nil {
			e.Wrap(ErrAuthConfig, "auth.type=basic but basic config missing")
		} else {
			e.If(len(c.Auth.Basic.Users) == 0, ErrAuthConfig, "auth.basic.users must not be empty")
			for i, u := range c.Auth.Basic.Users {
				e.If(u.Username == "" || u.Password == "", ErrAuthConfig, "auth.basic.users[%d] requires username and password", i)
			}
		}
	default:
		e.Wrapf(ErrAuthConfig, "auth.type %q invalid (use none|token|basic)", c.Auth.Type)
	}

	e.If(!strings.HasPrefix(c.Server.BasePath, "/"), ErrServerConfig, "server.basePath must start with '/'")
	e.If(len(c.Endpoints) == 0, ErrEndpointConfig, "at least one endpoint required")

	seen := map[string]struct{}{}
	for i, ep := range c.Endpoints {
		scope := fmt.Sprintf("endpoints[%d]", i)

		e.If(!isHTTPMethod(ep.Method), ErrEndpointConfig, "%s.method %q invalid", scope, ep.Method)
		e.If(!strings.HasPrefix(ep.Path, "/"), ErrEndpointConfig, "%s.path must start with '/'", scope)
		e.If(len(ep.Responses) == 0, ErrEndpointConfig, "%s must have at least one response variant", scope)

		if epHasNoWhen(ep) {
			key := strings.ToUpper(ep.Method) + " " + ep.Path
			if _, ok := seen[key]; ok {
				e.Wrapf(ErrEndpointConfig, "%s duplicate endpoint without 'when': %s", scope, key)
			}
			seen[key] = struct{}{}
		}

		if ep.Validate != nil {
			if ep.Validate.ContentType != "" && !validContentType(ep.Validate.ContentType) {
				e.Wrapf(ErrEndpointConfig, "%s.validate.contentType %q invalid", scope, ep.Validate.ContentType)
			}
			if ep.Validate.SchemaFile != "" && !fileExists(ep.Validate.SchemaFile) {
				e.Wrapf(ErrSchemaRef, "%s.validate.schemaFile %q not found", scope, ep.Validate.SchemaFile)
			}
		}

		for j, rv := range ep.Responses {
			rscope := fmt.Sprintf("%s.responses[%d]", scope, j)
			e.If(rv.Status < 100 || rv.Status > 599, ErrEndpointConfig, "%s.status %d out of range", rscope, rv.Status)

			both := (rv.Body != "" && rv.BodyFile != "") || (rv.Body == "" && rv.BodyFile == "")
			e.If(both, ErrEndpointConfig, "%s: set exactly one of body or bodyFile", rscope)

			if rv.BodyFile != "" && !fileExists(rv.BodyFile) {
				e.Wrapf(ErrEndpointConfig, "%s.bodyFile %q not found", rscope, rv.BodyFile)
			}
		}
	}

	return e.Err()
}

func isHTTPMethod(s string) bool {
	switch strings.ToUpper(s) {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS":
		return true
	default:
		return false
	}
}

func validContentType(ct string) bool {
	_, _, err := mime.ParseMediaType(ct)
	return err == nil
}

func fileExists(p string) bool {
	if p == "" {
		return false
	}

	if _, err := os.Stat(p); err != nil {
		return false
	}

	return true
}

func epHasNoWhen(ep Endpoint) bool {
	for _, r := range ep.Responses {
		if r.When != nil && (len(r.When.Query) > 0 || len(r.When.Header) > 0) {
			return false
		}
	}

	return true
}
