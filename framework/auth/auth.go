// Package auth provides authentication for MCP server transports.
package auth

import (
	"context"
	"net/http"
)

// Authenticator validates incoming HTTP requests. For non-HTTP transports
// (e.g., stdio), authentication is skipped entirely.
type Authenticator interface {
	// Authenticate validates the request and returns a context enriched
	// with identity information, or an error if authentication fails.
	Authenticate(ctx context.Context, r *http.Request) (context.Context, error)
}

// contextKey is an unexported type used for context value keys in this package,
// preventing collisions with keys defined in other packages.
type contextKey string

// clientIDKey is the context key for the authenticated client identifier.
const clientIDKey contextKey = "client_id"

// WithClientID returns a copy of ctx with the given client identifier attached.
func WithClientID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, clientIDKey, id)
}

// ClientID extracts the client identifier from ctx. It returns an empty string
// if no client ID has been set.
func ClientID(ctx context.Context) string {
	id, _ := ctx.Value(clientIDKey).(string)
	return id
}
