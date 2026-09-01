# Installation and verification

Download the matching `linux`, `darwin`, or `windows` archive for `amd64` or
`arm64` only from the
[GitHub releases](https://github.com/hkx5414375/scaffold-agent/releases).
Every release includes SHA-256 checksums, a CycloneDX SBOM, a release manifest,
and signed GitHub Artifact Attestations backed by Sigstore.

Verify an asset before extracting it:

```bash
gh attestation verify scaffold-agent_1.0.0_linux_amd64.tar.gz \
  --repo hkx5414375/scaffold-agent
sha256sum -c SHA256SUMS --ignore-missing
```

Extract `scaffold-agent` or `scaffold-agent.exe` into a fixed user-controlled
directory on `PATH`, then run:

```bash
scaffold-agent version --json
scaffold-agent doctor --json
scaffold-agent conformance --json
scaffold-agent benchmark --json
```

Register `scaffold-agent mcp` using the relevant host instructions under
[`integrations/`](../integrations/README.md). Never put a model API key in the
Scaffold Agent MCP entry.
