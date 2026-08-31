# ADR 0028: Java email notifications through durable jobs

- Status: Accepted
- Date: 2026-09-01

## Context

The Java reference stack needs the same notification boundary as Go without
duplicating queue, retry, lease, or SMTP security logic in every generated feature.
SMTP credentials and queued message bodies are sensitive, while Java agents may
otherwise be tempted to add an arbitrary public send endpoint or initialize SMTP in
the web process.

## Decision

1. Java `notifications` version `0.1.0` depends on `background-jobs` `^0.1.0`.
2. `NotificationService.enqueueEmail` validates and normalizes a single-recipient
   message, requires a stable idempotency key, caps combined UTF-8 bodies at 512 KiB,
   and queues `notifications.email.deliver` without exposing an HTTP endpoint.
3. The SMTP handler is created only in worker mode. Missing or invalid runtime
   configuration fails worker startup instead of silently dropping delivery.
4. `SMTP_ADDRESS`, `SMTP_FROM`, `SMTP_TLS_MODE`, and the optional paired
   `SMTP_USERNAME`/`SMTP_PASSWORD` are runtime-only. Implicit TLS or required
   STARTTLS, TLS 1.2 or newer, hostname verification, and bounded connection, read,
   and write timeouts are mandatory.
5. Delivery supports UTF-8 text, HTML, and multipart alternative messages. Errors
   stored by the job system do not contain message bodies or SMTP credentials.
6. Queued message content remains in `background_jobs.payload_json` until the
   deployment's database retention policy removes it.

## Consequences

- Go and Java expose the same capability and durable-delivery semantics through
  language-native service names.
- Worker deployment requires explicit SMTP configuration; the normal web process
  does not.
- Provider templates, receipts, SMS, push, and payload-retention administration stay
  outside this version.
