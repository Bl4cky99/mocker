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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Bl4cky99/mocker/internal/config"
	"github.com/Bl4cky99/mocker/internal/httpx"
)

func capture(target **os.File, fn func()) string {
	old := *target
	r, w, err := os.Pipe()
	Expect(err).NotTo(HaveOccurred())
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

var _ = Describe("Execute", func() {
	var oldArgs []string

	BeforeEach(func() {
		oldArgs = os.Args
	})

	AfterEach(func() {
		os.Args = oldArgs
	})

	It("prints usage and exits 2 when no args are given", func() {
		os.Args = []string{"mocker"}
		var code int
		stderr := capture(&os.Stderr, func() {
			code = Execute("1.0.0", "deadbeef", "today")
		})
		Expect(code).To(Equal(2))
		Expect(stderr).To(ContainSubstring("Usage:"))
	})

	It("shows help output and exits 0 with --help", func() {
		os.Args = []string{"mocker", "--help"}
		var code int
		stdout := capture(&os.Stdout, func() {
			code = Execute("1.0.0", "deadbeef", "today")
		})
		Expect(code).To(Equal(0))
		Expect(stdout).To(ContainSubstring("Commands:"))
	})

	It("shows version info with the version subcommand", func() {
		os.Args = []string{"mocker", "version"}
		var code int
		stdout := capture(&os.Stdout, func() {
			code = Execute("1.2.3", "cafebabe", "yesterday")
		})
		Expect(code).To(Equal(0))
		Expect(stdout).To(ContainSubstring("1.2.3"))
		Expect(stdout).To(ContainSubstring("cafebabe"))
	})

	It("delegates 'serve' to runServer with the correct args", func() {
		os.Args = []string{"mocker", "serve", "--flag"}
		prev := runServer
		defer func() { runServer = prev }()

		var gotVersion, gotCommit, gotDate string
		var gotArgs []string
		runServer = func(version, commit, date string, args []string) int {
			gotVersion, gotCommit, gotDate = version, commit, date
			gotArgs = append([]string(nil), args...)
			return 42
		}

		Expect(Execute("v", "c", "d")).To(Equal(42))
		Expect(gotVersion).To(Equal("v"))
		Expect(gotCommit).To(Equal("c"))
		Expect(gotDate).To(Equal("d"))
		Expect(gotArgs).To(Equal([]string{"--flag"}))
	})

	It("delegates 'validate' to runValidate with the correct args", func() {
		os.Args = []string{"mocker", "validate", "--config"}
		prev := runValidate
		defer func() { runValidate = prev }()

		var gotArgs []string
		runValidate = func(args []string) int {
			gotArgs = append([]string(nil), args...)
			return 7
		}

		Expect(Execute("v", "c", "d")).To(Equal(7))
		Expect(gotArgs).To(Equal([]string{"--config"}))
	})

	It("exits 2 and prints an error for an unknown command", func() {
		os.Args = []string{"mocker", "mystery"}
		var code int
		stderr := capture(&os.Stderr, func() {
			code = Execute("v", "c", "d")
		})
		Expect(code).To(Equal(2))
		Expect(stderr).To(ContainSubstring("unknown command"))
	})
})

var _ = Describe("cmdServer", func() {
	It("exits 2 on flag parse error", func() {
		var code int
		stderr := capture(&os.Stderr, func() {
			code = cmdServer("v", "c", "d", []string{"--nope"})
		})
		Expect(code).To(Equal(2))
		Expect(stderr).To(ContainSubstring("flag provided"))
	})

	It("exits 1 and logs when config cannot be loaded", func() {
		prev := loadConfig
		loadConfig = func(string) (*config.Config, error) { return nil, errors.New("boom") }
		defer func() { loadConfig = prev }()

		prevNotify := notifyContext
		notifyContext = func(ctx context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
			return context.WithCancel(ctx)
		}
		defer func() { notifyContext = prevNotify }()

		var code int
		stdout := capture(&os.Stdout, func() {
			code = cmdServer("v", "c", "d", []string{"--config", "cfg"})
		})
		Expect(code).To(Equal(1))
		Expect(stdout).To(ContainSubstring("load config"))
	})

	It("runs successfully and shuts down gracefully on signal", func() {
		prevLoad := loadConfig
		cfg := &config.Config{
			Server: config.ServerConfig{Addr: ":8080", BasePath: "/api"},
			Auth: config.AuthConfig{
				Type:  "token",
				Token: &config.TokenAuthConfig{Header: "X", Prefix: "Bearer", Tokens: []string{"tok"}},
			},
		}
		loadConfig = func(path string) (*config.Config, error) {
			Expect(path).To(Equal("cfg.yaml"))
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

		var code int
		stdout := capture(&os.Stdout, func() {
			code = cmdServer("1.0.0", "deadbeef", "today",
				[]string{"--config", "cfg.yaml", "--addr", ":9000", "--log-level", "debug", "--pretty", "--version"})
		})
		Expect(code).To(Equal(0))
		Expect(fake.listenCount).To(Equal(1))
		Expect(fake.shutdownCount).To(Equal(1))
		Expect(gotCfg.Server.Addr).To(Equal(":9000"))
		Expect(fake.shutdownCtx).NotTo(BeNil())
		deadline, hasDeadline := fake.shutdownCtx.Deadline()
		Expect(hasDeadline).To(BeTrue())
		Expect(deadline.After(time.Now())).To(BeTrue())
		Expect(gotOpts).To(HaveLen(3))
		Expect(stdout).NotTo(BeEmpty())
	})

	It("exits 0 and logs the error when ListenAndServe returns an unexpected error", func() {
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

		var code int
		stdout := capture(&os.Stdout, func() {
			code = cmdServer("v", "c", "d", []string{})
		})
		Expect(code).To(Equal(0))
		Expect(stdout).To(ContainSubstring("server error"))
	})

	It("exits 1 when newHTTPServer returns an error", func() {
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

		var code int
		stdout := capture(&os.Stdout, func() {
			code = cmdServer("v", "c", "d", nil)
		})
		Expect(code).To(Equal(1))
		Expect(stdout).To(ContainSubstring("init server"))
	})

	It("exits 1 when graceful shutdown fails", func() {
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

		var code int
		stdout := capture(&os.Stdout, func() {
			code = cmdServer("v", "c", "d", nil)
		})
		Expect(code).To(Equal(1))
		Expect(stdout).To(ContainSubstring("graceful shutdown failed"))
	})
})

var _ = Describe("cmdValidate", func() {
	It("exits 0 and prints 'config ok' on success", func() {
		prev := loadConfig
		loadConfig = func(string) (*config.Config, error) { return &config.Config{}, nil }
		defer func() { loadConfig = prev }()

		var code int
		stdout := capture(&os.Stdout, func() {
			code = cmdValidate([]string{"--config", "cfg"})
		})
		Expect(code).To(Equal(0))
		Expect(stdout).To(ContainSubstring("config ok"))
	})

	It("exits 2 on flag parse error", func() {
		var code int
		stderr := capture(&os.Stderr, func() {
			code = cmdValidate([]string{"--config", "cfg", "--extra"})
		})
		Expect(code).To(Equal(2))
		Expect(stderr).To(ContainSubstring("flag provided"))
	})

	It("exits 2 and shows usage when --config is missing", func() {
		var code int
		stderr := capture(&os.Stderr, func() {
			code = cmdValidate([]string{})
		})
		Expect(code).To(Equal(2))
		Expect(stderr).To(ContainSubstring("Usage"))
	})

	It("exits 1 and reports the error when config load fails", func() {
		prev := loadConfig
		loadConfig = func(string) (*config.Config, error) { return nil, errors.New("bad") }
		defer func() { loadConfig = prev }()

		var code int
		stderr := capture(&os.Stderr, func() {
			code = cmdValidate([]string{"--config", "cfg"})
		})
		Expect(code).To(Equal(1))
		Expect(stderr).To(ContainSubstring("invalid config"))
	})
})

var _ = Describe("parseLevel", func() {
	DescribeTable("maps string to slog.Level",
		func(input string, want slog.Level) {
			Expect(parseLevel(input)).To(Equal(want))
		},
		Entry("debug", "debug", slog.LevelDebug),
		Entry("warn", "warn", slog.LevelWarn),
		Entry("error", "error", slog.LevelError),
		Entry("info (default)", "info", slog.LevelInfo),
	)
})
