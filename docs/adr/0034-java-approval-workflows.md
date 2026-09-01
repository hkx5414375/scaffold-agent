# ADR 0034: Java portable single-stage approval workflows

- Status: Accepted
- Date: 2026-09-01

## Context

Java projects generated from the same Blueprint must preserve the Go approval
contract instead of inventing Spring-specific states, authorization rules, or
transaction boundaries. PostgreSQL and MySQL must also reject the same conflicts
and retain the same evidence.

## Decision

1. Java `approval-workflows` version `0.1.0` requires exactly one generated
   business entity and the ordered `approval` workflow states `pending`,
   `approved`, `rejected`, and `cancelled`.
2. Submission verifies the business subject inside the current organization and
   atomically stores the pending request, immutable event, and audit event. A
   dialect-specific unique constraint permits only one pending request per scoped
   subject.
3. Requesters cannot approve or reject their own requests. Only the requester may
   cancel a pending request, and rejection requires a non-empty reason.
4. Every terminal operation supplies the current version. The database update
   repeats the pending-state, version, organization, and actor-rule predicates so
   stale or racing transitions commit no evidence.
5. Submitter and reviewer permissions remain separate. The shared administration
   UI and OpenAPI contract use the same routes, payloads, statuses, and decimal
   version strings as the Go adapter.
6. A decision records evidence but never changes the generated business entity.

## Consequences

- Go and Java agents can select the same capability without relearning workflow
  semantics or database-specific concurrency rules.
- Multi-stage routing, delegation, quorum, timers, escalation, and implicit
  business effects remain outside this version.
