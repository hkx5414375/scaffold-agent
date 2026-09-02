# Scaffold Agent

[简体中文](README.zh-CN.md)

Scaffold Agent is a model-neutral local tool agent for AI coding assistants. It turns versioned project blueprints and reusable capability packs into deterministic, testable, and upgradeable full-stack applications.

The project is intentionally not an AI model, chat UI, or model gateway. OpenAI GPT through Codex, Claude Code, Kimi K3, GLM, DeepSeek, and other models run in MCP-capable coding hosts and provide the reasoning; Scaffold Agent provides stable software-engineering facts and safe file operations.

## Status

Six Agent protocol profiles and token budgets are now executable release gates; every model uses the same MCP tool contract and Engine implementation.

The first published stable release is `v1.0.1`. The `v1.0.0` tag failed closed at the SBOM attestation gate and intentionally produced no GitHub Release. The protocol core, deterministic filesystem transactions, stable JSON CLI, six-tool MCP server, and Go, Java 21/Spring Boot, and Python 3.12+/FastAPI adapters are implemented for PostgreSQL and MySQL. All three backends generate the same locked Vue administration and Nuxt 4 SSR storefront contracts and support `commerce-catalog` `0.1.0`, `customer-accounts` `0.1.0`, `commerce-operations` `0.1.0`, `crm-core` `0.1.0`, and `erp-inventory` `0.1.0`. Commerce operations cover deterministic pricing, versioned carts, idempotent checkout and payment events, orders, fulfillment, returns, refunds, reconciliation, campaigns, coupons, and an explicit no-network sandbox gateway. Cross-backend OpenAPI, permissions, schemas, migrations, behavior, administration, and storefront gates keep Go, Java, and Python aligned.

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

## Install the Engine

Stable releases provide signed amd64/arm64 archives for Linux, macOS, and Windows, SHA-256 checksums, a CycloneDX SBOM, and Sigstore-backed provenance. Verify the download before putting the single `scaffold-agent` executable on `PATH`. See [installation](docs/installation.md), [host integration](integrations/README.md), and the [upgrade policy](docs/upgrade-policy.md).

## Development

Prerequisites:

- Go 1.27 or newer in the 1.27 release line
- Git
- Node.js 22.12 or newer when validating generated frontends
- Node.js 24.11 or newer when validating generated Nuxt storefronts
- Python 3.12 or 3.13 and uv 0.12.8 when validating generated Python services

Run the baseline checks:

```bash
go test ./...
go vet ./...
go run ./cmd/scaffold-agent doctor --json
go run ./cmd/scaffold-agent conformance --json
go run ./cmd/scaffold-agent benchmark --json
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
go run ./cmd/scaffold-agent validate --project-root ./examples/file-task-service-python --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/cache-task-service-python --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/job-admin-task-service-python --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/observability-task-service-python --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/csv-transfer-task-service-python --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/approval-task-service-python --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/storefront-foundation --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/catalog-store-go --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/catalog-store-java --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/catalog-store-python --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/customer-store-go --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/customer-store-java --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/customer-store-python --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/crm-service-go --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/crm-service-java --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/crm-service-python --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/inventory-service-go --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/inventory-service-go-mysql --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/inventory-service-java --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/inventory-service-python --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/inventory-service-python-mysql --blueprint scaffold.yaml
```

See [CONTRIBUTING.md](CONTRIBUTING.md), [the Agent interface](docs/agent-interface.md), [cross-Agent conformance](docs/agent-conformance.md), [token benchmarks](docs/token-benchmarks.md), [support policy](SUPPORT.md), [the architecture decision](docs/adr/0001-foundation.md), and [the roadmap](docs/roadmap.md) before contributing.

## License

Apache-2.0
