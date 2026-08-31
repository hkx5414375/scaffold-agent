# Scaffold Agent repository instructions

## Product boundary

- Scaffold Agent is a model-neutral local tool agent. Do not add model SDKs, model API keys, chat UIs, or hidden network dependencies to the core.
- Generated applications must run without Scaffold Agent.
- Keep the core independent from Go, Java, Python, database, and UI adapters.

## Required reading

Before changing implementation, read:

1. `README.md`
2. `docs/adr/0001-foundation.md`
3. `docs/coding-standards.md`
4. The package documentation and tests for the affected subsystem

## Implementation rules

- Prefer the Go standard library. Add a dependency only when it replaces substantial, security-sensitive, or protocol-specific code.
- Keep domain packages independent from CLI, MCP, filesystem, and template adapters.
- Use explicit types and stable error codes at public boundaries.
- Never silently overwrite user-owned files or execute scripts supplied by capability packs.
- Resolve and validate every target path against the project root before filesystem mutation.
- Public code, schemas, error messages, and comments are English. User documentation may have English and Chinese versions.
- Do not create interfaces that have only one implementation unless they define a real external boundary or test seam.

## Verification

Before committing, run:

```text
go test ./...
go vet ./...
go run ./cmd/scaffold-agent doctor --json
```

Formatting must be clean according to `gofmt`.

## Security

- Do not commit credentials, tokens, private project blueprints, customer data, absolute user paths, or generated backups.
- Telemetry is opt-in only and must never contain source, paths, blueprints, logs, business data, accounts, or secrets.

