# Testing And Validation

Compilation is not enough. Validate the MCP behavior end to end.

## Minimum Validation

Run:

```bash
go test ./...
go vet ./...
govulncheck ./...
```

If `govulncheck` is unavailable, note that explicitly.

## Default End-To-End Test Pattern

Use in-memory transports to connect a client and server:

```go
ctx := context.Background()
t1, t2 := mcp.NewInMemoryTransports()

serverSession, err := server.Connect(ctx, t1, nil)
if err != nil {
    t.Fatal(err)
}
defer serverSession.Close()

client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v1.0.0"}, nil)
clientSession, err := client.Connect(ctx, t2, nil)
if err != nil {
    t.Fatal(err)
}
defer clientSession.Close()
```

Then call the tool through MCP, not directly through Go function calls.

## What To Assert

At minimum assert:
- the client connects successfully
- the tool call returns no protocol error
- `res.IsError` is false for the success path
- returned content or structured output contains the expected value

Also add one failure-path test when the tool has meaningful validation or lookup failure behavior.

## Validation Script

Use `scripts/validate_server.sh <project-dir>` for the common checks.

That script is deliberately simple so smaller models can rely on one command instead of remembering the Go validation sequence.
