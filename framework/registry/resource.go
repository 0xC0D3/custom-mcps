package registry

import (
	"context"

	"github.com/0xC0D3/custom-mcps/framework/protocol"
)

// ResourceHandler handles a resources/read request for a specific URI.
// It receives the requested URI and returns the read result or an error.
type ResourceHandler func(ctx context.Context, uri string) (*protocol.ReadResourceResult, error)

// ResourceDefinition holds a resource's metadata and handler.
type ResourceDefinition struct {
	// Info contains the resource's URI, name, description, and MIME type.
	Info protocol.ResourceInfo
	// Handler is the function invoked when the resource is read.
	Handler ResourceHandler
}

// RegisterResource registers a resource handler. The resource is keyed by
// its Info.URI field. If a resource with the same URI already exists it is
// silently overwritten.
func (r *Registry) RegisterResource(def ResourceDefinition) {
	r.resources[def.Info.URI] = &def
}
