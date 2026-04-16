package registry

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/0xC0D3/custom-mcps/framework/protocol"
	"github.com/0xC0D3/custom-mcps/framework/schema"
)

// ToolHandler is the function signature for handling a tools/call request.
// It receives the raw JSON parameters and returns a call result or an error.
type ToolHandler func(ctx context.Context, params json.RawMessage) (*protocol.CallToolResult, error)

// ToolDefinition holds a tool's metadata, schema, and handler.
type ToolDefinition struct {
	// Info contains the tool's name, description, and input schema.
	Info protocol.ToolInfo
	// Handler is the function invoked when the tool is called.
	Handler ToolHandler
}

// RegisterTool registers a typed tool handler. T is the input struct whose
// struct tags are used to generate the JSON Schema automatically via
// schema.Generate[T](). The typed handler receives a decoded T value instead
// of raw JSON, making tool implementations type-safe.
//
// If a tool with the same name already exists it is silently overwritten.
// This is the preferred registration method for most tools.
func RegisterTool[T any](r *Registry, name, description string, handler func(ctx context.Context, input T) (*protocol.CallToolResult, error)) {
	s := schema.Generate[T]()

	inputSchema, err := json.Marshal(s)
	if err != nil {
		panic(fmt.Sprintf("registry: failed to marshal schema for tool %q: %v", name, err))
	}

	wrapped := func(ctx context.Context, params json.RawMessage) (*protocol.CallToolResult, error) {
		var input T
		if len(params) > 0 {
			if err := json.Unmarshal(params, &input); err != nil {
				return nil, fmt.Errorf("invalid parameters for tool %q: %w", name, err)
			}
		}
		return handler(ctx, input)
	}

	r.tools[name] = &ToolDefinition{
		Info: protocol.ToolInfo{
			Name:        name,
			Description: description,
			InputSchema: inputSchema,
		},
		Handler: wrapped,
	}
}

// RegisterToolRaw registers a tool with a pre-built ToolInfo and raw handler.
// Use this for dynamic tools or when struct tags are not sufficient to
// describe the input schema.
//
// If a tool with the same name already exists it is silently overwritten.
func (r *Registry) RegisterToolRaw(def ToolDefinition) {
	r.tools[def.Info.Name] = &def
}
