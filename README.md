<a id="readme-top"></a>

<br />
<div align="center">
    <a href="https://github.com/Bl4cky99/mocker">
        <img src="README_ASSETS/logo.png" width="600">
    </a>
    <h3>mocker</h3>
    <p align="center">
        A file-driven mock <b>HTTP API server</b> with templated responses, schema validation, and pluggable auth for local development and integration tests.
        <br/>
        Define endpoints in YAML/JSON, serve multiple variants, enforce request contracts, and explore responses with real-time templating helpers.
        <br/><br/>
        <a href="https://github.com/Bl4cky99/mocker/issues/new?template=bug_report.yml">Report Bug</a>
        &middot;
        <a href="https://github.com/Bl4cky99/mocker/issues/new?template=feature_request.yml">Request Feature</a>
        <br/><br/>
    </p>
</div>

<details>
<summary>Table of Contents</summary>
<ol>
  <li><a href="#features">Features</a></li>
  <li><a href="#installation">Installation</a>
    <ul>
      <li><a href="#install-go">Go install (recommended)</a></li>
      <li><a href="#install-release">GitHub Releases</a></li>
      <li><a href="#install-build">Build from source</a></li>
      <li><a href="#install-docker">Docker (GHCR)</a></li>
    </ul>
  </li>
  <li><a href="#quickstart">Quickstart</a></li>
  <li><a href="#configuration">Configuration</a>
    <ul>
      <li><a href="#config-server">Server settings</a></li>
      <li><a href="#config-auth">Authentication</a></li>
      <li><a href="#config-endpoints">Endpoints</a></li>
      <li><a href="#config-variants">Response variants</a></li>
      <li><a href="#config-template">Template data & helpers</a></li>
      <li><a href="#config-validation">Request validation</a></li>
    </ul>
  </li>
  <li><a href="#examples">Examples</a></li>
  <li><a href="#cli">CLI reference</a></li>
  <li><a href="#troubleshooting">Troubleshooting</a></li>
  <li><a href="#compatibility">Compatibility</a></li>
  <li><a href="#roadmap">Roadmap</a></li>
  <li><a href="#faq">FAQ</a></li>
  <li><a href="#developer-documentation">Developer Documentation</a>
    <ul>
      <li><a href="#architecture">Architecture</a></li>
      <li><a href="#tests">Tests</a></li>
      <li><a href="#minimal-dev-loop">Minimal dev loop</a></li>
    </ul>
  </li>
  <li><a href="#license">License</a></li>
</ol>
</details>

---

## <span id="features">Features</span>

- **Declarative mocks**: describe endpoints, variants, and contracts in a single YAML or JSON file; runtime validation rejects misconfigured responses early.
- **Variant matching**: choose responses by method, path params, query strings, or request headers with deterministic fallback rules.
- **Templated bodies**: inline Go templates (or external files) get live request data such as path parameters, headers, and the current timestamp; reuse helpers like `{{ json . }}` for quick payloads.
- **Schema-aware inputs**: optional JSON Schema validation (Draft 2020) and `Content-Type` checks let you enforce request payloads before returning mock data.
- **Built-in auth**: enable bearer-token or HTTP basic authentication with constant-time comparisons, or disable auth entirely for open mocks.
- **Production-like behaviour**: configurable response delays, global default headers, request IDs, and structured logs mimic real services during integration tests.
- **Developer-friendly CLI**: `mocker serve` starts the server with pretty logs, `mocker validate` verifies configurations, and `--version` prints build metadata at startup.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## <span id="installation">Installation</span>

### <span id="install-go">Go install (recommended)</span>

```bash
go install github.com/Bl4cky99/mocker/cmd/mocker@latest

# Run from any folder
mocker serve -c ./config.yaml
```

Requires Go **1.25+**. The binary is placed in `$(go env GOPATH)/bin` (add it to your `$PATH`).

### <span id="install-release">GitHub Releases</span>

Tagged builds ship as tarballs for each supported architecture. Download and verify a release binary:

```bash
VERSION=v1.2.3
OS=linux
ARCH=amd64

BASE=https://github.com/Bl4cky99/mocker/releases/download/${VERSION}
FILE=mocker-${VERSION}-${OS}-${ARCH}.tar.gz

curl -sSLO ${BASE}/${FILE}
curl -sSLO ${BASE}/${FILE}.sha256
sha256sum -c ${FILE}.sha256

tar -xzf ${FILE}
chmod +x mocker-${OS}-${ARCH}
./mocker-${OS}-${ARCH} serve -c config.yaml -p
```

Rename the binary or move it to a directory on your `$PATH` (for example `/usr/local/bin`).

---

### <span id="install-build">Build from source</span>

Clone the repository and use the provided Makefile (or plain `go build`).

```bash
make build           # produces ./bin/mocker with git version metadata
make serve           # builds & starts the server using config.yaml

# or without Makefile
GOFLAGS="-trimpath" go build -o bin/mocker ./cmd/mocker
```

Environment variables at build time (`VERSION`, `COMMIT`, `DATE`) are embedded into the binary. Override them when cutting releases.

---

### <span id="install-docker">Docker (GHCR)</span>

Tagged pushes trigger an automated build that publishes `ghcr.io/bl4cky99/mocker` (currently linux/amd64). Use semantic version tags or the floating aliases `latest`, `vX`, and `vX.Y`.

```bash
IMAGE=ghcr.io/bl4cky99/mocker
TAG=v1.2.3    # or latest

docker pull ${IMAGE}:${TAG}

docker run --rm -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  ${IMAGE}:${TAG} serve -c /app/config.yaml -p
```

The container entrypoint runs `mocker serve -c /app/config.yaml -p`; mount your config there or override the command.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## <span id="quickstart">Quickstart</span>

1. Copy the sample config (`examples/example.yaml`) to `config.yaml` and customise it.
2. Start the server (pretty logs enabled):

```bash
mocker serve -c config.yaml -p
```

3. Exercise endpoints with `curl` or your favourite API client:

```bash
curl -i -H "Authorization: Bearer devtoken123" \
  "http://localhost:1337/api/users/42?verbose=true"
```

> Tip: run `mocker validate -c config.yaml` in your CI pipeline to catch schema errors or missing files before deployment.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## <span id="configuration">Configuration</span>

Configuration files accept **YAML (`.yml`, `.yaml`)** or **JSON (`.json`)**. Unknown fields are rejected, defaults are applied automatically, and the resolved configuration is validated before the server starts.

### <span id="config-server">Server settings</span>

```yaml
server:
  addr: ":1337"           # default :8080
  basePath: "/api"        # default "/"
  defaultHeaders:
    X-Powered-By: "mocker"
    Cache-Control: "no-store"
  cors:
    enabled: true
    allowOrigins: ["*"]
    allowMethods: ["GET", "POST"]
    allowHeaders: ["Authorization"]
```

- `addr`: listening address; override at runtime via `--addr`.
- `basePath`: mounted prefix (trimmed of trailing `/`). All endpoints are registered beneath it.
- `defaultHeaders`: applied to every response unless the handler has already set the header.
- `cors`: reserved for upcoming first-class CORS support. When enabled today it allows unauthenticated `OPTIONS` preflight requests while you manage the actual headers via `defaultHeaders`.

### <span id="config-auth">Authentication</span>

```yaml
auth:
  type: token         # "none" | "token" | "basic"
  token:
    header: "Authorization"
    prefix: "Bearer "
    tokens: ["devtoken123", "staging-secret"]

# or basic auth
# auth:
#   type: basic
#   basic:
#     users:
#       - username: "admin"
#         password: "password"
```

- `token`: constant-time comparison against the configured token list. Prefix is optional.
- `basic`: validates username/password pairs; responses include `WWW-Authenticate` when credentials are missing or wrong.
- `none`: disables auth entirely.

### <span id="config-endpoints">Endpoints</span>

Each endpoint specifies an HTTP method, path (chi-style parameters like `/users/{id}`), optional request validation, and one or more response variants.

Rules enforced during load:
- Paths must start with `/` and HTTP methods must be one of `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `OPTIONS`.
- Duplicate endpoints without `when` clauses are rejected to avoid ambiguous fallbacks.
- Every endpoint needs at least one response.

### <span id="config-variants">Response variants</span>

```yaml
responses:
  - when:
      query: { search: "alice" }
      header: { X-Debug: "1" }
    status: 200
    headers: { Content-Type: "application/json" }
    bodyFile: "./examples/bodies/users.list.json.tmpl"
  - status: 404
    headers: { Content-Type: "application/json" }
    body: '{ "error": "user not found" }'
    delayMs: 150
```

- `when.query`: exact match on query parameters.
- `when.header`: exact match on request headers (canonicalised names).
- Selection order: the first matching variant wins. If none match, the earliest variant without a `when` clause is used as fallback, otherwise the first variant is returned.
- `status`: defaults to `200` when omitted.
- `headers`: override or extend the global `defaultHeaders` for that response.
- `delayMs`: artificial latency before writing the response (cancelled if the request context ends).
- Exactly one of `body` (inline string) or `bodyFile` (path to template or raw file) must be set.

### <span id="config-template">Template data & helpers</span>

The renderer uses Go's `html/template` with `missingkey=default` and a growing set of helpers:

```go
{{ .Path.id }}            # chi path parameter
{{ .Query.verbose }}      # query parameter (string)
{{ index .Header "X-Correlation-Id" }}
{{ .NowRFC3339 }}         # timestamp injected per request
{{ json .Query }}         # helper -> JSON encode any value
```

- Inline templates (`body`) are parsed on each request; files (`bodyFile`) are cached and reloaded when their mtime changes.
- Headers are canonicalised (`X-Correlation-Id`), queries preference the first value, path params come from chi's URL params.

### <span id="config-validation">Request validation</span>

```yaml
validate:
  contentType: "application/json"
  schemaFile: "./examples/schemas/user.create.json"
```

- `contentType`: optional strict equality check for the request `Content-Type`.
- `schemaFile`: compile-on-start JSON Schema (Draft 2020). Files are cached per absolute path and reused across endpoints.
- Request bodies are limited to 1 MiB for schema validation to avoid runaway payloads.
- Validation errors result in `400 Bad Request` with the schema error message.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## <span id="examples">Examples</span>

Explore `/examples` for ready-to-run demos:

- `examples/example.yaml` - full sample covering auth, validation, templated responses, delays, and query/header routing.
- `examples/bodies/*.tmpl` - Go templates showing how to echo request data back to the caller.
- `examples/schemas/*.json` - JSON Schemas used to validate POST/PUT payloads.

Run the sample config directly:

```bash
cp examples/example.yaml config.yaml
mocker serve -c config.yaml -p

# Try different variants
curl -i -H "Authorization: Bearer devtoken123" \
  "http://localhost:1337/api/users?search=alice"

curl -i -H "Authorization: Bearer devtoken123" \
  -H "X-Debug: 1" "http://localhost:1337/api/orders/42"
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## <span id="cli">CLI reference</span>

```
mocker - local mock API server

Usage:
    mocker <command> [flags]

Commands:
    serve       Start the mock server (alias: mocker serve)
    validate    Validate a config file and exit
    version     Print version info
```

### `serve`

| Flag | Description |
|------|-------------|
| `-c, --config` | Path to config file (default `config.yaml`). |
| `-a, --addr` | Override server address from the config. |
| `-l, --log-level` | `debug`, `info`, `warn`, or `error` (default `info`). |
| `-p, --pretty` | Use human-readable text logs instead of JSON. |
| `--version` | Print build metadata at startup. |

### `validate`

| Flag | Description |
|------|-------------|
| `-c, --config` | **Required** path to config file to validate. |

Exit codes follow UNIX conventions: `0` on success, `1` for runtime errors, `2` for CLI misuse (missing flags, unknown commands, etc.).

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## <span id="troubleshooting">Troubleshooting</span>

- **"empty config path" or "read ...":** ensure the path passed to `--config` exists and has `.yaml`, `.yml`, or `.json` extension.
- **Schema compile errors:** paths inside `schemaFile` are resolved relative to the working directory. Use absolute paths or keep schemas next to your config.
- **"set exactly one of body or bodyFile":** every variant needs exactly one body source. Remove the redundant field.
- **Unexpected fallback response:** remember that variants without `when` clauses serve as fallbacks; put more specific matches earlier.
- **401 Unauthorized:** confirm the correct bearer token or basic credentials and header prefix. Prefix matching is case-sensitive.
- **Body file not found at runtime:** `bodyFile` paths are read on demand; missing files will log an error and return `500`. Keep mock payloads alongside your config or use absolute paths.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## <span id="compatibility">Compatibility</span>

- Go runtime: **1.25+**
- Platforms: tested on Linux and macOS; Windows is expected to work (chi + net/http).
- JSON Schema: compiled with [`github.com/santhosh-tekuri/jsonschema/v6`](https://pkg.go.dev/github.com/santhosh-tekuri/jsonschema/v6) using Draft 2020 defaults.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## <span id="roadmap">Roadmap</span>

- [ ] First-class CORS response headers derived from the `server.cors` block.  
- [ ] Hot reload / watch mode for configuration changes.
- [ ] Pluggable request matchers (e.g. regex, body predicates).
- [ ] Additional template helpers (UUIDs, random data, timestamps).
- [ ] Optional OpenAPI export to document configured endpoints.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## <span id="faq">FAQ</span>

**Q: How are multiple variants evaluated?**  
A: `mocker` walks the list once. The first variant whose `when` clause matches wins; if none match, the earliest variant without conditions becomes the fallback, otherwise the head of the list is used.

**Q: Can I return binary files or images?**  
A: Yes. Point `bodyFile` to any file on disk (e.g. PNG). The file is streamed as-is, so remember to set the matching `Content-Type` header in the variant.

**Q: Do templates have access to the request body?**  
A: Not yet. The renderer currently provides path params, query params, headers, and a timestamp. Body capture is on the roadmap.

**Q: How do I disable logging?**  
A: Use `--log-level error` (or `warn`). Structured logging remains active for observability but noise is reduced.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## <span id="developer-documentation">Developer Documentation</span>

### <span id="architecture">Architecture</span>

```
.
|-- cmd/mocker          # CLI entrypoint (serve, validate, version)
|-- internal/cli        # Command parsing, logging setup, signal handling
|-- internal/config     # Config structs, defaulting, validation helpers
|-- internal/httpx      # HTTP server, routing, middleware, response engine
|-- internal/auth       # Basic and token auth providers
|-- internal/render     # Template renderer with file caching & helpers
`-- internal/validate   # JSON Schema compilation and runtime checks
```

Key flow:
1. CLI loads configuration (`internal/config.Load`), applies defaults, and validates it.
2. HTTP server (`internal/httpx`) compiles schema validators, builds chi routes, and wires middleware (logging, auth, validation).
3. Requests resolve to a response variant, optional delay is applied, bodies are rendered via `internal/render`, and responses inherit default headers.

### <span id="tests">Tests</span>

Run the entire suite (race detector enabled by default in the Makefile):

```bash
go test ./...
make test        # equivalent with -race -timeout=2m

# Specific packages
go test ./internal/config -run TestLoad
```

### <span id="minimal-dev-loop">Minimal dev loop</span>

```
make build                             # compile binary with version metadata
./bin/mocker serve -c config.yaml -p   # start the server with pretty logs
curl -H 'Authorization: Bearer devtoken123' http://localhost:1337/api/healthz
./bin/mocker validate -c config.yaml   # ensure config stays valid
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## <span id="license">License</span>

This project is licensed under the **MIT License**.

- Copyright (c) 2025 [Jason Giese (Bl4cky99)](https://github.com/Bl4cky99)
- See the full text in [LICENSE](./LICENSE).

**Notes for users and integrators**
- Commercial use, modification, distribution, and private forks are permitted.
- Keep the copyright and permission notice from the MIT license in all copies/substantial portions.
- (Optional) Add an SPDX header to source files for tooling: `// SPDX-License-Identifier: MIT`.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

Happy mocking!
