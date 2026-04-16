package schema

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Generate produces a JSON Schema from the exported fields of struct type T.
// It reads json, mcp, and jsonschema struct tags to determine field names,
// required status, and schema attributes.
//
// The json tag controls the property name in the schema. The mcp tag controls
// whether a field is required. The jsonschema tag provides additional schema
// metadata such as description, enum values, defaults, and validation
// constraints.
func Generate[T any]() *JSONSchema {
	t := reflect.TypeOf((*T)(nil)).Elem()
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return &JSONSchema{Type: schemaType(t)}
	}

	root := &JSONSchema{
		Type:       "object",
		Properties: map[string]*JSONSchema{},
	}

	processStructFields(t, root)
	return root
}

// processStructFields iterates the fields of a struct type and populates the
// given root schema with properties derived from struct tags.
func processStructFields(t reflect.Type, root *JSONSchema) {
	for i := range t.NumField() {
		field := t.Field(i)

		// Skip unexported fields.
		if !field.IsExported() {
			continue
		}

		// Handle embedded (anonymous) structs by flattening.
		if field.Anonymous {
			ft := field.Type
			for ft.Kind() == reflect.Ptr {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				processStructFields(ft, root)
				continue
			}
		}

		// Determine the JSON property name from the json tag.
		name := field.Name
		if tag, ok := field.Tag.Lookup("json"); ok {
			parts := strings.SplitN(tag, ",", 2)
			if parts[0] == "-" {
				continue
			}
			if parts[0] != "" {
				name = parts[0]
			}
		}

		// Build the field schema from the Go type.
		fieldSchema := schemaFromType(field.Type)

		// Parse the mcp tag for required status.
		if mcp := field.Tag.Get("mcp"); mcp == "required" {
			root.Required = append(root.Required, name)
		}

		// Parse the jsonschema tag for additional attributes.
		if jsTag := field.Tag.Get("jsonschema"); jsTag != "" {
			applyJSONSchemaTag(jsTag, fieldSchema, field.Type)
		}

		root.Properties[name] = fieldSchema
	}
}

// schemaFromType builds a JSONSchema for the given reflect.Type, recursing
// into slices, maps, structs, and pointers as needed.
func schemaFromType(t reflect.Type) *JSONSchema {
	// Dereference pointers.
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Special-case time.Time.
	if t == reflect.TypeOf(time.Time{}) {
		return &JSONSchema{Type: "string", Format: "date-time"}
	}

	// Special-case json.RawMessage.
	if t == reflect.TypeOf(json.RawMessage{}) {
		return &JSONSchema{}
	}

	switch t.Kind() {
	case reflect.Slice:
		items := schemaFromType(t.Elem())
		return &JSONSchema{Type: "array", Items: items}

	case reflect.Map:
		valSchema := schemaFromType(t.Elem())
		return &JSONSchema{Type: "object", AdditionalProperties: valSchema}

	case reflect.Struct:
		s := &JSONSchema{
			Type:       "object",
			Properties: map[string]*JSONSchema{},
		}
		processStructFields(t, s)
		return s

	default:
		return &JSONSchema{Type: schemaType(t)}
	}
}

// schemaType maps a reflect.Type to its JSON Schema type string.
func schemaType(t reflect.Type) string {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "integer"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	default:
		return "string"
	}
}

// applyJSONSchemaTag parses a comma-separated jsonschema tag value and applies
// the key=value pairs to the given schema.
func applyJSONSchemaTag(tag string, s *JSONSchema, fieldType reflect.Type) {
	pairs := strings.Split(tag, ",")
	for _, pair := range pairs {
		key, value, ok := strings.Cut(pair, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch key {
		case "description":
			s.Description = value
		case "enum":
			parts := strings.Split(value, "|")
			s.Enum = make([]any, len(parts))
			for i, p := range parts {
				s.Enum[i] = p
			}
		case "default":
			s.Default = parseDefault(value, fieldType)
		case "minimum":
			if v, err := strconv.ParseFloat(value, 64); err == nil {
				s.Minimum = &v
			}
		case "maximum":
			if v, err := strconv.ParseFloat(value, 64); err == nil {
				s.Maximum = &v
			}
		case "minLength":
			if v, err := strconv.Atoi(value); err == nil {
				s.MinLength = &v
			}
		case "maxLength":
			if v, err := strconv.Atoi(value); err == nil {
				s.MaxLength = &v
			}
		case "pattern":
			s.Pattern = value
		case "format":
			s.Format = value
		}
	}
}

// parseDefault converts a string default value to an appropriate Go type
// based on the field's reflect.Type.
func parseDefault(value string, t reflect.Type) any {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Bool:
		b, _ := strconv.ParseBool(value)
		return b
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			return v
		}
		return value
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			return v
		}
		return value
	case reflect.Float32, reflect.Float64:
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			return v
		}
		return value
	default:
		return value
	}
}
