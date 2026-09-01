# ADR 0032: Java safe HTTP observability

- Status: Accepted
- Date: 2026-09-01

## Context

Generated Java services need portable request correlation, minimum HTTP metrics,
and a readiness result that cannot wait indefinitely on connection acquisition.
Instrumentation written independently by coding agents can leak query strings,
bodies, headers, exception values, or unbounded user-controlled metric labels.

## Decision

1. Java `observability` version `0.1.0` supports PostgreSQL and MySQL without a
   database migration or telemetry-vendor dependency.
2. A servlet filter accepts only 8–64 ASCII letters, digits, hyphens, or
   underscores as `X-Request-ID`; otherwise it generates a random UUID. The safe
   identifier is returned in the response and stored as a request attribute.
3. Access logs contain only request ID, HTTP method, a control-free path bounded to
   2,048 code points, response status, and duration. Query strings, headers,
   bodies, and exception values are never read for logging.
4. `GET /metrics` emits aggregate Prometheus request, in-flight, failure, and
   duration values without labels. Metrics are process-local and reset on restart.
5. `GET /readyz` executes connection acquisition and validation in a cancellable
   virtual-thread task and bounds the entire public probe to two seconds. It returns
   only `ready` or `unavailable`; `GET /healthz` remains database-independent.
6. Operational endpoints require ingress or private-network protection rather than
   application authentication. Live database CI verifies readiness for both
   supported database drivers.

## Consequences

- Operators gain correlation, scrapeable minimum metrics, and a truthful bounded
  readiness signal without an exporter dependency.
- Route labels, tracing exporters, cross-replica aggregation, and alert policy
  remain explicit later extensions.
