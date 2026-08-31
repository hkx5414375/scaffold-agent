# ADR 0018: Atomic bounded CSV import and audited export

- Status: Accepted
- Date: 2026-09-01

## Context

Generated business applications repeatedly need spreadsheet-compatible transfer,
but ad hoc AI implementations often accept unbounded bodies, partially commit a
failed batch, omit tenant predicates, reuse broad CRUD permissions, or export
cells that a spreadsheet executes as formulas.

## Decision

1. `csv-import-export` version `0.1.0` is a first-party Go capability for
   PostgreSQL and MySQL and requires exactly one generated business entity.
2. The CSV header is the Blueprint's mutable field order. Boolean values are
   `true` or `false`, integers are base-10 int64 values, and datetimes are RFC3339.
   Blank optional cells become null; required cells cannot be blank.
3. A document is limited to 5 MiB and 1000 data rows. Parsing and type validation
   finish before persistence. All imported rows and one audit event commit in one
   database transaction; any invalid or conflicting row commits nothing.
4. Import is create-only and generates new identifiers. It does not silently
   update, delete, or match existing records.
5. Export executes one tenant-scoped, identifier-ordered bounded query and records
   an audit event. More than 1000 rows or 5 MiB returns an error without a partial
   HTTP document.
6. Exported strings beginning with spreadsheet formula markers or apostrophes use
   a reversible apostrophe escape understood by the generated importer.
7. `:import` and `:export` permissions are separate and granted only to the
   administrator role. HTTP, OpenAPI, and Vue actions use those exact permissions.

## Consequences

- Small and medium administrative transfers are portable and deterministic across
  both supported databases without generated applications depending on Scaffold
  Agent at runtime.
- The bounded synchronous contract deliberately rejects large data migrations;
  those need a future background-job and object-storage workflow.
- This version exports mutable fields for round-trip creation, not database IDs,
  versions, timestamps, deleted records, or arbitrary joined reports.
