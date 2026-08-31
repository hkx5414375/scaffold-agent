# ADR 0014: Durable file assets

- Status: Accepted
- Date: 2026-09-01

## Context

Generated applications commonly need uploads, metadata listing, downloads, and
deletion. Recreating this boundary with an AI agent is risky: naive implementations
buffer unbounded bodies, trust user-controlled paths, expose cross-tenant objects,
overwrite existing files, or leave metadata and binary content inconsistent.

## Decision

1. `file-assets` version `0.1.0` provides one model-neutral capability for Go,
   PostgreSQL, MySQL, OpenAPI, and the Vue administration application.
2. Binary objects live below `FILE_STORAGE_ROOT`; relational databases contain
   tenant-aware metadata, SHA-256 integrity values, lifecycle state, and audit events.
3. Uploads stream through a 10 MiB limit into a same-directory temporary file. The
   local store publishes with a create-only hard link, so existing objects are never
   overwritten and incomplete bytes are never visible.
4. Object keys are generated internally. The local adapter rejects absolute paths,
   traversal segments, backslashes, unsupported characters, and symlinked object
   directories.
5. Metadata creation follows object publication. A failed metadata transaction
   removes the object. Deletion first soft-deletes metadata; a failed object removal
   restores metadata so the operation remains retryable.
6. Every metadata query includes the authenticated organization when tenancy is
   enabled. A valid identifier from another organization is reported as not found.
7. Download responses use a standards-based attachment filename, disable content
   sniffing, and never expose the internal storage key.

## Consequences

- Generated services gain safe local storage without requiring an external object
  storage dependency during development.
- Local storage assumes one shared durable volume. A future adapter may implement
  the same `BlobStore` contract for S3-compatible services without changing use cases.
- Orphan reconciliation and delayed physical purge remain observability/job
  administration work; current mutation compensation covers synchronous failures.
