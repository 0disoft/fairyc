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
| Containers required by a compiler port | start | ready | Windows SDK-owned move-only `std::vec::Vec<T>` in ADRs 0388–0392; token/diagnostic storage and issue #44 reconciliation below; ordinary CI run 34325894239 at `a76b09dea75f7b953048a71c1d0481469d64ba24` | none for the bounded Windows AMD64 MSVC lexer candidate; broader container and target admission remain separate |
| `Result` and explicit propagation | start | ready | Language Result contract, Result Flow, MIR verification, interpreter and native fixture coverage | none |
| Resource moves and cleanup | start | ready | Resource Flow, move-only resources, explicit abilities/burn boundaries, MIR cleanup verification, interpreter traces, and native resource fixtures | none |
| C FFI bootstrap boundary | start | ready | `unsafe extern C fn`, target-aware scalar/pointer/aggregate ABI work, constrained-C backend and bidirectional native fixture lanes | none for bootstrap use; #47 remains a broader public tagged-aggregate ABI gate |
| Package build graph | start | ready | Stable workspace build/run/test, exact-root `fairy.toml`, lock validation, package interfaces and cache identity | none |
| Source maps and durable source identity | start | ready | Lossless source ownership, source spans, Atlas durable anchors, emitted-source mapping and diagnostic source-location contracts under `docs/atlas/` and compiler fixtures | none; #48 is editor tooling, not a bootstrap prerequisite |
| Diagnostics suitable for differential comparison | start | ready | Stable diagnostic catalog/codes, structured diagnostics, source anchors and compile-fail fixtures under `docs/diagnostics/` | none |
| Test runner suitable for bootstrap comparison | start | ready | Stable package test discovery/execution plus versioned list/report JSON documented by the CLI and exercised by native test fixtures | none |
| Reproducible two-stage bootstrap evidence | promotion | blocked | The Go compiler is the only authoritative stage-0 implementation; no Fairy compiler stage has been admitted | the first bounded candidate must produce matching stage1/stage2 artifacts and differential results |

## First component after the gate opens

The public vector release removes the absence of a typed sequence as a blocker; it
does not by itself prove that the compiler port's container requirements are met.
Assess the lossless lexer first, using the shipped `examples/std_vec_quickstart`
and `TestWindowsVectorStableCLI` as integration evidence, not as a compiler-workload
substitute. Mutable element access, iteration APIs, nested vectors and non-Windows
vector execution are not part of ADR 0390. Add only capabilities required by the
demonstrated port workload; do not make every possible container feature a new
self-hosting prerequisite. A workflow's test command or local native success alone
does not establish the required same-commit CI result.

ADR 0392 adds a maintained token-storage prerequisite probe with kind/trivia tags,
source-byte bounds and an owned cooked Buffer inside each vector element. Ordinary
Windows CLI execution and separate injected-failure/allocation-accounting tests
cover that representation. The subsequent record-transport assessment below adds
source reconstruction and diagnostic storage evidence without implementing a lexer
or implementing token recognition. The same-commit hosted evidence below now closes
the container start row for the bounded Windows candidate.

### Current lexer storage assessment

The Go scanner in `internal/lexer/lexer.go` appends completed `Token` and
`diagnostic.Diagnostic` values. It does not update previously appended elements.
`internal/lexer/token.go` reads their ranges to recover source spelling and
reconstruct the original bytes. `internal/lexer/documentation.go` also reads
adjacent tokens without modifying the sequence. This access pattern maps to the
released vector subset as follows:

| Reference requirement | Admitted representation and operation | Maintained evidence |
| --- | --- | --- |
| Immutable original bytes, including malformed UTF-8 | Owned Buffer with shared byte view | Native record roundtrip compares exact reconstructed output bytes |
| Token kind, trivia kind and byte range | Scalar fields in a directly named resource; `usize` for nonnegative byte offsets | Token field output compared with Go-produced records |
| Canonical `Token.Cooked` string | Owned Buffer field, retaining the reference string bytes without decoding or reinterpretation | Cooked length and every byte compared after vector growth |
| Ordered diagnostic code, repair, token text and range | Directly named resource with three owned Buffers and scalar offsets | Diagnostic field output compared after growing beyond the initial vector capacity |
| Append completed values, read by index, then destroy | Fallible push, owner-bound shared `at`, reverse cleanup | Public CLI roundtrip plus ADRs 0388–0392 ownership, failure and cleanup probes |

`TestWindowsVectorLexerRoundTrip` in
`internal/cli/vector_lexer_roundtrip_test.go` obtains records from the Go lexer,
generates Fairy source that owns them in separate token and diagnostic vectors,
then compares exact native output under low and optimized Windows MSVC. Its
bounded cases cover empty input, BOM, CRLF, documentation/block trivia, a Unicode
escaped literal, invalid UTF-8, and unterminated string/comment diagnostics.
The source reconstruction and diagnostic comparison run through ordinary `run`;
the optimized run uses the resulting lockfile through `--locked`.

This is evidence for container representation and access, not a differential
tokenizer: the Fairy program receives precomputed records. It does not establish
recognition, recovery, budget-exhaustion behavior, full corpus coverage, diagnostic
sorting, source discovery or stage1/stage2 reproducibility. Allocation-failure
evidence remains in the independent ownership probes, not the record comparison.
The current scanner's append/read pattern supplies no requirement for mutable
element access, iterator syntax or nested vectors; add those only if an actual
port operation demonstrates the need.

### Same-commit start evidence and issue #44 reconciliation

[Frontend run 34325894239](https://github.com/0disoft/fairyc/actions/runs/34325894239)
completed successfully for `main` commit
`a76b09dea75f7b953048a71c1d0481469d64ba24` on 2026-09-09. Its ordinary Windows
lane runs the complete Go suite without cached results and the Windows native
smoke. The Linux frontend, race checks and downstream C differential, sanitizer,
layout, ABI and product-build job also passed. This is evidence for that exact
commit, not a promise that later revisions cannot regress.

Issue #44's acceptance items map to the released Windows subset as follows:

| Acceptance item | Repository evidence |
| --- | --- |
| Bounded source, HIR, MIR and backend admission | ADRs 0384/0385 and 0388–0390; ordinary stable CLI vector execution in `internal/cli/vector_support_test.go` |
| Move-only layout and allocator ownership | SDK `internal/sdkstdlib/sources/optional/std/vec.fy`; private storage layout and SystemHeap ownership in ADR 0388 |
| Inert and admitted resource elements | `internal/compiler/vector_generic_test.go`, `vector_resource_test.go`, `vector_sdk_test.go` and `nested_resource_test.go` |
| Reserve/grow, push/pop, truncate/clear and borrowed access | SDK operations above; `internal/cli/vector_truncate_test.go`, `vector_support_test.go` and the nested-resource ownership probe |
| Initialized prefix and allocation-failure atomicity | `TestPrivateVectorNestedResourceTokenOwnership` forces initial/growth failures, checks intact returned donors, pop ownership, retained capacity, reverse cleanup and zero live allocations |
| Low and optimized generated-Fairy execution | Native compiler ownership probes and stable CLI token/diagnostic record roundtrips, both exercised by the successful Windows lane |

This reconciliation does not admit arbitrary user generic aggregates, mutable
views, iteration, nested vectors or non-Windows vector execution. The lexer
storage workload needs none of those features. Issue tracking state alone does
not control the gate: these maintained sources and same-commit CI establish its
bounded start evidence. Full tokenizer corpus comparison remains the acceptance
condition for the subsequent Fairy component, not a claim made by these probes.

The first Fairy-written compiler component is the lossless lexer/token stream producer, not the parser, type checker, MIR verifier, optimizer, backend, package resolver, or runtime.

The lexer is selected because its observable contract is bounded and can be compared byte-for-byte and token-for-token against the Go reference without introducing a second semantic authority. The initial port must consume the same source bytes and language revision and emit a canonical token record containing token kind, exact source range, cooked literal data where applicable, and diagnostics.

The first port must not add syntax, recovery rules, token kinds, preprocessing, host-dependent Unicode behavior, filesystem discovery, package semantics, or compiler plugins.

## Differential acceptance

A Fairy-written component is accepted only when the Go reference and Fairy candidate produce identical canonical results for the checked-in positive, malformed, previous-revision, and mutation corpora. Comparison covers output bytes or canonical records, diagnostics and source ranges, process status, and deterministic artifact hashes where artifacts exist.

Native success alone is insufficient. The candidate must also prove failure behavior against the same corpus. Any mismatch keeps the Go implementation authoritative and blocks promotion of that component.

## Bootstrap stages

The Fairy-written candidate now executes authenticated scalar/resource/struct
Vec storage, `std::bytes::Buffer`, String/CString and UTF-8 bodies, including
allocation failures and ownership cleanup. Authenticated `std::io` console
output/input bodies also execute against the original Windows runtime, with
exact effect authority and failure results independently verified. The
[candidate progress](../ops/candidate-module-progress-20261008.md)
records the exact native evidence. Copied byte iteration now also executes
through existing owned HIR/MIR loops, including one-time slice bounds and
resource exits. Fixed byte-array field subranges also preserve original storage and
execute in both candidate profiles. Local mixed-origin payload selections now retain every possible owner
in HIR and independently recomputed MIR. Indexed reachability, finalized-origin
caching and owner-indexed exclusive access checks pass the scanner's prior HIR
work limits without increasing the budget. Development and release candidates
compile original scanner and SDK bodies whose Token/cooked/diagnostic storage,
byte-budget diagnostics and cleanup run in both native C profiles. MIR expression
helpers also remove the development scanner's default-stack overflow. The
checked-in current/previous, malformed, mutation and budget corpora now match the
Go lexer through candidate-emitted C in both native profiles (7670 cases per
profile, plus malformed framing). This proves the scanner component scope.
Authenticated file SDK extern declarations and C record authority now also
pass development/release native API tests, including 14 rejected mutations.
A separate private Result encoding now preserves enum/record and
Option-record/record pairs without changing prior valid IDs or analysis bounds.
Original File SDK open/read/write/sync/close/replace bodies now also execute
with the Windows runtime in both candidate and native profiles, including
UTF-8/NUL data, buffer preservation and failure/effect cases. Project integration
remains a separate check. The actual bounded source/manifest loader is now
a standalone Fairy leaf; both candidate/native profiles prove its byte limits,
early errors and handle/allocation cleanup, and the prior CLI import cycle is
removed.
An opt-in SDK compaction pipeline now also compiles the authored file-token path
with the original SDK source owners. Exact source/token spans and all user
bodies survive compaction; existing join and work bounds are unchanged. Both
candidate profiles emit identical C (769317 bytes for the observed program),
and both native profiles reproduce the prior CLI's four file cases, including
their exact streams and asserted statuses. Unused invalid user code, private SDK
access and missing filesystem effects remain rejected. Byte-loop induction
spans no longer incorrectly extend unrelated call loans; real later-argument
conflicts still reject. Evidence:
`docs/ops/evidence/candidate-file-tokens-fixed-dev-20261009.json`,
`docs/ops/evidence/candidate-file-tokens-fixed-release-20261009.json` and
`docs/ops/evidence/candidate-byte-loop-calls-release-20261009.json`.
The complete command dispatcher and project-driver integration are still open.
Full-dispatcher self-source admission has now been observed directly: MIR
lowering was split into six parser-admitted units while preserving all original
function bodies and byte-identical generated C for the focused regression.
The current 142-file input still rejects at parsing; HIR model, type checking
and HIR lowering source splits remain. These observations retain the existing
budgets and do not establish whole-compiler native execution or promotion.
The owned HIR model has subsequently been split into a factory/query owner and
eight proof helpers. Its nine files now parse with the same guard bodies and
publication order. Existing thirteen drop-link mutations still reject; focused
generated C remains identical and passes twelve native executions. Type checking
and HIR lowering remain the two measured self-source parser blockers. The full
150-file input remains below 1MiB (1047406 joined bytes) and still rejects.
Project-driver and platform work remain. These scanner results do not establish
the complete compiler's stage1/stage2 or promotion gates.

`stage0` is the current Go compiler built by the supported Go toolchain.

`stage1` is produced by `stage0` from the admitted Fairy compiler sources.

`stage2` is produced by `stage1` from the exact same Fairy compiler sources and locked inputs.

The eventual bootstrap gate requires `stage1` and `stage2` to match under the repository's declared reproducibility policy for compiler artifacts and to produce the same differential fixture results as `stage0`. Target-specific nondeterministic metadata must be either eliminated or explicitly excluded by an already-versioned artifact contract before this comparison is accepted.

## Authority and rollback

Until all rows are ready and the two-stage evidence lands, the Go compiler remains the sole bootstrap and reference semantics implementation. A Fairy port is an experimental differential implementation and may be deleted without a compatibility promise.

If a prerequisite regresses, self-hosting promotion stops. Existing Go release and bootstrap paths remain the recovery path; no release may require the Fairy-written compiler while this gate is closed.

## Non-goals

This gate does not require replacing Go, implementing the full compiler in Fairy, stage-2 trust claims, trusting-trust mitigation, a second backend, LSP/SCIP, generic containers beyond the compiler's demonstrated needs, or freezing unstable syntax and ABI decisions.
