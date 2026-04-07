# Transports

Choose transport based on deployment shape, not preference.

## Stdio

Default choice for:
- local CLI workflows
- desktop MCP clients
- subprocess-based integrations

Use:

```go
if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
    log.Fatal(err)
}
```

This is the safest default for new local servers.

## Streamable HTTP

Use for:
- networked services
- remote deployment
- cases where the server is not launched as a subprocess by the client

When using HTTP:
- verify the client supports the chosen HTTP transport mode
- confirm session/resumption expectations
- add integration testing beyond plain compilation

If the project is network-facing and transport details matter, verify against the current official `examples/http` package before coding.

## In-Memory Transports

Use in tests:

```go
t1, t2 := mcp.NewInMemoryTransports()
serverSession, err := server.Connect(ctx, t1, nil)
clientSession, err := client.Connect(ctx, t2, nil)
```

Important:
- connect the server first
- then connect the client
- close sessions cleanly

This is the default way to prove your server actually speaks MCP correctly.

## Transport Selection Rule

Use this order unless the user says otherwise:
1. stdio for local tools
2. streamable HTTP for remote services
3. in-memory transport for tests
