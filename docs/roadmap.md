# Roadmap

## M0 — Repository and quality baseline

Independent repository, bilingual documentation, Go CLI skeleton, local verification, and cross-platform CI.

## M1 — Protocol core

Versioned Blueprint, Capability Pack, Plan, Result, Diagnostic, canonical JSON, content hashing, and dependency validation.

## M2 — Deterministic filesystem transactions

Preview, apply, rollback, ownership manifests, conflict detection, root containment, staging, and recovery journals.

## M3 — Agent interfaces

Stable CLI JSON, six MCP tools, pagination, compact result storage, and Codex/Claude Code/Kimi/generic MCP integration packages.

## M4 — Go and PostgreSQL reference application

- [x] Deterministic Go adapter and generated-module dependency lock.
- [x] Bounded PostgreSQL pool and serialized embedded migrations.
- [x] PBKDF2 password hashing, administrator bootstrap, browser sessions, API tokens, and audit events.
- [x] Generated-module formatting, dependency, unit-test, vet, apply, and ownership verification.
- [x] Permission-based RBAC with deny-by-default HTTP wrappers.
- [x] One stateful CRUD module with keyset pagination, optimistic concurrency, transactional audit, and generated RBAC.
- [x] OpenAPI 3.1 contract with authentication, permission, CRUD, pagination, and conflict semantics.
- [x] Vue/Element Plus administration UI generated from the same Blueprint contract and verified through lint, unit test, type check, format check, and production build.
- [x] PostgreSQL integration tests against an isolated schema in an ephemeral CI database.

## M5 — MySQL and platform capabilities

- [x] Go/MySQL 8.4 connection pool, named-lock migrations, identity, RBAC, audit, CRUD, and isolated integration gate.
- [x] Shared Blueprint, domain, HTTP, OpenAPI, and Vue contracts across PostgreSQL and MySQL.
- [x] Organization creation, discovery, membership-scoped authorization, tenant-aware administration, and cross-tenant data isolation.
- [x] Email-bound invitations, member administration, role changes, removals, and concurrent last-administrator protection.
- [x] Tenant-aware durable jobs, idempotent enqueue, leased workers, heartbeat, retry, and dead-letter state.
- [x] Reversible organization deactivation, reactivation, rename, ownership transfer, and owner protection.
- [x] Idempotent email notification enqueue and TLS-only SMTP worker delivery.
- [x] Tenant-aware file metadata, bounded streaming HTTP, atomic local object storage, and mutation compensation.
- [x] Cross-instance database TTL cache with tenant scopes, bounded JSON values, and bounded cleanup.
- [x] Payload-free job administration with tenant-scoped listing and audited dead-job retry.
- [x] HTTP request correlation, safe access logs, low-cardinality metrics, and bounded database readiness.
- [x] Atomic bounded CSV import/export with explicit permissions, tenant scope, audit, and administration UI actions.
- [x] Portable single-stage approval workflows with separation of duties, optimistic concurrency, immutable events, and tenant-safe administration.

## M6 — Language parity

- [x] Java 21/Spring Boot 4.1 Maven foundation for PostgreSQL and MySQL with deterministic output, health/readiness, and automated quality gates.
- [x] Java session/token identity, permission RBAC, transactional audit, dialect migrations, and live-database CI gates.
- [x] Java Blueprint CRUD parity with keyset pagination, optimistic concurrency, transactional audit, and PostgreSQL/MySQL generation gates.
- [x] Java OpenAPI and shared Vue/Element Plus administration parity with locked frontend build gates.
- [x] Java organization-tenancy 0.1 with atomic organization creation, membership RBAC, tenant-scoped CRUD, and administration selection.
- [x] Java organization-tenancy 0.2 with email-bound invitations, member administration, and concurrent last-administrator protection.
- [x] Java organization-tenancy 0.3 with ownership transfer, reversible deactivation, inactive authorization blocking, and owner-safe administration.
- [x] Java background-jobs 0.1 with scoped idempotency, skip-locked leasing, heartbeat renewal, bounded retry, dead state, and an independent worker mode.
- [x] Java notifications 0.1 with normalized idempotent enqueue, worker-only SMTP initialization, mandatory TLS, and bounded delivery timeouts.
- [x] Java file-assets 0.1 with tenant-scoped metadata, bounded multipart streaming, atomic local publication, safe downloads, and mutation compensation.
- [x] Java application-cache 0.1 with scoped bounded JSON, atomic dialect upsert, strict TTL, idempotent invalidation, and bounded cleanup.
- [x] Java job-administration 0.1 with payload-free metadata, tenant-scoped listing, dead-only retry, atomic audit, and shared administration UI.
- [x] Java observability 0.1 with bounded request IDs, safe access logs, label-free Prometheus metrics, and whole-probe readiness deadlines.
- [x] Java csv-import-export 0.1 with fixed field order, bounded parsing, atomic create-only import, spreadsheet-safe export, tenant scope, and audit.
- [x] Java approval-workflows 0.1 with subject verification, separation of duties, optimistic transitions, immutable events, tenant scope, and shared administration UI.
- [x] Java platform capability parity.
- [x] Python 3.12+/FastAPI foundation for PostgreSQL and MySQL with deterministic uv locks, Alembic migrations, liveness/readiness, and frozen quality gates.
- [x] Python session/token identity, permission RBAC, transactional audit, stable OpenAPI, SQLite repository contracts, and live PostgreSQL/MySQL CI gates.
- [x] Python Blueprint CRUD with five portable field types, keyset pagination, optimistic concurrency, transactional audit, stable OpenAPI, and dual-database gates.
- [ ] Python shared Vue administration, organization tenancy, and M5 platform capability parity.

## M7 — General business suite and storefront

Membership, CRM, ERP and inventory, commerce, marketing, payment abstraction, sandbox adapters, and Nuxt storefront.

## M8 — Cross-agent conformance

Codex, Claude Code, Kimi, GLM, DeepSeek, and generic MCP conformance plus measured token benchmarks.

## M9 — 1.0 release

Signed cross-platform releases, stable public schemas, installation guides, upgrade policy, and long-term support documentation.
