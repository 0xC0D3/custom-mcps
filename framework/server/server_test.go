package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/0xC0D3/custom-mcps/framework/protocol"
	"github.com/0xC0D3/custom-mcps/framework/registry"
	"github.com/0xC0D3/custom-mcps/framework/transport"
)

// testHelper holds the test server infrastructure.
type testHelper struct {
	server  *Server
	writer  io.WriteCloser
	scanner *bufio.Scanner
	cancel  context.CancelFunc
	done    chan error
}

func newTestHelper(t *testing.T, setup ...func(s *Server)) *testHelper {
	t.Helper()

	// clientWriter -> serverReader: client writes requests, server reads them.
	serverReader, clientWriter := io.Pipe()
	// serverWriter -> clientReader: server writes responses, client reads them.
	clientReader, serverWriter := io.Pipe()

	srv := New(
		WithTransport(transport.NewStdio(
			transport.WithStdioInput(serverReader),
			transport.WithStdioOutput(serverWriter),
		)),
		WithName("test-server"),
		WithVersion("0.1.0"),
	)

	for _, fn := range setup {
		fn(srv)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- srv.Run(ctx)
	}()

	scanner := bufio.NewScanner(clientReader)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	return &testHelper{
		server:  srv,
		writer:  clientWriter,
		scanner: scanner,
		cancel:  cancel,
		done:    done,
	}
}

func (h *testHelper) send(t *testing.T, msg any) {
	t.Helper()
	data, err := json.Marshal(msg)
	require.NoError(t, err)
	_, err = h.writer.Write(append(data, '\n'))
	require.NoError(t, err)
}

func (h *testHelper) recv(t *testing.T) json.RawMessage {
	t.Helper()
	ok := h.scanner.Scan()
	require.True(t, ok, "expected to read a response line")
	return json.RawMessage(h.scanner.Bytes())
}

func (h *testHelper) close(t *testing.T) {
	t.Helper()
	h.cancel()
	h.writer.Close()
	select {
	case err := <-h.done:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("server did not shut down in time")
	}
}

func (h *testHelper) initialize(t *testing.T) {
	t.Helper()
	h.send(t, protocol.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  json.RawMessage(`{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}`),
	})
	_ = h.recv(t) // consume initialize response

	h.send(t, protocol.Notification{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	})
}

func TestServer_Initialize(t *testing.T) {
	h := newTestHelper(t)
	defer h.close(t)

	h.send(t, protocol.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  json.RawMessage(`{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0"}}`),
	})

	raw := h.recv(t)

	var resp protocol.Response
	require.NoError(t, json.Unmarshal(raw, &resp))
	assert.Nil(t, resp.Error)

	var result protocol.InitializeResult
	require.NoError(t, json.Unmarshal(resp.Result, &result))

	assert.Equal(t, protocol.ProtocolVersion, result.ProtocolVersion)
	assert.Equal(t, "test-server", result.ServerInfo.Name)
	assert.Equal(t, "0.1.0", result.ServerInfo.Version)
}

func TestServer_Ping(t *testing.T) {
	h := newTestHelper(t)
	defer h.close(t)

	h.initialize(t)

	h.send(t, protocol.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  "ping",
	})

	raw := h.recv(t)

	var resp protocol.Response
	require.NoError(t, json.Unmarshal(raw, &resp))
	assert.Nil(t, resp.Error)
	assert.JSONEq(t, `{}`, string(resp.Result))
}

func TestServer_ToolsList(t *testing.T) {
	type EchoInput struct {
		Message string `json:"message" description:"The message to echo"`
	}

	h := newTestHelper(t, func(s *Server) {
		Tool(s, "echo", "Echoes input", func(ctx context.Context, input EchoInput) (*protocol.CallToolResult, error) {
			return &protocol.CallToolResult{
				Content: []protocol.Content{protocol.TextContent(input.Message)},
			}, nil
		})
	})
	defer h.close(t)

	h.initialize(t)

	h.send(t, protocol.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`3`),
		Method:  "tools/list",
	})

	raw := h.recv(t)

	var resp protocol.Response
	require.NoError(t, json.Unmarshal(raw, &resp))
	assert.Nil(t, resp.Error)

	var result protocol.ToolListResult
	require.NoError(t, json.Unmarshal(resp.Result, &result))
	require.Len(t, result.Tools, 1)
	assert.Equal(t, "echo", result.Tools[0].Name)
	assert.Equal(t, "Echoes input", result.Tools[0].Description)
}

func TestServer_ToolsCall(t *testing.T) {
	type AddInput struct {
		A int `json:"a"`
		B int `json:"b"`
	}

	h := newTestHelper(t, func(s *Server) {
		Tool(s, "add", "Adds two numbers", func(ctx context.Context, input AddInput) (*protocol.CallToolResult, error) {
			sum := input.A + input.B
			return &protocol.CallToolResult{
				Content: []protocol.Content{protocol.TextContent(fmt.Sprintf("%d", sum))},
			}, nil
		})
	})
	defer h.close(t)

	h.initialize(t)

	h.send(t, protocol.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`4`),
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"add","arguments":{"a":2,"b":3}}`),
	})

	raw := h.recv(t)

	var resp protocol.Response
	require.NoError(t, json.Unmarshal(raw, &resp))
	assert.Nil(t, resp.Error)

	var result protocol.CallToolResult
	require.NoError(t, json.Unmarshal(resp.Result, &result))
	require.Len(t, result.Content, 1)
	assert.Equal(t, "text", result.Content[0].Type)
	assert.Equal(t, "5", result.Content[0].Text)
}

func TestServer_ToolsCall_UnknownTool(t *testing.T) {
	h := newTestHelper(t)
	defer h.close(t)

	h.initialize(t)

	h.send(t, protocol.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`5`),
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"nonexistent"}`),
	})

	raw := h.recv(t)

	var resp protocol.Response
	require.NoError(t, json.Unmarshal(raw, &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, protocol.CodeInvalidParams, resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "nonexistent")
}

func TestServer_ResourcesList(t *testing.T) {
	h := newTestHelper(t, func(s *Server) {
		s.Registry().RegisterResource(registry.ResourceDefinition{
			Info: protocol.ResourceInfo{
				URI:         "file:///test.txt",
				Name:        "test.txt",
				Description: "A test file",
				MIMEType:    "text/plain",
			},
			Handler: func(ctx context.Context, uri string) (*protocol.ReadResourceResult, error) {
				return &protocol.ReadResourceResult{
					Contents: []protocol.ResourceContent{
						{URI: uri, MIMEType: "text/plain", Text: "hello"},
					},
				}, nil
			},
		})
	})
	defer h.close(t)

	h.initialize(t)

	h.send(t, protocol.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`6`),
		Method:  "resources/list",
	})

	raw := h.recv(t)

	var resp protocol.Response
	require.NoError(t, json.Unmarshal(raw, &resp))
	assert.Nil(t, resp.Error)

	var result protocol.ResourceListResult
	require.NoError(t, json.Unmarshal(resp.Result, &result))
	require.Len(t, result.Resources, 1)
	assert.Equal(t, "file:///test.txt", result.Resources[0].URI)
	assert.Equal(t, "test.txt", result.Resources[0].Name)
}

func TestServer_PromptsList(t *testing.T) {
	h := newTestHelper(t, func(s *Server) {
		s.Registry().RegisterPrompt(registry.PromptDefinition{
			Info: protocol.PromptInfo{
				Name:        "greet",
				Description: "Generates a greeting",
				Arguments: []protocol.PromptArgument{
					{Name: "name", Description: "Name to greet", Required: true},
				},
			},
			Handler: func(ctx context.Context, args map[string]string) (*protocol.GetPromptResult, error) {
				return &protocol.GetPromptResult{
					Description: "A greeting",
					Messages: []protocol.PromptMessage{
						{Role: "user", Content: protocol.Content{Type: "text", Text: "Hello " + args["name"]}},
					},
				}, nil
			},
		})
	})
	defer h.close(t)

	h.initialize(t)

	h.send(t, protocol.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`7`),
		Method:  "prompts/list",
	})

	raw := h.recv(t)

	var resp protocol.Response
	require.NoError(t, json.Unmarshal(raw, &resp))
	assert.Nil(t, resp.Error)

	var result protocol.PromptListResult
	require.NoError(t, json.Unmarshal(resp.Result, &result))
	require.Len(t, result.Prompts, 1)
	assert.Equal(t, "greet", result.Prompts[0].Name)
}

func TestServer_UnknownMethod(t *testing.T) {
	h := newTestHelper(t)
	defer h.close(t)

	h.initialize(t)

	h.send(t, protocol.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`8`),
		Method:  "foo/bar",
	})

	raw := h.recv(t)

	var resp protocol.Response
	require.NoError(t, json.Unmarshal(raw, &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, protocol.CodeMethodNotFound, resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "foo/bar")
}

func TestServer_InitializeCapabilities(t *testing.T) {
	type Input struct {
		X string `json:"x"`
	}

	h := newTestHelper(t, func(s *Server) {
		Tool(s, "t1", "a tool", func(ctx context.Context, input Input) (*protocol.CallToolResult, error) {
			return &protocol.CallToolResult{Content: []protocol.Content{protocol.TextContent("ok")}}, nil
		})
		s.Registry().RegisterResource(registry.ResourceDefinition{
			Info: protocol.ResourceInfo{URI: "r://1", Name: "r1"},
			Handler: func(ctx context.Context, uri string) (*protocol.ReadResourceResult, error) {
				return &protocol.ReadResourceResult{}, nil
			},
		})
		s.Registry().RegisterPrompt(registry.PromptDefinition{
			Info: protocol.PromptInfo{Name: "p1"},
			Handler: func(ctx context.Context, args map[string]string) (*protocol.GetPromptResult, error) {
				return &protocol.GetPromptResult{}, nil
			},
		})
	})
	defer h.close(t)

	h.send(t, protocol.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  json.RawMessage(`{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}`),
	})

	raw := h.recv(t)

	var resp protocol.Response
	require.NoError(t, json.Unmarshal(raw, &resp))
	assert.Nil(t, resp.Error)

	var result protocol.InitializeResult
	require.NoError(t, json.Unmarshal(resp.Result, &result))

	assert.NotNil(t, result.Capabilities.Tools, "should advertise tools capability")
	assert.NotNil(t, result.Capabilities.Resources, "should advertise resources capability")
	assert.NotNil(t, result.Capabilities.Prompts, "should advertise prompts capability")
}
