# ADR 0048: Python portable single-stage approval workflows

- Status: Accepted
- Date: 2026-09-01

## Context

Python projects generated from the same Blueprint must preserve the Go and Java
approval contract instead of inventing FastAPI-specific states, authorization
rules, or transaction boundaries. PostgreSQL and MySQL must reject the same
conflicts and retain the same evidence.

## Decision

1. Python `approval-workflows` version `0.1.0` requires exactly one generated
   business entity and the ordered `approval` workflow states `pending`,
   `approved`, `rejected`, and `cancelled`.
2. Submission verifies the scoped business subject and atomically stores the
   pending request, immutable event, and audit event. PostgreSQL and SQLite use a
   partial unique index; MySQL uses an indexed generated pending-subject column.
3. Requesters cannot approve or reject their own requests. Only the requester may
   cancel a pending request, and rejection requires a non-empty reason.
4. Every terminal operation supplies the current version. The conditional
   database update repeats organization, pending-state, version, and actor-rule
   predicates so stale or racing transitions commit no evidence.
5. An internal per-request sequence establishes event causality without exposing
   a new public field. Request and event persistence, plus audit persistence, share
   one SQLAlchemy transaction and roll back together.
6. Submitter and reviewer permissions remain separate. The shared administration
   UI and OpenAPI contract use the same routes, payloads, statuses, and decimal
   version strings as the Go and Java adapters.
7. A decision records evidence but never changes the generated business entity.

## Consequences

- Go, Java, and Python agents can select the same capability without relearning
  workflow semantics or database-specific concurrency rules.
- Multi-stage routing, delegation, quorum, timers, escalation, and implicit
  business effects remain outside this version.
