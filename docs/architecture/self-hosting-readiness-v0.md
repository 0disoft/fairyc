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
Type checking now also parses as eight responsibility-based files. Extracted
bodies and source-depth accounting are retained; focused C is byte-identical,
twelve native runs pass and eight type-error diagnostics match the prior driver.
The full input is now 157 files and 1054271 joined bytes, still rejected at
parsing. HIR lowering partitioning and source deduplication below the unchanged
1MiB budget remain prerequisites. Evidence:
`docs/ops/evidence/candidate-type-source-split-20261009.json` and
`docs/ops/evidence/candidate-dispatcher-admission-after-type-split-20261009.json`.
HIR lowering has subsequently been split into four parser-admitted files with
all five function bodies retained. The full dispatcher now passes parsing and
reaches phase 1 code 10 at the cumulative source cap: 160 files and 1055400
joined bytes. Source deduplication below 1MiB remains necessary before later
admission stages can be evaluated. Focused C is identical and twelve native
runs pass. Evidence:
`docs/ops/evidence/candidate-hir-lower-source-split-20261009.json` and
`docs/ops/evidence/candidate-dispatcher-admission-after-hir-lower-split-20261009.json`.
Shared CLI stages/output and typed HIR default factories subsequently reduced
the full input to 1048516 joined bytes across 160 files, below the same 1MiB cap.
The actual dispatcher now passes parsing and cumulative source admission but
rejects at module-program phase 1 code 9 on `mir_output::row`'s by-value
`usize[20]` signature. Source-set fixed-array type registration is still open.
Forty-six CLI cases retain exact streams/status, five generated C outputs match
the prior driver and twenty focused native runs pass. Only 60 source bytes of
headroom remain; this does not establish full-dispatcher compilation or
stage1/stage2. Evidence:
`docs/ops/evidence/candidate-cli-source-dedup-20261009.json` and
`docs/ops/evidence/candidate-dispatcher-admission-after-source-dedup-20261009.json`.
Source-set fixed-array signatures now retain canonical element identity and
bounded rank-one/rank-two lengths, including imported nominal/enum and Option
arrays. Three API tests pass in each native profile; five source-set programs
produce identical C across candidate profiles and pass twenty native runs.
The actual dispatcher now passes usize[20] signature registration and reaches
SDK compaction phase 8 code 11 at its reference-index cap. Current input is
160 files and 1046965 joined bytes, within 1MiB after moving AST-join notes out
of compiler source. Reference-index/work reduction and accurate limit/error
classification remain necessary. Evidence:
`docs/ops/evidence/candidate-module-array-signatures-parity-20261009.json` and
`docs/ops/evidence/candidate-dispatcher-admission-after-array-signatures-20261009.json`.
The reference cap now reports limit code 10 separately from allocation code
11. The development driver reproduces phase 8 code 10 at the same span on the
160-file, 1047016-byte input, confirming the index-cap blocker. Two existing
native SDK compaction regressions pass. Index reduction still remains before
full-dispatcher compilation can advance. Evidence:
`docs/ops/evidence/candidate-reachability-limit-classification-20261009.json`.
Per-name reachability storage has subsequently been replaced by checked
declaration expression ranges. A 17600-reference, 22-module native case retains
all user AST/source facts; SDK diagnostics and the four real file-token cases
remain exact with identical generated C. The full 160-file input (1046901
joined bytes) passes the previous reference-index blocker and now reaches the
phase-8 lookup-work cap in `numbers.fy`. Catalog lookup reduction remains open.
Evidence: `docs/ops/evidence/candidate-reachability-ranges-20261009.json` and
`docs/ops/evidence/candidate-dispatcher-admission-reach-ranges-20261009.json`.
Module-local catalog ranges and actual probe accounting now pass name
reachability on the 160-file, 1048411-byte dispatcher. Two native catalog and
three SDK compaction regressions pass in development. The next phase-8 work
blocker occurs during AST copying at `std/vec.fy:0..0`; full-dispatcher C/native
execution and stage1/stage2 comparison remain open. Evidence:
`docs/ops/evidence/candidate-catalog-ranges-20261009.json` and
`docs/ops/evidence/candidate-dispatcher-admission-catalog-ranges-20261009.json`.
Token-qualified local-name detection and cached SDK function ownership further
advance the 1048534-byte dispatcher to AST copying at `bodies.fy:0..0`.
Three development native regressions pass, including 17600 long local names
under 50000 reachability work units. Moving unchanged user AST owners is next.
Evidence: `docs/ops/evidence/candidate-ast-owner-cache-20261009.json` and
`docs/ops/evidence/candidate-dispatcher-admission-owner-cache-20261009.json`.
The opt-in CLI now moves unchanged user AST/scanner owners after all SDK
replacement copies and transfer capacity are ready. Three development native
tests retain exact observations and validation, including a 22-module move
under 50000 work units. The 160-file dispatcher (1048460 joined bytes) passes
phase 8 and reaches phase 2 code 10 at the joined scanner's storage rejection.
Cooked-storage reduction is next; joined AST/HIR/MIR and stage1/stage2 remain
open. Evidence: `docs/ops/evidence/candidate-ast-owner-transfer-20261009.json`
and `docs/ops/evidence/candidate-dispatcher-admission-in-place-20261009.json`.

The joined scanner now shares equal canonical cooked buffers, preserving token
kinds, spans, diagnostics and snapshot indices. Its original 65536-byte pool
and 16777216 work bounds remain fixed. Mutable scalar-array field views through
record parameters retain the original owner. Eight development native API
tests and fourteen array-view low/optimized native runs pass; file-token
stdout/stderr/status remain identical to the reference in two native profiles.
The 160-file dispatcher is now 1045810 joined bytes. Its 69949 logical cooked
bytes occupy 41226 owned bytes, with zero scanner diagnostics. The next observed
failure is phase 2 code 10 at module 18 (`module_cli.fy`): compacted cumulative
AST entries reach 1049 against the unchanged 1024-entry bound. Partitioning or
reducing those arenas is next; complete joined AST/HIR/MIR, full-dispatcher C,
and stage1/stage2 are still open. Evidence:
`docs/ops/evidence/candidate-cooked-storage-20261009.json`,
`docs/ops/evidence/candidate-dispatcher-admission-cooked-intern-20261009.json`,
`docs/ops/evidence/candidate-file-tokens-cooked-intern-20261009.json` and
`docs/ops/evidence/candidate-mutable-record-array-view-native-20261009.json`.

The AST partition layer now preserves the actual 160-module dispatcher in
twelve independently bounded joined owners. The 1046245-byte full input still
fits 1MiB; all 152020 expressions and 11509 entries are retained across groups.
Eight development native API regressions pass. Each group remains within the
original joined AST bounds, and range-join errors retain full-source module
coordinates. This does not yet connect groups to body binding, HIR/MIR, C output
or final linking. The current command dispatcher still uses one joined owner.
Cross-unit declaration identity is the next compiler connection; stage1/stage2
and promotion remain open. Architecture:
`docs/architecture/candidate-compilation-units-v0.md`; evidence:
`docs/ops/evidence/candidate-module-units-20261009.json`.

Cross-unit declaration binding now retains original module/declaration/span
coordinates in a separate owned target table, with alias sharing and private
lookup enforcement. Stale provider bytes outside the selected unit are rejected.
Five development native regressions pass, and the actual 160-module dispatcher
(1046447 joined bytes) binds all twelve ranges while preserving 152032 expressions
and 11513 entries. Measured work peaks at 134589 against the existing 1048576
bound. Original parser/signature owners are not retained by the Bound result.
Body name/type checking must still import provider layouts and signatures using
these coordinates; HIR/MIR, C and CLI integration remain open. Evidence:
`docs/ops/evidence/candidate-module-unit-bind-20261009.json`.

Owned provider interfaces now retain canonical field shapes and callable
signatures with their declaration flags, foreign link names, resource abilities
and declared cleanup authority. Generic function interfaces remain deferred
templates. Field queries enforce resource privacy, and fresh catalog lookups
revalidate retained bindings. Six development native regressions pass, including
queries after source/metadata removal and stale/private provider rejection.
All 160 modules (1046575 bytes) import interfaces for thirteen groups; maxima
are 131 shapes, 910 fields, 198 callables and 401 type nodes within existing
bounds. Actual unit body typing must still consume these interfaces and resolve
local annotations/templates before HIR/MIR/C/CLI integration. Evidence:
`docs/ops/evidence/candidate-module-unit-interfaces-20261009.json`.

Unit interfaces now own canonical annotation roots for local entries,
declaration results, fields, casts, explicit type arguments and byte-loop
binders. Resolution uses original module coordinates; generic declaration roots
remain explicitly deferred. Eight development native regressions and all
thirteen dispatcher groups pass (10289 concrete roots, 69 deferred, maximum 409
graph nodes). Actual expression/body typing and template specialization must
consume this mapping next; remaining CLI/HIR/MIR/C integration and stage proofs
stay open. Evidence: `docs/ops/evidence/candidate-module-unit-annotations-20261009.json`.

Owned explicit generic signature instances now preserve provider coordinates,
ordered canonical arguments, declaration flags and per-expression request
identities. Thirteen native regressions pass, including SDK resource admission
and failure rollback. Initial AST-only groups exposed specialization pressure at module 37
(E4503); the semantic refinement below resolves that pressure. Body instantiation
and all unit compiler consumers remain open. Evidence:
`docs/ops/evidence/candidate-module-unit-generics-20261009.json`.

Semantic range admission now resolves specialization pressure by splitting only
code10/E4503 ranges. Two new native planner regressions pass, and the thirteen
prior kernel regressions retain matching source hashes. All 160 dispatcher
modules (1048186 bytes) now admit fifteen ranges after two splits; at most 61
instances occur in one range under the existing 64 cap. Per-range instance
counts sum to 476 for 1281 explicit requests; fourteen generic-body requests
remain deferred. This completes current-input range/interface signature
planning. Owned template body cloning, nested requests, structural cross-unit
instance deduplication, expression/body typing, all IR/backend/CLI consumers and
stage proofs remain open. Evidence: `docs/ops/evidence/candidate-module-unit-plan-20261009.json`.

Transitive generic function request graphs now resolve forwarded arguments and
private provider targets, reject specialization recursion/depth exhaustion, and
own SDK intrinsic and Vec aggregate records at original expression coordinates.
Ten native regressions and all eighteen dispatcher units pass (160 modules,
1048277 bytes, maximum 57 function instances per unit). Per-unit counts sum to
592 function instances, with 116 nested function edges, 594 intrinsic records
and 100 aggregate records. Provider function and SDK Vec cleanup AST ownership,
resource cleanup specialization, expression/body typing and all IR/backend/CLI
consumers remain open. Evidence:
`docs/ops/evidence/candidate-module-unit-generic-dependencies-20261009.json`.

Requested generic function AST ownership now passes development native tests
(7/7) and the full 160-module source corpus in 25 bounded units. Provider source
streams, body-local node links, canonical annotation overlays, signatures and
per-instance nested request edges are retained after original owners are
released. The original plus cloned AST sizes remain inside existing admission
bounds. SDK Vec cleanup AST ownership,
resource cleanup contexts and implicit drop dependencies, local expression/body
typing, structural cross-unit links, layouts/effects, HIR/MIR/C/CLI and actual
stage1/stage2 remain open. Evidence:
`docs/ops/evidence/candidate-module-unit-bodies-20261009.json`.

Required resource and SDK Vec cleanup AST contexts now pass development native
tests (7/7) and the 160-module, 1048472-byte corpus as 26 bounded units. Declared
effects/abilities and compiler-owned dropping receiver facts are retained;
cleanup annotations and explicit generic request edges are owned separately.
The authenticated SDK take-slot leaf is folded only in cleanup contexts with
checked signature/body structure and retained unsafe provenance to preserve the
16-per-template quota. Implicit/nested drop dependencies, transitive cleanup
effects, body symbol/expression typing and inline provenance consumption,
cross-unit links/layouts, HIR/MIR/C/CLI and actual stage1/stage2 remain open.
Evidence: `docs/ops/evidence/candidate-module-unit-cleanups-20261009.json`.

Type-level nested drop requirements now use an immutable graph of owned
interfaces. Contained field/Option/Result/array/Vec effects and no-drop abilities
propagate to a fixed point; references and pointers have no drop edges.
Unprojected nominal metadata is explicitly incomplete and cannot be queried as
a proven drop requirement. Unit admission includes this bounded graph without
raising type, dependency or work limits. Cleanup-body locals, inferred types,
move-aware obligations and new generic-body local field projection still need
body typing/ownership consumers. Cross-unit links/layouts, HIR/MIR/C/CLI and
actual stage1/stage2 remain open. Evidence:
`docs/ops/evidence/candidate-module-unit-drops-20261009.json`.
Development check/probe check/build and seven native tests pass; the current
160-module, 1048472-byte corpus retains 26 bounded units, with drop work at most
6036 within the unchanged 1048576 bound. This does not establish executable
drop lowering or whole dispatcher native compilation.

Requested generic functions and cleanup contexts now collect explicit annotation
and named constructor type demands before body cloning. Newly discovered
private types request provider field shapes and cleanup contexts; cleanup
helpers feed back into the same bounded closure. Original context/type root/
expression coordinates and diagnostic spans are retained. Development check
and eight native regressions pass. Ordinary-call result inference, body symbol/
expression typing, move-aware obligations and inline provenance consumption
remain open, along with cross-unit links/layouts, HIR/MIR/C/CLI and stage1/stage2.
Evidence: `docs/ops/evidence/candidate-module-unit-body-types-20261009.json`.

Owned lexical binding for concrete bodies (2026-10-09):

`module_unit_names`, `module_unit_name_model` and `module_unit_name_walk` resolve
requested generic function and resource/SDK Vec cleanup ASTs. Current source
stamps, canonical graphs, ordered arguments and provider streams are checked
before publishing any result. Parameters, let/var declarations, payload patterns,
byte iteration, regions and the dropping receiver have separate symbol kinds.
Explicit annotation types remain canonical; inferred locals and payload types
remain unresolved. Entry/pattern/use tables and provider module/declaration
coordinates retain original expression IDs and spans.

Cleanup `this` is kind 10 with a resource type and no ordinary by-value parameter.
References record lexical unsafe function/block context; SDK storage names are
classified only in authenticated std::vec source. The safe `take_slot<T>` wrapper
has an unsafe intrinsic in its body: its source call can have `unsafe_context`
false while the separate request edge retains `inline_unsafe` provenance.
Binding records these facts without granting unsafe authority or performing
expression/flow checks. Ordinary unit bodies, expression typing, implicit drop
ownership/effect checks, inline provenance consumption, structural links,
layouts, HIR/MIR/C/CLI and full dispatcher native/stage1/stage2 remain pending.
Public language/SDK ABI, SDK sources, DB, CI and runners are unchanged. Evidence:
`docs/ops/evidence/candidate-module-unit-names-20261009.json`.

| Observation | Value |
| --- | --- |
| Development native tests | 7/7 passed |
| Dispatcher source corpus | 160 modules; 1048472 bytes |
| Admitted units | 26; 39 attempts; 13 splits |
| Per-unit function / cleanup totals | 625 / 763 |
| Per-unit symbol / reference totals | 3963 / 10161 |
| Maximum symbols / references | 260 / 648 |
| Maximum resolved-unit name work | 10639 / 1048576 |
| Maximum combined entries | 1015 / 1024 |

Lexical binding for ordinary unit bodies (2026-10-09):

module_unit_ordinary_names binds every non-generic function body within an
admitted original module range, reusing module_unit_body_ast and
module_unit_name_walk with borrowed original scanner streams. The temporary
body AST is discarded, while immutable resolved name tables retain the
original module and declaration, expression, and entry IDs and spans. Explicit
canonical annotation roots are read from the exact same bounded interface
range, and interface range mismatches and stale source stamps are rejected
before publication. Generic templates stay deferred, and requested generic and
cleanup binding stays in module_unit_names. Unit admission accounts for
combined ordinary, generic, and cleanup symbols, references, and name work
under the existing 16384, 16384, and 1048576 bounds, with the existing 128
active locals and depth 64 unchanged. Expression type checking, ordinary-call
result inference, move-aware drop obligations, unsafe and inline provenance
consumption, cross-unit structural dedup and layout, HIR, MIR, C, and CLI, and
the full dispatcher native, stage1, and stage2 remain unfinished. The public
language and SDK ABI, SDK sources, DB, CI, and runners are unchanged.

Evidence: `docs/ops/evidence/candidate-module-unit-ordinary-names-20261009.json`.


| Observation | Value |
| --- | --- |
| Development native tests | 7/7 passed |
| Dispatcher source corpus / units | 160 modules; 1048562 bytes; 26 units |
| Ordinary functions / symbols / references | 1175 / 7479 / 48136 |
| Maximum combined symbols / references | 723 / 4094 (each <= 16384) |
| Maximum combined name work | 751853 / 1048576 |
| Maximum combined entries / type nodes | 1015 / 1024; 2242 / 4096 |
| Observation execution timeout | Initial 300s run timed out after 23 units; final 600s selection passed all 26 |

Declared call results and interface closure (2026-10-09):

The fairy compiler's module_unit_calls records declared-call canonical results
and parameters, along with provider and original expression coordinates. It
also records effects, unsafe, and foreign flags after lexical binding.
module_unit_call_closure repeatedly seeds unseen ordinary callable signatures
into the existing bounded field and cleanup closure. Cleanup helper calls can
request more result types. Local name shadows and stale input are rejected.
Generic calls, closed builtins, and variant constructors stay separate.
Existing arena bounds remain unchanged, with at most 64 rounds and 1048576
units of call-projection work. Full expression and argument type matching,
inferred local types, ownership and effects, unit HIR/MIR/C/CLI, and self-
hosting all remain pending.

Public language/SDK ABI, SDK sources, DB, CI and runners are unchanged.
Evidence: `docs/ops/evidence/candidate-module-unit-call-results-20261009.json`.


| Observation | Value |
| --- | --- |
| Development native tests | 8/8 passed |
| Dispatcher source corpus / units | 160 modules; 1048562 bytes; 26 units |
| Per-unit declared / specialized call-site totals | 11189 / 2622 |
| Per-unit variant / closed-builtin totals | 610 / 4 |
| Maximum closure rounds / work | 2 / 64; 305763 / 1048576 |
| Maximum combined entries / type nodes | 1015 / 1024; 2242 / 4096 |
| Native closure-chain fixture | New call-result fields and cleanup-returned resources close after at least 3 rounds |
| Array reference regression | Kind-24 element links use right; a declared call beneath a cast retains its callee binding |
| Remaining expression consumers | Argument/operator matching, inferred-local symbol types, ownership and unsafe/inline checks |
| Deferred checks | Release, Linux, full dispatcher native and stage1/stage2; no push |

`stage0` is the current Go compiler built by the supported Go toolchain.

`stage1` is produced by `stage0` from the admitted Fairy compiler sources.

`stage2` is produced by `stage1` from the exact same Fairy compiler sources and locked inputs.

The eventual bootstrap gate requires `stage1` and `stage2` to match under the repository's declared reproducibility policy for compiler artifacts and to produce the same differential fixture results as `stage0`. Target-specific nondeterministic metadata must be either eliminated or explicitly excluded by an already-versioned artifact contract before this comparison is accepted.

## Authority and rollback

Until all rows are ready and the two-stage evidence lands, the Go compiler remains the sole bootstrap and reference semantics implementation. A Fairy port is an experimental differential implementation and may be deleted without a compatibility promise.

If a prerequisite regresses, self-hosting promotion stops. Existing Go release and bootstrap paths remain the recovery path; no release may require the Fairy-written compiler while this gate is closed.

## Non-goals

This gate does not require replacing Go, implementing the full compiler in Fairy, stage-2 trust claims, trusting-trust mitigation, a second backend, LSP/SCIP, generic containers beyond the compiler's demonstrated needs, or freezing unstable syntax and ABI decisions.

## Compilation-unit known value analysis (2026-10-09)

`module_unit_values` collects generic, cleanup and ordinary contexts. Its
immutable value, local and return tables retain provider module/declaration,
original expression/entry IDs and source spans. Declared and explicitly
specialized function results seed known values. Direct literals take local or
return context through parentheses; integer range errors are retained.
Unannotated locals infer from known initializer types, and local references and
parentheses propagate those types. Only resolved returns are matched.
Operators, casts, fields, arguments, variants, joins and builtin result typing
remain deferred, along with ownership, effects and unsafe/inline validation.
Existing canonical IDs are preserved; missing primitive/literal descriptors
append to a separate bounded graph. Stale source stamps are rejected before
publication. Existing arenas, 64 fixed-point rounds and 1048576 work remain
bounded, with code-10 admission using the existing range-split planner.

| Observation | Value |
| --- | --- |
| Development native tests | 8/8 passed |
| Dispatcher input snapshot | 160 modules; 1048562 bytes |
| Admitted units / attempts / splits | 26 / 39 / 13 |
| Per-unit context / expression totals | 2563 / 186137 |
| Known / unresolved / non-value expression totals | 58300 / 68976 / 58861 |
| Local / inferred local totals | 11442 / 539 |
| Return / checked return totals | 4783 / 2876 |
| Maximum value work / graph | 754886 / 1048576; 427 / 4096 |
| Maximum combined entries / type syntax nodes | 1015 / 1024; 2242 / 4096 |
| Preserved local / iteration bounds | 128 active locals; 64 fixed-point rounds |


The same dispatcher input snapshot is reused for comparison. The new value
modules are tested as implementation code, but are not added to the 1MiB input
snapshot. This observation is not full current-source dispatcher compilation.
Argument/operator consumers and move-aware cleanup, unsafe/inline provenance,
structural cross-unit links/layouts and unit HIR/MIR/C/CLI remain unfinished.
Public language/SDK ABI, SDK sources, DB, CI and runners are unchanged.
Release, Linux, full dispatcher native and stage1/stage2 remain deferred; no push.
Evidence: `docs/ops/evidence/candidate-module-unit-values-20261009.json`.

## Compilation-unit scalar operator typing (2026-10-09)

`module_unit_value_scalar` types builtin integer arithmetic, bitwise/shift,
comparisons, boolean logic/negation and signed unary operators within local
initializer and return roots. Pure integer literal trees remain flexible; a
typed peer anchors their scalar type while original source links retain
execution order. Direct signed i32/i64 minimum operands retain a unary minimum
marker. Unknown field/cast/builtin anchors keep literal peers unresolved, avoiding
premature i32 range errors. A primitive descriptor cache per value plan replaces
repeated graph scans. Existing canonical IDs, arenas, 64 rounds and 1MiB work
remain bounded across ordinary, requested generic and cleanup contexts.

| Observation | Result |
| --- | --- |
| Development check | 185 source files; 2298 MIR functions; passed |
| Selected Fairy native tests | 8/8 passed |
| Reference fixture acceptance / selected primary code and span | 8/8 matched; additional cascade output retained |
| Unsupported reference case | Experimental exit 2 and span, distinct from an E diagnostic |
| Integer range correction | E1013, replacing the earlier internal 1008 |

Call arguments and other expression/control/place/ownership consumers, inline
and unsafe provenance, unit HIR/MIR/C/CLI and self-hosting remain unfinished.
Full dispatcher input observations are grouped with the next argument unit;
this unit makes no new full-corpus claim. Release, Linux, full dispatcher native,
stage1/stage2 and push were not run. Public language/SDK ABI, SDK sources, DB,
CI and runners are unchanged.
Evidence: `docs/ops/evidence/candidate-module-unit-operators-20261009.json`.

## Compilation-unit argument matching (2026-10-09)

Fairy C's module_unit_argument_collect/types now adds immutable invocation and argument records across generic, cleanup, and ordinary contexts, preserving the provider module/declaration, specialization instance, ordered original expression IDs, and spans. Declared and explicit generic calls validate arity (E1010) and known argument types (E1006), along with contextual scalar literals/operators and range (E1013); positional labels are only allowed for same-module private non-generic ordinary functions per ADR0396, requiring the exact name at the same position or else E1014. Converting ref mut T to ref T records a shared reborrow, while ownership/place validation remains pending, and unknown field, cast, borrow, builtin, and variant types stay unresolved. Existing arena, work, depth, and round limits are unchanged, and code10 splits only oversized units.

| Observation | Result |
| --- | --- |
| Development check | 187 source files; 2338 MIR functions; passed |
| Selected development native tests | 9/9 passed |
| Reference acceptance/code/span | 10 matching cases; one recorded callable-value difference |
| Fixed dispatcher input snapshot | 160 modules; 1048562 bytes |
| Units / attempts / splits | 26 / 39 / 13 |
| Invocation / checked invocation totals | 12590 / 6167 |
| Argument / checked argument totals | 29822 / 20527 |
| Shared reborrow argument records | 402 |
| Maximum calls / arguments | 919 / 2687; each bounded to 16384 |
| Maximum value work / graph | 737134 / 1048576; 427 / 4096 |


The observation reuses the fixed 160-module, 1MiB input snapshot; new analysis
modules are compiled as implementation code. It does not establish full
current-source dispatcher compilation. Arguments whose field/cast/borrow/variant
or builtin types are unresolved stay pending, even when the call result is known.
The reference compiler accepts the callable-name argument fixture; the candidate
retains its existing non-value callable classification and reports E1006. That
case is recorded as a difference, not counted among the matching fixtures.
Move-aware cleanup, ownership/place and unsafe/inline validation, structural
cross-unit deduplication/layouts and unit HIR/MIR/C/CLI remain pending.
Release, Linux, full dispatcher native, stage1/stage2 and push were not run.
Public language/SDK ABI, SDK sources, DB, CI and runners are unchanged.
Evidence: `docs/ops/evidence/candidate-module-unit-arguments-20261009.json`.

## Compilation-unit scalar casts (2026-10-09)

module_unit_value_model.Expression now keeps the canonical cast target separate from the contextual expected/result type. module_unit_value_collect seeds these targets from owned generic/cleanup annotations or the original ordinary source type offsets, while retaining provider original expression/type IDs and spans and leaving stale input checks unchanged. module_unit_value_scalar applies the existing scalar_casts::allowed rules, including widening to i64 and the exact i32 / c::int bridge. Operands are checked without target context, so oversized unsuffixed literals still fail E1013. Unsupported cast pairs keep the experimental code2 and the full cast span rather than an invented E1052. Known casts propagate to inferred locals, scalar peer operators, call arguments, and returns. Unknown operands and raw-pointer conversions remain pending until the field/place/unsafe work. The existing graph/arena/depth64/round64/work1MiB bounds are unchanged. Full dispatcher observation is deferred to the next field/place unit; this does not claim a new corpus pass, stage1/stage2, or Release/Linux/push.

| Observation | Result |
| --- | --- |
| Development check | 187 source files; 2340 MIR functions; passed |
| Selected development Fairy native tests | 8/8 passed |
| Reference acceptance/code/span | 7/7 matched |
| Unsupported conversion | Experimental code 2 and full cast span; no invented E1052 |
| Retained bounds | 4096 types; 16384 expressions; depth/rounds 64; work 1048576 |


Full dispatcher observation is bundled with the next field/place unit; this unit
claims no new corpus pass. Field/place, raw-pointer casts and unsafe provenance,
move-aware cleanup, structural cross-unit links/layouts and HIR/MIR/C/CLI remain
pending. Release, Linux, full dispatcher native, stage1/stage2 and push were not
run. Public language/SDK ABI, SDK sources, DB, CI and runners are unchanged.
Evidence: `docs/ops/evidence/candidate-module-unit-casts-20261009.json`.

## Compilation-unit field access typing (2026-10-09)

module_unit_value_field now resolves known nominal fields from owned interfaces, preserving the canonical owner and type, the provider module and declaration, and the original field index alongside the original expression ID and span in an immutable value table. Shared and mutable receiver borrows are unwrapped for nominal field lookup, while resource fields outside the provider are rejected as E0103, with the internal helper 6201 normalized only by the consumer. Unknown operands remain pending, and missing, nonaggregate, or otherwise unsupported member uses fall back to the experimental code2 over the full expression span. Closed builtin fields cover ref str, byte and scalar views' len and is_empty, fixed-array len, and Result is_ok, and the work integrates with scalar peer context, casts, inferred locals, arguments, and returns. Measured shape and name comparisons are charged to the existing work budget; valid known return mismatches are now E1009 while locals and arguments stay E1006. Borrow, place, ownership, and unsafe consumers as well as pointer-cast consumers remain pending, and the existing arena, depth64, round64, and work1MiB bounds are unchanged, with code10 range splitting retained.

| Observation | Result |
| --- | --- |
| Development check | 188 source files; 2344 MIR functions; passed |
| Selected development native cases | 11 passed; 9 unchanged cases reused and 2 corrected-coordinate cases rerun |
| Reference acceptance/code/span | 9/9 matched, including imported field privacy |
| Fixed dispatcher snapshot | 160 modules; 1048562 bytes |
| Detailed observations | 24 units before the 900s process timeout; remaining 2 units passed separately |
| Units / admission attempts / splits | 26 / 39 / 13 |
| Field / checked field totals | 14031 / 6085 |
| Nominal provider field records | 6015 |
| Cast / checked scalar cast totals | 1789 / 52 |
| Argument / checked argument totals | 29822 / 25740 |
| Maximum value work / graph | 747467 / 1048576; 427 / 4096 |


The native reuse receipt retains the original 9-pass/2-failure report and proves
that the only test-file change corrected two expected declaration IDs from 1 to
2. The parser counts the module declaration; compiler/implementation/SDK/lock
identities and the other nine fixtures are unchanged. Both corrected cases were
rerun natively. No failed execution is labeled as a passing execution.
The first process admitted all 26 units, then completed detailed assertions for
24 before its 900-second deadline. The final two ranges (145..152 and 152..160)
passed with the same compiler/manifest/implementation. All ranges continuously
cover modules 0..160; the initial timeout is retained as a failed execution.
This combined observation uses the same bounded 160-module input snapshot; new unit
modules are compiled as implementation code. Full current-source dispatcher
native and stage1/stage2 remain pending. Borrow/dereference/index/slice and
place/ownership/unsafe consumers, raw-pointer casts, move-aware cleanup,
structural cross-unit links/layouts and HIR/MIR/C/CLI remain unfinished.
Release, Linux and push were not run. Public language/SDK ABI, SDK sources,
DB, CI and runners are unchanged.
Evidence: `docs/ops/evidence/candidate-module-unit-fields-20261009.json`.

## Named safe reference and place projection (2026-10-09)

The module_unit_value_place project mainly targets scalar, nominal, and resource references plus nested nominal fields. Safe reference borrow and dereference types reach inferred locals, declared and generic call arguments, scalar casts, and returns, and canonical shared/mutable types are appended to the existing 4096-type graph. Place metadata keeps the whole local owner root and scope, the expression projection ID, mutability, indirect access, and the resource-field boundary, while reference aliases propagate owner provenance across bounded fixed-point rounds. Immutable and nonplace mutable borrows and mutable resource-field borrows emit E1008 at the target span, whereas unsupported shared nonplace borrows and non-reference dereferences use experimental code2 at the whole expression span; ref *r and ref (x) stay unsupported under the reference compiler, so general reborrow support is not claimed. Field grouping retains its named root, array/view/indexed borrows and unsafe pointer dereferences remain explicitly pending, and this is type/provenance projection rather than lifetime, conflict, or move verification; the existing program/work 1MiB, entries 1024, types 4096, activeLocals 128, depth 64, and rounds 64 are unchanged.

| Observation | Result |
| --- | --- |
| Development check | 189 source files; 2350 MIR functions; passed |
| Selected development native cases | 11 passed; unchanged ten cases reused with a source/hash proof, corrected cleanup case rerun |
| Reference acceptance/code/span | 14/14 matched, including generic and cleanup provider context |
| Bounds | program/work 1048576; types 4096; entries 1024; active locals 128; depth/rounds 64 |

The unit only consumes initializer, call-argument and return expressions. Standalone
expression/discard typing, lifetime/conflict/move checking, array/view/index/slice
borrows, unsafe pointer consumers, cross-unit links/layouts and HIR/MIR/C/CLI remain
pending. The standalone implementation receipt excludes corpus telemetry; the
separate full fixed-corpus observation is documented below. Full current dispatcher native and stage1/stage2, release,
Linux and push were not run. Public language/SDK ABI, SDK sources, DB, CI and runners
are unchanged. Evidence: `docs/ops/evidence/candidate-module-unit-places-20261009.json`.


## Safe-reference corpus observation (2026-10-09)

The fixed dispatcher snapshot completed admission and all detailed assertions in
one development execution. Original expression IDs/spans, nominal provider field
coordinates, canonical borrow types and whole-owner root/scope metadata were
checked. The input snapshot hashes are unchanged from the field-unit observation;
new unit modules are compiled as implementation inputs outside this 1MiB snapshot.

| Observation | Result |
| --- | --- |
| Input / units | 160 modules; 1048562 bytes; 26 units |
| Admission attempts / splits | 39 / 13 |
| Borrow expressions / typed borrows | 2891 / 1422 |
| Dereferences / typed safe dereferences | 388 / 254 |
| Expressions retaining owner root/scope | 51532 |
| Checked arguments | 27324 / 29822; previous field-unit result 25740 |
| Checked fields / nominal provider records | 6486 / 6416; previous checked fields 6085 |
| Checked scalar casts | 52 / 1789 |
| Maximum value work / graph | 823678 / 1048576; 427 / 4096 |
| Detailed coverage | All ranges continuously cover 0..160; execution passed without timeout |

Array/view/index/slice borrows, standalone expression/discard typing, lifetime,
conflict and move checking, unsafe/pointer provenance, cross-unit links/layouts,
unit HIR/MIR/C/CLI, full current-source dispatcher native and stage1/stage2 remain
pending. Release, Linux and push were not run. Public language/SDK ABI, SDK source,
DB, CI and runner contracts are unchanged.
Evidence: `docs/ops/evidence/candidate-module-unit-places-corpus-20261009.json`.

## Indexed array and scalar-view values (2026-10-09)

module_unit_value_index resolves known scalar views and admitted fixed-array indexed values with canonical element types. Index operands require usize; E1006 preserves the operand span, an unsupported receiver becomes experimental code2 at the receiver span, and a return mismatch stays E1009. An index place keeps its original owner root/scope and expression projection, and view mutability comes from the view type rather than the binding.

Named array and view element borrows feed canonical shared/mutable references, casts, inferred locals, arguments and returns. Temporary, call and raw-pointer roots never gain borrow address identity. Nested arrays sit outside the current language contract and are rejected by the value consumer even when an older internal graph can represent them. The generic provider and owned cleanup contexts reuse it. Existing limits are unchanged, and the compiler logic is entirely Fairy. Array literal/view-construction/slice, nominal view interfaces, lifetime/conflict/move, unsafe/pointer, HIR/MIR/C/CLI and stage1/stage2 remain pending.

| Observation | Result |
| --- | --- |
| Development check | 190 source files; 2352 MIR functions; passed |
| Selected development native cases | 5 passed; four identical cases reused and the corrected provider cleanup case rerun |
| Reference acceptance / code / span | 12/12 matched |
| Retained bounds | types 4096; expressions/calls/arguments 16384; entries 1024; active locals 128; depth/rounds 64; work 1048576 |

This observation only covers known receiver types in initializer, argument and
return consumers. Declaration/shape admission remains separate: resource array
fields and nested array type annotations still need earlier interface rejection.
No new full-corpus observation is claimed; it is bundled with array/view/slice
construction. Release, Linux, full current dispatcher native, stage1/stage2 and
push were not run. Public language/SDK ABI, SDK sources, DB, CI and runners are
unchanged. Evidence: `docs/ops/evidence/candidate-module-unit-indexes-20261009.json`.

## Canonical array literal projection (2026-10-09)

A new `module_unit_value_array` module checks list and repeat array literals against canonical graph types. The first uncontextualized element anchors the type; explicit array expectations contextualize each element, while an array length mismatch is reported by the existing local/argument E1006 or return E1009 consumer. The count must be a literal in 1..4096; an unsupported count produces code2 at the count span, empty and unsupported element arrays produce code2 at the whole literal, and a mismatched element produces E1006 with scalar overflow producing E1013, both preserving the element span. List edges are collected once and evaluated in source order, with the repeated initializer evaluated once. `module_unit_value_shapes` validates nonempty acyclic nominal field graphs, including permitted inline arrays, and rejects resources, raw pointers, borrows, tags, and nested array elements. The 64-depth and 1 MiB work limit are retained; comparisons are charged and the canonical graph stays at 4096. Known arrays feed inferred locals, indexing, declared and generic call arguments, and returns, with provider expression IDs and spans unchanged. The unknown record initializer remains pending, and no general standalone expression, lifetime, or move completion is claimed. View construction, slices, nominal-view representation, earlier declaration rejection, unsafe and pointer handling, HIR/MIR/C/CLI, and stage1/stage2 remain. There is no new full corpus pass, release, Linux build, or push.

| Observation | Result |
| --- | --- |
| Development check | 192 source files; 2355 MIR functions; passed |
| Selected development native cases | 5 passed; four identical cases reused and the corrected shape-rejection case rerun |
| Reference acceptance / code / span | 15/15 matched |
| Literal order / identity | Source-order list edges, one repeated initializer, canonical element and length retained |
| Bounds | graph 4096; literal elements 1..4096; work 1048576; active shape depth 64 |

The original resource-bearing struct negative fixture was rejected earlier by
reference declaration analysis with E1002. It was replaced with a legal
pointer-bearing struct whose array literal is rejected by the value-shape gate.
General declaration rejection, record construction, standalone expression,
loan lifetime/conflict/move, view construction/slices, nominal view interfaces,
unsafe/pointers and downstream HIR/MIR/C/CLI remain incomplete. Full corpus
observation is bundled with the next view/slice unit. Release, Linux, full current
dispatcher native, stage1/stage2 and push were not run. Public language/SDK ABI,
SDK sources, DB, CI and runners are unchanged.
Evidence: `docs/ops/evidence/candidate-module-unit-arrays-20261009.json`.

## Scalar array views and slices (2026-10-09)

module_unit_value_view now covers all existing byte, bool, and integer view families. Direct ref or ref mut of a named scalar array takes the exact contextual view type; without a matching view context it keeps a canonical reference to the array. An explicit ref over a grouped named array or a plain struct array-field range yields a view slice. Ordinary slices of existing views keep their shared or exclusive type and the whole owner root and scope across aliases. Omitted bounds stay as zero source IDs; explicit bounds are usize in receiver, start, end order; half-open bounds are checked at later runtime and no static E is invented. ref view[..] is unsupported because the compiler does not borrow temporary descriptors. Mutable-to-shared shortening applies only to call arguments, not local declarations.

E1008 points at the target span for the immutable mutable-borrow and the nonplain field subview; E1006 points at the bound, argument, or local span; code2 keeps the receiver or whole-borrow span. Field owners with plain shapes are validated for array-field views. The bounds wrapper is non-value syntax and original expression IDs and spans are preserved. Limits graph4096, work1MiB, locals128, depth/round64 are unchanged. This is type and provenance work only: lifetimes, conflicts, moves, and child generations remain incomplete, and nominal view interface type representation is still pending with no claim of completion. No unobserved corpus numbers, no release, Linux, or push, and no HIR, MIR, C, CLI, stage1, or stage2.

| Observation | Result |
| --- | --- |
| Development check | 193 source files; 2366 MIR functions; passed |
| Selected development native cases | 9/9 passed |
| Reference acceptance / code / span | 17/17 matched |
| Aliases / bounds | Whole-owner root/scope retained; omitted bounds keep zero IDs; explicit usize bounds typed in order |
| Call shortening | Mutable scalar views can shorten to shared call arguments; local declarations remain exact |
| Bounds | types 4096; work 1048576; active locals 128; depth/rounds 64 |

The implementation receipt covers scalar view families and contextual consumers.
Nominal view interfaces and nominal array-field borrowing remain pending, as do
record/control/variant/builtin and standalone expressions, earlier declaration
rejection, loan generations/lifetime/conflict/move, unsafe/pointer provenance,
move-aware cleanup, cross-unit links/layouts and HIR/MIR/C/CLI. The separate full
fixed-corpus observation is still running. Full current dispatcher native and
stage1/stage2, release, Linux and push were not run. Public language/SDK ABI,
SDK source, DB, CI and runner contracts are unchanged.
Evidence: `docs/ops/evidence/candidate-module-unit-views-20261009.json`.
