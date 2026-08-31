# Scaffold Agent

[简体中文](README.zh-CN.md)

Scaffold Agent is a model-neutral local tool agent for AI coding assistants. It turns versioned project blueprints and reusable capability packs into deterministic, testable, and upgradeable full-stack applications.

The project is intentionally not an AI model, chat UI, or model gateway. Codex, Claude Code, Kimi Code, and other MCP-capable coding agents provide the reasoning; Scaffold Agent provides stable software-engineering facts and safe file operations.

## Status

Scaffold Agent is under active construction. The repository currently contains the quality baseline and the first executable CLI skeleton. Public schemas and generation behavior are not stable yet.

## Design goals

- Save tokens by reusing mature capability packs instead of regenerating common infrastructure.
- Produce identical output from identical blueprints, pack versions, and project state.
- Keep generated applications independent from Scaffold Agent at runtime.
- Support Go, Java, and Python backends with shared Vue admin and Nuxt storefront contracts.
- Support PostgreSQL and MySQL without leaking database-specific behavior into blueprints.
- Preview every write, preserve user-owned extension code, and provide recoverable upgrades.

## Initial workflow

```text
query -> plan -> preview -> apply -> verify
```

The first end-to-end milestone targets a Go modular monolith with PostgreSQL, session and token authentication, RBAC, audit logging, and one stateful CRUD module.

## Development

Prerequisites:

- Go 1.27 or newer in the 1.27 release line
- Git

Run the baseline checks:

```bash
go test ./...
go vet ./...
go run ./cmd/scaffold-agent doctor --json
```

See [CONTRIBUTING.md](CONTRIBUTING.md), [the architecture decision](docs/adr/0001-foundation.md), and [the roadmap](docs/roadmap.md) before contributing.

## License

Apache-2.0

