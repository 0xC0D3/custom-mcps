package registry

import (
	"context"

	"github.com/0xC0D3/custom-mcps/framework/protocol"
)

// PromptHandler handles a prompts/get request. It receives the prompt
// arguments as a string map and returns the prompt result or an error.
type PromptHandler func(ctx context.Context, args map[string]string) (*protocol.GetPromptResult, error)

// PromptDefinition holds a prompt's metadata and handler.
type PromptDefinition struct {
	// Info contains the prompt's name, description, and argument definitions.
	Info protocol.PromptInfo
	// Handler is the function invoked when the prompt is retrieved.
	Handler PromptHandler
}

// RegisterPrompt registers a prompt handler. The prompt is keyed by its
// Info.Name field. If a prompt with the same name already exists it is
// silently overwritten.
func (r *Registry) RegisterPrompt(def PromptDefinition) {
	r.prompts[def.Info.Name] = &def
}
