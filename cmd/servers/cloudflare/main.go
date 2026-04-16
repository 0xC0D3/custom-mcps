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
	if err := run(); err != nil {
		slog.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := loadConfig()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Build transport based on config.
	var t transport.Transport
	switch cfg.Transport {
	case "http":
		t = transport.NewStreamableHTTP(transport.WithAddr(cfg.Addr))
	default:
		t = transport.NewStdio()
	}

	// Build server options.
	opts := []server.Option{
		server.WithName("cloudflare-mcp"),
		server.WithVersion("0.1.0"),
		server.WithTransport(t),
	}
	if cfg.MCPAuthToken != "" {
		opts = append(opts, server.WithAuthenticator(auth.BearerWithTokens(cfg.MCPAuthToken)))
	}

	srv := server.New(opts...)
	registerTools(srv)

	return srv.Run(ctx)
}
