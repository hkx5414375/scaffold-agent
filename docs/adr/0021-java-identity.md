# ADR 0021: Java identity, RBAC, and transactional audit

- Status: Accepted
- Date: 2026-09-01

## Context

Java projects need the same authentication and authorization guarantees as the Go
reference without copying framework-specific security code on every AI generation.
Storing raw tokens, auditing outside the credential transaction, or returning
different public errors would make the language target unsafe and non-portable.

## Decision

1. `java-service` `0.2.0` adds PostgreSQL and MySQL Flyway migrations for users,
   roles, permissions, sessions, API tokens, and security audit events.
2. Passwords use self-describing PBKDF2-HMAC-SHA256 hashes with 600,000 iterations,
   16 random salt bytes, 32 derived bytes, constant-time comparison, and a dummy
   verification path for missing accounts.
3. Browser sessions expire after 24 hours and use an HttpOnly, SameSite=Lax cookie
   that is secure by default. API tokens expire after 90 days. Only SHA-256 token
   digests are stored; raw secrets are returned once.
4. One MVC interceptor authenticates every protected `/api/v1` route, gives bearer
   credentials precedence, and enforces explicit stable permission annotations.
5. Successful login and API-token creation commit the credential and audit event in
   the same transaction. Denied login and authorization attempts also append bounded,
   control-character-free audit metadata.
6. Bootstrap credentials are environment-only, must be supplied together, and are
   removed after the administrator exists. Startup rejects a disabled or non-admin
   account using the bootstrap email.
7. Public errors remain stable and generic. Unknown internal exceptions are logged
   by type without returning or logging database messages that may contain data.
8. Generated integration tests create only a random PostgreSQL schema or MySQL
   database, migrate twice, exercise both credential types and audit behavior, and
   remove that isolated scope. CI runs both database variants.

## Consequences

- AI agents can rely on one generated identity contract instead of reconstructing
  password, cookie, bearer, permission, and audit behavior from framework defaults.
- The Java adapter is ready for permission-protected business modules, but tenant
  authorization and M5 platform capabilities remain later additive slices.
