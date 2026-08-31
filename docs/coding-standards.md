# Coding standards

These rules apply Alibaba engineering principles through concise, language-native automation.

## Universal rules

- Names must express business meaning. Avoid unexplained abbreviations and generic utility dumping grounds.
- Separate transport, application, domain, persistence, and external integration responsibilities.
- Validate external input before business execution. Authorization and tenant checks happen before mutation.
- Never swallow errors. Public failures use stable codes and safe messages; internal causes remain available for diagnostics.
- Logs are structured and must not contain passwords, tokens, payment secrets, private source, or customer data.
- Database changes use versioned migrations. Transactions belong to application use cases, not HTTP handlers.
- Add an interface only for a real replacement boundary, protocol adapter, or test seam.
- Prefer deleting obsolete code over retaining commented-out implementations.

## Go baseline

- `gofmt` is authoritative.
- Pass `go vet`; add `golangci-lint` and `gosec` to CI when their pinned tool versions are introduced.
- Wrap errors with operation context and preserve the cause.
- Accept `context.Context` at I/O and long-running boundaries.
- Avoid package names such as `common`, `base`, and `utils`.

## Generated stacks

- Java uses Spotless, Checkstyle, SpotBugs, Maven Enforcer, and ArchUnit. Historical P3C rules are guidance, not a blanket gate.
- Python uses Ruff formatting and linting, strict mypy, and Bandit.
- Vue and Nuxt use ESLint, `eslint-plugin-vue`, Prettier, and `vue-tsc`.

