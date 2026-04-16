package schema

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate_BasicTypes(t *testing.T) {
	type Input struct {
		Name    string  `json:"name"`
		Age     int     `json:"age"`
		Score   float64 `json:"score"`
		Active  bool    `json:"active"`
	}

	s := Generate[Input]()

	assert.Equal(t, "object", s.Type)
	require.Contains(t, s.Properties, "name")
	require.Contains(t, s.Properties, "age")
	require.Contains(t, s.Properties, "score")
	require.Contains(t, s.Properties, "active")

	assert.Equal(t, "string", s.Properties["name"].Type)
	assert.Equal(t, "integer", s.Properties["age"].Type)
	assert.Equal(t, "number", s.Properties["score"].Type)
	assert.Equal(t, "boolean", s.Properties["active"].Type)
}

func TestGenerate_RequiredTag(t *testing.T) {
	type Input struct {
		Name     string `json:"name"     mcp:"required"`
		Optional string `json:"optional"`
		Age      int    `json:"age"      mcp:"required"`
		Desc     string `json:"desc"     mcp:"optional"`
	}

	s := Generate[Input]()

	assert.Equal(t, []string{"name", "age"}, s.Required)
}

func TestGenerate_JsonSchemaTag(t *testing.T) {
	type Input struct {
		Name   string  `json:"name"   jsonschema:"description=The user name,minLength=1,maxLength=100"`
		Role   string  `json:"role"   jsonschema:"description=User role,enum=admin|editor|viewer,default=viewer"`
		Score  float64 `json:"score"  jsonschema:"minimum=0,maximum=100"`
		Email  string  `json:"email"  jsonschema:"format=email,pattern=^.+@.+$"`
		Count  int     `json:"count"  jsonschema:"default=10"`
		Active bool    `json:"active" jsonschema:"default=true"`
	}

	s := Generate[Input]()

	// description
	assert.Equal(t, "The user name", s.Properties["name"].Description)

	// minLength / maxLength
	require.NotNil(t, s.Properties["name"].MinLength)
	require.NotNil(t, s.Properties["name"].MaxLength)
	assert.Equal(t, 1, *s.Properties["name"].MinLength)
	assert.Equal(t, 100, *s.Properties["name"].MaxLength)

	// enum
	assert.Equal(t, []any{"admin", "editor", "viewer"}, s.Properties["role"].Enum)

	// default string
	assert.Equal(t, "viewer", s.Properties["role"].Default)

	// default int (parsed as float64)
	assert.Equal(t, float64(10), s.Properties["count"].Default)

	// default bool
	assert.Equal(t, true, s.Properties["active"].Default)

	// minimum / maximum
	require.NotNil(t, s.Properties["score"].Minimum)
	require.NotNil(t, s.Properties["score"].Maximum)
	assert.Equal(t, float64(0), *s.Properties["score"].Minimum)
	assert.Equal(t, float64(100), *s.Properties["score"].Maximum)

	// format / pattern
	assert.Equal(t, "email", s.Properties["email"].Format)
	assert.Equal(t, "^.+@.+$", s.Properties["email"].Pattern)
}

func TestGenerate_SliceType(t *testing.T) {
	type Inner struct {
		ID int `json:"id"`
	}
	type Input struct {
		Tags   []string `json:"tags"`
		Counts []int    `json:"counts"`
		Items  []Inner  `json:"items"`
	}

	s := Generate[Input]()

	// []string
	require.Contains(t, s.Properties, "tags")
	assert.Equal(t, "array", s.Properties["tags"].Type)
	require.NotNil(t, s.Properties["tags"].Items)
	assert.Equal(t, "string", s.Properties["tags"].Items.Type)

	// []int
	require.Contains(t, s.Properties, "counts")
	assert.Equal(t, "array", s.Properties["counts"].Type)
	require.NotNil(t, s.Properties["counts"].Items)
	assert.Equal(t, "integer", s.Properties["counts"].Items.Type)

	// []struct
	require.Contains(t, s.Properties, "items")
	assert.Equal(t, "array", s.Properties["items"].Type)
	require.NotNil(t, s.Properties["items"].Items)
	assert.Equal(t, "object", s.Properties["items"].Items.Type)
	require.Contains(t, s.Properties["items"].Items.Properties, "id")
	assert.Equal(t, "integer", s.Properties["items"].Items.Properties["id"].Type)
}

func TestGenerate_MapType(t *testing.T) {
	type Input struct {
		Metadata map[string]string `json:"metadata"`
	}

	s := Generate[Input]()

	require.Contains(t, s.Properties, "metadata")
	assert.Equal(t, "object", s.Properties["metadata"].Type)
	require.NotNil(t, s.Properties["metadata"].AdditionalProperties)
	assert.Equal(t, "string", s.Properties["metadata"].AdditionalProperties.Type)
}

func TestGenerate_NestedStruct(t *testing.T) {
	type Address struct {
		Street string `json:"street"`
		City   string `json:"city"`
	}
	type Input struct {
		Name    string  `json:"name"`
		Address Address `json:"address"`
	}

	s := Generate[Input]()

	require.Contains(t, s.Properties, "address")
	addr := s.Properties["address"]
	assert.Equal(t, "object", addr.Type)
	require.Contains(t, addr.Properties, "street")
	require.Contains(t, addr.Properties, "city")
	assert.Equal(t, "string", addr.Properties["street"].Type)
	assert.Equal(t, "string", addr.Properties["city"].Type)
}

func TestGenerate_PointerField(t *testing.T) {
	type Input struct {
		Name  *string `json:"name"`
		Count *int    `json:"count"`
	}

	s := Generate[Input]()

	require.Contains(t, s.Properties, "name")
	require.Contains(t, s.Properties, "count")
	assert.Equal(t, "string", s.Properties["name"].Type)
	assert.Equal(t, "integer", s.Properties["count"].Type)
}

func TestGenerate_TimeField(t *testing.T) {
	type Input struct {
		CreatedAt time.Time  `json:"createdAt"`
		UpdatedAt *time.Time `json:"updatedAt"`
	}

	s := Generate[Input]()

	require.Contains(t, s.Properties, "createdAt")
	assert.Equal(t, "string", s.Properties["createdAt"].Type)
	assert.Equal(t, "date-time", s.Properties["createdAt"].Format)

	require.Contains(t, s.Properties, "updatedAt")
	assert.Equal(t, "string", s.Properties["updatedAt"].Type)
	assert.Equal(t, "date-time", s.Properties["updatedAt"].Format)
}

func TestGenerate_JsonDash(t *testing.T) {
	type Input struct {
		Visible string `json:"visible"`
		Hidden  string `json:"-"`
	}

	s := Generate[Input]()

	assert.Contains(t, s.Properties, "visible")
	assert.NotContains(t, s.Properties, "Hidden")
	assert.NotContains(t, s.Properties, "-")
}

func TestGenerate_UnexportedField(t *testing.T) {
	type Input struct {
		Public  string `json:"public"`
		private string //nolint:unused
	}

	s := Generate[Input]()

	assert.Contains(t, s.Properties, "public")
	assert.Len(t, s.Properties, 1)
}

func TestGenerate_EmbeddedStruct(t *testing.T) {
	type Base struct {
		ID   string `json:"id"   mcp:"required"`
		Name string `json:"name"`
	}
	type Input struct {
		Base
		Extra string `json:"extra"`
	}

	s := Generate[Input]()

	// Embedded fields should be flattened into the parent.
	assert.Contains(t, s.Properties, "id")
	assert.Contains(t, s.Properties, "name")
	assert.Contains(t, s.Properties, "extra")
	assert.Equal(t, []string{"id"}, s.Required)
}

func TestGenerate_ComplexExample(t *testing.T) {
	type DNSLookupInput struct {
		Domain     string   `json:"domain"     mcp:"required" jsonschema:"description=The domain name to look up"`
		RecordType string   `json:"recordType" mcp:"required" jsonschema:"description=DNS record type,enum=A|AAAA|CNAME|MX|TXT"`
		Nameserver string   `json:"nameserver"                jsonschema:"description=Custom nameserver,default=1.1.1.1"`
		Timeout    int      `json:"timeout"                   jsonschema:"description=Timeout in seconds,minimum=1,maximum=30"`
		Tags       []string `json:"tags"                      jsonschema:"description=Optional tags"`
	}

	s := Generate[DNSLookupInput]()

	// Verify overall structure.
	assert.Equal(t, "object", s.Type)
	assert.Equal(t, []string{"domain", "recordType"}, s.Required)
	assert.Len(t, s.Properties, 5)

	// domain
	assert.Equal(t, "string", s.Properties["domain"].Type)
	assert.Equal(t, "The domain name to look up", s.Properties["domain"].Description)

	// recordType
	assert.Equal(t, "string", s.Properties["recordType"].Type)
	assert.Equal(t, "DNS record type", s.Properties["recordType"].Description)
	assert.Equal(t, []any{"A", "AAAA", "CNAME", "MX", "TXT"}, s.Properties["recordType"].Enum)

	// nameserver
	assert.Equal(t, "string", s.Properties["nameserver"].Type)
	assert.Equal(t, "1.1.1.1", s.Properties["nameserver"].Default)

	// timeout
	assert.Equal(t, "integer", s.Properties["timeout"].Type)
	require.NotNil(t, s.Properties["timeout"].Minimum)
	require.NotNil(t, s.Properties["timeout"].Maximum)
	assert.Equal(t, float64(1), *s.Properties["timeout"].Minimum)
	assert.Equal(t, float64(30), *s.Properties["timeout"].Maximum)

	// tags
	assert.Equal(t, "array", s.Properties["tags"].Type)
	require.NotNil(t, s.Properties["tags"].Items)
	assert.Equal(t, "string", s.Properties["tags"].Items.Type)

	// Verify JSON marshaling produces valid output.
	data, err := json.MarshalIndent(s, "", "  ")
	require.NoError(t, err)

	var roundTripped map[string]any
	err = json.Unmarshal(data, &roundTripped)
	require.NoError(t, err)
	assert.Equal(t, "object", roundTripped["type"])

	props, ok := roundTripped["properties"].(map[string]any)
	require.True(t, ok)
	assert.Len(t, props, 5)
}

func TestGenerate_RawMessage(t *testing.T) {
	type Input struct {
		Data json.RawMessage `json:"data"`
	}

	s := Generate[Input]()

	require.Contains(t, s.Properties, "data")
	assert.Equal(t, "", s.Properties["data"].Type)
}
