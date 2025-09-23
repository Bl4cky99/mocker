// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package config

type Config struct {
	Server    ServerConfig `yaml:"server" json:"server"`
	Auth      AuthConfig   `yaml:"auth" json:"auth"`
	Endpoints []Endpoint   `yaml:"endpoints" json:"endpoints"`
}

type ServerConfig struct {
	Addr           string            `yaml:"addr" json:"addr"`
	BasePath       string            `yaml:"basePath" json:"basePath"`
	DefaultHeaders map[string]string `yaml:"defaultHeaders" json:"defaultHeaders"`
	CORS           *CORSConfig       `yaml:"cors,omitempty" json:"cors,omitempty"`
}

type CORSConfig struct {
	Enabled      bool     `yaml:"enabled" json:"enabled"`
	AllowOrigins []string `yaml:"allowOrigins" json:"allowOrigins"`
	AllowMethods []string `yaml:"allowMethods" json:"allowMethods"`
	AllowHeaders []string `yaml:"allowHeaders" json:"allowHeaders"`
}

type AuthConfig struct {
	// "none" | "token" | "basic"
	Type  string           `yaml:"type"  json:"type"`
	Token *TokenAuthConfig `yaml:"token,omitempty" json:"token,omitempty"`
	Basic *BasicAuthConfig `yaml:"basic,omitempty" json:"basic,omitempty"`
}

type TokenAuthConfig struct {
	Header string   `yaml:"header" json:"header"`
	Prefix string   `yaml:"prefix" json:"prefix"`
	Tokens []string `yaml:"tokens" json:"tokens"`
}

type BasicAuthConfig struct {
	Users []BasicUser `yaml:"users" json:"users"`
}
type BasicUser struct {
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
}

type Endpoint struct {
	Method    string            `yaml:"method"    json:"method"`
	Path      string            `yaml:"path"      json:"path"`
	Validate  *ValidateSpec     `yaml:"validate,omitempty" json:"validate,omitempty"`
	Responses []ResponseVariant `yaml:"responses" json:"responses"`
}

type ValidateSpec struct {
	ContentType string `yaml:"contentType" json:"contentType"`
	SchemaFile  string `yaml:"schemaFile"  json:"schemaFile"`
}

type ResponseVariant struct {
	When     *WhenClause       `yaml:"when,omitempty"   json:"when,omitempty"`
	Status   int               `yaml:"status"           json:"status"`
	Headers  map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	Body     string            `yaml:"body,omitempty"    json:"body,omitempty"`
	BodyFile string            `yaml:"bodyFile,omitempty" json:"bodyFile,omitempty"`
	DelayMs  int               `yaml:"delayMs,omitempty" json:"delayMs,omitempty"`
}

type WhenClause struct {
	Query  map[string]string `yaml:"query,omitempty"  json:"query,omitempty"`
	Header map[string]string `yaml:"header,omitempty" json:"header,omitempty"`
}
