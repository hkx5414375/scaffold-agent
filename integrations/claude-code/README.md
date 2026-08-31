# Claude Code

Register Scaffold Agent for the current Claude Code scope:

```bash
claude mcp add --transport stdio scaffold-agent -- /absolute/path/to/scaffold-agent mcp
```

For a project-scoped configuration, copy [`mcp.json.example`](mcp.json.example) to `.mcp.json` in the generated project after replacing the executable path. Claude Code asks users to trust project-scoped MCP commands before starting them.

Reference: [Claude Code MCP documentation](https://code.claude.com/docs/en/mcp).
