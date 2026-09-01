# ADR 0036: Python Blueprint CRUD

- Status: Accepted
- Date: 2026-09-01

## Context

The Python foundation needs a complete business vertical slice before AI agents can
turn Blueprint entities into useful services. Generating only table definitions or
HTTP handlers would force each model to invent validation, authorization, optimistic
concurrency, audit, pagination, and database behavior again.

## Decision

1. `python-crud` `0.1.0` accepts at most one module containing exactly one entity.
   Unsupported multiple modules, workflows, pages, unsafe identifiers, incomplete
   permission sets, and non-portable field constraints fail before planning.
2. Portable fields are `string`, `text`, `bool`, decimal-string `int64`, and
   offset-aware `datetime`. Required, optional, and portable unique constraints are
   emitted from the same Blueprint facts for Pydantic, SQLAlchemy, Alembic, OpenAPI,
   repository tests, and HTTP tests.
3. Create, full replacement, and delete commit their audit event in the same
   transaction. Read and bounded list operations use an opaque ID keyset cursor.
4. Updates and deletes require a positive decimal-string version. A single conditional
   mutation increments the version; missing records return 404 and stale versions or
   unique collisions return 409 without exposing driver messages.
5. Every route uses the generated `module:entity:action` permission through the
   existing deny-by-default identity dependency. The administrator receives those
   four permissions in the business migration.
6. PostgreSQL and MySQL projects pass the same frozen dependency, Ruff, strict mypy,
   Bandit, branch coverage, architecture, HTTP, SQLite repository, and live database
   gates.

## Consequences

- An AI agent can describe one entity and receive a complete executable CRUD slice
  without recreating cross-cutting security or persistence code.
- The deliberately narrow first slice preserves deterministic output while multiple
  modules, tenancy, administration UI, and platform capabilities remain additive work.
