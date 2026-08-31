# ADR 0004: First Blueprint-driven CRUD slice

- Status: Accepted
- Date: 2026-08-31

## Context

The reference generator needs to prove that Blueprint business data becomes a
complete vertical feature rather than disconnected models or SQL snippets. The
first slice must stay small enough that unsupported semantics are visible.

## Decision

1. The first slice accepts exactly one module and one entity. Wider module graphs are rejected until cross-entity semantics are defined.
2. The supported portable field types are `string`, `text`, `bool`, `int64`, and `datetime`; required, optional, and unique constraints are preserved.
3. Every entity receives create, read, list, replace, and delete HTTP operations, bounded keyset pagination, and a monotonically increasing version used for optimistic locking.
4. The Blueprint must explicitly declare create, read, update, and delete permission codes using `<module>:<entity>:<operation>`.
5. Generated HTTP routes authorize every operation with its matching permission. The administrator role receives the four permissions through a versioned migration.
6. Create, update, and delete persist the business change and audit event in the same PostgreSQL transaction.
7. Unique-value collisions and stale update/delete versions return one stable conflict category without leaking database details.
8. Generated Go source is formatted after template rendering, so arbitrary valid field lengths cannot produce unstable formatting.

## Consequences

- AI coding agents can request a useful business module by editing a compact Blueprint instead of generating transport, service, store, SQL, authorization, audit, and tests independently.
- Workflows, relationships, decimal and money semantics, OpenAPI, and generated pages remain explicit future work.
- The strict initial boundary can expand without pretending that ignored Blueprint fields were implemented.
