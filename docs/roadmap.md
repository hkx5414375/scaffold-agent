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
- [ ] Import/export.
- [ ] Portable approval workflows.

## M6 — Language parity

Java and Python adapters with the same Blueprint, transport, security, database, and quality contracts.

## M7 — General business suite and storefront

Membership, CRM, ERP and inventory, commerce, marketing, payment abstraction, sandbox adapters, and Nuxt storefront.

## M8 — Cross-agent conformance

Codex, Claude Code, Kimi, GLM, DeepSeek, and generic MCP conformance plus measured token benchmarks.

## M9 — 1.0 release

Signed cross-platform releases, stable public schemas, installation guides, upgrade policy, and long-term support documentation.
