# Release procedure

Only maintainers publish a release. The tag-triggered workflow is the sole
binary publication path.

1. Confirm `main` is clean and all CI jobs pass.
2. Re-run `go test ./...`, `go vet ./...`, `conformance`, and `benchmark`.
3. Confirm public schema snapshot changes are either absent or explicitly
   compatible under the upgrade policy.
4. Update `CHANGELOG.md`, installation instructions, support policy, and the
   target version in release notes.
5. Create an annotated tag from the reviewed commit:

   ```bash
   git tag -a v1.0.1 -m "Scaffold Agent v1.0.1"
   git push origin v1.0.1
   ```

6. The release workflow rebuilds and tests the source, produces deterministic
   Linux/macOS/Windows archives for amd64/arm64, generates checksums, a release
   manifest and CycloneDX SBOM, signs provenance and SBOM statements with the
   workflow's short-lived OIDC identity, verifies one packaged binary, and then
   creates the GitHub release.
7. From a separate checkout, download one release archive and verify it using
   the public installation instructions.

Never publish binaries built on a maintainer workstation, bypass a failed
attestation, reuse a release tag, or store a signing key in repository secrets.
