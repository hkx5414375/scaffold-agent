# Agent interface

Scaffold Agent exposes the same application services through JSON CLI commands and a newline-delimited JSON-RPC MCP server. The transport never calls an AI model and never changes the meaning of a Blueprint.

## MCP tools

| Tool | Purpose | Project code writes |
| --- | --- | --- |
| `scaffold_query` | Return compact support, workflow, or managed-project facts | No |
| `scaffold_plan` | Validate a Blueprint and store an immutable generated plan | No generated-file writes |
| `scaffold_preview` | Page through changes and obtain the exact `apply_token` | No |
| `scaffold_apply` | Apply the previewed plan with ownership and hash checks | Yes |
| `scaffold_verify` | Hash managed files and store pageable findings | No generated-file writes |
| `scaffold_result` | Read one bounded page of a stored result | No |

The required workflow is:

```text
scaffold_query -> scaffold_plan -> scaffold_preview -> scaffold_apply -> scaffold_verify
```

`scaffold_apply` refuses calls without the `apply_token` produced for the exact immutable Plan. Large change sets and verification findings are returned through opaque cursors instead of being inserted into model context all at once.

The six-tool surface and storage contracts are implemented. The Go adapter generates a PostgreSQL- or MySQL-backed HTTP service with embedded migrations, session and token authentication, permission-based RBAC, audit events, business CRUD, OpenAPI, Vue administration, and the M5 platform capabilities. The Java adapter generates a Java 21/Spring Boot 4.1 Maven service for PostgreSQL or MySQL and now has parity with the Go platform capability set. The Python 3.12+/FastAPI adapter generates a PostgreSQL or MySQL identity foundation plus one complete Blueprint CRUD entity with deterministic uv locks, Alembic migrations, bounded health/readiness, digest-only credentials, permission RBAC, transactional audit, keyset pagination, optimistic concurrency, stable OpenAPI, organization tenancy 0.1, the shared Vue/Element Plus administration project, and locked quality gates. Unsupported Python capability selections return explicit stable generation errors and never produce partial output.

See `examples/task-service/scaffold.yaml` for the Go business-module contract and
`examples/task-service-java/scaffold.yaml` for the Java-equivalent contract.
Every generated service includes `api/openapi.yaml`, including stable operation IDs,
authentication schemes, permission extensions, request schemas, response schemas,
pagination, and optimistic-lock inputs. Agents should read that contract before
opening transport implementation files.

Generated administration projects use the OpenAPI document as their transport
contract and keep `int64` business values as decimal strings so JavaScript never
silently rounds database identifiers or counters. Before accepting a generated
administration change, run its locked install, lint, unit tests, type check,
production build, and format check as documented in the generated README.

Selecting `organization-tenancy` version `0.1.0` on Go, Java, or Python adds organization creation and
discovery, membership-scoped permission checks, dialect-specific persistence,
tenant-aware administration state, and organization predicates on every generated
business mutation and query. Agents should select an organization once and send
its identifier through `X-Organization-ID`; they do not need to rediscover the
tenant propagation rules from handlers or SQL stores.

The Go and Java `0.2.0` versions additionally generate organization member discovery, 72-hour
email-bound invitation tokens, invitation acceptance, role changes, removals,
last-administrator protection, additive database migrations, OpenAPI operations,
and a member administration view. The raw invitation token is returned once for
delivery; agents must not search the database for it because only its digest is
stored.

The Go and Java `0.3.0` versions add an explicit owner, organization rename, atomic ownership
transfer, reversible deactivation, and reactivation. Transfer only targets an
existing member and promotes that identity to administrator. Agents must not add
a destructive organization-delete shortcut or bypass owner protection. An
inactive organization remains visible to its members but cannot authorize tenant
business requests.

Project capability selections pin exact versions. Transitive dependencies may use
semantic-version ranges; the Engine deterministically selects the highest release
that satisfies the complete graph and records every exact result in the capability
lock. Agents should reuse that lock instead of re-resolving dependency versions.

Selecting `background-jobs` version `0.1.0` generates a database-backed queue and
an independent `cmd/worker`. Feature code should enqueue a bounded JSON payload
through `jobs.Service` with a stable idempotency key. Handlers receive the verified
organization identifier on `jobs.Job`; they must honor context cancellation and
must never log the payload. Retry, lease renewal, dead-letter decisions, and
expired-lease recovery are Engine-provided behavior, not code for an agent to copy.

Selecting `notifications` version `0.1.0` automatically resolves
`background-jobs` `0.1.x`. Feature code calls the generated notification service
(`notifications.Service.EnqueueEmail` in Go or `NotificationService.enqueueEmail`
in Java) with a stable idempotency key; it must not expose a generic unauthorised email
endpoint. The worker requires TLS-only SMTP environment configuration. Agents
must treat queued message bodies as sensitive persisted data and must never put
SMTP credentials in a Blueprint, source file, job payload, or model context.

Selecting `file-assets` version `0.1.0` generates tenant-aware metadata,
streaming multipart upload, bounded keyset listing, attachment download, reversible
metadata deletion, and atomic local object storage. Feature code must use the
generated file service (`files.Service` in Go or `FileAssetService` in Java); it
must not derive object paths from user filenames or expose
`Asset.StorageKey`. `FILE_STORAGE_ROOT` belongs to runtime configuration. Agents
may replace the `BlobStore` adapter but must preserve the 10 MiB request bound,
create-only publication, SHA-256 metadata, compensation, and not-found behavior
for cross-tenant identifiers.

Selecting `application-cache` version `0.1.0` generates a cross-instance database
TTL cache with bounded JSON values. Feature code uses `cache.Service` in Go or
`CacheService` in Java, should namespace keys, select a stable TTL, and treat a
miss as the common missing, expired, or cross-tenant result. Agents must not add a
generic cache HTTP endpoint, place
secrets in cached values, bypass organization scope, or run unbounded cleanup.
The generated purge method accepts a bounded maintenance batch; a later store
adapter may use Redis while preserving the same service contract.

Selecting `job-administration` version `0.1.0` automatically resolves
`background-jobs` `0.1.x`. Administrators can inspect bounded job metadata and
retry only dead jobs. Payloads and deduplication keys are deliberately absent from
all public types. Agents must not add payload viewing or editing, retry running
jobs, bypass organization scope, or update queue rows outside the generated
transactional service.

Selecting `observability` version `0.1.0` adds validated or generated request IDs,
structured access logs without query strings or bodies, low-cardinality
Prometheus counters, and a two-second database readiness probe. `GET /metrics`
contains only aggregate process values; `GET /readyz` returns a generic 503 and
never exposes a database error. Agents should propagate `X-Request-ID` when it is
available, keep user-derived values out of metric labels, and must not add request
bodies, credentials, query strings, or panic values to logs.

Selecting `csv-import-export` version `0.1.0` requires one generated business
entity and adds a Blueprint-derived CSV header, typed parser, header-only template,
atomic create-only import, and audited export. Import and export have separate
administrator permissions. A document is limited to 5 MiB and 1000 rows; one
invalid or conflicting row rolls back the complete import, and an oversized
export returns no partial document. Exported string cells are reversibly escaped
to prevent spreadsheet formula execution. Agents must use the generated field
order and RFC3339 datetimes, must not turn import into an unaudited upsert, and
must not bypass organization scope or the generated limits.

Selecting `approval-workflows` version `0.1.0` on Go or Java requires one generated business
entity and one explicit module workflow named `approval` whose ordered states are
`pending`, `approved`, `rejected`, and `cancelled`. The generated service verifies
the tenant-scoped subject at submission, permits at most one pending request per
subject, prevents requesters from approving or rejecting their own requests, and
uses optimistic versions for every terminal transition. Only the requester may
cancel a pending request. Submitter and reviewer routes have separate permissions;
each successful submission or transition commits the request, an immutable event,
and a security audit atomically. Agents must not add implicit subject mutation,
bypass separation of duties, overwrite event history, or infer extra workflow
states that are absent from the Blueprint.

## STDIO protocol

Run the server with:

```bash
scaffold-agent mcp
```

The server supports MCP protocol versions `2025-11-25`, `2025-06-18`, `2025-03-26`, and `2024-11-05`. Each STDIO message is one UTF-8 JSON-RPC object terminated by a newline. Standard output is reserved for protocol messages; operational failures go to standard error.

Codex can register the local executable with:

```bash
codex mcp add scaffold-agent -- /absolute/path/to/scaffold-agent mcp
```

See the [OpenAI Codex MCP documentation](https://developers.openai.com/codex/mcp) and the [MCP transport specification](https://modelcontextprotocol.io/specification/2025-11-25/basic/transports).

## JSON CLI

All workflow commands emit the versioned `scaffold-agent.io/result/v1alpha1` envelope:

```bash
scaffold-agent query --topic support
scaffold-agent validate --project-root /path/to/project --blueprint scaffold.yaml
scaffold-agent preview --project-root /path/to/project --plan-id plan_...
scaffold-agent apply --project-root /path/to/project --plan-id plan_... --apply-token apply_...
scaffold-agent verify --project-root /path/to/project
scaffold-agent result --project-root /path/to/project --result-id result_... --cursor ...
```

Operator-only recovery commands are also available:

```bash
scaffold-agent rollback --project-root /path/to/project --plan-id plan_...
scaffold-agent recover --project-root /path/to/project --plan-id plan_...
```
