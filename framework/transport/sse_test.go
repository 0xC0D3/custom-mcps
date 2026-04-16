package transport

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newSSETestServer creates an httptest.Server wired to an SSETransport's handlers.
//
//nolint:unparam // opts kept for future use.
func newSSETestServer(t *testing.T, handler MessageHandler, opts ...SSEOption) (*httptest.Server, *SSETransport) {
	t.Helper()
	tr := NewSSE(opts...)

	mux := http.NewServeMux()
	mux.HandleFunc(tr.sseEndpoint, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		tr.handleSSE(context.Background(), w, r) //nolint:contextcheck
	})

	mux.HandleFunc(tr.msgEndpoint, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		tr.handleMessages(context.Background(), w, r, handler) //nolint:contextcheck
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

func TestSSE_GetSSEReturnsEndpointEvent(t *testing.T) {
	handler := func(ctx context.Context, raw json.RawMessage) json.RawMessage {
		return nil
	}

	ts, _ := newSSETestServer(t, handler)

	resp, err := http.Get(ts.URL + "/sse")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

	// Read the initial endpoint event.
	scanner := bufio.NewScanner(resp.Body)
	var eventLine, dataLine string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			eventLine = strings.TrimPrefix(line, "event: ")
		}
		if strings.HasPrefix(line, "data: ") {
			dataLine = strings.TrimPrefix(line, "data: ")
			break
		}
	}

	assert.Equal(t, "endpoint", eventLine)
	assert.Contains(t, dataLine, `"endpoint":"/messages?sessionId=`)
}

func TestSSE_PostMessages_ValidSession(t *testing.T) {
	handler := func(ctx context.Context, raw json.RawMessage) json.RawMessage {
		return json.RawMessage(`{"jsonrpc":"2.0","id":1,"result":"ok"}`)
	}

	ts, tr := newSSETestServer(t, handler)

	// Create a session directly.
	s := tr.sessions.Create()

	// POST a message with the session ID.
	body := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"test"}`)
	resp, err := http.Post(ts.URL+"/messages?sessionId="+s.id, "application/json", body)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
}

func TestSSE_PostMessages_InvalidSession(t *testing.T) {
	handler := func(ctx context.Context, raw json.RawMessage) json.RawMessage {
		return nil
	}

	ts, _ := newSSETestServer(t, handler)

	body := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"test"}`)
	resp, err := http.Post(ts.URL+"/messages?sessionId=nonexistent", "application/json", body)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestSSE_PostMessages_MissingSessionId(t *testing.T) {
	handler := func(ctx context.Context, raw json.RawMessage) json.RawMessage {
		return nil
	}

	ts, _ := newSSETestServer(t, handler)

	body := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"test"}`)
	resp, err := http.Post(ts.URL+"/messages", "application/json", body)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
