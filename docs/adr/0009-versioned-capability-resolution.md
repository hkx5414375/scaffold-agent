# ADR 0009: Versioned capability resolution

- Status: Accepted
- Date: 2026-08-31

## Context

An Engine catalog eventually contains several releases of the same capability.
Keeping only the latest release would make an older, pinned Blueprint fail as soon
as the Engine adds a new capability version. Selecting dependencies greedily can
also reject a valid graph when a lower version satisfies the combined constraints.

## Decision

1. The capability catalog stores packs by capability name and exact semantic version.
2. Every project selection pins one exact version. An omitted version is rejected.
3. Transitive dependency ranges select the highest available version satisfying the complete constraint intersection.
4. Resolution uses deterministic backtracking, so a later dependency constraint can select a lower compatible release instead of producing a false conflict.
5. Output remains dependency-first and sorted independently of map iteration order.
6. Missing versions report the requested constraints, their source packs, and the available versions in one compact diagnostic.
7. Dependency cycles and declared capability conflicts remain hard errors.

## Consequences

- The Engine can retain old capability releases while adding new ones.
- A Blueprint and capability lock continue to identify the exact generated behavior.
- Catalog implementations must keep every supported pack document instead of replacing a name with its newest release.
