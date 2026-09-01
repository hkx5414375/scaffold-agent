# ADR 0035: Python FastAPI identity foundation

- Status: Accepted
- Date: 2026-09-01

## Context

The Python adapter must provide the same deterministic planning, security boundaries,
database support, and generated-project independence as Go and Java. Letting an AI
assistant assemble framework defaults for every project would repeat authentication,
migration, dependency, and test decisions while consuming tokens and producing
different outcomes across models.

## Decision

1. `python-service` `0.1.0` targets Python 3.12 and 3.13 with FastAPI 0.141.1,
   Pydantic 2.13.5, synchronous SQLAlchemy Core 2.0.52, Alembic 1.19.1, and Uvicorn.
   Synchronous route functions match the synchronous database driver boundary and
   avoid presenting blocking calls as asynchronous work.
2. PostgreSQL uses Psycopg 3.3.5 and MySQL uses PyMySQL 1.2.0. Each generated dialect
   receives a complete `uv.lock`; `uv sync --frozen` and `uv lock --check` reject
   dependency drift.
3. Startup migrates the database before serving traffic, composes a bounded
   pre-pinged pool, and exposes database-independent `/healthz` plus generic
   `/readyz` status without infrastructure details.
4. Passwords use bounded PBKDF2-HMAC-SHA256. Browser sessions and API tokens expire,
   only SHA-256 credential digests are persisted, bearer credentials take precedence,
   and all protected operations require explicit permissions.
5. Login and token creation commit credential state and audit evidence in one
   transaction. Denied authentication and authorization append bounded audit values
   with control characters removed.
6. HTTP, application service, persistence, and schema migration responsibilities are
   separate. The application service imports neither FastAPI nor SQLAlchemy.
7. Generated projects enforce Ruff formatting and linting, strict mypy, Bandit,
   pytest branch coverage of at least 90%, architecture tests, a stable OpenAPI file,
   and repository contracts. CI additionally executes migrations and repository
   behavior against PostgreSQL and MySQL.
8. Administration UI, business modules, and capability selections fail explicitly
   until their complete Python vertical slices are implemented.

## Consequences

- Python is a registered Engine backend for secure identity-only services on both
  databases; unsupported parity claims cannot produce partial output.
- Generated services have no runtime dependency on Scaffold Agent and do not need an
  AI model to reconstruct identity, migration, dependency, or quality infrastructure.
- CRUD, shared Vue administration, tenancy, and platform capabilities remain the
  next additive Python slices in M6.
