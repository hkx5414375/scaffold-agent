# ADR 0031: Java background job administration

- Status: Accepted
- Date: 2026-09-01

## Context

Java workers need a safe operational surface for routine dead-job recovery. Direct
queue-table access can leak email or business payloads, cross an organization
boundary, retry running work, and omit audit evidence.

## Decision

1. Java `job-administration` version `0.1.0` depends on `background-jobs`
   `^0.1.0` and supports PostgreSQL and MySQL.
2. `GET /api/v1/jobs` returns at most 100 stable, cursor-paginated metadata rows
   and accepts only the five queue states as an optional filter.
3. The public Java record, HTTP response, OpenAPI schema, and shared Vue type omit
   payload JSON, scope keys, and deduplication keys.
4. `POST /api/v1/jobs/{id}/retry` accepts only a scoped `dead` row. One database
   transaction locks the row, resets attempts and lease state, queues it, and
   inserts the `jobs:retry` audit event.
5. `jobs:read` and `jobs:manage` are granted only to the administrator role.
   Tenant projects require the authenticated organization for both list and retry.
6. The shared administration view provides status filtering, refresh, bounded
   pagination, and a retry action enabled only for dead jobs.

## Consequences

- Java operators can recover dead work without database credentials or payload
  access.
- Cancellation, payload viewing or editing, and forced takeover of running jobs
  remain deliberately unsupported.
- Live PostgreSQL and MySQL CI jobs exercise the generated transaction and tenant
  predicates in addition to Maven, frontend, and static quality gates.
