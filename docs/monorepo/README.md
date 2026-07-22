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

- `cmd/fairyc` owns command assembly; `internal/` packages own compiler stages; `runtime/` and `stdlib/` own target-visible code; `spec/` owns the normative grammar plus parser, semantic, resource, Region, Result, MIR, MIR-verifier, and resolved-package-graph fixture data, and later versioned executable contracts.
- Detailed ownership and import direction live in `docs/monorepo/package-ownership.md` and `docs/monorepo/workspace-boundaries.md`.
- Internal Go packages are not public APIs before a stable language revision. Public contracts are the documented CLI, language revision, explicit C ABI, runtime ABI, standard-library APIs, and emitted schema families.
- Internal MIR and caches use exact compatibility and are discarded rather than migrated. Runtime, standard-library, package, target, and Atlas formats follow `docs/compatibility/versioning-v0.md`.
- `internal/workspaceloader` owns local-exact graph construction; `experimental build-workspace`
  and `run-workspace` consume that graph through the closed Windows native path without creating a
  general monorepo task runner.

## Review Blockers

- Cross-package changes lack ownership and dependency impact review.
- Workspace scripts or package boundaries drift from documented contracts.
