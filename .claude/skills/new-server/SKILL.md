---
name: new-server
description: Scaffold a new MCP server under cmd/servers/. Creates main.go, tools.go, config.go, Dockerfile, and README.md following the established framework patterns.
user-invocable: true
---

# New MCP Server Scaffold

Scaffold a new MCP server application under `cmd/servers/<name>/`.

## Workflow

1. Ask for the server name (lowercase, hyphen-separated)
2. Ask what the server will do (1-2 sentences)
3. Ask which tools it should start with (name + description for each)
4. Ask for required environment variables (API keys, config)

## Files to Create

### `cmd/servers/<name>/main.go`

Follow the cloudflare server pattern:

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "os/signal"
    "syscall"

    "github.com/0xC0D3/custom-mcps/framework/auth"
    "github.com/0xC0D3/custom-mcps/framework/server"
    "github.com/0xC0D3/custom-mcps/framework/transport"
)

func main() {
    cfg := loadConfig()

    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    var t transport.Transport
    switch cfg.Transport {
    case "http":
        t = transport.NewStreamableHTTP(transport.WithAddr(cfg.Addr))
    default:
        t = transport.NewStdio()
    }

    opts := []server.ServerOption{
        server.WithName("<name>-mcp"),
        server.WithVersion("0.1.0"),
        server.WithTransport(t),
    }
    // Add auth if token configured

    srv := server.New(opts...)
    registerTools(srv)

    if err := srv.Run(ctx); err != nil {
        slog.Error("server exited with error", "error", err)
        os.Exit(1)
    }
}
```

### `cmd/servers/<name>/config.go`

Environment-based config with `<PREFIX>_TRANSPORT`, `<PREFIX>_ADDR`, and server-specific vars.

### `cmd/servers/<name>/tools.go`

Define tool input structs with `mcp` and `jsonschema` tags. Use `server.Tool[T]()` for registration:

```go
type MyToolInput struct {
    Field string `json:"field" mcp:"required" jsonschema:"description=What this field does"`
}

func registerTools(srv *server.Server) {
    server.Tool(srv, "tool_name", "Tool description", handleMyTool)
}
```

### `cmd/servers/<name>/Dockerfile`

Multi-stage build following the cloudflare pattern.

### `cmd/servers/<name>/README.md`

Document tools, environment variables, and usage examples.

## After Scaffolding

1. Run `go build ./cmd/servers/<name>/` to verify compilation
2. Test via stdio: `echo '<init json>' | go run ./cmd/servers/<name>/`
3. Update the root `README.md` servers section
