# ADR 0029: Java durable file assets

- Status: Accepted
- Date: 2026-09-01

## Context

The Java reference stack needs safe uploads, metadata, downloads, and deletion
without asking each coding agent to rebuild path validation, size bounds, tenant
filters, filesystem publication, and cross-resource compensation.

## Decision

1. Java `file-assets` version `0.1.0` provides Spring MVC endpoints, relational
   metadata, a `BlobStore` boundary, a local filesystem adapter, OpenAPI, and the
   shared Vue administration view.
2. Servlet multipart limits cap files at 10 MiB and spool parts outside application
   memory. `FileAssetService` streams the part into the selected blob adapter and
   persists its size and SHA-256 integrity value.
3. Local objects live below runtime-only `FILE_STORAGE_ROOT`. Generated internal
   keys are validated independently from user filenames; absolute paths, traversal,
   backslashes, unsupported key characters, and symbolic-link directories are
   rejected.
4. Uploads write a same-directory temporary object, force its content, and publish
   through a create-only hard link. Existing objects are never overwritten and
   incomplete bytes are never visible at the final key.
5. Metadata creation follows object publication. A failed metadata transaction
   removes the object. Deletion first soft-deletes metadata and its audit event; a
   failed object deletion restores metadata so the request remains retryable.
6. Every metadata operation includes the authenticated organization when tenancy is
   selected. Foreign-tenant identifiers return the same not-found result as unknown
   identifiers.
7. Download responses use UTF-8 attachment disposition and `nosniff`. JSON and
   OpenAPI expose public metadata and SHA-256, never the internal storage key.

## Consequences

- Go and Java now share the same file capability and administration contract.
- Local multi-instance deployments require a shared durable volume. S3-compatible
  adapters may replace `BlobStore` only if they preserve bounded streaming,
  create-only publication, and compensation behavior.
- Orphan reconciliation and delayed physical retention remain later background-job
  and administration work.
