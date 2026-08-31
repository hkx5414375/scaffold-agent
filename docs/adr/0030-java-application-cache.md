# ADR 0030: Java cross-instance application cache

- Status: Accepted
- Date: 2026-09-01

## Context

Java features need a small cache contract without generating divergent process-local
maps or requiring Redis for basic derived values. The contract must remain coherent
across replicas and prevent unbounded keys, values, lifetimes, and cleanup work.

## Decision

1. Java `application-cache` version `0.1.0` stores JSON in the selected PostgreSQL
   or MySQL database and exposes no HTTP or administration endpoint.
2. `CacheService` limits UTF-8 keys and tenant scope values to 191 bytes, encoded
   values to 256 KiB, and TTL to the positive interval from one nanosecond through
   30 days.
3. Tenant projects require and use the organization identifier as `scope_key`.
   Global projects always use `global` and ignore caller-supplied organization text.
4. Reads return only entries whose expiry is later than the service clock. Missing,
   expired, and cross-tenant entries share `CacheException.Kind.MISS`.
5. Set uses dialect-native atomic upsert. Delete is idempotent. Explicit expiry
   cleanup is limited to 10,000 rows per call and removes oldest expirations first.
6. Values are encoded and decoded by the generated Jackson configuration. Business
   code must namespace keys and must not cache secrets.

## Consequences

- Generated Java features gain one replica-coherent cache without extra runtime
  infrastructure.
- PostgreSQL and MySQL retain dialect-specific upsert and bounded-delete SQL behind
  one repository contract.
- A later Redis adapter may replace the repository only if scope, size, TTL, miss,
  delete, and bounded-cleanup semantics remain unchanged.
