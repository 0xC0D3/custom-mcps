package auth

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRequest(headers map[string]string) *http.Request {
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	return r
}

func TestBearerWithTokens(t *testing.T) {
	tests := []struct {
		name      string
		tokens    []string
		headers   map[string]string
		wantErr   bool
		errSubstr string
		wantID    string
	}{
		{
			name:   "valid bearer token",
			tokens: []string{"supersecrettoken"},
			headers: map[string]string{
				"Authorization": "Bearer supersecrettoken",
			},
			wantErr: false,
			wantID:  "bearer:supe",
		},
		{
			name:   "wrong bearer token",
			tokens: []string{"supersecrettoken"},
			headers: map[string]string{
				"Authorization": "Bearer wrongtoken",
			},
			wantErr:   true,
			errSubstr: "invalid",
		},
		{
			name:      "missing header",
			tokens:    []string{"supersecrettoken"},
			headers:   map[string]string{},
			wantErr:   true,
			errSubstr: "missing",
		},
		{
			name:   "X-API-Key header works",
			tokens: []string{"myapikey1"},
			headers: map[string]string{
				"X-API-Key": "myapikey1",
			},
			wantErr: false,
			wantID:  "bearer:myap",
		},
		{
			name:   "Authorization takes precedence over X-API-Key",
			tokens: []string{"tokenA"},
			headers: map[string]string{
				"Authorization": "Bearer tokenA",
				"X-API-Key":     "tokenB",
			},
			wantErr: false,
			wantID:  "bearer",
		},
		{
			name:   "multiple valid tokens - first matches",
			tokens: []string{"alpha123", "beta1234"},
			headers: map[string]string{
				"Authorization": "Bearer alpha123",
			},
			wantErr: false,
			wantID:  "bearer:alph",
		},
		{
			name:   "multiple valid tokens - second matches",
			tokens: []string{"alpha123", "beta1234"},
			headers: map[string]string{
				"Authorization": "Bearer beta1234",
			},
			wantErr: false,
			wantID:  "bearer:beta",
		},
		{
			name:   "short token uses default client id",
			tokens: []string{"abc"},
			headers: map[string]string{
				"Authorization": "Bearer abc",
			},
			wantErr: false,
			wantID:  "bearer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := BearerWithTokens(tt.tokens...)
			ctx, err := a.Authenticate(context.Background(), newRequest(tt.headers))
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantID, ClientID(ctx))
		})
	}
}

func TestBearerEnvVar(t *testing.T) {
	const envKey = "TEST_AUTH_TOKEN_FOR_MCP"

	t.Run("reads token from env", func(t *testing.T) {
		t.Setenv(envKey, "envtoken1234")
		a := Bearer(envKey)
		ctx, err := a.Authenticate(context.Background(), newRequest(map[string]string{
			"Authorization": "Bearer envtoken1234",
		}))
		require.NoError(t, err)
		assert.Equal(t, "bearer:envt", ClientID(ctx))
	})

	t.Run("empty env always fails", func(t *testing.T) {
		t.Setenv(envKey, "")
		a := Bearer(envKey)
		_, err := a.Authenticate(context.Background(), newRequest(map[string]string{
			"Authorization": "Bearer anytoken",
		}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no valid tokens configured")
	})

	t.Run("unset env always fails", func(t *testing.T) {
		// Ensure the variable is not set.
		t.Setenv(envKey, "")
		a := Bearer(envKey)
		_, err := a.Authenticate(context.Background(), newRequest(map[string]string{
			"Authorization": "Bearer anything",
		}))
		require.Error(t, err)
	})
}

func TestConstantTimeComparison(t *testing.T) {
	// Verify that constantTimeEqual behaves correctly for equal and unequal inputs.
	assert.True(t, constantTimeEqual("secret", "secret"))
	assert.False(t, constantTimeEqual("secret", "Secret"))
	assert.False(t, constantTimeEqual("secret", "secre"))
	assert.False(t, constantTimeEqual("secret", "secret!"))
	assert.False(t, constantTimeEqual("", "x"))
	assert.True(t, constantTimeEqual("", ""))
}

func TestNoop(t *testing.T) {
	a := Noop()

	t.Run("always passes", func(t *testing.T) {
		ctx, err := a.Authenticate(context.Background(), newRequest(nil))
		require.NoError(t, err)
		assert.Equal(t, "anonymous", ClientID(ctx))
	})

	t.Run("passes with nil request", func(t *testing.T) {
		ctx, err := a.Authenticate(context.Background(), nil)
		require.NoError(t, err)
		assert.Equal(t, "anonymous", ClientID(ctx))
	})
}

func TestContextHelpers(t *testing.T) {
	t.Run("round-trip", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithClientID(ctx, "user-42")
		assert.Equal(t, "user-42", ClientID(ctx))
	})

	t.Run("empty context returns empty string", func(t *testing.T) {
		assert.Equal(t, "", ClientID(context.Background()))
	})

	t.Run("overwrite", func(t *testing.T) {
		ctx := WithClientID(context.Background(), "first")
		ctx = WithClientID(ctx, "second")
		assert.Equal(t, "second", ClientID(ctx))
	})
}
