package transport

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAuthenticator is a test authenticator.
type mockAuthenticator struct {
	shouldFail bool
}

func (m *mockAuthenticator) Authenticate(ctx context.Context, r *http.Request) (context.Context, error) {
	if m.shouldFail {
		return ctx, errors.New("auth failed")
	}
	return ctx, nil
}

// newStreamableTestServer creates an httptest.Server wired to a StreamableHTTPTransport's handler.
func newStreamableTestServer(t *testing.T, handler MessageHandler, opts ...StreamableOption) (*httptest.Server, *StreamableHTTPTransport) {
	t.Helper()
	tr := NewStreamableHTTP(opts...)

	mux := http.NewServeMux()
	mux.HandleFunc(tr.endpoint, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			tr.handlePost(context.Background(), w, r, handler) //nolint:contextcheck
		case http.MethodGet:
			tr.handleGet(context.Background(), w, r) //nolint:contextcheck
		case http.MethodDelete:
			tr.handleDelete(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	if tr.healthPath != "" {
		mux.HandleFunc(tr.healthPath, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		})
	}

	var h http.Handler = mux
	for i := len(tr.httpMws) - 1; i >= 0; i-- {
		h = tr.httpMws[i](h)
	}

	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	return ts, tr
}

func TestStreamable_PostValidRequest(t *testing.T) {
	handler := func(ctx context.Context, raw json.RawMessage) json.RawMessage {
		return json.RawMessage(`{"jsonrpc":"2.0","id":1,"result":"hello"}`)
	}

	ts, _ := newStreamableTestServer(t, handler)

	body := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"test"}`)
	resp, err := http.Post(ts.URL+"/mcp", "application/json", body)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("Mcp-Session-Id"))
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"jsonrpc":"2.0","id":1,"result":"hello"}`, string(respBody))
}

func TestStreamable_PostWithAuth(t *testing.T) {
	handler := func(ctx context.Context, raw json.RawMessage) json.RawMessage {
		return json.RawMessage(`{"jsonrpc":"2.0","id":1,"result":"ok"}`)
	}

	// Test with failing auth.
	ts, _ := newStreamableTestServer(t, handler, WithStreamableAuth(&mockAuthenticator{shouldFail: true}))

	body := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"test"}`)
	resp, err := http.Post(ts.URL+"/mcp", "application/json", body)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestStreamable_PostWithAuthSuccess(t *testing.T) {
	handler := func(ctx context.Context, raw json.RawMessage) json.RawMessage {
		return json.RawMessage(`{"jsonrpc":"2.0","id":1,"result":"ok"}`)
	}

	ts, _ := newStreamableTestServer(t, handler, WithStreamableAuth(&mockAuthenticator{shouldFail: false}))

	body := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"test"}`)
	resp, err := http.Post(ts.URL+"/mcp", "application/json", body)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestStreamable_GetSSEStream(t *testing.T) {
	handler := func(ctx context.Context, raw json.RawMessage) json.RawMessage {
		return json.RawMessage(`{"jsonrpc":"2.0","id":1,"result":"ok"}`)
	}

	ts, tr := newStreamableTestServer(t, handler)

	// First create a session via POST.
	body := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"init"}`)
	postResp, err := http.Post(ts.URL+"/mcp", "application/json", body)
	require.NoError(t, err)
	sessionID := postResp.Header.Get("Mcp-Session-Id")
	postResp.Body.Close()
	require.NotEmpty(t, sessionID)

	// Now GET the SSE stream.
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/mcp", nil)
	require.NoError(t, err)
	req.Header.Set("Mcp-Session-Id", sessionID)

	client := &http.Client{}
	getResp, err := client.Do(req)
	require.NoError(t, err)
	defer getResp.Body.Close()

	assert.Equal(t, http.StatusOK, getResp.StatusCode)
	assert.Equal(t, "text/event-stream", getResp.Header.Get("Content-Type"))

	// Send a message via the transport and read it from the SSE stream.
	notification := json.RawMessage(`{"jsonrpc":"2.0","method":"test/notify"}`)
	err = tr.Send(context.Background(), notification)
	require.NoError(t, err)

	scanner := bufio.NewScanner(getResp.Body)
	var dataLine string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			dataLine = strings.TrimPrefix(line, "data: ")
			break
		}
	}
	assert.JSONEq(t, `{"jsonrpc":"2.0","method":"test/notify"}`, dataLine)
}

func TestStreamable_DeleteSession(t *testing.T) {
	handler := func(ctx context.Context, raw json.RawMessage) json.RawMessage {
		return json.RawMessage(`{"jsonrpc":"2.0","id":1,"result":"ok"}`)
	}

	ts, _ := newStreamableTestServer(t, handler)

	// Create a session.
	body := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"init"}`)
	postResp, err := http.Post(ts.URL+"/mcp", "application/json", body)
	require.NoError(t, err)
	sessionID := postResp.Header.Get("Mcp-Session-Id")
	postResp.Body.Close()
	require.NotEmpty(t, sessionID)

	// Delete the session.
	req, err := http.NewRequest(http.MethodDelete, ts.URL+"/mcp", nil)
	require.NoError(t, err)
	req.Header.Set("Mcp-Session-Id", sessionID)

	client := &http.Client{}
	delResp, err := client.Do(req)
	require.NoError(t, err)
	delResp.Body.Close()
	assert.Equal(t, http.StatusNoContent, delResp.StatusCode)

	// Subsequent GET should fail.
	getReq, err := http.NewRequest(http.MethodGet, ts.URL+"/mcp", nil)
	require.NoError(t, err)
	getReq.Header.Set("Mcp-Session-Id", sessionID)

	getResp, err := client.Do(getReq)
	require.NoError(t, err)
	getResp.Body.Close()
	assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
}

func TestStreamable_HealthEndpoint(t *testing.T) {
	handler := func(ctx context.Context, raw json.RawMessage) json.RawMessage {
		return nil
	}

	ts, _ := newStreamableTestServer(t, handler)

	resp, err := http.Get(ts.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"status":"ok"}`, string(body))
}
