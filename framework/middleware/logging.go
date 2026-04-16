package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"
)

// rpcMethodExtractor is a minimal struct for extracting the method field from
// a JSON-RPC message without fully parsing it.
type rpcMethodExtractor struct {
	Method string `json:"method"`
}

// rpcErrorExtractor is a minimal struct for checking whether a JSON-RPC
// response contains an error field.
type rpcErrorExtractor struct {
	Error json.RawMessage `json:"error"`
}

// Logging returns middleware that logs each JSON-RPC request.
// It extracts the method from the raw JSON and logs the method name, request
// duration, and whether the response contains an error.
func Logging(logger *slog.Logger) Middleware {
	return func(next MessageHandler) MessageHandler {
		return func(ctx context.Context, raw json.RawMessage) json.RawMessage {
			var req rpcMethodExtractor
			_ = json.Unmarshal(raw, &req)

			start := time.Now()
			resp := next(ctx, raw)
			duration := time.Since(start)

			hasError := false
			if resp != nil {
				var errCheck rpcErrorExtractor
				if json.Unmarshal(resp, &errCheck) == nil && errCheck.Error != nil {
					hasError = true
				}
			}

			logger.InfoContext(ctx, "rpc request",
				slog.String("method", req.Method),
				slog.Duration("duration", duration),
				slog.Bool("has_error", hasError),
			)

			return resp
		}
	}
}
