# ADR 0006: Generated Vue administration UI

- Status: Accepted
- Date: 2026-08-31

## Context

AI coding agents should not repeatedly recreate login flows, API error handling,
CRUD forms, pagination, and frontend quality configuration for every service. A
generated frontend must still follow the same Blueprint and transport semantics
as the generated backend, remain deterministic, and be safe across JavaScript's
numeric limits.

## Decision

1. `admin_ui: element-plus` generates a Vue 3, Element Plus, Pinia, TypeScript, and Vite project under `web/admin`.
2. Dependencies are pinned exactly and include a committed npm lockfile. The generated project requires Node.js 22.12 or later.
3. Browser requests send the generated HttpOnly session cookie, decode stable API error envelopes, and restore the current principal before rendering protected views.
4. The first Blueprint module and entity produce typed list, create, replace, and delete screens with keyset pagination and optimistic-lock versions.
5. PostgreSQL `bigint` values, including optimistic-lock versions, cross JSON and OpenAPI as decimal strings. This prevents JavaScript precision loss while preserving exact values for the Go service.
6. Generated administration code must pass ESLint, Vitest, `vue-tsc`, a Vite production build, and Prettier checks after a locked install.
7. Continuous integration repeats the complete generation and frontend quality gate on Node.js 24 and audits the committed dependency lock.

## Consequences

- Agents receive a mature administration surface by selecting one Blueprint value instead of regenerating common frontend infrastructure.
- Backend entities, OpenAPI schemas, TypeScript models, forms, permissions, pagination, and conflict behavior are produced by one immutable Plan.
- The initial slice deliberately supports one business entity and no storefront; unsupported shapes fail explicitly until the protocol and capability packs expand.
