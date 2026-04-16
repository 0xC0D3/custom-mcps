package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/0xC0D3/custom-mcps/framework/protocol"
)

// rpcIDExtractor is a minimal struct for extracting the id field from a
// JSON-RPC request so that recovery can build a proper error response.
type rpcIDExtractor struct {
	ID json.RawMessage `json:"id"`
}

// Recovery returns middleware that recovers from panics in downstream handlers.
// When a panic occurs it logs the error and returns a JSON-RPC internal error
// response with the original request id.
func Recovery(logger *slog.Logger) Middleware {
	return func(next MessageHandler) MessageHandler {
		return func(ctx context.Context, raw json.RawMessage) (resp json.RawMessage) {
			defer func() {
				if r := recover(); r != nil {
					logger.ErrorContext(ctx, "panic recovered in handler",
						slog.Any("panic", r),
					)

					var req rpcIDExtractor
					_ = json.Unmarshal(raw, &req)

					rpcErr := protocol.NewInternalError(fmt.Errorf("internal error: %v", r))
					errResp := protocol.NewErrorResponse(req.ID, rpcErr)
					resp, _ = json.Marshal(errResp)
				}
			}()
			return next(ctx, raw)
		}
	}
}
