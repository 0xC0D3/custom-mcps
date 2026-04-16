// Package protocol defines JSON-RPC 2.0 message types and MCP (Model Context
// Protocol) data structures used by the server framework.
package protocol

import "fmt"

// Standard JSON-RPC 2.0 error codes.
const (
	// CodeParseError indicates that invalid JSON was received by the server.
	CodeParseError = -32700

	// CodeInvalidRequest indicates that the JSON sent is not a valid Request object.
	CodeInvalidRequest = -32600

	// CodeMethodNotFound indicates that the method does not exist or is not available.
	CodeMethodNotFound = -32601

	// CodeInvalidParams indicates invalid method parameter(s).
	CodeInvalidParams = -32602

	// CodeInternalError indicates an internal JSON-RPC error.
	CodeInternalError = -32603
)

// RPCError represents a JSON-RPC 2.0 error object.
type RPCError struct {
	// Code is the integer error code.
	Code int `json:"code"`
	// Message is a short description of the error.
	Message string `json:"message"`
	// Data contains additional information about the error.
	Data any `json:"data,omitempty"`
}

// Error implements the error interface for RPCError.
func (e *RPCError) Error() string {
	if e.Data != nil {
		return fmt.Sprintf("rpc error %d: %s (%v)", e.Code, e.Message, e.Data)
	}
	return fmt.Sprintf("rpc error %d: %s", e.Code, e.Message)
}

// NewParseError creates an RPCError for parse failures.
func NewParseError(msg string) *RPCError {
	return &RPCError{Code: CodeParseError, Message: msg}
}

// NewMethodNotFound creates an RPCError indicating the given method is not available.
func NewMethodNotFound(method string) *RPCError {
	return &RPCError{Code: CodeMethodNotFound, Message: fmt.Sprintf("method not found: %s", method)}
}

// NewInvalidParams creates an RPCError for invalid method parameters.
func NewInvalidParams(msg string) *RPCError {
	return &RPCError{Code: CodeInvalidParams, Message: msg}
}

// NewInternalError creates an RPCError wrapping an internal error.
func NewInternalError(err error) *RPCError {
	return &RPCError{Code: CodeInternalError, Message: err.Error()}
}
