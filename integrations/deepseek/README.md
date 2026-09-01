# DeepSeek

DeepSeek is the model layer. Install Scaffold Agent in the MCP-capable coding
host that runs DeepSeek, such as Codex-compatible, Claude-compatible, OpenCode,
or another local Agent host. A direct DeepSeek API request cannot start a local
MCP process by itself.

For a host using the common `mcpServers` shape, copy
[`mcp.json.example`](mcp.json.example) into its user or trusted-project
configuration after replacing the executable path. Hosts exposing DeepSeek
through an OpenAI- or Anthropic-compatible model endpoint still use the same
Scaffold Agent MCP entry.

Do not put the DeepSeek API key in the Scaffold Agent entry. The Engine is
model-neutral and needs no model credential. Keep approval enabled for
`scaffold_apply`.

Verify the local protocol path without spending model quota:

```bash
scaffold-agent conformance --json
```
