---
name: go-mcp-creator
description: Expert assistant for building Model Context Protocol (MCP) servers in Go using the official SDK. Use when creating new MCP servers, adding tools/resources/prompts to existing Go servers, or troubleshooting Go MCP implementations.
---

# Go MCP Creator

This skill enables the development of high-quality, type-safe Model Context Protocol (MCP) servers using the official Go SDK (`github.com/modelcontextprotocol/go-sdk`).

## Core Capabilities

### 1. Project Initialization & Setup
Quickly scaffold a new Go MCP project and manage dependencies.
- **Scaffolding**: Use `scripts/scaffold_server.py <name>` to create a new project.
- **Manual Setup**:
  ```bash
  go mod init github.com/username/project
  go get github.com/modelcontextprotocol/go-sdk/mcp
  ```
- **API Exploration**: Use `go doc` to view the latest SDK documentation:
  ```bash
  go doc github.com/modelcontextprotocol/go-sdk/mcp
  go doc github.com/modelcontextprotocol/go-sdk/mcp.ServerOptions
  ```

### 2. Tool Implementation
Define and register tools with proper type safety and JSON schema validation.
- **Pattern**: Use structs with `json` tags and `mcp.AddTool`.
- **Reference**: See [references/sdk-patterns.md](references/sdk-patterns.md) for detailed tool registration examples.

### 3. Resource & Prompt Management
Expose data as resources or guide LLM behavior with prompts.
- **Action**: Use `mcp.AddResource` and `mcp.AddPrompt`.
- **Reference**: See [references/sdk-patterns.md](references/sdk-patterns.md) for resource/prompt patterns.

### 4. Transport Configuration
Configure communication via Stdio or HTTP.
- **Guideline**: Prefer Stdio for CLI tools and local integration; use HTTP for networked services.

## Best Practices

- **API Discovery**: Frequently use `go doc` to understand the full capabilities of the SDK structs and methods.
- **Type-Safe Design**: Always use structs with JSON schema tags for tool inputs/outputs.
- **Error Handling**: Use `mcp.NewToolResultError` for user-facing errors and wrap internal errors.
- **Context Usage**: Ensure all long-running operations respect `context.Context` cancellation.
- **Graceful Shutdown**: Handle signals for clean server termination.

## Troubleshooting

- **Validation Errors**: Check JSON schema definitions against the input structs.
- **Transport Issues**: Ensure the transport matches the client's expectation (e.g., Gemini CLI uses Stdio).
- **Dependency Issues**: Run `go mod tidy` to ensure the Go environment is clean.

---

For detailed code snippets and API references, see:
- [sdk-patterns.md](references/sdk-patterns.md) - Official SDK patterns and examples.