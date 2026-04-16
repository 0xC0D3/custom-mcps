# Custom MCP Servers

A Go monorepo containing a reusable **MCP (Model Context Protocol) server framework** and multiple MCP servers built on it. Each server is an independent application that imports the shared framework, registers tools/resources/prompts via struct tags, and runs.

## Quick Start

```go
package main

import (
    "context"
    "os"
    "os/signal"

    "github.com/0xC0D3/custom-mcps/framework/protocol"
    "github.com/0xC0D3/custom-mcps/framework/server"
    "github.com/0xC0D3/custom-mcps/framework/transport"
)

type GreetInput struct {
    Name string `json:"name" mcp:"required" jsonschema:"description=Name to greet"`
}

func main() {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
    defer stop()

    srv := server.New(
        server.WithName("hello-mcp"),
        server.WithVersion("1.0.0"),
        server.WithTransport(transport.NewStdio()),
    )

    server.Tool(srv, "greet", "Say hello", func(ctx context.Context, input GreetInput) (*protocol.CallToolResult, error) {
        return &protocol.CallToolResult{
            Content: []protocol.Content{protocol.TextContent("Hello, " + input.Name + "!")},
        }, nil
    })

    srv.Run(ctx)
}
```

## Repository Structure

```
.
├── framework/              # Reusable MCP server framework library
│   ├── protocol/           # JSON-RPC 2.0 + MCP message types
│   ├── schema/             # Struct tag -> JSON Schema generator
│   ├── auth/               # Pluggable authentication
│   ├── registry/           # Tool/Resource/Prompt registration
│   ├── transport/          # Stdio, Streamable HTTP, SSE transports
│   ├── middleware/         # MCP-level and HTTP-level middleware
│   └── server/             # Server builder with functional options
├── cmd/servers/            # Independent MCP server applications
│   └── cloudflare/         # Cloudflare DNS management server
├── go.mod
└── go.sum
```

See [`framework/README.md`](framework/README.md) for full framework documentation.

## Servers

### Cloudflare DNS (`cmd/servers/cloudflare/`)

MCP server for managing Cloudflare DNS zones and records.

**Tools:**
- `list_zones` — List DNS zones with pagination
- `list_dns_records` — List DNS records for a zone with filtering
- `create_dns_record` — Create a new DNS record

**Environment Variables:**

| Variable | Default | Description |
|----------|---------|-------------|
| `CF_TRANSPORT` | `stdio` | Transport mode: `stdio` or `http` |
| `CF_ADDR` | `:8080` | Listen address for HTTP mode |
| `CF_API_TOKEN` | — | Cloudflare API token |
| `CF_MCP_TOKEN` | — | Bearer token for MCP auth (optional) |

**Run:**

```bash
# Stdio mode (for Claude Code, Cursor, etc.)
go run ./cmd/servers/cloudflare/

# HTTP mode (for network access)
CF_TRANSPORT=http CF_MCP_TOKEN=mysecret go run ./cmd/servers/cloudflare/
```

## Build & Test

```bash
# Build a server
go build ./cmd/servers/cloudflare/

# Run all tests (107 tests)
go test ./... -race -count=1

# Run framework tests only
go test ./framework/... -race -count=1

# Run a single package
go test ./framework/schema/... -v

# Lint
golangci-lint run ./...
```

## Creating a New Server

1. Create a directory under `cmd/servers/<name>/`
2. Define tool input structs with `mcp` and `jsonschema` tags
3. Wire the server in `main.go`:

```go
srv := server.New(
    server.WithName("my-server"),
    server.WithVersion("1.0.0"),
    server.WithTransport(transport.NewStreamableHTTP()),
    server.WithAuthenticator(auth.Bearer("MY_TOKEN_ENV")),
)
server.Tool(srv, "my_tool", "Does something", myHandler)
srv.Run(ctx)
```

See the [framework README](framework/README.md) for the full API reference.

## Contributing

Feel free to use and modify these servers. You can contribute by suggesting features, fixes, and improvements.

## License

MIT
