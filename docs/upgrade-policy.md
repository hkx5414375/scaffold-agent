# Upgrade and compatibility policy

Scaffold Agent 1.x follows semantic versioning. Patch releases fix defects and
security issues; minor releases add backward-compatible capabilities. Breaking
CLI, MCP, Blueprint, Plan, Result, Manifest, or Capability Pack semantics
requires a new major version and wire API identifier.

Installing a new Engine binary never changes generated projects. Verify the new
release, keep the previous binary, and run `version`, `doctor`, `conformance`,
and `benchmark` before replacing the normal executable. Upgrade a managed
project only through query, `plan --action upgrade`, preview, apply with the
preview token, and verify. Capability locks change only when the Blueprint and
new immutable Plan explicitly select compatible versions.

The existing `v1alpha1` spelling is retained for early-user compatibility but
is frozen as the stable 1.x wire identifier beginning with Scaffold Agent
v1.0.1, the first published stable release. Optional compatible additions are allowed; removals, changed meanings,
new requirements, or changed tool semantics are not.

File rollback cannot reverse an already-applied business database migration.
Use the generated project's documented database migration policy for that case.
