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
- `0020-stage-0-region-mir-and-interpreter.md`: normal lexical Region placement, inner cleanup, reset trace, and `V1403`.
- `0021-stage-0-region-early-exit-cleanup.md`: scope-indexed Region reset for `return`, Result `?`, and nested exits.
- `0022-controlled-interpreter-io-harness.md`: opt-in in-memory stdout/stderr byte hooks with a shared output limit.
- `0023-explicit-checked-arithmetic-mir.md`: verified overflow and right-shift checks, exact trap edges, and Stage 0 left-shift rejection.
- `0024-stage-0-loop-control-and-cleanup.md`: nearest-loop `break` and `continue`, Flow obligations, and scope-indexed Region cleanup.
- `0025-verified-scalar-c-emission.md`: fail-closed verified-MIR emission for the provisional scalar ISO C11 transport.
- `0026-linux-scalar-c-differential-lane.md`: fail-closed Clang/GCC low/optimized native execution on Linux/amd64 with glibc.
- `0027-experimental-c-emission-cli.md`: deterministic raw C11 inspection from the canonical unstable CLI without file or toolchain side effects.
- `0028-experimental-linux-scalar-build-cli.md`: explicit Linux constrained-C executable builds through the shared C driver and embedded runtime.
- `0029-scalar-option-result-c-transport.md`: private ISO C11 structs and typed MIR flow for `Option<i32>` and `Result<i32, i32>`.
- `0030-executable-x86-64-linux-gnu-target-spec.md`: host-independent target facts, fail-closed maturity evidence, and deterministic experimental inspection.
- `0031-target-layout-engine-and-fixtures.md`: checked scalar, pointer, array, enum, nested-struct, offset, and padding layout execution without host assumptions.
- `0032-resource-definition-preservation-and-empty-drop-gate.md`: ordered resource definitions across AST, HIR, and MIR with fail-closed empty-drop execution.
- `0033-flat-i32-resource-c-transport.md`: target-laid-out private C11 transport for nonempty all-`i32` resources with empty drop bodies.
- `0034-resource-tagged-c-transport.md`: private target-laid-out C11 transport for resource-bearing built-in tags and typed Result propagation.
- `0035-native-drop-trace-differential.md`: closed source-linked drop instrumentation for exact interpreter-to-native C cleanup comparison without product-build observers.
- `0036-native-c-layout-probe.md`: deterministic private C11 probes for the nine valid target-layout cases across Clang/GCC and low/optimized profiles.
- `0037-sanitizer-assisted-c-backend-checks.md`: closed Clang/GCC ASan+UBSan execution of every admitted production C unit without exposing a product build profile.
- `0038-sysv-amd64-classifier-fixtures.md`: host-independent INTEGER/SSE/MEMORY classification, hidden returns, register exhaustion, and whole-aggregate rollback fixtures.
- `0039-bidirectional-cross-compiler-c-abi-probes.md`: separate Clang/GCC caller and callee objects in both directions for 56 low/optimized ABI crossings.
- `0040-provisional-system-heap-c-slice.md`: standard-library-owned fallible SystemHeap C transport and four-profile native probe without source API claims.
- `0041-failure-atomic-overlap-safe-buffer-c-slice.md`: initialized-byte Buffer growth, overlap append rebinding, overflow rollback, and zeroed resize C probe.
- `0042-single-active-region-arena-c-slice.md`: single-active Region generations, failure-atomic byte placement, reset invalidation, and Arena drop C probe.
- `0043-canonical-utf8-text-view-c-slice.md`: allocation-free canonical UTF-8 validation and code-point-boundary text slicing C probe.
- `0044-owned-string-and-cstring-c-slice.md`: failure-atomic Buffer-to-String ownership, immutable text views, and exact CString NUL invariants.
- `0045-linux-file-consuming-close-c-slice.md`: partial Linux I/O, write-all progress, sync, and never-retried consuming close.
- `0046-scalar-source-foreign-c-call.md`: exact-name scalar source FFI with unsafe provenance, fail-closed interpretation, and four-way native crossing.
- `0047-raw-pointer-target-scalar-foreign-call.md`: typed-null raw-pointer plus fixed-width and pointer-width source FFI transport with a separate four-way native crossing.
- `0048-source-callable-utf8-validation-transport.md`: exact allowlisted FY-to-real-stdlib-C empty-input UTF-8 validation crossing without a public wrapper claim.
- `0049-repr-c-output-record-source-transport.md`: flat source `@repr(C)` records, verified direct-local raw addresses, and a real stdlib C output mutation crossing.

Product scope remains owned by `docs/product/02-spec.md`. Validation names are owned by `VALIDATION.md`.

## Review Blockers

- The change invents a product domain without a source.
- The change weakens validation or skips required evidence.
- The change relies on generated, cache, or build output as source truth.
