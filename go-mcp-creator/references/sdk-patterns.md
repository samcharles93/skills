# MCP Go SDK Reference

This document provides detailed information on the key components and patterns of the official `github.com/modelcontextprotocol/go-sdk` package.

## API Exploration

Use `go doc` to explore the SDK directly from your terminal. This is the best way to get up-to-date information on types and functions.

### View Package Overview
```bash
go doc github.com/modelcontextprotocol/go-sdk/mcp
```

### View Specific Types
```bash
go doc github.com/modelcontextprotocol/go-sdk/mcp.ServerOptions
go doc github.com/modelcontextprotocol/go-sdk/mcp.Tool
```

## Core Components

### Server Creation
- `mcp.NewServer(implementation mcp.Implementation, options ...mcp.ServerOption)`: Initializes a new MCP server.
- `mcp.Implementation`: A struct containing server name and version.
- `mcp.WithLogging()`: Server option to enable logging.
- `mcp.WithCapabilities(caps mcp.ServerCapabilities)`: Declares server features (tools, resources, prompts).

### Transport
- `mcp.NewStdioTransport()`: Best for CLI-based tools.
- `mcp.NewHTTPServerTransport(path string, port int)`: Best for web services.
- `server.Serve(transport mcp.Transport)`: Starts the server loop.

## Tool Registration
Tools are the most common way to extend MCP.

```go
server.AddTool(mcp.Tool{
    Name:        "get_weather",
    Description: "Get the current weather for a location",
    InputSchema: mcp.JSonschema{
        Type: "object",
        Properties: map[string]interface{}{
            "location": map[string]interface{}{
                "type":        "string",
                "description": "The city and state, e.g. San Francisco, CA",
            },
        },
        Required: []string{"location"},
    },
}, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // 1. Unmarshal inputs
    var args struct {
        Location string `json:"location"`
    }
    if err := json.Unmarshal(request.Arguments, &args); err != nil {
        return nil, fmt.Errorf("invalid arguments: %w", err)
    }

    // 2. Perform logic
    weather, err := fetchWeather(ctx, args.Location)
    if err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("failed to fetch weather: %v", err)), nil
    }

    // 3. Return result
    return mcp.NewToolResultText(weather), nil
})
```

## Resource Registration
Resources provide read-only data (logs, files, states).

```go
server.AddResource(mcp.Resource{
    URI:         "logs://system",
    Name:        "System Logs",
    Description: "Real-time system logs",
    MimeType:    "text/plain",
}, func(ctx context.Context, request mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
    logs := getSystemLogs()
    return &mcp.ReadResourceResult{
        Contents: []mcp.ResourceContent{
            mcp.TextResourceContent{
                URI:      request.URI,
                MimeType: "text/plain",
                Text:     logs,
            },
        },
    }, nil
})
```

## Prompt Registration
Prompts guide the LLM's behavior or provide templates.

```go
server.AddPrompt(mcp.Prompt{
    Name:        "code_review",
    Description: "Review Go code for best practices",
    Arguments: []mcp.PromptArgument{
        {
            Name:        "code",
            Description: "The Go code to review",
            Required:    true,
        },
    },
}, func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
    code := request.Arguments["code"]
    return &mcp.GetPromptResult{
        Messages: []mcp.PromptMessage{
            mcp.NewPromptMessageUser(fmt.Sprintf("Please review this Go code:

%s", code)),
        },
    }, nil
})
```

## Error Handling Patterns
- Use `mcp.NewToolResultError(msg)` for expected tool errors (e.g., "city not found").
- Return a Go `error` from the handler for protocol-level errors (e.g., connection lost).
- Always check `ctx.Done()` or use `ctx` in outgoing calls to respect deadlines.
