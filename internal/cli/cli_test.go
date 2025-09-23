// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Bl4cky99/mocker/internal/config"
	"github.com/Bl4cky99/mocker/internal/httpx"
)

func capture(t *testing.T, target **os.File, fn func()) string {
	old := *target
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	*target = w
	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()

	_ = w.Close()
	*target = old
	out := <-done
	_ = r.Close()
	return out
}

func TestExecuteUsageNoArgs(t *testing.T) {
	old := os.Args
	os.Args = []string{"mocker"}
	defer func() { os.Args = old }()

	var code int
	stderr := capture(t, &os.Stderr, func() {
		code = Execute("1.0.0", "deadbeef", "today")
	})

	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	if !strings.Contains(stderr, "Usage:") {
		t.Fatalf("stderr missing usage, got %q", stderr)
	}
}

func TestExecuteHelp(t *testing.T) {
	old := os.Args
	os.Args = []string{"mocker", "--help"}
	defer func() { os.Args = old }()

	stdout := capture(t, &os.Stdout, func() {
		if code := Execute("1.0.0", "deadbeef", "today"); code != 0 {
			t.Fatalf("unexpected exit %d", code)
		}
	})

	if !strings.Contains(stdout, "Commands:") {
		t.Fatalf("help output missing commands: %q", stdout)
	}
}

func TestExecuteVersion(t *testing.T) {
	old := os.Args
	os.Args = []string{"mocker", "version"}
	defer func() { os.Args = old }()

	stdout := capture(t, &os.Stdout, func() {
		if code := Execute("1.2.3", "cafebabe", "yesterday"); code != 0 {
			t.Fatalf("unexpected exit %d", code)
		}
	})

	if !strings.Contains(stdout, "1.2.3") || !strings.Contains(stdout, "cafebabe") {
		t.Fatalf("version output missing data: %q", stdout)
	}
}

func TestExecuteServeDelegates(t *testing.T) {
	oldArgs := os.Args
	os.Args = []string{"mocker", "serve", "--flag"}
	defer func() { os.Args = oldArgs }()

	prev := runServer
	defer func() { runServer = prev }()

	var gotVersion, gotCommit, gotDate string
	var gotArgs []string
	runServer = func(version, commit, date string, args []string) int {
		gotVersion, gotCommit, gotDate = version, commit, date
		gotArgs = append([]string(nil), args...)
		return 42
	}

	code := Execute("v", "c", "d")
	if code != 42 {
		t.Fatalf("expected delegated exit 42, got %d", code)
	}
	if gotVersion != "v" || gotCommit != "c" || gotDate != "d" {
		t.Fatalf("unexpected version info: %s %s %s", gotVersion, gotCommit, gotDate)
	}
	if !reflect.DeepEqual(gotArgs, []string{"--flag"}) {
		t.Fatalf("args mismatch: %#v", gotArgs)
	}
}

func TestExecuteValidateDelegates(t *testing.T) {
	oldArgs := os.Args
	os.Args = []string{"mocker", "validate", "--config"}
	defer func() { os.Args = oldArgs }()

	prev := runValidate
	defer func() { runValidate = prev }()

	var gotArgs []string
	runValidate = func(args []string) int {
		gotArgs = append([]string(nil), args...)
		return 7
	}

	code := Execute("v", "c", "d")
	if code != 7 {
		t.Fatalf("expected exit 7, got %d", code)
	}
	if !reflect.DeepEqual(gotArgs, []string{"--config"}) {
		t.Fatalf("args mismatch: %#v", gotArgs)
	}
}

func TestExecuteUnknown(t *testing.T) {
	old := os.Args
	os.Args = []string{"mocker", "mystery"}
	defer func() { os.Args = old }()

	var code int
	stderr := capture(t, &os.Stderr, func() {
		code = Execute("v", "c", "d")
	})

	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Fatalf("stderr missing unknown command message: %q", stderr)
	}
}

func TestCmdServerFlagParseError(t *testing.T) {
	stderr := capture(t, &os.Stderr, func() {
		if code := cmdServer("v", "c", "d", []string{"--nope"}); code != 2 {
			t.Fatalf("expected exit 2, got %d", code)
		}
	})

	if !strings.Contains(stderr, "flag provided") {
		t.Fatalf("expected flag error, got %q", stderr)
	}
}

func TestCmdServerLoadConfigError(t *testing.T) {
	prevLoad := loadConfig
	loadConfig = func(string) (*config.Config, error) { return nil, errors.New("boom") }
	defer func() { loadConfig = prevLoad }()

	prevNotify := notifyContext
	notifyContext = func(ctx context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		return context.WithCancel(ctx)
	}
	defer func() { notifyContext = prevNotify }()

	stdout := capture(t, &os.Stdout, func() {
		if code := cmdServer("v", "c", "d", []string{"--config", "cfg"}); code != 1 {
			t.Fatalf("expected exit 1")
		}
	})

	if !strings.Contains(stdout, "load config") {
		t.Fatalf("missing load config error: %q", stdout)
	}
}

type fakeServer struct {
	cancel        context.CancelFunc
	listenErr     error
	shutdownErr   error
	listenCount   int
	shutdownCount int
	shutdownCtx   context.Context
}

func (f *fakeServer) ListenAndServe() error {
	f.listenCount++
	if f.cancel != nil {
		f.cancel()
	}
	return f.listenErr
}

func (f *fakeServer) Shutdown(ctx context.Context) error {
	f.shutdownCount++
	f.shutdownCtx = ctx
	return f.shutdownErr
}

func TestCmdServerSuccess(t *testing.T) {
	prevLoad := loadConfig
	cfg := &config.Config{
		Server: config.ServerConfig{Addr: ":8080", BasePath: "/api"},
		Auth:   config.AuthConfig{Type: "token", Token: &config.TokenAuthConfig{Header: "X", Prefix: "Bearer", Tokens: []string{"tok"}}},
	}
	loadConfig = func(path string) (*config.Config, error) {
		if path != "cfg.yaml" {
			t.Fatalf("unexpected config path %q", path)
		}
		clone := *cfg
		clone.Auth = cfg.Auth
		clone.Server = cfg.Server
		return &clone, nil
	}
	defer func() { loadConfig = prevLoad }()

	var cancel context.CancelFunc
	prevNotify := notifyContext
	notifyContext = func(ctx context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		ctx, cancel = context.WithCancel(ctx)
		return ctx, cancel
	}
	defer func() { notifyContext = prevNotify }()

	fake := &fakeServer{listenErr: http.ErrServerClosed}
	var gotCfg *config.Config
	var gotOpts []httpx.Option
	prevNew := newHTTPServer
	newHTTPServer = func(ctx context.Context, c *config.Config, opts ...httpx.Option) (httpServer, error) {
		gotCfg = c
		gotOpts = append([]httpx.Option(nil), opts...)
		fake.cancel = cancel
		return fake, nil
	}
	defer func() { newHTTPServer = prevNew }()

	stdout := capture(t, &os.Stdout, func() {
		if code := cmdServer("1.0.0", "deadbeef", "today", []string{"--config", "cfg.yaml", "--addr", ":9000", "--log-level", "debug", "--pretty", "--version"}); code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
	})

	if fake.listenCount != 1 || fake.shutdownCount != 1 {
		t.Fatalf("server calls listen=%d shutdown=%d", fake.listenCount, fake.shutdownCount)
	}
	if gotCfg.Server.Addr != ":9000" {
		t.Fatalf("addr override failed: %s", gotCfg.Server.Addr)
	}
	if fake.shutdownCtx == nil {
		t.Fatalf("shutdown ctx nil")
	}
	if deadline, ok := fake.shutdownCtx.Deadline(); !ok || deadline.Before(time.Now()) {
		t.Fatalf("expected shutdown deadline in future")
	}
	if len(gotOpts) != 3 {
		t.Fatalf("expected three options, got %d", len(gotOpts))
	}
	if stdout == "" {
		t.Fatalf("expected log output")
	}
}

func TestCmdServerListenErrorStops(t *testing.T) {
	prevLoad := loadConfig
	loadConfig = func(string) (*config.Config, error) {
		return &config.Config{Server: config.ServerConfig{}, Auth: config.AuthConfig{Type: "none"}}, nil
	}
	defer func() { loadConfig = prevLoad }()

	var cancel context.CancelFunc
	prevNotify := notifyContext
	notifyContext = func(ctx context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		ctx, cancel = context.WithCancel(ctx)
		return ctx, cancel
	}
	defer func() { notifyContext = prevNotify }()

	fake := &fakeServer{listenErr: errors.New("boom")}
	prevNew := newHTTPServer
	newHTTPServer = func(ctx context.Context, c *config.Config, opts ...httpx.Option) (httpServer, error) {
		fake.cancel = cancel
		return fake, nil
	}
	defer func() { newHTTPServer = prevNew }()

	stdout := capture(t, &os.Stdout, func() {
		if code := cmdServer("v", "c", "d", []string{}); code != 0 {
			t.Fatalf("expected exit 0 on graceful shutdown, got %d", code)
		}
	})
	if !strings.Contains(stdout, "server error") {
		t.Fatalf("expected server error log, got %q", stdout)
	}
}

func TestCmdServerNewServerError(t *testing.T) {
	prevLoad := loadConfig
	loadConfig = func(string) (*config.Config, error) {
		return &config.Config{Auth: config.AuthConfig{}}, nil
	}
	defer func() { loadConfig = prevLoad }()

	prevNotify := notifyContext
	notifyContext = func(ctx context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		return context.WithCancel(ctx)
	}
	defer func() { notifyContext = prevNotify }()

	prevNew := newHTTPServer
	newHTTPServer = func(context.Context, *config.Config, ...httpx.Option) (httpServer, error) {
		return nil, errors.New("boom")
	}
	defer func() { newHTTPServer = prevNew }()

	stdout := capture(t, &os.Stdout, func() {
		if code := cmdServer("v", "c", "d", nil); code != 1 {
			t.Fatalf("expected exit 1, got %d", code)
		}
	})

	if !strings.Contains(stdout, "init server") {
		t.Fatalf("missing init server error: %q", stdout)
	}
}

func TestCmdServerShutdownError(t *testing.T) {
	prevLoad := loadConfig
	loadConfig = func(string) (*config.Config, error) {
		return &config.Config{Auth: config.AuthConfig{}, Server: config.ServerConfig{}}, nil
	}
	defer func() { loadConfig = prevLoad }()

	var cancel context.CancelFunc
	prevNotify := notifyContext
	notifyContext = func(ctx context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		ctx, cancel = context.WithCancel(ctx)
		return ctx, cancel
	}
	defer func() { notifyContext = prevNotify }()

	fake := &fakeServer{listenErr: http.ErrServerClosed, shutdownErr: errors.New("fail")}
	prevNew := newHTTPServer
	newHTTPServer = func(ctx context.Context, c *config.Config, opts ...httpx.Option) (httpServer, error) {
		fake.cancel = cancel
		return fake, nil
	}
	defer func() { newHTTPServer = prevNew }()

	stdout := capture(t, &os.Stdout, func() {
		if code := cmdServer("v", "c", "d", nil); code != 1 {
			t.Fatalf("expected exit 1, got %d", code)
		}
	})

	if !strings.Contains(stdout, "graceful shutdown failed") {
		t.Fatalf("missing shutdown error log: %q", stdout)
	}
}

func TestCmdValidate(t *testing.T) {
	prevLoad := loadConfig
	defer func() { loadConfig = prevLoad }()

	loadConfig = func(string) (*config.Config, error) { return &config.Config{}, nil }
	stdout := capture(t, &os.Stdout, func() {
		if code := cmdValidate([]string{"--config", "cfg"}); code != 0 {
			t.Fatalf("expected exit 0, got %d", code)
		}
	})
	if !strings.Contains(stdout, "config ok") {
		t.Fatalf("missing success message: %q", stdout)
	}

	stderr := capture(t, &os.Stderr, func() {
		if code := cmdValidate([]string{"--config", "cfg", "--extra"}); code != 2 {
			t.Fatalf("expected exit 2 on parse error")
		}
	})
	if !strings.Contains(stderr, "flag provided") {
		t.Fatalf("expected flag error, got %q", stderr)
	}

	stderr = capture(t, &os.Stderr, func() {
		if code := cmdValidate([]string{}); code != 2 {
			t.Fatalf("expected exit 2 when config missing")
		}
	})
	if !strings.Contains(stderr, "Usage") {
		t.Fatalf("expected usage output")
	}

	loadConfig = func(string) (*config.Config, error) { return nil, errors.New("bad") }
	stderr = capture(t, &os.Stderr, func() {
		if code := cmdValidate([]string{"--config", "cfg"}); code != 1 {
			t.Fatalf("expected exit 1")
		}
	})
	if !strings.Contains(stderr, "invalid config") {
		t.Fatalf("expected invalid config message")
	}
}

func TestParseLevel(t *testing.T) {
	if lvl := parseLevel("debug"); lvl != slog.LevelDebug {
		t.Fatalf("expected debug level")
	}
	if lvl := parseLevel("warn"); lvl != slog.LevelWarn {
		t.Fatalf("expected warn level")
	}
	if lvl := parseLevel("error"); lvl != slog.LevelError {
		t.Fatalf("expected error level")
	}
	if lvl := parseLevel("info"); lvl != slog.LevelInfo {
		t.Fatalf("default level info")
	}
}

func TestMain(m *testing.M) {
	if runtime.GOOS == "windows" {
		os.Exit(m.Run())
	}
	os.Exit(m.Run())
}
