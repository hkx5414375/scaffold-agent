# ADR 0033: Java atomic CSV import and export

- Status: Accepted
- Date: 2026-09-01

## Context

Generated Java business modules need small spreadsheet-compatible transfers without
letting each coding agent invent parsing, tenant predicates, partial commit rules,
permissions, or formula-injection defenses.

## Decision

1. Java `csv-import-export` version `0.1.0` requires one generated business entity
   and supports PostgreSQL and MySQL.
2. CSV uses strict UTF-8 and the Blueprint field order. Boolean values are lowercase
   `true` or `false`, integers are base-10 int64, datetimes are RFC3339, and blank
   optional cells become null.
3. The service reads at most 5 MiB, parses at most 1,000 data rows, validates the
   complete document, and only then calls persistence. Import creates new IDs and
   never updates, deletes, or matches existing records.
4. Every imported row and one audit event commit in one transaction. A row conflict
   or audit failure rolls back the complete batch.
5. Export performs one identifier-ordered query limited to 1,001 rows inside its
   audit transaction. Actual encoded row sizes are checked against 5 MiB before the
   audit commits, so an oversized export produces no partial response.
6. Text beginning with an apostrophe or spreadsheet formula marker uses the same
   reversible apostrophe escape as the Go adapter.
7. `:import` and `:export` are separate administrator-only permissions. HTTP,
   OpenAPI, and the shared Vue administration actions use those exact permissions.

## Consequences

- Coding agents can add bounded administrative transfer by selecting one capability
  instead of generating another parser, transaction boundary, and UI workflow.
- Larger migrations remain asynchronous background-job and object-storage work;
  this synchronous contract must not be expanded by raising its limits.
