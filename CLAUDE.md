# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A Go monorepo containing a custom MCP (Model Context Protocol) server framework and multiple MCP servers built on it. Each server is an independent application that imports the shared framework.

- Go module: `github.com/0xC0D3/custom-mcps`
- Go version: 1.25.5

## Architecture

### Framework (`framework/`)

Layered library for building MCP servers. Dependency direction: `protocol`/`schema` → `auth`/`registry` → `transport` → `middleware` → `server`.

| Package | Purpose |
|---------|---------|
| `protocol/` | JSON-RPC 2.0 + MCP message types, error codes |
| `schema/` | Struct tag → JSON Schema generator (`mcp` and `jsonschema` tags) |
| `auth/` | `Authenticator` interface + Bearer token and Noop implementations |
| `registry/` | Tool/Resource/Prompt registration with generic `RegisterTool[T]()` |
| `transport/` | `Transport` interface + stdio, Streamable HTTP, and legacy SSE |
| `middleware/` | MCP-level (logging, recovery) and HTTP-level (CORS, rate limit, metrics) middleware |
| `server/` | Top-level `Server` builder with functional options, MCP dispatch loop |

### Servers (`cmd/servers/<name>/`)

Each subdirectory is an independent MCP server. Currently: `cloudflare/`.

Servers use struct tags for tool input schemas (preferred) or programmatic registration:

```go
server.Tool(srv, "tool_name", "description", func(ctx context.Context, input MyInput) (*protocol.CallToolResult, error) {
    return &protocol.CallToolResult{Content: []protocol.Content{protocol.TextContent("result")}}, nil
})
```

## Build & Run

```sh
go build ./cmd/servers/<name>/              # build a server
go run ./cmd/servers/<name>                 # run (stdio mode)
CF_TRANSPORT=http go run ./cmd/servers/cloudflare/  # run in HTTP mode
```

## Tests

```sh
go test ./framework/... -race -count=1      # framework tests (107 tests)
go test ./... -race -count=1                # all tests
go test ./framework/schema/... -v           # single package
```

## Linting

```sh
golangci-lint run ./...
```
