# Contributing

Thank you for helping build Scaffold Agent.

## Before opening a change

- Discuss public schema, capability protocol, or compatibility changes in an issue first.
- Keep changes focused and include tests for observable behavior.
- Update English and Chinese documentation together when user-facing behavior changes.
- Do not introduce model-provider dependencies into the core.

## Local verification

```bash
go test ./...
go vet ./...
go run ./cmd/scaffold-agent doctor --json
```

Run `gofmt` on every changed Go file. Commits should use a concise imperative subject. Public APIs follow semantic versioning once their schema is marked stable.

## Pull requests

A pull request must explain:

- The concrete problem being solved.
- Public behavior or schema changes.
- Security and compatibility impact.
- Tests and generated-project evidence.

