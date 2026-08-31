# ADR 0001: Foundation architecture

- Status: Accepted
- Date: 2026-08-31

## Context

Scaffold Agent must be callable by different coding agents while producing the same engineering result. It also needs to generate Go, Java, and Python applications without coupling the core to any one stack.

## Decision

1. Scaffold Agent is a local tool agent and does not call AI models.
2. The implementation is a Go monorepo with one `scaffold-agent` binary.
3. CLI and MCP are adapters over the same application services and public schemas.
4. Human-authored YAML is normalized into canonical JSON before hashing or planning.
5. Capability packs are declarative. They cannot run arbitrary installation scripts.
6. Generated files are either Agent-managed or user-owned. User-owned files are never silently overwritten.
7. Generated applications are modular monoliths and have no runtime dependency on Scaffold Agent.
8. PostgreSQL is implemented first; MySQL compatibility is established before Java and Python adapters.
9. The repository follows concise, language-native rules inspired by Alibaba engineering principles rather than enforcing the historical P3C rule set.

## Consequences

- Provider-specific prompts remain in thin integration packages.
- The core can be tested without an MCP client, a model, a database, or a template set.
- Cross-stack features require conformance tests before being marked stable.
- Complex AST merging is deferred; explicit extension points and whole-file ownership are the initial upgrade mechanism.

