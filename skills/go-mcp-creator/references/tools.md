# Tools

Use typed tools by default.

## Preferred Pattern

Use:

```go
type Input struct {
    Name string `json:"name" jsonschema:"the name to greet"`
}

type Output struct {
    Greeting string `json:"greeting" jsonschema:"the greeting returned to the caller"`
}

func Greet(
    ctx context.Context,
    req *mcp.CallToolRequest,
    input Input,
) (*mcp.CallToolResult, Output, error) {
    text := "Hello, " + input.Name + "!"
    return &mcp.CallToolResult{
        Content: []mcp.Content{&mcp.TextContent{Text: text}},
    }, Output{Greeting: text}, nil
}

mcp.AddTool(server, &mcp.Tool{
    Name:        "greet",
    Description: "Return a greeting for a person",
}, Greet)
```

Why this is the default:
- input schema is inferred from the Go input type
- output schema is inferred from the Go output type
- invalid inputs are rejected before the handler runs
- tool-originated errors are surfaced in MCP-aware ways by the typed handler path

## Handler Shape

Typed handlers use this form:

```go
func Handler(
    ctx context.Context,
    req *mcp.CallToolRequest,
    input In,
) (*mcp.CallToolResult, Out, error)
```

Use the return values like this:
- `*mcp.CallToolResult`: optional content payloads such as text
- `Out`: structured output for `structuredContent`
- `error`: exceptional or protocol-level failure

For a successful tool call, prefer returning a structured output and optional text content.

## Input Design

Use simple structs with:
- `json:"..."` tags for wire names
- `jsonschema:"..."` tags for model-facing descriptions

Example:

```go
type SearchInput struct {
    Query string `json:"query" jsonschema:"search phrase provided by the user"`
    Limit int    `json:"limit,omitempty" jsonschema:"maximum number of results to return"`
}
```

Keep inputs narrow. If a tool needs many independent branches of behavior, split it into multiple tools.

## Output Design

Prefer outputs that are:
- deterministic
- small
- directly useful to the client and model

Example:

```go
type SearchOutput struct {
    Count   int      `json:"count" jsonschema:"number of matching results"`
    Results []string `json:"results" jsonschema:"matching result titles"`
}
```

## When To Use Lower-Level `Server.AddTool`

Use the raw registration path only when you need:
- hand-authored JSON schema
- schema behavior that inferred Go types cannot express cleanly
- manual control over request decoding or validation

If you use the raw path, document why in code.

## Tool Design Rules

- one tool, one clear job
- prefer explicit parameters over free-form blobs
- respect `context.Context`
- return user-relevant failures as tool-visible failures, not hidden logs
- avoid tools that silently mutate remote state unless the action is explicit

## Small-Model Guardrail

For weaker models, do not ask them to invent handler signatures or schema layout from memory. Use `scripts/scaffold_tool.py`, then edit only:
- input fields
- output fields
- the TODO logic block
