# ADR 0050: Portable commerce catalog contract

- Status: Accepted
- Date: 2026-09-01

## Context

A useful storefront needs a real catalog contract rather than a generic CRUD
entity renamed to product. AI agents should not repeatedly rediscover SKU
normalization, money representation, publication state, public visibility,
optimistic concurrency, tenancy, and audit boundaries.

## Decision

1. `commerce-catalog` version `0.1.0` generates products independently from a
   project-defined business module. It can therefore compose with custom modules.
2. A product has a canonical upper-case ASCII SKU, name, bounded description,
   non-negative `int64` price in the currency's smallest unit, ISO-style upper-
   case three-letter currency, publication status, optimistic version, and
   timestamps. SKU uniqueness is organization-scoped when tenancy is enabled.
3. Publication follows only `draft -> active -> archived`. An archived product
   cannot be edited, republished, or deleted through a shortcut. Price and content
   changes retain the current publication status.
4. Administration operations require separate read and write permissions. Every
   create, update, publish, and archive mutation commits its audit event in the
   same transaction and repeats scope, state, and version predicates in SQL.
5. Public catalog routes require no administrator identity and return only active
   products. Tenant-enabled projects still require an explicit organization scope
   and hide inactive organizations and cross-organization identifiers.
6. Public and administration pages use bounded keyset pagination. Prices and
   versions cross JSON and OpenAPI as decimal strings.

## Consequences

- Catalog semantics become a reusable capability rather than Blueprint field
  conventions an agent must infer.
- Inventory availability, price lists, variants, media, tax, search, and channel
  host mapping remain separate composable capabilities.
- Go, Java, Python, PostgreSQL, MySQL, shared Vue administration, and Nuxt public
  catalog conformance gates now protect this contract.
