// Package main implements a Cloudflare DNS MCP server.
package main

import "os"

// Config holds the runtime configuration for the Cloudflare DNS MCP server.
type Config struct {
	// Transport selects the transport mode: "stdio" or "http".
	Transport string
	// Addr is the listen address for the HTTP transport (e.g. ":8080").
	Addr string
	// APIToken is the Cloudflare API token used for DNS operations.
	APIToken string
	// MCPAuthToken is the bearer token for authenticating MCP clients.
	// When empty, no authentication is required.
	MCPAuthToken string
}

// loadConfig reads configuration from environment variables and returns
// a Config with sensible defaults applied.
func loadConfig() Config {
	cfg := Config{
		Transport: envOrDefault("CF_TRANSPORT", "stdio"),
		Addr:      envOrDefault("CF_ADDR", ":8080"),
		APIToken:  os.Getenv("CF_API_TOKEN"),
		MCPAuthToken: os.Getenv("CF_MCP_TOKEN"),
	}
	return cfg
}

// envOrDefault returns the value of the named environment variable or
// the provided fallback if the variable is empty or unset.
func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
