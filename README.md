# Scaffold Agent

[简体中文](README.zh-CN.md)

Scaffold Agent is a model-neutral local tool agent for AI coding assistants. It turns versioned project blueprints and reusable capability packs into deterministic, testable, and upgradeable full-stack applications.

The project is intentionally not an AI model, chat UI, or model gateway. Codex, Claude Code, Kimi Code, and other MCP-capable coding agents provide the reasoning; Scaffold Agent provides stable software-engineering facts and safe file operations.

## Status

Scaffold Agent is under active construction. The protocol core, deterministic filesystem transactions, stable JSON CLI, six-tool MCP server, and the Go/PostgreSQL/MySQL generator are implemented. Generated Go applications include secure identity, permission RBAC, transactional audit, Blueprint-driven CRUD, OpenAPI 3.1, a build-tested Vue/Element Plus administration UI, and the completed M5 platform capability suite. The Java 21/Spring Boot adapter has platform parity, including CRUD, tenancy, durable jobs, notifications, file assets, cache, job administration, observability, CSV transfer, approvals, OpenAPI, and the shared administration UI. The Python 3.12+/FastAPI adapter now generates PostgreSQL or MySQL services with deterministic uv locks, Alembic migrations, bounded health/readiness, secure session/token identity, permission RBAC, transactional audit, stable OpenAPI, Blueprint-driven CRUD, organization tenancy, email-bound invitations, member administration, explicit ownership, ownership transfer, reversible deactivation, durable leased background jobs, idempotent TLS-only email notifications with an independent worker, tenant-aware file assets with bounded atomic storage, and a tenant-scoped cross-instance JSON cache, plus the same build-tested Vue/Element Plus administration UI. Generated Python projects enforce Ruff, strict mypy, Bandit, pytest coverage, and live-database gates. Remaining Python platform capabilities and Nuxt storefronts remain in progress; schemas stay experimental until 1.0.

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

The reference workflow generates a Go modular monolith with PostgreSQL or MySQL, session and token authentication, RBAC, audit logging, one stateful CRUD module, and an optional administration UI.

## Development

Prerequisites:

- Go 1.27 or newer in the 1.27 release line
- Git
- Node.js 22.12 or newer when validating generated frontends
- Python 3.12 or 3.13 and uv 0.12.8 when validating generated Python services

Run the baseline checks:

```bash
go test ./...
go vet ./...
go run ./cmd/scaffold-agent doctor --json
go run ./cmd/scaffold-agent query --topic support
go run ./cmd/scaffold-agent validate --project-root ./examples/minimal --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/task-service --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/task-service-mysql --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/tenant-task-service --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/worker-task-service --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/notification-task-service --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/file-task-service --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/cache-task-service --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/job-admin-task-service --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/observability-task-service --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/csv-transfer-task-service --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/approval-task-service --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/minimal-java --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/task-service-java --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/minimal-python --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/task-service-python --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/tenant-task-service-python --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/worker-task-service-python --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/notification-task-service-python --blueprint scaffold.yaml
```

See [CONTRIBUTING.md](CONTRIBUTING.md), [the Agent interface](docs/agent-interface.md), [the architecture decision](docs/adr/0001-foundation.md), and [the roadmap](docs/roadmap.md) before contributing.

## License

Apache-2.0
