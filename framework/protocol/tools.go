// Package protocol defines JSON-RPC 2.0 message types and MCP data structures.
package protocol

import "encoding/json"

// ToolInfo describes a tool that the server exposes.
type ToolInfo struct {
	// Name is the unique identifier for the tool.
	Name string `json:"name"`
	// Description is a human-readable description of what the tool does.
	Description string `json:"description,omitempty"`
	// InputSchema is the JSON Schema describing the tool's expected input.
	InputSchema json.RawMessage `json:"inputSchema"`
}

// ToolListResult is the result of a tools/list request.
type ToolListResult struct {
	// Tools is the list of available tools.
	Tools []ToolInfo `json:"tools"`
	// NextCursor is an opaque token for pagination. Empty when there are no more results.
	NextCursor string `json:"nextCursor,omitempty"`
}

// CallToolParams holds the parameters for a tools/call request.
type CallToolParams struct {
	// Name is the name of the tool to invoke.
	Name string `json:"name"`
	// Arguments contains the tool input as a JSON object.
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// CallToolResult holds the result of a tools/call request.
type CallToolResult struct {
	// Content contains the output content items produced by the tool.
	Content []Content `json:"content"`
	// IsError indicates whether the tool invocation resulted in an error.
	IsError bool `json:"isError,omitempty"`
}

// Content is a union type representing different kinds of content blocks
// returned by tools, prompts, and resources.
type Content struct {
	// Type identifies the content kind (e.g. "text", "image", "resource").
	Type string `json:"type"`
	// Text holds the textual content. Used when Type is "text" or "resource".
	Text string `json:"text,omitempty"`
	// MIMEType is the MIME type of the content. Used with "image" and "resource" types.
	MIMEType string `json:"mimeType,omitempty"`
	// Data holds base64-encoded binary data. Used when Type is "image".
	Data string `json:"data,omitempty"`
	// URI is the resource URI. Used when Type is "resource".
	URI string `json:"uri,omitempty"`
}

// TextContent creates a Content block of type "text" with the given text.
func TextContent(text string) Content {
	return Content{Type: "text", Text: text}
}

// ImageContent creates a Content block of type "image" with the given MIME type
// and base64-encoded data.
func ImageContent(mimeType, base64Data string) Content {
	return Content{Type: "image", MIMEType: mimeType, Data: base64Data}
}

// EmbeddedResourceContent creates a Content block of type "resource" with the
// given URI, MIME type, and text.
func EmbeddedResourceContent(uri, mimeType, text string) Content {
	return Content{Type: "resource", URI: uri, MIMEType: mimeType, Text: text}
}
