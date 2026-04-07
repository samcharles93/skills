# Troubleshooting

## Stale API Smells

If you see any of these, stop and verify against the current SDK:
- `server.Serve(...)`
- `mcp.NewStdioTransport()`
- `mcp.JSonschema`
- `mcp.NewToolResultError(...)`
- examples built around manual JSON unmarshalling for every tool

These are common signs that guidance came from an older SDK version.

## Common Failure Modes

### Client cannot use the server

Usually caused by:
- wrong transport
- server not actually running over the transport the client expects
- initialization path not exercised in tests

Fix:
- verify transport choice first
- reproduce with an in-memory transport test or real client

### Tool compiles but fails through MCP

Usually caused by:
- malformed inferred schema
- wrong field tags
- assumptions about content versus structured output

Fix:
- simplify the input struct
- add `jsonschema` descriptions
- return explicit text content plus structured output while debugging

### Resources or prompts not showing up

Usually caused by:
- not registering them on the server you actually run
- stale assumptions about client UI support
- transport or session mismatch during testing

Fix:
- verify registration and list behavior in an MCP client session

### Validation passes but production use fails

Usually caused by:
- only testing direct Go calls instead of MCP calls
- skipping client-compatible integration tests

Fix:
- add an in-memory end-to-end test
- then verify with one real client over the intended transport
