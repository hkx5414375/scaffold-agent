# Agent integrations

Every integration starts the same `scaffold-agent mcp` STDIO process. These files contain host configuration only; they do not fork tool behavior or add model SDK dependencies.

- [`codex/`](codex/) — Codex CLI, desktop app, and IDE extension
- [`claude-code/`](claude-code/) — Claude Code local or project-scoped MCP
- [`kimi-code/`](kimi-code/) — Kimi Code CLI user or project MCP configuration
- [`generic/`](generic/) — any MCP host that supports newline-delimited STDIO JSON-RPC

Replace `/absolute/path/to/scaffold-agent` with the installed executable path. On Windows, use the full `.exe` path with escaped backslashes when the host configuration is JSON.

Do not pre-approve `scaffold_apply` globally. It is intentionally annotated as a destructive write tool and also requires the token returned by `scaffold_preview`.
