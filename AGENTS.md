# AGENTS.md

Instructions for AI agents working with this codebase.

## Repository Context

This is a Go monorepo with a custom MCP (Model Context Protocol) server framework and servers built on it. The framework is in `framework/`, servers are in `cmd/servers/<name>/`.

## Before Writing Code

1. **Read the framework README** at `framework/README.md` for the full API reference
2. **Check existing patterns** in `cmd/servers/cloudflare/` for how servers use the framework
3. **Understand the layer boundaries** — never import outer packages from inner ones:
   - `protocol`/`schema` have zero internal dependencies
   - `registry` depends on `protocol` + `schema`
   - `transport` depends on `protocol` + `auth`
   - `middleware` depends on `protocol` (not `transport`, to avoid circular imports)
   - `server` depends on everything

## Framework Development Rules

- **No third-party dependencies** in framework packages (stdlib only). Tests may use `testify`.
- **Every exported symbol** must have a godoc comment.
- **Struct tag schema generation** is the preferred tool registration method. Programmatic registration (`RegisterToolRaw`) is the escape hatch.
- The `mcp` tag controls `required` status. The `jsonschema` tag controls all other schema attributes.
- **Functional options** pattern for all constructors (`New*(opts...)`)
- Tests are **table-driven** using `testify/assert` and `testify/require`. Generate mocks with `mockery`.
- Run tests with `-race` flag: `go test ./framework/... -race -count=1`

## Server Development Rules

- Each server lives in `cmd/servers/<name>/` with at minimum: `main.go`, `tools.go`, `config.go`
- **main.go** should be ~20-30 lines: load config, create transport, create server, register tools, run
- **tools.go** defines input structs with `mcp`/`jsonschema` tags and handler functions
- **config.go** reads environment variables with sensible defaults
- Each server should support both `stdio` and `http` transport via an env var
- Use `server.Tool[T]()` for registration (not raw registry methods) unless dynamic tools are needed

## Adding a New Tool to an Existing Server

1. Define the input struct in `tools.go` with appropriate tags
2. Write the handler function: `func handleX(ctx context.Context, input XInput) (*protocol.CallToolResult, error)`
3. Register in `registerTools()`: `server.Tool(srv, "tool_name", "description", handleX)`
4. Add tests if the handler has non-trivial logic

## Adding a New Server

1. `mkdir cmd/servers/<name>/`
2. Copy the pattern from `cmd/servers/cloudflare/`
3. Define tools with struct tags
4. Add a Dockerfile
5. Add a README.md documenting the server's tools and env vars
6. Update the root README.md servers section

## Transport Selection

| Transport | Constructor | When to Use |
|-----------|-------------|-------------|
| Stdio | `transport.NewStdio()` | Claude Code, Cursor, local clients |
| Streamable HTTP | `transport.NewStreamableHTTP()` | Network/cloud deployment, modern clients |
| Legacy SSE | `transport.NewSSE()` | Older MCP clients that don't support Streamable HTTP |

## Error Handling in Handlers

- Return `(*protocol.CallToolResult, error)` — the error is for transport/protocol failures
- For **tool-level errors** (bad input, API failures), return a result with `IsError: true`:
  ```go
  return &protocol.CallToolResult{
      Content: []protocol.Content{protocol.TextContent("error: invalid zone ID")},
      IsError: true,
  }, nil
  ```
- For **unexpected failures**, return the error directly — the framework wraps it as a JSON-RPC internal error

## Testing

- Framework: `go test ./framework/... -race -count=1` (107 tests currently)
- Single package: `go test ./framework/schema/... -v`
- End-to-end via stdio pipe:
  ```bash
  echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | go run ./cmd/servers/cloudflare/
  ```
