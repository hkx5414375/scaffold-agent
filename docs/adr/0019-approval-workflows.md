# ADR 0019: Portable single-stage approval workflows

- Status: Accepted
- Date: 2026-09-01

## Context

Generated business applications often need approval, but ad hoc AI implementations
commonly mix authorization with state mutation, allow self-approval, omit tenant
predicates, lose decision history, or introduce different states and meanings across
PostgreSQL and MySQL.

## Decision

1. `approval-workflows` version `0.1.0` is a first-party Go capability for
   PostgreSQL and MySQL and requires exactly one generated business entity.
2. The module must explicitly declare one workflow named `approval` with the ordered
   states `pending`, `approved`, `rejected`, and `cancelled`. Other shapes fail
   generation instead of being partially interpreted.
3. Submission verifies the subject in the current organization scope. At most one
   pending request may exist for the same scoped subject, enforced by each database.
4. A requester cannot approve or reject their own request. Only the requester may
   cancel a pending request. Rejection requires a reason.
5. Every terminal transition requires the current optimistic version and is
   conditionally enforced again by the database update, including the actor rule.
6. Submission and each successful transition atomically commit the request, one
   immutable event, and one audit event. A failed operation commits none of them.
7. Submitter permissions (`approvals:submit`, `approvals:mine`) are separate from
   reviewer permissions (`approvals:read`, `approvals:decide`). The user role receives
   only submitter permissions; the administrator role receives both sets.
8. Approval records a decision but does not mutate the subject. Business-specific
   effects must be an explicitly designed later capability or application handler.

## Consequences

- An AI can add a complete, portable approval boundary by selecting one capability
  and declaring four states instead of regenerating authorization, transactions,
  SQL, OpenAPI, tests, and administration screens.
- The initial contract intentionally excludes multi-stage routing, delegation,
  quorum, timers, escalation, dynamic forms, and implicit business side effects.
- Terminal requests remain as evidence and a subject may be submitted again after
  approval, rejection, or cancellation.
