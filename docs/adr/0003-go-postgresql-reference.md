# ADR 0003: Go and PostgreSQL reference slice

- Status: Accepted
- Date: 2026-08-31

## Context

The first generator must prove that a model-neutral Blueprint can produce useful,
secure application code rather than only an empty framework. It must also avoid
claiming support for Blueprint selections that are not implemented yet.

## Decision

1. The first Go slice supports PostgreSQL plus both session and API-token authentication.
2. Generated services use `pgx/v5` with a bounded pool and have no runtime dependency on Scaffold Agent.
3. Ordered SQL migrations are embedded in the binary, serialized with a PostgreSQL advisory lock, and recorded in a migration ledger.
4. Passwords use self-describing PBKDF2-SHA256 hashes. Raw session and API tokens are returned once; only SHA-256 digests are persisted.
5. Browser sessions use HttpOnly, SameSite=Lax cookies. Secure cookies are enabled by default and may be disabled explicitly for local HTTP.
6. Security-relevant login and token-creation outcomes are written as audit events. A session or token is removed when its success audit cannot be recorded.
7. The adapter rejects MySQL, frontends, incomplete authentication selections, capabilities, and business modules until their complete vertical slices exist.
8. The generated dependency graph is pinned with both `go.mod` and `go.sum`.
9. Roles, permissions, and role-permission assignments are relational data. Protected HTTP handlers require a stable permission code and deny access unless the store confirms that assignment.
10. Generated integration tests use a random, explicitly created schema and run migrations, identity persistence, RBAC, CRUD, optimistic locking, and transactional audit checks against PostgreSQL 18 in CI.

## Consequences

- The minimal Blueprint now generates a compilable, testable identity service rather than a health endpoint alone.
- PostgreSQL must be available to run the generated service and opt-in integration test, but unit tests remain database-independent.
- The reference slice now has a real database gate without requiring local Docker for ordinary generator development.
- MySQL, Java, and Python cannot accidentally appear successful through partial output.
