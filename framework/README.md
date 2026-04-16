# MCP Server Framework

A Go library for building [Model Context Protocol](https://modelcontextprotocol.io/) servers with full spec support. Define tools with struct tags, pick a transport, and run.

## Features

- **Full MCP spec**: Tools, Resources, Prompts, initialization handshake, capability negotiation
- **Three transports**: Stdio (local), Streamable HTTP (modern), SSE (legacy)
- **Struct tag schemas**: Define tool inputs with `mcp` and `jsonschema` tags — JSON Schema generated automatically
- **Pluggable auth**: Bearer token built-in, custom authenticators via interface
- **Composable middleware**: Built-in logging and recovery; opt-in rate limiting, CORS, metrics
- **Functional options**: Clean builder API with sensible defaults
- **Zero external dependencies** in the framework core (only `testify` for tests)

## Architecture

```
protocol  schema       ← Foundation (no internal deps)
    \      /
    registry    auth   ← Core logic
       \       /
      transport        ← Wire protocols
         |
      middleware       ← Cross-cutting concerns
          \
          server       ← Public API (composes everything)
```

Dependency direction is strictly enforced: inner packages never import outer packages.

---

## Package Reference

### `server` — Top-Level API

The only package most developers need to import directly.

```go
import "github.com/0xC0D3/custom-mcps/framework/server"
```

#### Creating a Server

```go
srv := server.New(
    server.WithName("my-mcp-server"),
    server.WithVersion("1.0.0"),
    server.WithTransport(transport.NewStreamableHTTP()),
    server.WithAuthenticator(auth.Bearer("API_TOKEN")),
    server.WithInstructions("This server manages DNS records."),
)
```

#### Functional Options

| Option | Default | Description |
|--------|---------|-------------|
| `WithName(name)` | `"mcp-server"` | Server name in initialize response |
| `WithVersion(version)` | `"1.0.0"` | Server version |
| `WithTransport(t)` | Stdio | Transport implementation |
| `WithLogger(logger)` | `slog.Default()` | Structured logger |
| `WithAuthenticator(a)` | `auth.Noop()` | Request authenticator |
| `WithMiddleware(mws...)` | — | Append MCP-level middleware |
| `WithHTTPMiddleware(mws...)` | — | Append HTTP-level middleware |
| `WithTLS(cert, key)` | — | Enable TLS for HTTP transports |
| `WithAddress(addr)` | — | Listen address for HTTP transports |
| `WithGracefulShutdown(timeout)` | `30s` | Shutdown drain timeout |
| `WithInstructions(text)` | — | Instructions in initialize response |

#### Registering Tools

**Preferred: Struct tags (automatic JSON Schema)**

```go
type DNSLookupInput struct {
    Domain     string `json:"domain"     mcp:"required" jsonschema:"description=Domain to look up"`
    RecordType string `json:"recordType" mcp:"required" jsonschema:"description=Record type,enum=A|AAAA|CNAME|MX|TXT"`
    Timeout    int    `json:"timeout"                   jsonschema:"description=Timeout in seconds,minimum=1,maximum=30,default=5"`
}

server.Tool(srv, "dns_lookup", "Look up DNS records", func(ctx context.Context, input DNSLookupInput) (*protocol.CallToolResult, error) {
    // input is already decoded and validated
    return &protocol.CallToolResult{
        Content: []protocol.Content{protocol.TextContent("result here")},
    }, nil
})
```

**Escape hatch: Programmatic registration**

```go
srv.Registry().RegisterToolRaw(registry.ToolDefinition{
    Info: protocol.ToolInfo{
        Name:        "dynamic_tool",
        Description: "A dynamically registered tool",
        InputSchema: customSchemaJSON,
    },
    Handler: func(ctx context.Context, params json.RawMessage) (*protocol.CallToolResult, error) {
        // raw JSON params
        return &protocol.CallToolResult{
            Content: []protocol.Content{protocol.TextContent("done")},
        }, nil
    },
})
```

#### Registering Resources

```go
srv.Registry().RegisterResource(registry.ResourceDefinition{
    Info: protocol.ResourceInfo{
        URI:         "config://settings",
        Name:        "Server Settings",
        Description: "Current server configuration",
        MIMEType:    "application/json",
    },
    Handler: func(ctx context.Context, uri string) (*protocol.ReadResourceResult, error) {
        return &protocol.ReadResourceResult{
            Contents: []protocol.ResourceContent{
                {URI: uri, MIMEType: "application/json", Text: `{"debug": false}`},
            },
        }, nil
    },
})
```

#### Registering Prompts

```go
srv.Registry().RegisterPrompt(registry.PromptDefinition{
    Info: protocol.PromptInfo{
        Name:        "analyze-dns",
        Description: "Analyze DNS configuration",
        Arguments: []protocol.PromptArgument{
            {Name: "domain", Description: "Domain to analyze", Required: true},
        },
    },
    Handler: func(ctx context.Context, args map[string]string) (*protocol.GetPromptResult, error) {
        return &protocol.GetPromptResult{
            Description: "DNS analysis for " + args["domain"],
            Messages: []protocol.PromptMessage{
                {Role: "user", Content: protocol.TextContent("Analyze DNS for " + args["domain"])},
            },
        }, nil
    },
})
```

#### Running the Server

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

if err := srv.Run(ctx); err != nil {
    slog.Error("server exited", "error", err)
    os.Exit(1)
}
```

---

### `schema` — Struct Tag to JSON Schema

Generates JSON Schema from Go struct tags. Used internally by `RegisterTool[T]`, but also available directly.

```go
import "github.com/0xC0D3/custom-mcps/framework/schema"

s := schema.Generate[MyInput]()
```

#### `mcp` Tag

| Value | Effect |
|-------|--------|
| `required` | Adds field to JSON Schema `required` array |
| `optional` or omitted | Field is optional (default) |

#### `jsonschema` Tag

Comma-separated `key=value` pairs:

| Attribute | Example | JSON Schema Field |
|-----------|---------|-------------------|
| `description=...` | `description=The domain name` | `description` |
| `enum=A\|B\|C` | `enum=A\|AAAA\|CNAME` | `enum` (pipe-delimited) |
| `default=...` | `default=1.1.1.1` | `default` (type-aware parsing) |
| `minimum=N` | `minimum=1` | `minimum` |
| `maximum=N` | `maximum=100` | `maximum` |
| `minLength=N` | `minLength=3` | `minLength` |
| `maxLength=N` | `maxLength=255` | `maxLength` |
| `pattern=...` | `pattern=^[a-z]+$` | `pattern` |
| `format=...` | `format=email` | `format` |

#### Go Type Mapping

| Go Type | JSON Schema Type | Notes |
|---------|-----------------|-------|
| `string` | `"string"` | |
| `int`, `int8`–`int64`, `uint`–`uint64` | `"integer"` | |
| `float32`, `float64` | `"number"` | |
| `bool` | `"boolean"` | |
| `[]T` | `"array"` | `items` recursed |
| `map[string]T` | `"object"` | `additionalProperties` recursed |
| nested `struct` | `"object"` | `properties` recursed |
| `*T` | same as `T` | pointer = Go-optional |
| `time.Time` | `"string"` | `format: "date-time"` auto-set |
| `json.RawMessage` | (omitted) | represents `any` |

Unexported fields, `json:"-"` fields, and embedded structs (flattened) are handled correctly.

---

### `transport` — Wire Protocols

```go
import "github.com/0xC0D3/custom-mcps/framework/transport"
```

All transports implement the `Transport` interface:

```go
type Transport interface {
    Start(ctx context.Context, handler MessageHandler) error
    Send(ctx context.Context, msg json.RawMessage) error
    Close() error
}
```

#### Stdio

For local use — Claude Code, Cursor, and other clients that launch servers as subprocesses.

```go
t := transport.NewStdio()
// Options: WithStdioInput(r), WithStdioOutput(w), WithStdioLogger(l)
```

- Reads newline-delimited JSON from stdin (1MB buffer)
- Writes responses as JSON lines to stdout
- No authentication (local process)

#### Streamable HTTP

Modern MCP transport — single endpoint supporting POST, GET (SSE), and DELETE.

```go
t := transport.NewStreamableHTTP(
    transport.WithAddr(":8080"),
    transport.WithEndpoint("/mcp"),
    transport.WithStreamableAuth(auth.Bearer("TOKEN_ENV")),
    transport.WithStreamableTLS("cert.pem", "key.pem"),
    transport.WithHealthPath("/health"),
)
```

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/mcp` | Send JSON-RPC request, receive response |
| GET | `/mcp` | Open SSE stream for server notifications |
| DELETE | `/mcp` | Terminate a session |
| GET | `/health` | Health check (`{"status":"ok"}`) |

Session management via `Mcp-Session-Id` header.

#### SSE (Legacy)

Two-endpoint pattern for older MCP clients.

```go
t := transport.NewSSE(
    transport.WithSSEAddr(":8080"),
    transport.WithSSEAuth(auth.Bearer("TOKEN_ENV")),
)
```

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/sse` | Open SSE stream (initial event contains POST endpoint URL) |
| POST | `/messages?sessionId=...` | Send JSON-RPC request |
| GET | `/health` | Health check |

---

### `auth` — Authentication

```go
import "github.com/0xC0D3/custom-mcps/framework/auth"
```

#### Built-in Implementations

```go
// No authentication (stdio, development)
auth.Noop()

// Bearer token from environment variable
auth.Bearer("API_TOKEN_ENV")

// Bearer token from fixed values
auth.BearerWithTokens("token1", "token2")
```

Bearer auth checks headers in order: `Authorization: Bearer <token>`, then `X-API-Key`. Uses constant-time comparison to prevent timing attacks.

#### Custom Authenticator

Implement the `Authenticator` interface:

```go
type Authenticator interface {
    Authenticate(ctx context.Context, r *http.Request) (context.Context, error)
}
```

Use `auth.WithClientID(ctx, id)` to attach client identity and `auth.ClientID(ctx)` to extract it.

---

### `middleware` — Cross-Cutting Concerns

```go
import "github.com/0xC0D3/custom-mcps/framework/middleware"
```

Two levels of middleware:

#### MCP-Level (all transports)

Wraps `MessageHandler` — operates on JSON-RPC messages regardless of transport.

```go
// Built-in (always active)
middleware.Logging(logger)    // Logs method, duration, error status
middleware.Recovery(logger)   // Catches panics, returns JSON-RPC error

// Composing
chain := middleware.Chain(mw1, mw2, mw3) // first = outermost
```

#### HTTP-Level (HTTP transports only)

Wraps `http.Handler` — standard HTTP middleware pattern.

```go
// Opt-in
middleware.RateLimit(10, 20)          // 10 req/s, burst 20, per client IP
middleware.CORS("*")                  // Allow all origins
middleware.CORS("https://example.com") // Specific origin
middleware.Metrics("/metrics")        // JSON metrics at /metrics
```

Add HTTP middleware via server options:

```go
srv := server.New(
    server.WithHTTPMiddleware(
        middleware.CORS("*"),
        middleware.RateLimit(100, 200),
    ),
)
```

---

### `protocol` — MCP Message Types

```go
import "github.com/0xC0D3/custom-mcps/framework/protocol"
```

Provides all JSON-RPC 2.0 and MCP-specific types. Most commonly used:

#### Content Helpers

```go
protocol.TextContent("Hello, world!")
protocol.ImageContent("image/png", base64Data)
protocol.EmbeddedResourceContent(uri, mimeType, text)
```

#### Error Codes

```go
protocol.CodeParseError      // -32700
protocol.CodeInvalidRequest  // -32600
protocol.CodeMethodNotFound  // -32601
protocol.CodeInvalidParams   // -32602
protocol.CodeInternalError   // -32603
```

#### Error Helpers

```go
protocol.NewParseError("malformed JSON")
protocol.NewMethodNotFound("unknown/method")
protocol.NewInvalidParams("missing required field")
protocol.NewInternalError(err)
```

---

### `registry` — Registration

```go
import "github.com/0xC0D3/custom-mcps/framework/registry"
```

Usually accessed via `srv.Registry()` rather than directly. Stores tools, resources, and prompts. Registration should complete before `srv.Run()`.

Duplicate registrations silently overwrite the previous entry.

---

## Complete Example

A minimal but complete MCP server:

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log/slog"
    "os"
    "os/signal"
    "syscall"

    "github.com/0xC0D3/custom-mcps/framework/auth"
    "github.com/0xC0D3/custom-mcps/framework/protocol"
    "github.com/0xC0D3/custom-mcps/framework/server"
    "github.com/0xC0D3/custom-mcps/framework/transport"
)

type CalculateInput struct {
    Operation string  `json:"operation" mcp:"required" jsonschema:"description=Math operation,enum=add|subtract|multiply|divide"`
    A         float64 `json:"a"         mcp:"required" jsonschema:"description=First operand"`
    B         float64 `json:"b"         mcp:"required" jsonschema:"description=Second operand"`
}

func handleCalculate(ctx context.Context, input CalculateInput) (*protocol.CallToolResult, error) {
    var result float64
    switch input.Operation {
    case "add":
        result = input.A + input.B
    case "subtract":
        result = input.A - input.B
    case "multiply":
        result = input.A * input.B
    case "divide":
        if input.B == 0 {
            return &protocol.CallToolResult{
                Content: []protocol.Content{protocol.TextContent("error: division by zero")},
                IsError: true,
            }, nil
        }
        result = input.A / input.B
    }

    text, _ := json.Marshal(map[string]float64{"result": result})
    return &protocol.CallToolResult{
        Content: []protocol.Content{protocol.TextContent(string(text))},
    }, nil
}

func main() {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    srv := server.New(
        server.WithName("calculator-mcp"),
        server.WithVersion("1.0.0"),
        server.WithTransport(transport.NewStreamableHTTP(
            transport.WithAddr(":9090"),
            transport.WithStreamableAuth(auth.Bearer("CALC_TOKEN")),
        )),
    )

    server.Tool(srv, "calculate", "Perform a math operation", handleCalculate)

    slog.Info("starting calculator MCP server", "addr", ":9090")
    if err := srv.Run(ctx); err != nil {
        slog.Error("server exited", "error", err)
        os.Exit(1)
    }
}
```

Run it:

```bash
CALC_TOKEN=mysecret go run .

# In another terminal:
curl -X POST http://localhost:9090/mcp \
  -H "Authorization: Bearer mysecret" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"curl","version":"1.0"}}}'
```

## Testing

```bash
go test ./... -race -count=1         # 107 tests
go test ./schema/... -v              # single package
go test ./server/... -race -v        # server integration tests
```
