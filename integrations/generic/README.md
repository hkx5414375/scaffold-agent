# Generic MCP host

Configure a local STDIO server with:

- Command: `/absolute/path/to/scaffold-agent`
- Arguments: `mcp`
- Transport: STDIO
- Startup timeout: at least 10 seconds
- Tool timeout: at least 120 seconds for generated-project verification

Hosts that accept the common `mcpServers` JSON shape can start from
[`mcp.json.example`](mcp.json.example).

The host must send one UTF-8 JSON-RPC object per line. Scaffold Agent negotiates MCP versions `2025-11-25`, `2025-06-18`, `2025-03-26`, and `2024-11-05`.

The server advertises six tools and returns both `structuredContent` and the same serialized JSON in a text content block for compatibility. See the [MCP lifecycle](https://modelcontextprotocol.io/specification/2025-11-25/basic/lifecycle), [STDIO transport](https://modelcontextprotocol.io/specification/2025-11-25/basic/transports), and [tool result](https://modelcontextprotocol.io/specification/2025-11-25/server/tools) specifications.
