// Package transport provides MCP server transport implementations.
package transport

import (
	"context"
	"encoding/json"
	"net/http"
)

// MessageHandler processes an incoming JSON-RPC message and returns
// an optional response. For notifications (no id), return nil.
type MessageHandler func(ctx context.Context, raw json.RawMessage) json.RawMessage

// Transport defines how an MCP server communicates with clients.
type Transport interface {
	// Start begins accepting connections/messages. It blocks until
	// ctx is cancelled or a fatal error occurs.
	Start(ctx context.Context, handler MessageHandler) error

	// Send pushes a server-initiated message (notification) to connected clients.
	Send(ctx context.Context, msg json.RawMessage) error

	// Close performs graceful shutdown of the transport.
	Close() error
}

// HTTPMiddleware wraps an http.Handler with additional behavior.
// Used for transport-level concerns on HTTP-based transports only.
type HTTPMiddleware func(http.Handler) http.Handler
