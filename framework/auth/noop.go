package auth

import (
	"context"
	"net/http"
)

// noopAuth is an authenticator that always succeeds.
type noopAuth struct{}

// Noop returns an Authenticator that always succeeds, setting the client
// identifier to "anonymous". Use it for stdio transports or development
// environments where authentication is not required.
func Noop() Authenticator {
	return &noopAuth{}
}

// Authenticate always returns a context with the client ID set to "anonymous".
func (n *noopAuth) Authenticate(ctx context.Context, _ *http.Request) (context.Context, error) {
	return WithClientID(ctx, "anonymous"), nil
}
