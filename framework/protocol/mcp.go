// Package protocol defines JSON-RPC 2.0 message types and MCP data structures.
package protocol

// ProtocolVersion is the MCP protocol version implemented by this package.
const ProtocolVersion = "2025-03-26"

// Implementation describes a client or server implementation.
type Implementation struct {
	// Name is the human-readable name of the implementation.
	Name string `json:"name"`
	// Version is the semantic version of the implementation.
	Version string `json:"version"`
}

// ServerCapabilities describes the optional capabilities a server supports.
type ServerCapabilities struct {
	// Tools indicates tool-related capabilities, if supported.
	Tools *ToolCapability `json:"tools,omitempty"`
	// Resources indicates resource-related capabilities, if supported.
	Resources *ResourceCapability `json:"resources,omitempty"`
	// Prompts indicates prompt-related capabilities, if supported.
	Prompts *PromptCapability `json:"prompts,omitempty"`
	// Logging indicates logging support. An empty struct signals support.
	Logging *struct{} `json:"logging,omitempty"`
}

// ToolCapability describes server capabilities related to tools.
type ToolCapability struct {
	// ListChanged indicates whether the server emits notifications when the tool list changes.
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourceCapability describes server capabilities related to resources.
type ResourceCapability struct {
	// Subscribe indicates whether the server supports resource subscriptions.
	Subscribe bool `json:"subscribe,omitempty"`
	// ListChanged indicates whether the server emits notifications when the resource list changes.
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptCapability describes server capabilities related to prompts.
type PromptCapability struct {
	// ListChanged indicates whether the server emits notifications when the prompt list changes.
	ListChanged bool `json:"listChanged,omitempty"`
}

// ClientCapabilities describes the optional capabilities a client supports.
type ClientCapabilities struct {
	// Roots indicates root-related capabilities, if supported.
	Roots *RootCapability `json:"roots,omitempty"`
	// Sampling indicates sampling support. An empty struct signals support.
	Sampling *struct{} `json:"sampling,omitempty"`
}

// RootCapability describes client capabilities related to roots.
type RootCapability struct {
	// ListChanged indicates whether the client emits notifications when roots change.
	ListChanged bool `json:"listChanged,omitempty"`
}

// InitializeParams holds the parameters sent by the client during initialization.
type InitializeParams struct {
	// ProtocolVersion is the protocol version the client supports.
	ProtocolVersion string `json:"protocolVersion"`
	// Capabilities describes the client's optional capabilities.
	Capabilities ClientCapabilities `json:"capabilities"`
	// ClientInfo describes the client implementation.
	ClientInfo Implementation `json:"clientInfo"`
}

// InitializeResult holds the result returned by the server during initialization.
type InitializeResult struct {
	// ProtocolVersion is the protocol version the server has selected.
	ProtocolVersion string `json:"protocolVersion"`
	// Capabilities describes the server's optional capabilities.
	Capabilities ServerCapabilities `json:"capabilities"`
	// ServerInfo describes the server implementation.
	ServerInfo Implementation `json:"serverInfo"`
	// Instructions is an optional human-readable description of how to use the server.
	Instructions string `json:"instructions,omitempty"`
}
