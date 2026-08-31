# ADR 0016: Background job administration

- Status: Accepted
- Date: 2026-09-01

## Context

Reliable workers still need an operational surface. Direct database edits can
bypass tenant scope, retry a running job, leak queued email or business payloads,
and omit audit evidence.

## Decision

1. `job-administration` version `0.1.0` depends on `background-jobs` `0.1.x`.
2. Administrators can list bounded job metadata and filter by validated status.
   Payload, scope, and deduplication keys never appear in service, HTTP, OpenAPI,
   or Vue response types.
3. Only `dead` jobs are retryable. Retry locks the row, resets attempts and lease
   state, returns it to `queued`, and writes an audit event in the same transaction.
4. Every read and retry includes organization scope when tenancy is enabled.
5. `jobs:read` and `jobs:manage` are granted only to the administrator role.
6. The Vue page supports status filtering, refresh, bounded pagination, and an
   enabled retry action only for dead jobs.

## Consequences

- Operators no longer need direct queue-table access for routine dead-job recovery.
- Cancellation, payload inspection, payload editing, and forced running-job
  takeover remain intentionally unavailable.
