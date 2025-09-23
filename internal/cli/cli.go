// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Bl4cky99/mocker/internal/auth"
	"github.com/Bl4cky99/mocker/internal/config"
	"github.com/Bl4cky99/mocker/internal/httpx"
	"github.com/Bl4cky99/mocker/internal/render"
)

type httpServer interface {
	ListenAndServe() error
	Shutdown(context.Context) error
}

var (
	loadConfig    = config.Load
	newHTTPServer = func(ctx context.Context, cfg *config.Config, opts ...httpx.Option) (httpServer, error) {
		return httpx.New(ctx, cfg, opts...)
	}
	notifyContext = signal.NotifyContext
	runServer     = cmdServer
	runValidate   = cmdValidate
)

const usageHeader = `mocker - local mock API server

Usage:
	mocker <command> [flags]

Commands:
	server Start the mock server
	validate Validate a config file and exit
	version Print version info
	
Run 'mocker <command> --help' for command-specific flags.
`

func Execute(version, commit, date string) int {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usageHeader)
		return 2
	}

	switch os.Args[1] {
	case "-h", "-help", "--help", "help":
		fmt.Fprint(os.Stdout, usageHeader)
		return 0
	case "serve":
		return runServer(version, commit, date, os.Args[2:])
	case "validate":
		return runValidate(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Printf("mocker %s (commit %s, built %s)\n", version, commit, date)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usageHeader)
		return 2
	}
}

func cmdServer(version, commit, date string, args []string) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), `Usage: mocker serve [flags]
	
Flags:
	-c, --config string		Path to config file (yaml|yml|json) (default "config.yaml")
	-a, --addr string		Override server address (e.g. :9000)
	-l, --log-level string 		Log level: debug|info|warn|error (default: "info")
	-p, --pretty			Human-readable logs instead of JSON
	    --version			Print version on startup
`)
	}
	cfgPath := fs.String("config", "config.yaml", "")
	fs.StringVar(cfgPath, "c", *cfgPath, "path to config file")

	addr := fs.String("addr", "", "")
	fs.StringVar(addr, "a", *addr, "ovverride server address")

	logLevel := fs.String("log-level", "info", "")
	fs.StringVar(logLevel, "l", *logLevel, "log level (debug|info|warn|error)")

	pretty := fs.Bool("pretty", false, "")
	fs.BoolVar(pretty, "p", *pretty, "human-readable logs")

	printVersion := fs.Bool("version", false, "")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err.Error())
		return 2
	}

	level := parseLevel(*logLevel)
	var handler slog.Handler
	if *pretty {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}
	log := slog.New(handler).With("svc", "mocker", "version", version, "commit", commit)

	if *printVersion {
		log.Info("version", "version", version, "commit", commit, "date", date)
	}

	ctx, stop := notifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := loadConfig(*cfgPath)
	if err != nil {
		log.Error("load config", "path", *cfgPath, "err", err)
		return 1
	}
	if *addr != "" {
		cfg.Server.Addr = *addr
	}

	var prov auth.Provider
	switch cfg.Auth.Type {
	case "token":
		prov = auth.NewTokenAuth(cfg.Auth.Token.Header, cfg.Auth.Token.Prefix, cfg.Auth.Token.Tokens)
	case "basic":
		users := make(map[string]string, len(cfg.Auth.Basic.Users))
		for _, u := range cfg.Auth.Basic.Users {
			users[u.Username] = u.Password
		}
		prov = auth.NewBasicAuth(users, "mocker")
	}
	r := render.New()

	srv, err := newHTTPServer(ctx, cfg, httpx.WithLogger(log), httpx.WithAuth(prov, cfg.Auth.Type), httpx.WithRenderer(r))
	if err != nil {
		log.Error("init server", "err", err)
		return 1
	}

	go func() {
		log.Info("server starting", "addr", cfg.Server.Addr, "basePath", cfg.Server.BasePath)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "err", err)
			stop()
		}
	}()

	<-ctx.Done()

	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	log.Info("shutting down...")
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Error("graceful shutdown failed", "err", err)
		return 1
	}
	log.Info("bye")
	return 0
}

func cmdValidate(args []string) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), `Usage: mocker validate -c <file>

Flags:
	-c, --config string		Path to config file (yaml|yml|json) (required)
`)
	}
	cfgPath := fs.String("config", "", "")
	fs.StringVar(cfgPath, "c", *cfgPath, "path to config file")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err.Error())
		return 2
	}

	if *cfgPath == "" {
		fs.Usage()
		return 2
	}

	if _, err := loadConfig(*cfgPath); err != nil {
		fmt.Fprintf(os.Stderr, "invalid config: %v\n", err)
		return 1
	}

	fmt.Fprintln(os.Stdout, "config ok")
	return 0
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
