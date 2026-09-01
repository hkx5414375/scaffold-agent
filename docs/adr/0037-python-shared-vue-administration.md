# ADR 0037: Python shared Vue administration

- Status: Accepted
- Date: 2026-09-01

## Context

Python CRUD needs a usable administration surface, but a Python-specific frontend
copy would create three drifting implementations and force AI agents to spend tokens
reconciling equivalent Go, Java, and Python screens.

## Decision

1. The Python adapter renders the existing `vue-admin` `0.2.0` templates directly.
   It supplies the same backend-neutral project, business field, route, permission,
   and feature flags used by Go and Java.
2. `/api/v1/auth/login` and `/api/v1/auth/me` both return the shared
   `{ "principal": ... }` shape. Logout, flat safe error envelopes, decimal-string
   versions and `int64` fields, keyset pages, and full replacement payloads also
   preserve the shared transport contract.
3. Blueprint `string`, `text`, `bool`, `int64`, and `datetime` fields map to the same
   TypeScript types, default values, inputs, labels, optional handling, CRUD actions,
   and pagination behavior across all three backends.
4. Generated administration projects are locked by npm and must pass ESLint, Vitest,
   TypeScript production build, and Prettier checks in the shared frontend CI gate.
5. The adapter records every frontend file under the existing `vue-admin` ownership
   boundary. Python does not fork or shadow shared templates.

## Consequences

- AI agents can request `admin_ui: element-plus` for Python and receive the same
  maintained interface as Go and Java without generating frontend boilerplate.
- Future shared administration fixes apply to all three backends through one template
  package and one build matrix.
