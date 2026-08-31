# ADR 0005: Generated OpenAPI contract

- Status: Accepted
- Date: 2026-08-31

## Context

Different coding agents should not need to inspect every HTTP handler to discover
routes, request shapes, authentication, permissions, pagination, and conflicts.
That repeated exploration wastes tokens and can produce inconsistent clients.

## Decision

1. Every Go service includes a deterministic OpenAPI 3.1 document at `api/openapi.yaml`.
2. The contract is generated from the same normalized data used by HTTP, domain, persistence, and migration templates.
3. Protected operations declare both session-cookie and bearer-token security schemes.
4. Permission-protected operations expose their stable code through `x-required-permission`.
5. CRUD schemas preserve field types, required inputs, response metadata, pagination, and optimistic-lock versions.
6. Generator tests parse the rendered YAML and assert the selected business path; generated-module tests continue to compile and vet the corresponding handlers.

## Consequences

- Agents and generated frontends can use one compact transport truth source before reading implementation code.
- A Blueprint change updates server code and the machine-readable contract in the same immutable Plan.
- Full request/response conformance tests remain a later gate; syntax and structural generation are covered now.
