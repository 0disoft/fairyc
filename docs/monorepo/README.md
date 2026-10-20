# Monorepo

- Status: Draft
- Repository Type: monorepo

## Repository Type Contract

This repository type owns workspace boundaries, package ownership, dependency policy, and change coordination.

## Source of Truth

- Product decision: `docs/product/02-spec.md`
- Technical owner: UNASSIGNED
- Related ADRs: `docs/adr/0001-initial-architecture-boundaries.md`, `docs/adr/0002-contract-source-of-truth.md`, `docs/adr/0008-versioned-contract-boundaries.md`, and `docs/adr/0009-stage-0-grammar-and-parser-contract.md`

## Current boundary

- `cmd/fairyc` owns command assembly; `internal/` packages own compiler stages and the current target-visible sources: `internal/cruntime` holds the embedded C runtime sources, `internal/cstdlib` holds the target standard-library C sources beneath the closed symbol contract in `internal/cstdlibcontract`, and `internal/sdkstdlib/sources/std` holds the bundled Fairy standard-library sources. `internal/cbackend`, `internal/nativesupport`, and `internal/cdriver` own C emission, native source selection, and native execution. `spec/` owns the normative grammar plus parser, semantic, resource, Region, Result, MIR, MIR-verifier, and resolved-package-graph fixture data, and later versioned executable contracts.
- The planned `runtime/`, `std/`, and `tests/` top-level directories do not exist yet; `docs/monorepo/workspace-boundaries.md` records that layout as a target shape.
- Detailed ownership and import direction live in `docs/monorepo/package-ownership.md` and `docs/monorepo/workspace-boundaries.md`.
- Internal Go packages are not public APIs before a stable language revision. Public contracts are the documented CLI, language revision, explicit C ABI, runtime ABI, standard-library APIs, and emitted schema families.
- Internal MIR and caches use exact compatibility and are discarded rather than migrated. Runtime, standard-library, package, target, and Atlas formats follow `docs/compatibility/versioning-v0.md`.
- `internal/workspaceloader` owns local-exact graph construction and the explicit singular-package
  semantic-test overlay. Stable check/build/run/emit consume only the production graph; stable test
  overlays root `tests/` by default or one root-reachable local package selected by manifest UID
  before verified-MIR interpretation. Experimental
  build-workspace and run-workspace consume the production graph through the closed Windows native
  path without creating a general monorepo task runner.

## Review Blockers

- Cross-package changes lack ownership and dependency impact review.
- Workspace scripts or package boundaries drift from documented contracts.
