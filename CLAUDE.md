# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A Go monorepo containing a reusable MCP (Model Context Protocol) server framework (`framework/`) and multiple MCP servers built on it (`cmd/servers/<name>/`). Each server imports the framework, registers tools/resources/prompts via struct tags, and runs.

- Go module: `github.com/0xC0D3/custom-mcps`
- Go version: 1.25.5

## Architecture

### Framework (`framework/`)

Layered library with strict dependency direction: `protocol`/`schema` -> `auth`/`registry` -> `transport` -> `middleware` -> `server`.

| Package | Purpose |
|---------|---------|
| `protocol/` | JSON-RPC 2.0 + MCP message types, error codes, content helpers |
| `schema/` | `Generate[T]()` — struct tags (`mcp`, `jsonschema`) to JSON Schema |
| `auth/` | `Authenticator` interface + `Bearer(envKey)`, `BearerWithTokens(...)`, `Noop()` |
| `registry/` | `RegisterTool[T]()` (generic, preferred), `RegisterToolRaw()` (escape hatch), resource/prompt registration |
| `transport/` | `Transport` interface + `NewStdio()`, `NewStreamableHTTP()`, `NewSSE()` |
| `middleware/` | MCP-level: `Logging`, `Recovery`. HTTP-level: `RateLimit`, `CORS`, `Metrics` |
| `server/` | `New(opts...)` builder, `Tool[T]()` generic helper, `Run(ctx)` dispatch loop |

### Servers (`cmd/servers/<name>/`)

Each subdirectory is an independent MCP server. Currently: `cloudflare/` (DNS management with 3 tools).

## Key Patterns

### Tool Registration (preferred — struct tags)

```go
type MyInput struct {
    Name string `json:"name" mcp:"required" jsonschema:"description=The name"`
    Type string `json:"type" mcp:"required" jsonschema:"enum=A|B|C"`
}
server.Tool(srv, "my_tool", "description", handler)
```

### Struct Tag Reference

- `mcp:"required"` — marks field as required in JSON Schema
- `jsonschema:"description=...,enum=A|B|C,default=X,minimum=N,maximum=N,minLength=N,maxLength=N,pattern=...,format=..."`

### Transport Selection

```go
transport.NewStdio()                      // local/subprocess
transport.NewStreamableHTTP(WithAddr(":8080"))  // network (modern)
transport.NewSSE(WithSSEAddr(":8080"))    // network (legacy)
```

## Build & Run

```sh
go build ./cmd/servers/<name>/
go run ./cmd/servers/<name>                          # stdio mode
CF_TRANSPORT=http go run ./cmd/servers/cloudflare/   # HTTP mode
```

## Tests

```sh
go test ./framework/... -race -count=1    # framework (107 tests)
go test ./... -race -count=1              # everything
go test ./framework/schema/... -v         # single package
```

## Linting

```sh
golangci-lint run ./...
```

## Adding a New Server

1. Create `cmd/servers/<name>/` with `main.go`, `tools.go`, `config.go`
2. Define input structs with `mcp`/`jsonschema` tags
3. Wire: `server.New(opts...) -> server.Tool(srv, ...) -> srv.Run(ctx)`
4. Each server should have its own `Dockerfile` and `README.md`

## Conventions

- Framework packages have zero third-party dependencies (stdlib only, testify for tests)
- Servers use `CF_` prefix for Cloudflare env vars; new servers should use their own prefix
- Error handling: `fmt.Errorf("operation: %w", err)` wrapping, `errors.Is`/`errors.As`
- All exported symbols have godoc comments
- Table-driven tests with testify/assert and testify/require
