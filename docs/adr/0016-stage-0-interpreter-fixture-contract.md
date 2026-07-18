# ADR 0016: Stage 0 interpreter fixture contract

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-18
- Product source: `docs/product/02-spec.md`
- MIR source: `docs/mir/mir-v0.md`
- Fixture source: `spec/fixtures/interpreter/manifest-v0.json`

## Context

The MIR interpreter must become the first executable semantics oracle without quietly inventing behavior that the verified MIR does not contain. A broad runtime harness would prematurely mix scalar execution, tagged payloads, resource drops, Region cleanup, I/O, process arguments, and host behavior. A direct HIR evaluator would be cheaper, but it would create a second semantic path beside MIR and let backend-relevant lowering defects escape the oracle.

## Decision

Version the first interpreter fixture surface as `fairyc.interpreter-fixtures/v0`. Every case starts from Fairy C source and must complete parser, semantic, Flow, MIR lowering, and MIR verification before interpretation. The first closed slice accepts one explicitly named zero-argument module `main() -> i32` entry and observes either an exact `i32` return or one canonical trap category. The experimental CLI selects the input file's declared module and its `main`; this does not settle future package entry selection.

The initial cases cover literal return, scalar function calls with left-to-right arguments, checked addition, boolean `if` branches, source short-circuiting, scalar `while`, division-by-zero trapping, and the bounded interpreter step limit. Later v0 cases extend the same manifest with scalar and resource-bearing `Result` and `Option` construction, discriminant branches, copy or move payload extraction, Result propagation, resource construction and field reads, whole-root call moves, RHS-before-drop reassignment, consuming-close success/failure ownership, active-variant cleanup, and deterministic source-linked drop traces. The interpreter consumes the same mutable-local MIR handed to later backends. It does not evaluate HIR or reconstruct source operators after MIR lowering.

## Consequences

- A fixture pass proves the complete source-to-interpreter path for the listed scalar, built-in tagged, and whole-resource subset.
- The CLI may expose the same path under an explicitly unstable `experimental run` command without defining stable `fairyc run` semantics.
- User drop-body effects, Region reset, controlled I/O, process output, arguments, and host integration remain explicit deferrals.
- A new interpreter behavior requires synchronized MIR representation, verifier coverage, fixture schema or cases, CLI behavior when exposed, and documentation.

## Rejected alternatives

- Interpreting HIR directly: duplicates executable semantics and can disagree with backends.
- Treating Go tests without a manifest as the contract: hides fixture identity and deferrals in implementation code.
- Starting with host I/O or OS emulation: expands the oracle before scalar control and traps are proven.
