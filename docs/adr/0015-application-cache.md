# ADR 0015: Cross-instance application cache

- Status: Accepted
- Date: 2026-09-01

## Context

AI-generated features frequently add ad hoc process maps or introduce Redis only
for simple derived-value caching. Process-local maps diverge across replicas and
often omit tenant scope, expiration bounds, payload limits, and cleanup. Requiring
another infrastructure service by default also makes generated projects harder to
run and test.

## Decision

1. `application-cache` version `0.1.0` stores bounded JSON values in the selected
   PostgreSQL or MySQL database and is coherent across service instances.
2. Keys contain at most 191 bytes, values at most 256 KiB, and TTL is strictly
   positive and no longer than 30 days.
3. Tenant-enabled projects use the organization identifier as the scope key. Global
   projects force the literal `global` scope and ignore caller-supplied organization
   text.
4. Reads return only values whose expiration is after the store-provided comparison
   time. Missing, expired, and cross-tenant keys share `cache.ErrMiss`.
5. Set is an atomic dialect-specific upsert. Delete is idempotent. Expired cleanup
   is explicitly bounded to 10,000 rows per call through `PurgeExpired`.
6. The capability is internal application infrastructure and exposes no generic
   cache HTTP endpoint or administration UI.

## Consequences

- Generated features reuse one stable API without requiring Redis for correctness.
- The database cache favors portability and cross-instance semantics over maximum
  throughput. A later adapter may implement the same `Store` contract with Redis.
- Operators or a generated maintenance job must invoke bounded expired-row cleanup.
