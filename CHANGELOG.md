# Changelog

All notable changes follow [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and semantic versioning.

## [1.0.1] - 2026-09-02

### Fixed

- Add a deterministic CycloneDX `serialNumber` so the official
  `actions/attest@v4` SBOM parser accepts and signs release SBOMs.

### Release

- First published stable release. The `v1.0.0` tag failed closed at the SBOM
  attestation gate and produced no GitHub Release; the tag remains unchanged as
  an auditable record.

## [1.0.0] - 2026-09-02

### Added

- Model-neutral Blueprint, Capability Pack, Plan, Result, Diagnostic, ownership
  manifest, deterministic filesystem transaction, rollback, and recovery core.
- Stable JSON CLI and six-tool STDIO MCP server with strict schemas, bounded
  pagination, immutable preview tokens, and compact stored results.
- Go, Java 21/Spring Boot, and Python 3.12+/FastAPI generators for PostgreSQL and
  MySQL, plus shared Vue administration and Nuxt storefront foundations.
- Reusable tenancy, jobs, notifications, file, cache, observability, CSV,
  approval, commerce catalog, customer account, commerce operations, CRM, and
  ERP inventory packs. Commerce operations include deterministic pricing,
  versioned carts, idempotent checkout and payment events, orders, fulfillment,
  returns, refunds, reconciliation, campaigns, coupons, and a no-network
  sandbox payment gateway.
- Cross-language API, migration, behavior, frontend, Agent-host, and token-budget
  conformance gates.
- Deterministic six-platform release archives, CycloneDX SBOM, checksums,
  Sigstore-backed provenance, installation, upgrade, and support policies.

### Security

- Project-root containment, ownership hashes, conflict refusal, staged atomic
  writes, recovery journals, no telemetry by default, and no model credentials
  or model calls in the Engine.

[1.0.1]: https://github.com/hkx5414375/scaffold-agent/releases/tag/v1.0.1
[1.0.0]: https://github.com/hkx5414375/scaffold-agent/tree/v1.0.0
