---
name: go-mcp-creator
description: Expert assistant for building Model Context Protocol (MCP) servers in Go using the official SDK. Use when creating a new Go MCP server, adding typed tools/resources/prompts/templates to an existing server, choosing stdio versus HTTP transport, testing MCP behavior end to end, or troubleshooting official go-sdk integrations.
---

# Go MCP Creator

Build and maintain Go MCP servers with the official SDK: `github.com/modelcontextprotocol/go-sdk/mcp`.

Default assumption: use the official SDK's typed path first, then drop to lower-level APIs only when you have a specific reason.

## Quick Start

For a new server, start with the scaffold script instead of hand-writing boilerplate:

```bash
python3 scripts/scaffold_server.py my-server \
  --path /tmp \
  --module github.com/example/my-server
```

That creates a small, current stdio server with:
- typed `mcp.AddTool(...)`
- `server.Run(context.Background(), &mcp.StdioTransport{})`
- an in-memory end-to-end test
- a pinned SDK version

Then validate it:

```bash
scripts/validate_server.sh /tmp/my-server
```

If you need to add another tool to an existing server, generate a stub instead of freehanding it:

```bash
python3 scripts/scaffold_tool.py summarize_text \
  --package main \
  --output /tmp/my-server/summarize_text.go
```

The script prints the exact `mcp.AddTool(...)` line to wire into `main.go`.

## Default Workflow

### 1. Pick the narrowest successful path

Use this order by default:
1. `scripts/scaffold_server.py` for a new project
2. `scripts/scaffold_tool.py` for a new typed tool
3. edit only the generated TODO blocks
4. run `scripts/validate_server.sh`
5. if needed, add resources/prompts/templates using the focused references below

This matters for smaller models and unfamiliar codebases: prefer generated structure over bespoke implementation.

Do not start with `go doc` in the skill directory itself. This folder is not a Go module. If API inspection is needed, scaffold or enter a real Go module that already depends on `github.com/modelcontextprotocol/go-sdk`, then run `go doc` there.

### 2. Prefer typed tools

Default to:

```go
mcp.AddTool(server, &mcp.Tool{Name: "...", Description: "..."}, Handler)
```

Use typed `AddTool` unless you specifically need:
- custom raw JSON Schema not easily inferred from Go types
- behavior that requires the lower-level `Server.AddTool`

Read [references/tools.md](references/tools.md) before adding or refactoring tools.

### 3. Choose transport deliberately

- `stdio`: default for local CLI integrations and desktop clients
- streamable HTTP: default for networked deployments
- in-memory transports: default for tests

Read [references/transports.md](references/transports.md) when transport choice matters.

### 4. Validate behavior, not just compilation

Minimum completion bar:
- `go test ./...`
- `go vet ./...`
- `govulncheck ./...` if available
- at least one end-to-end MCP test using `mcp.NewInMemoryTransports()`

Read [references/testing.md](references/testing.md) before claiming the server works.

## What To Build First

### New server

Use the scaffold script, then:
1. rename the sample tool only if the domain is already clear
2. keep one working tool until tests pass
3. add more tools after the first end-to-end call works

### Existing server

Inspect first:
- where the server is created
- which transport is used
- whether tools are added with `mcp.AddTool` or lower-level APIs
- whether tests already use in-memory transports

Then extend the existing style unless it is clearly stale.

## Resources, Prompts, Templates

After tools are stable:
- add read-only resources for data the model should inspect
- add prompts for reusable, parameterized prompt building
- add resource templates when URIs are dynamic or patterned

Read [references/resources-prompts.md](references/resources-prompts.md) before adding these features.

## Troubleshooting Rules

- If guidance in old code conflicts with the current SDK, prefer the current SDK.
- If you see `server.Serve(...)`, `mcp.NewStdioTransport()`, `mcp.JSonschema`, or `mcp.NewToolResultError(...)`, treat that code as suspect and verify against current docs.
- If a generated example compiles but client integration fails, check transport mismatch first.
- If a tool works in unit code but not through MCP, add or fix an end-to-end in-memory transport test.

Read [references/troubleshooting.md](references/troubleshooting.md) when debugging.

## Versioning And Security

Pin a current `go-sdk` release in new projects. At minimum, avoid versions before `v1.3.1`; the Go vulnerability database currently reports a vulnerability affecting earlier releases.

When updating an existing MCP server:
- inspect `go.mod`
- upgrade deliberately
- rerun `go test`, `go vet`, and `govulncheck`

## Focused References

- [references/tools.md](references/tools.md): typed tools, schemas, handler shape, when to use raw handlers
- [references/resources-prompts.md](references/resources-prompts.md): resources, prompts, templates, and when each fits
- [references/transports.md](references/transports.md): stdio, streamable HTTP, in-memory testing transport
- [references/testing.md](references/testing.md): end-to-end test patterns and validation checklist
- [references/troubleshooting.md](references/troubleshooting.md): stale API detection and debugging checklist
