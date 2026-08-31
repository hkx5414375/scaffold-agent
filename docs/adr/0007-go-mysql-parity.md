# ADR 0007: Go and MySQL parity

- Status: Accepted
- Date: 2026-08-31

## Context

Database selection belongs in the Blueprint, but it must not change identity,
authorization, audit, CRUD, OpenAPI, or administration semantics. Claiming MySQL
support requires generated code and a live database gate, not only alternate SQL
type names.

## Decision

1. The Go adapter accepts `database.engine: mysql` and shares its domain, service, HTTP, OpenAPI, and Vue templates with PostgreSQL.
2. Generated MySQL services use `database/sql` and `go-sql-driver/mysql` v1.10.0 with bounded connections, UTC timestamps, and parsed time values.
3. MySQL migrations are embedded, ordered, recorded in the same migration ledger, and serialized with a connection-scoped named lock.
4. Embedded migration files are split into individual statements without enabling multi-statement execution on the application pool. Generated tests cover quoted semicolons and malformed input.
5. Migration DDL and seed data are idempotent because MySQL may commit DDL implicitly. A failed run can safely retry before recording the migration version.
6. MySQL persistence preserves session and token digest handling, deny-by-default RBAC, transactional business audit, keyset pagination, unique conflicts, and optimistic locking.
7. Continuous integration generates a complete application and runs its live integration suite against MySQL 8.4 LTS in an isolated random database.
8. Portable unique constraints on `text` fields are rejected for MySQL until the Blueprint exposes an explicit indexed-length policy.
9. Generated business identifiers are quoted with the selected SQL dialect, so valid Blueprint names cannot collide with database keywords.

## Consequences

- One Blueprint can switch between PostgreSQL and MySQL without changing API or administration clients.
- Database-specific code is limited to pools, migrations, SQL migrations, persistence stores, dependency locks, and live-test setup.
- Platform capability packs can target both relational engines; later ADRs define the completed organization, files, jobs, and approval workflow slices.
