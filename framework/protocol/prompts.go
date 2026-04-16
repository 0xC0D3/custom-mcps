// Package protocol defines JSON-RPC 2.0 message types and MCP data structures.
package protocol

// PromptInfo describes a prompt that the server exposes.
type PromptInfo struct {
	// Name is the unique identifier for the prompt.
	Name string `json:"name"`
	// Description is a human-readable description of what the prompt does.
	Description string `json:"description,omitempty"`
	// Arguments lists the arguments the prompt accepts.
	Arguments []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument describes a single argument accepted by a prompt.
type PromptArgument struct {
	// Name is the argument name.
	Name string `json:"name"`
	// Description is a human-readable description of the argument.
	Description string `json:"description,omitempty"`
	// Required indicates whether the argument must be provided.
	Required bool `json:"required,omitempty"`
}

// PromptListResult is the result of a prompts/list request.
type PromptListResult struct {
	// Prompts is the list of available prompts.
	Prompts []PromptInfo `json:"prompts"`
	// NextCursor is an opaque token for pagination. Empty when there are no more results.
	NextCursor string `json:"nextCursor,omitempty"`
}

// GetPromptParams holds the parameters for a prompts/get request.
type GetPromptParams struct {
	// Name is the name of the prompt to retrieve.
	Name string `json:"name"`
	// Arguments is a map of argument values keyed by argument name.
	Arguments map[string]string `json:"arguments,omitempty"`
}

// GetPromptResult holds the result of a prompts/get request.
type GetPromptResult struct {
	// Description is a human-readable description of the prompt.
	Description string `json:"description,omitempty"`
	// Messages is the list of messages produced by the prompt.
	Messages []PromptMessage `json:"messages"`
}

// PromptMessage represents a single message in a prompt result.
type PromptMessage struct {
	// Role is the message role (e.g. "user", "assistant").
	Role string `json:"role"`
	// Content is the message content.
	Content Content `json:"content"`
}
