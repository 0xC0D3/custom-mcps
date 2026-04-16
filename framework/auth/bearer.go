package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"os"
	"strings"
)

// bearerAuth validates requests using bearer tokens or API keys.
type bearerAuth struct {
	tokens []string
	envKey string
}

// Bearer creates an Authenticator that validates bearer tokens read from an
// environment variable. The variable named by envKey is read on every call to
// Authenticate so that rotated values are picked up without a restart.
// If the environment variable is empty or unset, authentication always fails.
func Bearer(envKey string) Authenticator {
	return &bearerAuth{envKey: envKey}
}

// BearerWithTokens creates an Authenticator that validates against a fixed set
// of tokens provided at construction time.
func BearerWithTokens(tokens ...string) Authenticator {
	t := make([]string, len(tokens))
	copy(t, tokens)
	return &bearerAuth{tokens: t}
}

// Authenticate extracts a token from the Authorization header (Bearer scheme)
// or the X-API-Key header and compares it against the configured valid tokens
// using constant-time comparison.
func (b *bearerAuth) Authenticate(ctx context.Context, r *http.Request) (context.Context, error) {
	token := extractToken(r)
	if token == "" {
		return ctx, errors.New("auth: missing authentication token: provide Authorization: Bearer <token> or X-API-Key header")
	}

	valid := b.validTokens()
	if len(valid) == 0 {
		return ctx, errors.New("auth: no valid tokens configured")
	}

	for _, v := range valid {
		if constantTimeEqual(token, v) {
			id := clientIDFromToken(token)
			return WithClientID(ctx, id), nil
		}
	}

	return ctx, errors.New("auth: invalid authentication token")
}

// validTokens returns the set of tokens to validate against.
func (b *bearerAuth) validTokens() []string {
	if len(b.tokens) > 0 {
		return b.tokens
	}
	if b.envKey != "" {
		if v := os.Getenv(b.envKey); v != "" {
			return []string{v}
		}
	}
	return nil
}

// extractToken pulls the token from the request, checking the Authorization
// header first (Bearer scheme), then falling back to X-API-Key.
func extractToken(r *http.Request) string {
	if auth := r.Header.Get("Authorization"); auth != "" {
		const prefix = "Bearer "
		if strings.HasPrefix(auth, prefix) {
			return strings.TrimSpace(auth[len(prefix):])
		}
	}
	return r.Header.Get("X-API-Key")
}

// constantTimeEqual compares two strings in constant time to prevent timing
// attacks.
func constantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// clientIDFromToken derives a non-sensitive client identifier from a token.
// If the token is long enough, it uses a short prefix; otherwise it falls back
// to "bearer".
func clientIDFromToken(token string) string {
	const minLen = 8
	const prefixLen = 4
	if len(token) >= minLen {
		return "bearer:" + token[:prefixLen]
	}
	return "bearer"
}
