# ADR 0017: Safe HTTP observability and database readiness

- Status: Accepted
- Date: 2026-09-01

## Context

Generated services need enough operational evidence to correlate failures, count
requests, and distinguish a live process from one that cannot use its database.
AI-generated instrumentation can otherwise leak credentials, request bodies,
query strings, panic values, or unbounded user-controlled metric labels.

## Decision

1. `observability` version `0.1.0` is a first-party Go capability for PostgreSQL
   and MySQL, with no database migration or third-party telemetry dependency.
2. Middleware accepts only bounded ASCII request IDs and otherwise generates a
   cryptographically random value. The value is returned in `X-Request-ID` and
   stored in request context.
3. Structured access logs contain only request ID, method, URL path, status, and
   duration. Panic values, query strings, headers, and bodies are excluded.
4. Panic recovery returns a generic JSON 500 response. Process-local Prometheus
   output exposes only aggregate request, in-flight, failure, and duration values,
   without user-derived labels.
5. `GET /readyz` checks the configured database with a two-second context deadline
   and returns only `ready` or `unavailable`. `GET /healthz` remains a liveness
   check that does not depend on downstream services.
6. `GET /metrics` and `GET /readyz` do not require application authentication.
   Deployments that expose the service publicly must protect these operational
   endpoints at their ingress or private-network boundary.

## Consequences

- Operators receive a portable minimum signal set without committing generated
  applications to one telemetry vendor.
- Metrics reset when the process restarts and do not aggregate across replicas;
  a Prometheus-compatible collector or later adapter can provide persistence.
- Route-level labels, distributed tracing, exporters, and service-level alerts
  remain explicit future extensions rather than unsafe generated defaults.
