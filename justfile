# SPDX-License-Identifier: MIT
# Copyright (c) 2026 Jason Giese (Bl4cky99)

binary_name := "mocker"
pkg         := "./cmd/mocker"
out_dir     := "bin"
config      := "config.yaml"
coverfile   := "coverage.out"

version := `git describe --tags --always --dirty 2>/dev/null || echo dev`
commit  := `git rev-parse --short HEAD 2>/dev/null || echo none`
date    := `date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo unknown`

ldflags := "-X 'main.version=" + version + "' -X 'main.commit=" + commit + "' -X 'main.date=" + date + "'"

# List all available recipes
help:
    @just --list

# Build binary into ./bin/ with version metadata
build:
    mkdir -p {{out_dir}}
    go build -ldflags "{{ldflags}}" -o {{out_dir}}/{{binary_name}} {{pkg}}

# Build & run 'serve' with pretty logs (override config via --set config=...)
run: build
    ./{{out_dir}}/{{binary_name}} serve -c {{config}} -p

# Build & start the server (no pretty logs)
serve: build
    ./{{out_dir}}/{{binary_name}} serve -c {{config}}

# Validate config file (override config via --set config=...)
validate: build
    ./{{out_dir}}/{{binary_name}} validate -c {{config}}

# Run tests with race detector
test:
    go test -race -timeout=2m ./...

# Run tests with race detector and verbose output
test-verbose:
    go test -race -timeout=2m -v ./...

# Run tests using the Ginkgo test framework
test-ginkgo:
    ginkgo -v ./...

# Run tests with coverage and show summary
cover:
    go test -coverprofile={{coverfile}} ./...
    go tool cover -func={{coverfile}} | tail -n 1

# Generate HTML coverage report
cover-html: cover
    go tool cover -html={{coverfile}} -o {{out_dir}}/coverage.html

# Format code with go fmt (and goimports if installed)
fmt:
    go fmt ./...
    -goimports -w .

# Static analysis with go vet
vet:
    go vet ./...

# Sync go.mod/go.sum
tidy:
    go mod tidy

# Remove build artifacts
clean:
    rm -rf {{out_dir}} {{coverfile}}

# Print build-time version values
version:
    @echo "version={{version}} commit={{commit}} date={{date}}"
