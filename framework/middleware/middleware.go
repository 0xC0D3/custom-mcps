// Package middleware provides MCP-level and HTTP-level middleware
// for MCP server transports.
package middleware

import (
	"context"
	"encoding/json"
	"net/http"
)

// MessageHandler processes a JSON-RPC message and returns an optional response.
// This type mirrors transport.MessageHandler to avoid circular imports.
type MessageHandler func(ctx context.Context, raw json.RawMessage) json.RawMessage

// Middleware wraps a MessageHandler with additional behavior.
// It operates at the MCP/JSON-RPC message level, independent of transport.
type Middleware func(next MessageHandler) MessageHandler

// Chain composes multiple middleware into a single middleware.
// Middleware are applied in order: the first in the list is the outermost wrapper.
func Chain(mws ...Middleware) Middleware {
	return func(final MessageHandler) MessageHandler {
		for i := len(mws) - 1; i >= 0; i-- {
			final = mws[i](final)
		}
		return final
	}
}

// HTTPMiddleware wraps an http.Handler with additional behavior.
// It is used for HTTP transport-level concerns such as CORS and rate limiting.
type HTTPMiddleware func(http.Handler) http.Handler

// ChainHTTP composes multiple HTTP middleware into a single middleware.
// Middleware are applied in order: the first in the list is the outermost wrapper.
func ChainHTTP(mws ...HTTPMiddleware) HTTPMiddleware {
	return func(final http.Handler) http.Handler {
		for i := len(mws) - 1; i >= 0; i-- {
			final = mws[i](final)
		}
		return final
	}
}
