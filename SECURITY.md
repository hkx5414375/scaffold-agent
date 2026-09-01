# Security Policy

## Supported versions

The latest stable 1.x minor receives security fixes. The immediately previous minor receives security fixes for six months after its successor is released. See [SUPPORT.md](SUPPORT.md).

## Reporting a vulnerability

Do not open a public issue for vulnerabilities that could expose source code, credentials, generated projects, or filesystem access. Use GitHub private vulnerability reporting for this repository after the public repository is available.

Reports should include affected versions, reproduction steps, impact, and any suggested mitigation. Please do not include real credentials or private customer data.

## Security principles

- Local stdio is the default MCP transport.
- Filesystem writes are restricted to an explicitly selected project root.
- Capability packs cannot execute arbitrary installation scripts.
- Telemetry is disabled until the user explicitly opts in.
- Generated secrets are never committed; only documented placeholders may be created.

