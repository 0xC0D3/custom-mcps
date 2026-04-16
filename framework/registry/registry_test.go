package registry_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/0xC0D3/custom-mcps/framework/protocol"
	"github.com/0xC0D3/custom-mcps/framework/registry"
)

// echoInput is a simple struct used as a tool input in tests.
type echoInput struct {
	Message string `json:"message" mcp:"required" jsonschema:"description=The message to echo"`
	Count   int    `json:"count"   jsonschema:"description=Repeat count,minimum=1"`
}

func TestRegisterTool_ListTools(t *testing.T) {
	r := registry.New()

	registry.RegisterTool(r, "echo", "Echo a message", func(_ context.Context, input echoInput) (*protocol.CallToolResult, error) {
		return &protocol.CallToolResult{
			Content: []protocol.Content{protocol.TextContent(input.Message)},
		}, nil
	})

	tools := r.ListTools()
	require.Len(t, tools, 1)

	info := tools[0]
	assert.Equal(t, "echo", info.Name)
	assert.Equal(t, "Echo a message", info.Description)

	// Verify the generated schema contains expected properties.
	var schema map[string]any
	require.NoError(t, json.Unmarshal(info.InputSchema, &schema))
	assert.Equal(t, "object", schema["type"])

	props, ok := schema["properties"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, props, "message")
	assert.Contains(t, props, "count")

	// "message" should be in the required list.
	requiredList, ok := schema["required"].([]any)
	require.True(t, ok)
	assert.Contains(t, requiredList, "message")
}

func TestRegisterTool_GetTool(t *testing.T) {
	r := registry.New()

	registry.RegisterTool(r, "greet", "Say hello", func(_ context.Context, input echoInput) (*protocol.CallToolResult, error) {
		return &protocol.CallToolResult{
			Content: []protocol.Content{protocol.TextContent("hello " + input.Message)},
		}, nil
	})

	def := r.GetTool("greet")
	require.NotNil(t, def)
	assert.Equal(t, "greet", def.Info.Name)

	// Unknown tool returns nil.
	assert.Nil(t, r.GetTool("nonexistent"))
}

func TestRegisterTool_HandlerDecoding(t *testing.T) {
	r := registry.New()

	var captured echoInput
	registry.RegisterTool(r, "capture", "Capture input", func(_ context.Context, input echoInput) (*protocol.CallToolResult, error) {
		captured = input
		return &protocol.CallToolResult{
			Content: []protocol.Content{protocol.TextContent(input.Message)},
		}, nil
	})

	def := r.GetTool("capture")
	require.NotNil(t, def)

	params := json.RawMessage(`{"message":"hello","count":3}`)
	result, err := def.Handler(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "hello", captured.Message)
	assert.Equal(t, 3, captured.Count)
	assert.Len(t, result.Content, 1)
	assert.Equal(t, "hello", result.Content[0].Text)
}

func TestRegisterTool_HandlerInvalidJSON(t *testing.T) {
	r := registry.New()

	registry.RegisterTool(r, "bad", "Bad input", func(_ context.Context, input echoInput) (*protocol.CallToolResult, error) {
		return nil, nil
	})

	def := r.GetTool("bad")
	require.NotNil(t, def)

	_, err := def.Handler(context.Background(), json.RawMessage(`{invalid`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid parameters")
}

func TestRegisterToolRaw(t *testing.T) {
	r := registry.New()

	called := false
	r.RegisterToolRaw(registry.ToolDefinition{
		Info: protocol.ToolInfo{
			Name:        "raw-tool",
			Description: "A raw tool",
			InputSchema: json.RawMessage(`{"type":"object"}`),
		},
		Handler: func(_ context.Context, _ json.RawMessage) (*protocol.CallToolResult, error) {
			called = true
			return &protocol.CallToolResult{
				Content: []protocol.Content{protocol.TextContent("raw")},
			}, nil
		},
	})

	def := r.GetTool("raw-tool")
	require.NotNil(t, def)
	assert.Equal(t, "raw-tool", def.Info.Name)

	result, err := def.Handler(context.Background(), nil)
	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, "raw", result.Content[0].Text)
}

func TestRegisterResource(t *testing.T) {
	r := registry.New()

	r.RegisterResource(registry.ResourceDefinition{
		Info: protocol.ResourceInfo{
			URI:         "file:///config.json",
			Name:        "Config",
			Description: "Application configuration",
			MIMEType:    "application/json",
		},
		Handler: func(_ context.Context, uri string) (*protocol.ReadResourceResult, error) {
			return &protocol.ReadResourceResult{
				Contents: []protocol.ResourceContent{
					{URI: uri, MIMEType: "application/json", Text: `{"key":"value"}`},
				},
			}, nil
		},
	})

	resources := r.ListResources()
	require.Len(t, resources, 1)
	assert.Equal(t, "file:///config.json", resources[0].URI)
	assert.Equal(t, "Config", resources[0].Name)

	def := r.GetResource("file:///config.json")
	require.NotNil(t, def)

	result, err := def.Handler(context.Background(), "file:///config.json")
	require.NoError(t, err)
	require.Len(t, result.Contents, 1)
	assert.Equal(t, `{"key":"value"}`, result.Contents[0].Text)

	// Unknown URI returns nil.
	assert.Nil(t, r.GetResource("file:///unknown"))
}

func TestRegisterPrompt(t *testing.T) {
	r := registry.New()

	r.RegisterPrompt(registry.PromptDefinition{
		Info: protocol.PromptInfo{
			Name:        "summarize",
			Description: "Summarize text",
			Arguments: []protocol.PromptArgument{
				{Name: "text", Description: "Text to summarize", Required: true},
				{Name: "style", Description: "Summary style"},
			},
		},
		Handler: func(_ context.Context, args map[string]string) (*protocol.GetPromptResult, error) {
			return &protocol.GetPromptResult{
				Description: "Summarize the given text",
				Messages: []protocol.PromptMessage{
					{Role: "user", Content: protocol.TextContent("Summarize: " + args["text"])},
				},
			}, nil
		},
	})

	prompts := r.ListPrompts()
	require.Len(t, prompts, 1)
	assert.Equal(t, "summarize", prompts[0].Name)
	require.Len(t, prompts[0].Arguments, 2)
	assert.True(t, prompts[0].Arguments[0].Required)

	def := r.GetPrompt("summarize")
	require.NotNil(t, def)

	result, err := def.Handler(context.Background(), map[string]string{"text": "hello world"})
	require.NoError(t, err)
	require.Len(t, result.Messages, 1)
	assert.Equal(t, "Summarize: hello world", result.Messages[0].Content.Text)

	// Unknown prompt returns nil.
	assert.Nil(t, r.GetPrompt("nonexistent"))
}

func TestDuplicateRegistration_Overwrites(t *testing.T) {
	r := registry.New()

	// Register a tool, then overwrite it.
	registry.RegisterTool(r, "dup", "first", func(_ context.Context, _ echoInput) (*protocol.CallToolResult, error) {
		return &protocol.CallToolResult{Content: []protocol.Content{protocol.TextContent("first")}}, nil
	})
	registry.RegisterTool(r, "dup", "second", func(_ context.Context, _ echoInput) (*protocol.CallToolResult, error) {
		return &protocol.CallToolResult{Content: []protocol.Content{protocol.TextContent("second")}}, nil
	})

	tools := r.ListTools()
	require.Len(t, tools, 1)
	assert.Equal(t, "second", tools[0].Description)

	result, err := r.GetTool("dup").Handler(context.Background(), json.RawMessage(`{"message":"x"}`))
	require.NoError(t, err)
	assert.Equal(t, "second", result.Content[0].Text)

	// Same for resources.
	r.RegisterResource(registry.ResourceDefinition{
		Info: protocol.ResourceInfo{URI: "r://dup", Name: "first"},
	})
	r.RegisterResource(registry.ResourceDefinition{
		Info: protocol.ResourceInfo{URI: "r://dup", Name: "second"},
	})
	resources := r.ListResources()
	require.Len(t, resources, 1)
	assert.Equal(t, "second", resources[0].Name)

	// Same for prompts.
	r.RegisterPrompt(registry.PromptDefinition{
		Info: protocol.PromptInfo{Name: "dup", Description: "first"},
	})
	r.RegisterPrompt(registry.PromptDefinition{
		Info: protocol.PromptInfo{Name: "dup", Description: "second"},
	})
	prompts := r.ListPrompts()
	require.Len(t, prompts, 1)
	assert.Equal(t, "second", prompts[0].Description)
}

func TestEmptyRegistry(t *testing.T) {
	r := registry.New()

	assert.Empty(t, r.ListTools())
	assert.Empty(t, r.ListResources())
	assert.Empty(t, r.ListPrompts())
	assert.Nil(t, r.GetTool("any"))
	assert.Nil(t, r.GetResource("any"))
	assert.Nil(t, r.GetPrompt("any"))
}
