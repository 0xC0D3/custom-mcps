// Package protocol defines JSON-RPC 2.0 message types and MCP data structures.
package protocol

import "encoding/json"

// JSONRPCVersion is the JSON-RPC protocol version supported by this package.
const JSONRPCVersion = "2.0"

// Request represents a JSON-RPC 2.0 request message.
type Request struct {
	// JSONRPC specifies the JSON-RPC protocol version; always "2.0".
	JSONRPC string `json:"jsonrpc"`
	// ID is the request identifier. It may be a string or integer.
	ID json.RawMessage `json:"id"`
	// Method is the name of the method to be invoked.
	Method string `json:"method"`
	// Params holds the parameter values for the method, if any.
	Params json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response message.
type Response struct {
	// JSONRPC specifies the JSON-RPC protocol version; always "2.0".
	JSONRPC string `json:"jsonrpc"`
	// ID matches the identifier of the request this response belongs to.
	ID json.RawMessage `json:"id"`
	// Result contains the result of the method invocation on success.
	Result json.RawMessage `json:"result,omitempty"`
	// Error contains the error object on failure.
	Error *RPCError `json:"error,omitempty"`
}

// Notification represents a JSON-RPC 2.0 notification (a request without an id).
type Notification struct {
	// JSONRPC specifies the JSON-RPC protocol version; always "2.0".
	JSONRPC string `json:"jsonrpc"`
	// Method is the name of the method to be invoked.
	Method string `json:"method"`
	// Params holds the parameter values for the method, if any.
	Params json.RawMessage `json:"params,omitempty"`
}

// NewResponse creates a successful Response by marshaling the given result value.
func NewResponse(id json.RawMessage, result any) (*Response, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Result:  raw,
	}, nil
}

// NewErrorResponse creates an error Response from the given RPCError.
func NewErrorResponse(id json.RawMessage, rpcErr *RPCError) *Response {
	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Error:   rpcErr,
	}
}

// ParseMessage parses raw JSON data and returns a *Request, *Notification, or
// *Response depending on the presence of the "id" and "method" fields.
//
// The rules are:
//   - Has "method" and "id" → *Request
//   - Has "method" but no "id" → *Notification
//   - Has "id" but no "method" → *Response
//   - Otherwise → error
func ParseMessage(data []byte) (any, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, NewParseError(err.Error())
	}

	_, hasID := raw["id"]
	_, hasMethod := raw["method"]

	switch {
	case hasMethod && hasID:
		var req Request
		if err := json.Unmarshal(data, &req); err != nil {
			return nil, NewParseError(err.Error())
		}
		return &req, nil

	case hasMethod && !hasID:
		var notif Notification
		if err := json.Unmarshal(data, &notif); err != nil {
			return nil, NewParseError(err.Error())
		}
		return &notif, nil

	case hasID && !hasMethod:
		var resp Response
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, NewParseError(err.Error())
		}
		return &resp, nil

	default:
		return nil, NewParseError("message must contain either 'id' or 'method'")
	}
}
