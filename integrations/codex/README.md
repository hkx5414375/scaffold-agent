# OpenAI GPT through Codex

The recommended registration command is:

```bash
codex mcp add scaffold-agent -- /absolute/path/to/scaffold-agent mcp
```

Equivalent `config.toml` configuration:

```toml
[mcp_servers.scaffold-agent]
command = "/absolute/path/to/scaffold-agent"
args = ["mcp"]
startup_timeout_sec = 10
tool_timeout_sec = 120
```

Codex stores user configuration in `~/.codex/config.toml`. A trusted project may instead use `.codex/config.toml`. The desktop app, CLI, and IDE extension share MCP configuration on the same Codex host.

Scaffold Agent keeps its initialization instructions below 512 characters so the
complete safety workflow is available while Codex is deciding whether to load a
tool. No OpenAI API key is passed to the Engine.

Reference: [OpenAI Codex MCP documentation](https://developers.openai.com/codex/mcp).
