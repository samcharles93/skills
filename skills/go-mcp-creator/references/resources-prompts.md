# Resources, Prompts, And Templates

Tools are usually the first feature to ship. Add resources, prompts, or templates only when they solve a specific problem.

## Resources

Use resources for read-only data the model may need to inspect or reference.

Good fits:
- logs
- generated reports
- configuration snapshots
- file-like content

Use a resource when the client should be able to list and read data directly.

If a resource may not exist, return `mcp.ResourceNotFoundError(uri)`.

## Resource Templates

Use resource templates when the URI is patterned or parameterized.

Good fits:
- `file:///reports/{name}`
- `build://runs/{id}`
- `doc://page/{slug}`

Templates are better than individually registering many predictable resources.

## Prompts

Use prompts for reusable prompt-building logic.

Good fits:
- code review prompts
- summarization prompts
- report drafting prompts

Prompts should be parameterized and predictable. Keep arguments explicit.

## Default Order

For a new server:
1. ship one typed tool
2. add resources if the tool depends on read-only data
3. add prompts if users repeatedly need the same prompt shape
4. add resource templates if URIs become patterned

## Compatibility Guidance

Across MCP clients, tools tend to be the most universally useful starting point.

Resources, prompts, and templates are still important, but if client support or UX is uncertain:
- ship the tool first
- verify client behavior
- then expand capability surface
