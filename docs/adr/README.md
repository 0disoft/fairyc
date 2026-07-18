# Architecture Decisions

- Status: Draft
- Owner: UNASSIGNED

## Purpose

This document captures the durable design contract for Architecture Decisions.
It is intentionally a scaffold and should be filled with project-specific decisions as they become known.

## Current decisions

- `0001-initial-architecture-boundaries.md`: Go Stage 0 compiler, explicit MIR, C reference backend, and deferred native backends.
- `0002-contract-source-of-truth.md`: authority of product, specification, MIR, target, CLI, and generated surfaces.
- `0003-stage-0-safety-boundary.md`: deliberately restricted ownership, borrowing, regions, traps, and FFI.
- `0004-atlas-semantic-snapshot-boundary.md`: one-way compiler-fact export and trust separation.
- `0005-stage-0-surface-and-module-boundary.md`: minimal syntax, one-file modules, alias imports, and restricted enums.
- `0006-c-abi-v0-and-first-conformance-target.md`: no native Fairy ABI and `x86_64-linux-gnu` as the first planned conformance target.
- `0007-runtime-stdlib-and-allocator-boundary.md`: compiler, thin-runtime, standard-library, and allocation ownership.
- `0008-versioned-contract-boundaries.md`: independent compatibility vector and fail-closed mismatch behavior.
- `0009-stage-0-grammar-and-parser-contract.md`: `.fy`, literal spelling, precedence, module identity, parser architecture, and bounded recovery.
- `0010-stage-0-semantic-fixture-contract.md`: phase-owned semantic fixtures, canonical assertions, byte spans, and diagnostic cascade policy.
- `0011-stage-0-resource-flow-fixture-contract.md`: whole-root resource states, cleanup obligations, moves, joins, and loop mismatch fixtures.
- `0012-stage-0-region-and-result-fixture-contracts.md`: Region origins and restrictions plus Result handling and propagation fixture boundaries.
- `0013-stage-0-mir-fixture-contract.md`: source-linked verified-MIR observations without freezing internal graph identity.
- `0014-resolved-package-graph-fixture-contract.md`: package-instance equality, direct dependency visibility, and explicit standard-library injection without public manifest syntax.
- `0015-mir-verifier-mutation-fixture-contract.md`: verified named seeds and isolated mutation recipes covering every current `Vxxxx` invariant.
- `0016-stage-0-interpreter-fixture-contract.md`: verified-MIR scalar returns and canonical traps as the first executable semantics oracle.
- `0017-stage-0-result-constructors-and-match.md`: qualified compiler-known Result construction and exhaustive shallow Result matching.
- `0018-stage-0-option-constructors-and-match.md`: qualified compiler-known Option construction and exhaustive shallow Option matching.
- `0019-stage-0-result-must-use-dataflow.md`: local, parameter, branch, loop, overwrite, and early-exit Result obligations.

Product scope remains owned by `docs/product/02-spec.md`. Validation names are owned by `VALIDATION.md`.

## Review Blockers

- The change invents a product domain without a source.
- The change weakens validation or skips required evidence.
- The change relies on generated, cache, or build output as source truth.
