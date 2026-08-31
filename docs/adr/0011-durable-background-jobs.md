# ADR 0011: Durable background jobs

- Status: Accepted
- Date: 2026-08-31

## Context

Files, notifications, imports, exports, approvals, and long-running business work
need one reliable asynchronous execution model. Recreating a queue in every
generated feature wastes agent context and commonly introduces duplicate work,
unbounded retries, lost leases, or cross-tenant execution.

## Decision

1. `background-jobs` version `0.1.0` is a first-party Go capability for PostgreSQL and MySQL.
2. Enqueue validates bounded JSON payloads and uses `(scope, type, idempotency key)` to return an existing job instead of duplicating work.
3. Tenant-enabled projects require an organization identifier in both the service and database schema. Non-tenant projects use an explicit global scope.
4. Workers atomically claim jobs with `FOR UPDATE SKIP LOCKED`, a named lease owner, and a bounded lease expiry. Expired running jobs are reclaimable.
5. A running handler renews its lease every third of the lease duration. Completion and failure require the current, unexpired lease.
6. Handler failures use bounded exponential retry. Exhausted jobs enter a durable `dead` state instead of looping forever.
7. Handler panics become ordinary job failures. Logs include job and worker metadata but never payload content.
8. Generated applications receive an independent `cmd/worker` process and a harmless `system.noop` extension example.
9. PostgreSQL and MySQL integration gates verify enqueue idempotency, leasing, retry, completion, tenant propagation, and stale-lease rejection.

## Consequences

- Later capability packs can enqueue work without defining their own queue tables or worker lifecycle.
- Handlers must honor context cancellation because losing a heartbeat cancels their execution context.
- Administrative job inspection, manual replay, retention cleanup, scheduling, and metrics remain later observability additions.
