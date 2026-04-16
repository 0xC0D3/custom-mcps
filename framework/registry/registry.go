// Package registry manages tool, resource, and prompt registration
// for MCP servers.
//
// A Registry stores definitions for tools, resources, and prompts,
// each paired with a handler function. It is safe for concurrent read
// access after registration is complete. All registration should happen
// before the server starts accepting requests.
package registry

import (
	"github.com/0xC0D3/custom-mcps/framework/protocol"
)

// Registry stores registered tools, resources, and prompts.
// It is safe for concurrent read access after registration is complete.
// Registration should happen before the server starts.
type Registry struct {
	tools     map[string]*ToolDefinition
	resources map[string]*ResourceDefinition
	prompts   map[string]*PromptDefinition
}

// New creates an empty Registry ready for registration.
func New() *Registry {
	return &Registry{
		tools:     make(map[string]*ToolDefinition),
		resources: make(map[string]*ResourceDefinition),
		prompts:   make(map[string]*PromptDefinition),
	}
}

// ListTools returns all registered tools as a slice of protocol.ToolInfo.
// The order of the returned slice is not guaranteed.
func (r *Registry) ListTools() []protocol.ToolInfo {
	result := make([]protocol.ToolInfo, 0, len(r.tools))
	for _, def := range r.tools {
		result = append(result, def.Info)
	}
	return result
}

// GetTool returns a tool definition by name, or nil if not found.
func (r *Registry) GetTool(name string) *ToolDefinition {
	return r.tools[name]
}

// ListResources returns all registered resources as a slice of protocol.ResourceInfo.
// The order of the returned slice is not guaranteed.
func (r *Registry) ListResources() []protocol.ResourceInfo {
	result := make([]protocol.ResourceInfo, 0, len(r.resources))
	for _, def := range r.resources {
		result = append(result, def.Info)
	}
	return result
}

// GetResource returns a resource definition by URI, or nil if not found.
func (r *Registry) GetResource(uri string) *ResourceDefinition {
	return r.resources[uri]
}

// ListPrompts returns all registered prompts as a slice of protocol.PromptInfo.
// The order of the returned slice is not guaranteed.
func (r *Registry) ListPrompts() []protocol.PromptInfo {
	result := make([]protocol.PromptInfo, 0, len(r.prompts))
	for _, def := range r.prompts {
		result = append(result, def.Info)
	}
	return result
}

// GetPrompt returns a prompt definition by name, or nil if not found.
func (r *Registry) GetPrompt(name string) *PromptDefinition {
	return r.prompts[name]
}
