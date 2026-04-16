package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStdioTransport_RequestResponse(t *testing.T) {
	in, out := &bytes.Buffer{}, &bytes.Buffer{}

	// Write a valid JSON-RPC request to the input.
	reqMsg := `{"jsonrpc":"2.0","id":1,"method":"test","params":{}}` + "\n"
	in.WriteString(reqMsg)

	transport := NewStdio(
		WithStdioInput(in),
		WithStdioOutput(out),
	)

	handlerCalled := false
	err := transport.Start(context.Background(), func(ctx context.Context, raw json.RawMessage) json.RawMessage {
		handlerCalled = true
		resp := `{"jsonrpc":"2.0","id":1,"result":"ok"}`
		return json.RawMessage(resp)
	})

	require.NoError(t, err)
	assert.True(t, handlerCalled, "handler should have been called")

	// Verify response line appeared on output.
	outStr := strings.TrimSpace(out.String())
	assert.JSONEq(t, `{"jsonrpc":"2.0","id":1,"result":"ok"}`, outStr)
}

func TestStdioTransport_Notification_NoResponse(t *testing.T) {
	in, out := &bytes.Buffer{}, &bytes.Buffer{}

	// A notification has no id.
	notif := `{"jsonrpc":"2.0","method":"notifications/test","params":{}}` + "\n"
	in.WriteString(notif)

	transport := NewStdio(
		WithStdioInput(in),
		WithStdioOutput(out),
	)

	handlerCalled := false
	err := transport.Start(context.Background(), func(ctx context.Context, raw json.RawMessage) json.RawMessage {
		handlerCalled = true
		return nil // notification, no response
	})

	require.NoError(t, err)
	assert.True(t, handlerCalled)
	assert.Empty(t, out.String(), "no response should be written for notifications")
}

func TestStdioTransport_ContextCancel(t *testing.T) {
	// Use a pipe so the scanner blocks waiting for input.
	pr, pw := io.Pipe()
	out := &bytes.Buffer{}

	transport := NewStdio(
		WithStdioInput(pr),
		WithStdioOutput(out),
	)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- transport.Start(ctx, func(ctx context.Context, raw json.RawMessage) json.RawMessage {
			return nil
		})
	}()

	// Give it a moment to start, then cancel.
	time.Sleep(50 * time.Millisecond)
	cancel()
	// Close the pipe writer so the scanner unblocks.
	pw.Close()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after context cancel")
	}
}

func TestStdioTransport_MalformedJSON(t *testing.T) {
	in, out := &bytes.Buffer{}, &bytes.Buffer{}

	in.WriteString("this is not json\n")

	transport := NewStdio(
		WithStdioInput(in),
		WithStdioOutput(out),
	)

	handlerCalled := false
	err := transport.Start(context.Background(), func(ctx context.Context, raw json.RawMessage) json.RawMessage {
		handlerCalled = true
		// The handler receives the raw bytes; it's up to it to handle malformed input.
		return json.RawMessage(`{"jsonrpc":"2.0","id":null,"error":{"code":-32700,"message":"parse error"}}`)
	})

	require.NoError(t, err)
	assert.True(t, handlerCalled, "handler should be called even with malformed JSON")
	assert.NotEmpty(t, out.String())
}

func TestStdioTransport_Send(t *testing.T) {
	out := &bytes.Buffer{}
	transport := NewStdio(
		WithStdioInput(&bytes.Buffer{}),
		WithStdioOutput(out),
	)

	msg := json.RawMessage(`{"jsonrpc":"2.0","method":"server/notify","params":{}}`)
	err := transport.Send(context.Background(), msg)
	require.NoError(t, err)

	outStr := strings.TrimSpace(out.String())
	assert.JSONEq(t, `{"jsonrpc":"2.0","method":"server/notify","params":{}}`, outStr)
}

func TestStdioTransport_Close(t *testing.T) {
	transport := NewStdio()
	assert.NoError(t, transport.Close())
}
