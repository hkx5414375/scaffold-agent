# ADR 0010: Organization member administration

- Status: Accepted
- Date: 2026-08-31

## Context

Tenant isolation is not operationally complete when only the creator can belong to
an organization. Generated applications need a reusable way to invite identities,
assign organization roles, remove access, and preserve a viable administrator
without asking each coding agent to design security-sensitive token and transaction
behavior again.

## Decision

1. `organization-tenancy` version `0.2.0` extends, rather than rewrites, version `0.1.0`.
2. Administrators can list members, create invitations, change member roles, and remove members through scoped HTTP routes and the generated Vue administration UI.
3. Invitations are bound to a normalized email, expire after 72 hours, and return the raw acceptance token only once. Persistence stores a SHA-256 digest.
4. Acceptance requires an authenticated identity with the invited email. Invalid, expired, accepted, and wrong-email tokens share one public error.
5. Member changes lock the organization row. Concurrent demotions and removals therefore cannot remove the final administrator.
6. Successful and denied mutations write audit events in the same transaction that establishes the result.
7. PostgreSQL and MySQL use dialect-specific additive migration `000060`; the applied `000050` tenant migration remains byte-identical.
8. Version `0.1.0` remains selectable, while `0.2.0` adds separate member service, transport, persistence, OpenAPI, tests, and administration outputs.

## Consequences

- A generated project has a usable organization access lifecycle before notification delivery is added.
- Invitation delivery stays outside this capability; callers must deliver the one-time token securely until a notification pack is selected.
- Organization rename, deletion, ownership transfer, and tenant-aware background work remain later additions.
