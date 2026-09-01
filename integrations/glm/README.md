# GLM

GLM is the model layer. Install Scaffold Agent in the MCP-capable coding host
that runs GLM, such as Claude Code, Kimi Code, OpenCode, or another local Agent.
The host must support local STDIO MCP servers.

For a host using the common `mcpServers` shape, copy
[`mcp.json.example`](mcp.json.example) into its user or trusted-project
configuration after replacing the executable path. If GLM is configured as the
model provider inside Claude Code or Kimi Code, use that host's documented
configuration path instead.

Do not put the GLM API key in the Scaffold Agent entry. The Engine calls no
model service and needs no model credential. Keep approval enabled for
`scaffold_apply`; the preview token is a consistency guard, not a replacement
for the host's user approval.

Verify the local protocol path without spending model quota:

```bash
scaffold-agent conformance --json
```
