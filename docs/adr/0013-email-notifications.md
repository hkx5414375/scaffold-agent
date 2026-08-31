# ADR 0013: Email notifications through durable jobs

- Status: Accepted
- Date: 2026-09-01

## Context

Invitations, approvals, exports, and commerce events need asynchronous delivery.
Giving every coding agent a separate SMTP loop would duplicate retry and
idempotency logic, risk header injection, and commonly log message contents or
credentials.

## Decision

1. `notifications` version `0.1.0` is a first-party Go capability that depends on `background-jobs` `^0.1.0`.
2. Feature code submits a normalized email plus a required stable idempotency key. The generated service validates the mailbox, blocks header injection, bounds the combined body to 512 KiB, and queues job type `notifications.email.deliver`.
3. The worker initializes environment-backed handlers before polling. Initialization is exactly once and fails startup when notification configuration is invalid.
4. SMTP configuration is runtime-only: `SMTP_ADDRESS`, `SMTP_FROM`, `SMTP_TLS_MODE`, and the optional `SMTP_USERNAME`/`SMTP_PASSWORD` pair. TLS is mandatory through implicit TLS or STARTTLS with TLS 1.2 or newer.
5. The SMTP transport honors context deadlines, composes UTF-8 text, HTML, or multipart alternative messages, and never logs or includes message bodies in errors.
6. Delivery reuses job idempotency, leases, heartbeat, bounded retries, and dead-letter state. No public “send arbitrary email” HTTP endpoint is generated; business capabilities invoke the internal service after their own authorization.
7. Message content is stored in the durable job payload until application retention removes it. Deployments must apply their database encryption, access, backup, and retention policy to that payload.

## Consequences

- Adding notifications automatically resolves and locks the reliable job dependency.
- Future SMS, push, webhook, and template capabilities can reuse the same queue contract without changing SMTP behavior.
- Provider-specific delivery receipts, templates, and job-retention administration remain later capability versions.
