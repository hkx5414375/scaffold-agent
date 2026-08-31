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

The six-tool surface and storage contracts are implemented. The Go adapter generates a PostgreSQL- or MySQL-backed HTTP service with embedded migrations, session and token authentication, permission-based RBAC, and audit events. It can also generate the first complete CRUD slice from one Blueprint module and entity. When `admin_ui: element-plus` is selected, the same Plan generates `web/admin` with login, session restoration, Blueprint-driven CRUD, cursor pagination, optimistic-lock updates, stable API errors, and an exact dependency lock. Unsupported storefronts, authentication selections, field types, workflows, pages, and wider module shapes are rejected instead of being silently omitted. Java and Python currently return the stable `generator.adapter.unavailable` diagnostic.

See `examples/task-service/scaffold.yaml` for the currently supported business-module contract.
Every generated service includes `api/openapi.yaml`, including stable operation IDs,
authentication schemes, permission extensions, request schemas, response schemas,
pagination, and optimistic-lock inputs. Agents should read that contract before
opening transport implementation files.

Generated administration projects use the OpenAPI document as their transport
contract and keep `int64` business values as decimal strings so JavaScript never
silently rounds database identifiers or counters. Before accepting a generated
administration change, run its locked install, lint, unit tests, type check,
production build, and format check as documented in the generated README.

Selecting `organization-tenancy` version `0.1.0` adds organization creation and
discovery, membership-scoped permission checks, dialect-specific persistence,
tenant-aware administration state, and organization predicates on every generated
business mutation and query. Agents should select an organization once and send
its identifier through `X-Organization-ID`; they do not need to rediscover the
tenant propagation rules from handlers or SQL stores.

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
