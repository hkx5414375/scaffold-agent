# ADR 0012: Organization lifecycle and ownership

- Status: Accepted
- Date: 2026-08-31

## Context

Member administration prevents the final administrator from disappearing, but it
does not identify who may suspend a tenant or hand ultimate control to another
identity. Deleting an organization would cascade business data and makes an
unsafe default for generated applications.

## Decision

1. `organization-tenancy` version `0.3.0` extends version `0.2.0` with one explicit owner and reversible organization state.
2. The creator becomes both owner and administrator. Ownership is separate from the reusable `admin` and `user` RBAC roles.
3. An active administrator may rename an organization. Only the current owner may transfer ownership, deactivate, or reactivate it.
4. Ownership transfer requires an existing member, atomically promotes that member to administrator, and leaves the previous owner as an administrator.
5. The current owner cannot be demoted or removed until ownership has transferred. Organization-row locks serialize ownership, role, removal, and state mutations.
6. Deactivation retains all tenant data but makes membership authorization fail. The owner may still reactivate through an authenticated store-authorized route.
7. Existing leased and queued background jobs are allowed to finish after deactivation; new user-scoped requests cannot enqueue more work because tenant authorization is blocked.
8. Successful and denied lifecycle mutations write audit events transactionally. The Vue administration UI exposes rename, ownership transfer, deactivation, reactivation, inactive-state labels, and owner protection.
9. PostgreSQL and MySQL use additive migration `000070`; migrations `000050` and `000060` remain byte-identical.

## Consequences

- Generated applications get a recoverable tenant lifecycle rather than a destructive delete endpoint.
- Capabilities can treat tenant authorization as proof that an organization is active.
- Permanent organization erasure and data-retention policy remain explicit application-specific operations.
