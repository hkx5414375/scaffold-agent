# ADR 0008: Organization tenancy capability

- Status: Accepted
- Date: 2026-08-31

## Context

Files, jobs, notifications, approvals, and business suites cannot be safely
composed until every request and database operation has an explicit tenant
boundary. Requiring each coding agent to rediscover and thread that boundary
through authentication, SQL, OpenAPI, and frontend state would waste tokens and
create isolation defects.

## Decision

1. `organization-tenancy` version `0.1.0` is a first-party Go capability for PostgreSQL and MySQL.
2. An authenticated identity can create and list organizations. Creation atomically adds the creator as an administrator and records an audit event.
3. Protected business routes require `X-Organization-ID`. The scoped authorizer resolves membership, applies the member role's permission assignments, audits denials, and passes the verified organization to the use case.
4. Generated business entities, queries, mutations, keyset pages, optimistic locks, and unique indexes include the verified organization identifier.
5. Cross-tenant reads deliberately return the existing not-found error instead of revealing that a record exists elsewhere.
6. PostgreSQL and MySQL receive dialect-specific organization migrations and stores. Generated live integration tests prove that records cannot be read through a different organization.
7. The Vue administration project restores a valid organization selection, sends the organization header automatically, supports first-organization creation, and remounts business views when the selection changes.
8. The capability accepts no configuration in version `0.1.0`; unknown configuration is rejected instead of silently ignored.

## Consequences

- Later capability packs can depend on one stable tenant context instead of defining their own headers and membership tables.
- Global identity operations such as creating a personal API token keep global RBAC, while business routes use organization membership RBAC.
- Membership invitations, member administration, organization deletion, and tenant-aware background jobs remain explicit follow-up work.
