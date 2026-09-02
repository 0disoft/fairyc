# Self-hosting readiness v0

## Purpose

This document is the executable ownership checklist for the roadmap self-hosting gate. It does not start a Fairy-written compiler and it does not weaken the Go compiler's role as bootstrap and reference semantics implementation.

A Fairy-written compiler-component PR may start only when every `start` row below is `ready` on the same `main` commit. Promotion of that candidate into a required bootstrap path additionally requires every `promotion` row to be `ready`. Evidence must originate from Fairy source where the row claims source or native behavior. C-only probes are supporting evidence, not substitutes for generated Fairy execution.

## Gate rule

The start gate is closed while any `start` row is `blocked`. A row becomes `ready` only when its linked source, fixture, or command is maintained by the repository and the ordinary CI lanes exercise the claimed behavior. The first bounded candidate may provide the evidence for a `promotion` row; that row is not a circular prerequisite for starting the candidate. A future regression re-closes the affected gate.

| Prerequisite | Gate | Status | Current evidence | Blocking work |
| --- | --- | --- | --- | --- |
| Modules and imports | start | ready | Package/module parsing, workspace resolution, public interfaces, locks, and package build/run/test contracts under `docs/package/`, `docs/cli/`, `internal/package*`, and compiler frontend fixtures | none |
| Owned strings and byte text | start | ready | `std::text`, `std::bytes::Buffer`, generated Buffer/String transfer fixtures, runtime contracts under `docs/runtime/runtime-stdlib-v0.md` | none |
| Containers required by a compiler port | start | blocked | Fixed arrays, byte views, Buffer, Option, and Result exist, but the roadmap deliberately has no general move-only typed sequence | #44 after #26 and #27 |
| `Result` and explicit propagation | start | ready | Language Result contract, Result Flow, MIR verification, interpreter and native fixture coverage | none |
| Resource moves and cleanup | start | ready | Resource Flow, move-only resources, explicit abilities/burn boundaries, MIR cleanup verification, interpreter traces, and native resource fixtures | none |
| C FFI bootstrap boundary | start | ready | `unsafe extern C fn`, target-aware scalar/pointer/aggregate ABI work, constrained-C backend and bidirectional native fixture lanes | none for bootstrap use; #47 remains a broader public tagged-aggregate ABI gate |
| Package build graph | start | ready | Stable workspace build/run/test, exact-root `fairy.toml`, lock validation, package interfaces and cache identity | none |
| Source maps and durable source identity | start | ready | Lossless source ownership, source spans, Atlas durable anchors, emitted-source mapping and diagnostic source-location contracts under `docs/atlas/` and compiler fixtures | none; #48 is editor tooling, not a bootstrap prerequisite |
| Diagnostics suitable for differential comparison | start | ready | Stable diagnostic catalog/codes, structured diagnostics, source anchors and compile-fail fixtures under `docs/diagnostics/` | none |
| Test runner suitable for bootstrap comparison | start | ready | Stable package test discovery/execution plus versioned list/report JSON documented by the CLI and exercised by native test fixtures | none |
| Reproducible two-stage bootstrap evidence | promotion | blocked | The Go compiler is the only authoritative stage-0 implementation; no Fairy compiler stage has been admitted | the first bounded candidate must produce matching stage1/stage2 artifacts and differential results |

## First component after the gate opens

The first Fairy-written compiler component is the lossless lexer/token stream producer, not the parser, type checker, MIR verifier, optimizer, backend, package resolver, or runtime.

The lexer is selected because its observable contract is bounded and can be compared byte-for-byte and token-for-token against the Go reference without introducing a second semantic authority. The initial port must consume the same source bytes and language revision and emit a canonical token record containing token kind, exact source range, cooked literal data where applicable, and diagnostics.

The first port must not add syntax, recovery rules, token kinds, preprocessing, host-dependent Unicode behavior, filesystem discovery, package semantics, or compiler plugins.

## Differential acceptance

A Fairy-written component is accepted only when the Go reference and Fairy candidate produce identical canonical results for the checked-in positive, malformed, previous-revision, and mutation corpora. Comparison covers output bytes or canonical records, diagnostics and source ranges, process status, and deterministic artifact hashes where artifacts exist.

Native success alone is insufficient. The candidate must also prove failure behavior against the same corpus. Any mismatch keeps the Go implementation authoritative and blocks promotion of that component.

## Bootstrap stages

`stage0` is the current Go compiler built by the supported Go toolchain.

`stage1` is produced by `stage0` from the admitted Fairy compiler sources.

`stage2` is produced by `stage1` from the exact same Fairy compiler sources and locked inputs.

The eventual bootstrap gate requires `stage1` and `stage2` to match under the repository's declared reproducibility policy for compiler artifacts and to produce the same differential fixture results as `stage0`. Target-specific nondeterministic metadata must be either eliminated or explicitly excluded by an already-versioned artifact contract before this comparison is accepted.

## Authority and rollback

Until all rows are ready and the two-stage evidence lands, the Go compiler remains the sole bootstrap and reference semantics implementation. A Fairy port is an experimental differential implementation and may be deleted without a compatibility promise.

If a prerequisite regresses, self-hosting promotion stops. Existing Go release and bootstrap paths remain the recovery path; no release may require the Fairy-written compiler while this gate is closed.

## Non-goals

This gate does not require replacing Go, implementing the full compiler in Fairy, stage-2 trust claims, trusting-trust mitigation, a second backend, LSP/SCIP, generic containers beyond the compiler's demonstrated needs, or freezing unstable syntax and ABI decisions.
