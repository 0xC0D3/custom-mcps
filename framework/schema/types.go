// Package schema provides JSON Schema generation from Go struct types.
//
// It supports struct tag-driven schema construction using json, mcp, and
// jsonschema tags, producing JSON Schema 2020-12 compatible output suitable
// for MCP tool input definitions.
package schema

// JSONSchema represents a JSON Schema 2020-12 document or sub-schema.
type JSONSchema struct {
	Type                 string                 `json:"type"`
	Properties           map[string]*JSONSchema `json:"properties,omitempty"`
	Required             []string               `json:"required,omitempty"`
	Description          string                 `json:"description,omitempty"`
	Enum                 []any                  `json:"enum,omitempty"`
	Default              any                    `json:"default,omitempty"`
	Minimum              *float64               `json:"minimum,omitempty"`
	Maximum              *float64               `json:"maximum,omitempty"`
	MinLength            *int                   `json:"minLength,omitempty"`
	MaxLength            *int                   `json:"maxLength,omitempty"`
	Pattern              string                 `json:"pattern,omitempty"`
	Format               string                 `json:"format,omitempty"`
	Items                *JSONSchema            `json:"items,omitempty"`
	AdditionalProperties *JSONSchema            `json:"additionalProperties,omitempty"`
}
