// Package protocol defines JSON-RPC 2.0 message types and MCP data structures.
package protocol

// ResourceInfo describes a concrete resource exposed by the server.
type ResourceInfo struct {
	// URI is the unique identifier for the resource.
	URI string `json:"uri"`
	// Name is the human-readable name of the resource.
	Name string `json:"name"`
	// Description is a human-readable description of the resource.
	Description string `json:"description,omitempty"`
	// MIMEType is the MIME type of the resource content.
	MIMEType string `json:"mimeType,omitempty"`
}

// ResourceTemplate describes a URI template for dynamically-addressable resources.
type ResourceTemplate struct {
	// URITemplate is the URI template (RFC 6570) for matching resources.
	URITemplate string `json:"uriTemplate"`
	// Name is the human-readable name of the resource template.
	Name string `json:"name"`
	// Description is a human-readable description of the resource template.
	Description string `json:"description,omitempty"`
	// MIMEType is the MIME type of resources matching this template.
	MIMEType string `json:"mimeType,omitempty"`
}

// ResourceListResult is the result of a resources/list request.
type ResourceListResult struct {
	// Resources is the list of available resources.
	Resources []ResourceInfo `json:"resources"`
	// NextCursor is an opaque token for pagination. Empty when there are no more results.
	NextCursor string `json:"nextCursor,omitempty"`
}

// ResourceTemplateListResult is the result of a resources/templates/list request.
type ResourceTemplateListResult struct {
	// ResourceTemplates is the list of available resource templates.
	ResourceTemplates []ResourceTemplate `json:"resourceTemplates"`
	// NextCursor is an opaque token for pagination. Empty when there are no more results.
	NextCursor string `json:"nextCursor,omitempty"`
}

// ReadResourceParams holds the parameters for a resources/read request.
type ReadResourceParams struct {
	// URI is the URI of the resource to read.
	URI string `json:"uri"`
}

// ReadResourceResult holds the result of a resources/read request.
type ReadResourceResult struct {
	// Contents contains the resource content items.
	Contents []ResourceContent `json:"contents"`
}

// ResourceContent represents the content of a single resource.
type ResourceContent struct {
	// URI is the URI of the resource.
	URI string `json:"uri"`
	// MIMEType is the MIME type of the content.
	MIMEType string `json:"mimeType,omitempty"`
	// Text holds the textual content of the resource.
	Text string `json:"text,omitempty"`
	// Blob holds base64-encoded binary content of the resource.
	Blob string `json:"blob,omitempty"`
}
