package protocol

import (
	"encoding/json"
	"testing"
)

func TestRequestRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		req  Request
	}{
		{
			name: "with params",
			req: Request{
				JSONRPC: JSONRPCVersion,
				ID:      json.RawMessage(`1`),
				Method:  "tools/list",
				Params:  json.RawMessage(`{"cursor":"abc"}`),
			},
		},
		{
			name: "without params",
			req: Request{
				JSONRPC: JSONRPCVersion,
				ID:      json.RawMessage(`"req-1"`),
				Method:  "initialize",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var got Request
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got.JSONRPC != tt.req.JSONRPC {
				t.Errorf("JSONRPC = %q, want %q", got.JSONRPC, tt.req.JSONRPC)
			}
			if got.Method != tt.req.Method {
				t.Errorf("Method = %q, want %q", got.Method, tt.req.Method)
			}
			if string(got.ID) != string(tt.req.ID) {
				t.Errorf("ID = %s, want %s", got.ID, tt.req.ID)
			}
			if string(got.Params) != string(tt.req.Params) {
				t.Errorf("Params = %s, want %s", got.Params, tt.req.Params)
			}
		})
	}
}

func TestResponseRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		resp Response
	}{
		{
			name: "success",
			resp: Response{
				JSONRPC: JSONRPCVersion,
				ID:      json.RawMessage(`1`),
				Result:  json.RawMessage(`{"tools":[]}`),
			},
		},
		{
			name: "error",
			resp: Response{
				JSONRPC: JSONRPCVersion,
				ID:      json.RawMessage(`2`),
				Error:   &RPCError{Code: CodeMethodNotFound, Message: "not found"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.resp)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var got Response
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got.JSONRPC != tt.resp.JSONRPC {
				t.Errorf("JSONRPC = %q, want %q", got.JSONRPC, tt.resp.JSONRPC)
			}
			if string(got.ID) != string(tt.resp.ID) {
				t.Errorf("ID = %s, want %s", got.ID, tt.resp.ID)
			}
			if string(got.Result) != string(tt.resp.Result) {
				t.Errorf("Result = %s, want %s", got.Result, tt.resp.Result)
			}
			if (got.Error == nil) != (tt.resp.Error == nil) {
				t.Errorf("Error nil mismatch: got %v, want %v", got.Error, tt.resp.Error)
			}
			if got.Error != nil && got.Error.Code != tt.resp.Error.Code {
				t.Errorf("Error.Code = %d, want %d", got.Error.Code, tt.resp.Error.Code)
			}
		})
	}
}

func TestNotificationRoundTrip(t *testing.T) {
	notif := Notification{
		JSONRPC: JSONRPCVersion,
		Method:  "notifications/initialized",
	}
	data, err := json.Marshal(notif)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Notification
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Method != notif.Method {
		t.Errorf("Method = %q, want %q", got.Method, notif.Method)
	}
	if got.Params != nil {
		t.Errorf("Params = %s, want nil", got.Params)
	}
}

func TestParseMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType string
		wantErr  bool
	}{
		{
			name:     "request",
			input:    `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
			wantType: "*protocol.Request",
		},
		{
			name:     "notification",
			input:    `{"jsonrpc":"2.0","method":"notifications/initialized"}`,
			wantType: "*protocol.Notification",
		},
		{
			name:     "response with result",
			input:    `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-03-26"}}`,
			wantType: "*protocol.Response",
		},
		{
			name:     "response with error",
			input:    `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"not found"}}`,
			wantType: "*protocol.Response",
		},
		{
			name:    "invalid json",
			input:   `{broken`,
			wantErr: true,
		},
		{
			name:    "no id or method",
			input:   `{"jsonrpc":"2.0"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := ParseMessage([]byte(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			var gotType string
			switch msg.(type) {
			case *Request:
				gotType = "*protocol.Request"
			case *Notification:
				gotType = "*protocol.Notification"
			case *Response:
				gotType = "*protocol.Response"
			default:
				t.Fatalf("unexpected type %T", msg)
			}
			if gotType != tt.wantType {
				t.Errorf("type = %s, want %s", gotType, tt.wantType)
			}
		})
	}
}

func TestErrorHelpers(t *testing.T) {
	tests := []struct {
		name     string
		err      *RPCError
		wantCode int
	}{
		{"parse error", NewParseError("bad json"), CodeParseError},
		{"method not found", NewMethodNotFound("foo/bar"), CodeMethodNotFound},
		{"invalid params", NewInvalidParams("missing name"), CodeInvalidParams},
		{"internal error", NewInternalError(errForTest("boom")), CodeInternalError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.wantCode {
				t.Errorf("Code = %d, want %d", tt.err.Code, tt.wantCode)
			}
			// Verify it implements error.
			var _ error = tt.err
			if tt.err.Error() == "" {
				t.Error("Error() returned empty string")
			}
		})
	}
}

// errForTest is a simple error type for testing.
type errForTest string

func (e errForTest) Error() string { return string(e) }

func TestContentHelpers(t *testing.T) {
	tests := []struct {
		name     string
		content  Content
		wantType string
	}{
		{
			name:     "text content",
			content:  TextContent("hello"),
			wantType: "text",
		},
		{
			name:     "image content",
			content:  ImageContent("image/png", "iVBOR..."),
			wantType: "image",
		},
		{
			name:     "resource content",
			content:  EmbeddedResourceContent("file:///a.txt", "text/plain", "data"),
			wantType: "resource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.content.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", tt.content.Type, tt.wantType)
			}
			// Round-trip through JSON.
			data, err := json.Marshal(tt.content)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var got Content
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got.Type != tt.content.Type {
				t.Errorf("round-trip Type = %q, want %q", got.Type, tt.content.Type)
			}
		})
	}
}

func TestInitializeResultOmitempty(t *testing.T) {
	result := InitializeResult{
		ProtocolVersion: ProtocolVersion,
		Capabilities:    ServerCapabilities{},
		ServerInfo:      Implementation{Name: "test", Version: "1.0"},
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}

	// Instructions should be omitted when empty.
	if _, ok := raw["instructions"]; ok {
		t.Error("expected 'instructions' to be omitted when empty")
	}

	// Capabilities sub-fields should be omitted when nil.
	var caps map[string]json.RawMessage
	if err := json.Unmarshal(raw["capabilities"], &caps); err != nil {
		t.Fatalf("unmarshal capabilities: %v", err)
	}
	for _, field := range []string{"tools", "resources", "prompts", "logging"} {
		if _, ok := caps[field]; ok {
			t.Errorf("expected capabilities.%s to be omitted when nil", field)
		}
	}
}

func TestNewResponse(t *testing.T) {
	id := json.RawMessage(`1`)
	resp, err := NewResponse(id, map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("NewResponse: %v", err)
	}
	if resp.JSONRPC != JSONRPCVersion {
		t.Errorf("JSONRPC = %q, want %q", resp.JSONRPC, JSONRPCVersion)
	}
	if string(resp.ID) != `1` {
		t.Errorf("ID = %s, want 1", resp.ID)
	}
	if resp.Error != nil {
		t.Errorf("Error should be nil, got %v", resp.Error)
	}

	var result map[string]string
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("result[key] = %q, want %q", result["key"], "value")
	}
}

func TestNewErrorResponse(t *testing.T) {
	id := json.RawMessage(`"abc"`)
	rpcErr := NewMethodNotFound("tools/call")
	resp := NewErrorResponse(id, rpcErr)

	if resp.JSONRPC != JSONRPCVersion {
		t.Errorf("JSONRPC = %q, want %q", resp.JSONRPC, JSONRPCVersion)
	}
	if resp.Result != nil {
		t.Errorf("Result should be nil, got %s", resp.Result)
	}
	if resp.Error == nil {
		t.Fatal("Error should not be nil")
	}
	if resp.Error.Code != CodeMethodNotFound {
		t.Errorf("Error.Code = %d, want %d", resp.Error.Code, CodeMethodNotFound)
	}
}
