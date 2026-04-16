---
name: add-tool
description: Add a new tool to an existing MCP server. Guides through defining the input struct with mcp/jsonschema tags, writing the handler, and registering it.
user-invocable: true
---

# Add Tool to MCP Server

Add a new tool to an existing server under `cmd/servers/<name>/`.

## Workflow

1. Ask which server to add the tool to (list existing servers)
2. Ask for tool name (snake_case)
3. Ask for tool description
4. Ask what inputs the tool needs (name, type, required?, description)

## Steps

### 1. Define Input Struct

Add to the server's `tools.go`. Use struct tags:

```go
type <ToolName>Input struct {
    // mcp:"required" for required fields
    // jsonschema tag for: description, enum, default, minimum, maximum, pattern, format
    Field string `json:"field" mcp:"required" jsonschema:"description=Field description"`
}
```

**Tag quick reference:**
- `mcp:"required"` — field is required
- `jsonschema:"description=..."` — field description
- `jsonschema:"enum=A|B|C"` — allowed values (pipe-separated)
- `jsonschema:"default=X"` — default value
- `jsonschema:"minimum=N,maximum=M"` — numeric range
- `jsonschema:"minLength=N,maxLength=M"` — string length
- `jsonschema:"pattern=^[a-z]+$"` — regex pattern
- `jsonschema:"format=email"` — format hint

### 2. Write Handler

```go
func handle<ToolName>(ctx context.Context, input <ToolName>Input) (*protocol.CallToolResult, error) {
    // Implement tool logic
    // For errors the AI should see, return IsError: true
    // For unexpected failures, return the error
    return &protocol.CallToolResult{
        Content: []protocol.Content{protocol.TextContent("result")},
    }, nil
}
```

### 3. Register

Add to `registerTools()` in `tools.go`:

```go
server.Tool(srv, "tool_name", "Tool description", handle<ToolName>)
```

### 4. Verify

```bash
go build ./cmd/servers/<name>/
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}
{"jsonrpc":"2.0","method":"notifications/initialized"}
{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | go run ./cmd/servers/<name>/ 2>/dev/null
```

Confirm the new tool appears in the `tools/list` response with the correct schema.
