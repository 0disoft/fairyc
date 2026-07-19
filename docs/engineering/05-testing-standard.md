# Testing Standard

- Status: Draft
- Owner: UNASSIGNED
- Semantic sources: `spec/grammar/stage0.ebnf`, `spec/fixtures/parser/manifest-v0.json`, `spec/fixtures/semantic/manifest-v0.json`, `spec/fixtures/resource/manifest-v0.json`, `spec/fixtures/region/manifest-v1.json`, `spec/fixtures/result/manifest-v0.json`, `spec/fixtures/mir/manifest-v0.json`, `spec/fixtures/mir-verifier/manifest-v0.json`, `spec/fixtures/interpreter/manifest-v0.json`, `spec/fixtures/package/manifest-v0.json`, `docs/language/`, `docs/mir/`, `docs/package/`, `docs/abi/`, `docs/runtime/`, `docs/compatibility/`, `docs/diagnostics/catalog-v0.md`, `docs/atlas/`
- Merge-blocking validation: `VALIDATION.md`

The Go frontend executes every case in `spec/fixtures/parser/manifest-v0.json` through
`internal/parserfixture`. Source, lexer, syntax, AST, budget, containment, and manifest behavior
also have focused unit tests. Lexer and parser fuzz targets run their deterministic seed corpus
during ordinary Go tests and assert lossless reconstruction, stable output, valid spans, bounded
exhaustion, and error-free AST lowering. Experimental CLI tests cover argument routing, reserved
stable commands, deterministic content-safe JSON, schema synchronization, human diagnostics,
bounded file reads, source-root escape rejection, output failure, clean syntax checks, semantic
success, semantic diagnostics, unsupported-subset rejection, closed native-build options,
cross-platform artifact-path rejection, no-overwrite promotion, symbolic-link escape rejection,
and ineligible-host failure before artifact creation.
Target-description tests separately cover structural validation, power-of-two layouts, closed and
sorted scalar and feature sets, immutable lookup copies, unknown-target rejection, deterministic
host-independent JSON, false maturity claims, and top-level JSON-schema synchronization.
`internal/layoutfixture` additionally executes all 25 v0 layout cases across the 16-case Linux GNU
and nine-case Windows MSVC manifests with strict JSON decoding, exact target binding, checked
size/alignment arithmetic, declaration-order offsets and padding, closed error codes, sorted unique
case IDs, and pass/fail expectation-shape validation.

Experimental semantic CLI coverage also verifies explicit multi-file module sets, input-order
determinism, cross-file visibility diagnostics, duplicate rejection, bounded file counts, and a
repository-owned qualified-import example.

`internal/semanticfixture` executes all 18 cases in `spec/fixtures/semantic/manifest-v0.json`
against name resolution, typed HIR, and local safe-borrow Flow. It verifies exact failure phase,
complete diagnostic set, related spans, selected resolution coordinates, selected canonical type
facts, borrow origins, last uses, escape facts, and loop-context diagnostics. `internal/resourcefixture` executes all 12 cases
in `spec/fixtures/resource/manifest-v0.json`, comparing complete-root states, cleanup obligations,
and the complete deterministic `E11xx` set after parser, resolver, and type setup.
`internal/regionfixture` executes all 6 cases in `spec/fixtures/region/manifest-v1.json`, compares
hidden allocation origins and complete `E31xx` diagnostics, and admits the lexical Arena case only
after proving its root, dependency, lookup, instance class, provider role, source root, and exported
`std::alloc::Arena` against the referenced package graph. `internal/resultfixture` executes all 24
cases in `spec/fixtures/result/manifest-v0.json`, comparing qualified construction, exhaustive
shallow matching, handling modes, canonical payloads,
single-evaluation facts, and complete `E41xx` diagnostics. `internal/mirfixture` executes all 11
cases in `spec/fixtures/mir/manifest-v0.json` after Flow, mutable-local MIR lowering, and MIR
verification, comparing source-linked evaluation, call, move, Region, loop-control cleanup, Result-branch, trap, and
eliminated-form facts. `internal/mirverifyfixture` constructs all 10 named verifier seeds, requires
each baseline to pass, clones it, applies exactly one typed mutation, and proves all 21 cases return
the complete expected one-code result. `internal/interpreterfixture` executes all 56 interpreter
v0 cases through source loading, Flow, verified MIR, and the bounded interpreter, comparing exact
`i32` returns, scalar and resource-bearing `Option` and `Result` construction, every built-in match
variant, both Result `?` paths, canonical trap categories, resource construction, whole-root call
moves, ordered resource reassignment, consuming-close ownership, active-variant cleanup,
Region fallthrough, direct-return, Result-propagation, nested, break, and continue reset ordering, source-linked drop traces, checked arithmetic and right-shift outcomes, exact fixture-only output bytes and order, or the stable interpreter step/output limits
category. `internal/cbackendfixture` executes all 18 cases in
`spec/fixtures/c-emission/manifest-v0.json` through source loading, Flow, verified MIR, interpreter
outcome comparison, and deterministic C11 emission, comparing target, dialect, canonical function
names, feature families, LF-only path-clean text, and repeated byte identity. It does not invoke a C
parser, compiler, assembler, linker, executable, sanitizer, layout probe, or ABI probe. Native C
behavior belongs to `internal/cbackendnative`, which consumes the same emitted units through the
shared C driver and scalar runtime. ABI, complete-runtime, and Atlas runners remain future
evidence requirements.

`internal/cstdlib` separately embeds the provisional SystemHeap implementation and its closed C
probe. Ordinary Go tests prove deterministic, LF-only source ownership and forbid raw `realloc` or
output. The Linux native runner compiles and executes the probe with Clang and GCC at low and
optimized profiles, checking zero-size normalization, fundamental alignment, growth and shrink
prefix preservation, unsupported-alignment rollback, release, and invalid output rejection. These
four configured executions are not source-level standard-library fixtures, deterministic OOM
injection, or hosted evidence until a Linux receipt is observed.

The Buffer probe adds ordinary append, no-op reserve identity, relocating initialized-prefix
self-append, length-overflow rollback, zeroed growth, truncate, clear, malformed state, and drop.
It verifies observable bytes, length, minimum capacity, and failure-state preservation without
freezing the geometric capacity choice or reading spare storage.

A separate generated-Fairy Buffer source probe protects the ABI connection rather than duplicating
the C-only behavior matrix. Its success case requires dependency-ordered nested `@repr(C)`
definitions, exact 24-byte child and 32-byte containing layouts, real initialize and zeroed-resize
mutation, observable zero bytes, and normalized drop. Backend unit boundaries reject recursive
value dependencies and a non-`@repr(C)` nested field.

The Arena probe adds single-active-Region rejection, zero-initialized and copied byte placement,
unsupported-alignment rollback, zero-size normalization, active-drop rejection, reset generation
advance, stale-use rejection, and final drop. It does not freeze node size, allocation grouping, or
an Arena growth policy.

A separate generated-Fairy Arena source probe protects the exact cross-translation-unit record
contract. It requires self-linked node pointers, the nested SystemHeap block, provider-proven
`uint64_t`/`unsigned long` identity, 32/24/32-byte node/Arena/Region layouts, a `void **` output slot,
observable zeroed bytes, released nodes, invalidated Region state, and reset/drop generations.

The Windows target product smoke covers a narrower source-level Arena lane. Its private FY records
use target-width `usize` for the C unit's eight-byte generation storage, call the five exact lifecycle
symbols, require generation 1 after initialization, one active Region and non-null allocation/node,
generation 2 plus cleared Region state after reset, and generation 3 after drop at low and optimized
MSVC profiles. It does not dereference the output pointer, so zeroed contents and the Arena failure
matrix remain owned by the Linux C-only and generated-source probes.

The text-view probe validates mixed-width text and rejects stray continuation bytes, two overlong
families, surrogate encodings, values above U+10FFFF, truncation, and obsolete leaders. It checks
valid and invalid byte-boundary slices, empty normalization, invalid pointer/length pairs, and
unchanged outputs on failure. It is byte-level UTF-8 evidence, not Unicode normalization or
grapheme behavior.

The owned-text probe checks String copy, immutable view, ordinary and self-source append, drop,
failed Buffer conversion with unchanged ownership and destination, successful consuming move, and
CString construction, bounded CStr view, exact trailing terminator, interior-NUL rejection,
unchanged failure output, and empty CString. It does not assert source Result layout or locale
conversion.

A separate generated-Fairy CStr source probe protects the exact cross-translation-unit view record
and real bounded validator call. Its success case requires a nonempty terminated byte sequence and a
logical length excluding the terminator. Its two failure cases distinguish interior NUL from a
missing terminator and require the previously valid output view to remain byte-for-byte observable.

A compiler-owned C-string source probe protects the literal path separately from that record-based
validator. The Fairy caller must preserve exact `fairy\0` storage through semantic analysis, MIR,
and C emission, perform the dedicated unsafe immutable-byte-pointer extraction, and let a separate C
callee read all six bytes. A second product-build smoke carries nonempty `&CStr` storage through the
ordinary CLI without a companion source. These are execution and artifact oracles, not proof of a
public safe CStr API.

A compiler-owned byte-string source probe separately protects exact non-terminated `fairy` bytes,
the private non-null `&[u8]` `{ data, length }` view, length five, typed-local unsafe data-address
extraction, and a separate C callee reading all five bytes. The emitter test also covers embedded
NUL, duplicate storage, and the length-zero non-null sentinel. A second product-build smoke carries
the view through the ordinary CLI. These are bounded-view transport oracles, not source indexing,
slicing, a public slice ABI, or safe foreign-borrow evidence.

The generated-Fairy owned-text source probe protects the success lifecycle through the exact nested
records and by-value operation result. String must copy `Fairy`, expose it, append it to
`FairyFairy`, expose the new bytes, and drop to canonical empty storage. CString must copy the text,
expose a five-byte CStr view followed immediately by one NUL, and drop to canonical empty storage.
Every high-level and allocation status must be zero. Existing C-only cases retain ownership of the
negative and rollback matrix rather than duplicating it in the source probe.

The generated-Fairy Linux File source probe protects exact result, File, and open-option records
plus by-value operation-result returns. It writes and synchronizes `FairyFile`, consumes close,
requires repeated close to report closed, reopens, performs three three-byte partial reads, observes
EOF with count zero, closes, and compares the reconstructed bytes and normalized handle. The C-only
Linux File probe retains direct partial-write, invalid-option, path, no-progress, `EINTR`, and forced
close-failure ownership rather than duplicating those negatives in the source probe.

The standard-library product-build smoke is distinct from source-call companion probes. Its `.fy`
source calls the real empty-input UTF-8 predicate without companion C, so success requires emitted
used-symbol metadata, minimal component resolution, the ordinary scalar-runtime build path, native
link, and exit 42 under Clang/GCC at low/optimized. Uncalled declaration filtering and component
dependency closure remain host-independent Go tests.

## Oracle order

1. Product specification and accepted ADRs define the intended boundary.
2. Versioned language and MIR fixtures become the executable contract.
3. The MIR verifier rejects structurally invalid compiler output.
4. The MIR interpreter provides the first executable semantics oracle.
5. Generated C is compared against the interpreter under the C compilers and profiles declared by the target fixture.
6. Later backends are compared against the same interpreter and C evidence rather than defining new behavior.

## Fixture classes

| Class | Proves |
| --- | --- |
| Parse pass and fail | `spec/fixtures/parser/` currently proves grammar acceptance, precedence trees, literal cooking, lossless reconstruction, recovery provenance, byte spans, and deterministic diagnostics through the Go parser runner |
| Semantic pass and fail | `spec/fixtures/semantic/` currently proves all 23 v0 cases: resolution coordinates, canonical types, local last-use loans, borrow return origins, Stage 0 signed-left-shift rejection, loop-control context rejection, admitted scalar foreign calls with distinct source/link identities, producer-to-consumer transport for all sixteen exact TargetSpec C scalars, `E5101` unsafe boundaries, `E5102` safe-reference exclusion, `E5106` malformed or duplicate link-name rejection, phase ownership, related spans, and deterministic cascade suppression |
| Module pass and fail | File-to-module identity, explicit aliases, forbidden import forms, cycles, and source visibility |
| Type and initialization fail | Inert/resource separation and initialized-local rules |
| Resource flow pass and fail | `spec/fixtures/resource/` currently proves all 12 v0 cases: whole-root moves, consuming tagged matches, live replacement, dead-slot reinstallation, scope obligations, branch, normal-loop, break-edge, and continue-edge states, and deterministic `E11xx` spans; conditional close ownership stays a runtime/MIR concern |
| Borrow-origin pass and fail | Local last-use loans, conflicts, single-origin return, and storage bans |
| Region pass and fail | `spec/fixtures/region/manifest-v1.json` currently proves all 6 v1 cases: parameter and package-resolved local origins, permitted parameter-origin returns, `E3101` local escape, capability restrictions, and no `NeedsDrop` placement |
| Result pass and fail | `spec/fixtures/result/` currently proves all 24 v0 cases: qualified scalar construction, exhaustive shallow matching, return, discard, propagation, local and parameter transfer, handled reassignment, all-branch handling, partial-branch rejection, loop back-edge, break, continue, and overwrite rejection, early-return obligations, exactly-once evaluation, and error compatibility; active resource payload traces are covered by MIR and interpreter fixtures |
| MIR normalization | `spec/fixtures/mir/` currently proves all 11 v0 cases through the Go MIR runner: source-linked evaluation order, call successors, ownership transfer including a resource-bearing Option match, RHS-before-drop resource reassignment with live/dead destinations, Region placement and reset ordering on fallthrough, direct return, Result propagation, nested, break, and continue exits, cleanup order, Result branching, division and checked-arithmetic trap paths, and eliminated source forms; graph identity remains excluded |
| Resolved package graph | `spec/fixtures/package/` proves instance equality, feature splits, direct visibility, module isolation, ambiguity, transitive hiding, and explicit standard-library injection without choosing public package-manager syntax |
| Built-in tagged resources | `Ok`, `Err`, `Some`, discard, match, propagation, early return, and nested active-payload drop traces |
| MIR verifier mutation | `spec/fixtures/mir-verifier/` currently proves all 10 named baselines and 21 isolated mutation cases, including missing resource definition and tagged-operation type `V1001`, fallthrough and break Region reset `V1403`, and missing checked-arithmetic `V1501`, through the Go verifier runner without serializing arbitrary MIR |
| MIR interpreter run, trap, and limit | `spec/fixtures/interpreter/` currently proves all 56 v0 cases: literal return, scalar call arithmetic, checked negation/addition/subtraction/multiplication, division/remainder, and right shift across trapping and non-trapping boundaries, boolean branching and logical not, source `&&` and `||` short-circuiting, scalar `while`, nearest-loop break and continue, scalar and resource-bearing `Option` and `Result` construction, every built-in match variant, scalar and resource-bearing Result `?` paths, resource construction and field reads, whole-root and tagged-payload moves, resource returns, ordered live/dead/self reassignment, consuming-close success/failure ownership, nested active-variant cleanup, deterministic drop traces, lexical Region placement and reset on fallthrough, conditional fallthrough, direct return, Result propagation, nested, break, and continue exits, exact manifest-authorized in-memory stdout/stderr bytes, order, and retained limit prefix, exact division-by-zero, signed-division-overflow, integer-overflow, and invalid-shift trap identity, and bounded step/output-limit termination; user drop-body effects, signed left shift, and host I/O remain deferred |
| Drop elaboration | Partial construction and cleanup-suffix merging; loop `break` and `continue` are fixture-backed |
| C emission | `spec/fixtures/c-emission/` currently proves 18 verified scalar, restricted inert-enum, flat-resource, and resource-tagged source-to-C11 cases, including private user-enum tag/payload/switch transport, private scalar and resource-bearing `Option`/`Result` transport, all-`i32` direct resources, typed propagation, deterministic target/dialect/function/feature metadata, target-layout assertions, LF-only path-clean output, and repeated byte identity; it does not prove C syntax acceptance or native behavior |
| C differential | The configured Linux runner covers all 18 C-emission cases across Clang/GCC and low/optimized profiles for 72 process-status plus exact source-linked drop-trace comparisons, recompiles and executes every production unit under Clang/GCC ASan+UBSan for 36 fail-closed sanitizer checks, exercises scalar, all sixteen exact TargetSpec C scalars through four individual and one twelve-type family fixture, typed-null FFI, register and hidden-memory `@repr(C)` struct results, an exact `void *`/`size_t` SystemHeap block, the exact nested Buffer record, exact self-linked Arena/Region records with a `void **` output slot, CStr validation, compiler-owned `fairy\0` C-string storage, compiler-owned five-byte `&[u8]` storage and view transport, owned String/CString lifecycles, and exact Linux File results/options/handle transport, links generated FY callers to the actual SystemHeap, Buffer, Arena, text-view, owned-text, and Linux File C units, then exercises the uninstrumented experimental build CLI with one resource-bearing program, one automatically linked UTF-8 standard-library call, one nonempty compiler-owned C-string program, and one compiler-owned byte-slice program under all 4 compiler/profile pairs; hosted execution is not yet observed, and output, wider returns, user drop-body effects, broader public layout/ABI, aggregate parameters, nested enum payload transport, C-scalar value semantics, UTF-8 literal execution, general slice operations, safe stdlib ownership/effects/errors, and the complete product runtime remain deferred |
| Target description | `internal/target` proves the `fairyc.target-spec/v0` structural invariants, canonical x86-64 Linux GNU and fail-closed Windows MSVC facts, immutable lookup, absent host paths, and fail-closed maturity evidence; `internal/cli` proves deterministic schema-shaped inspection, unknown-ID rejection, and an explicit `build-target` boundary. The Windows entry reports `scalar-and-llp64-c-scalar-c11`; a checked-in source is emitted after target selection, preserves four-byte `c::long`, rejects unavailable `c::ssize_t`, executes observed SystemHeap allocation/release, Buffer initialize/resize/drop, Arena/Region success lifecycle, String copy/view/append/drop, CString construction/view/drop, direct CStr validation, and the real empty-input UTF-8 validator, builds with local x64 MSVC, and returns 42. Both profiles use the documented 16-byte x64 `malloc` alignment ceiling and every other foreign symbol fails before toolchain discovery. Zeroed Buffer/Arena bytes, independent owned-text byte and terminator observation, owned-text and Arena failure matrices, broad or bidirectional source-level Windows FFI, complete independently inspected ABI/runtime evidence, and a hosted target receipt remain absent. |
| Target layout | `spec/fixtures/layout/` currently proves 16 host-independent Linux GNU v0 cases for language and C scalars plus nine Windows MSVC cases for LLP64-sensitive scalars, data and function pointers, explicit C-enum bases, a nested aggregate, and absent POSIX `c.ssize_t`; the 11 valid Linux cases generate one recursive C11 plus explicit POSIX-typedef probe configured for 4 Clang/GCC low/optimized executions, while the eight Windows pass cases generate a separate non-POSIX C11 unit compiled and executed twice by local x64 MSVC through the configured host smoke; full Windows scalar coverage and hosted receipts remain deferred evidence |
| C ABI classification and scalar source FFI | `spec/fixtures/c-abi/` proves 17 Linux SysV v0 cases for INTEGER/SSE/MEMORY eightbytes, register exhaustion, whole-aggregate rollback, and hidden results plus 15 Windows Microsoft-x64 cases for shared ordinal slots, fifth-argument stack transport, small-record integer transport, aggregate indirection, and hidden-return slot shifting; the 14 Linux pass cases generate 56 separate Clang/GCC crossings, while the 12 Windows pass cases generate separate caller/callee C11 units that local x64 MSVC crosses at low and optimized profiles. Distinct `@link_name`-mapped `i32 -> i32`, four individual C-scalar, one remaining twelve-type family, one register-plus-hidden struct-result, and one exact SystemHeap block source fixture each cross generated Linux callers and hand-written or project-owned C callees in both directions and profiles for 4 configured executions. None has an observed hosted result; the Windows probe is same-compiler C rather than Fairy source or independent/register-level evidence, and by-value aggregate parameters, retained callbacks, exports, C-scalar value semantics, and C-scalar raw-pointer lowering remain absent. |
| Windows PE executable and COFF object structure | `internal/pecoff` validates bounded synthetic positive and malformed AMD64 PE32+ images and raw COFF objects without host loading. The configured Windows smoke applies the executable parser to the retained target-neutral product executable, both declared-target profile executables, and both profiles of the layout, ABI, and unwind probes, for nine inspected images. Machine, PE32+ magic, executable/non-DLL/large-address flags, console subsystem, image/header/file/section alignment, unique ordered sections, bounded raw and virtual ranges, readable executable `.text`, and executable entry placement are required. Every MSVC build in the same smoke also parses each intermediate `.obj` before cleanup, requiring raw AMD64 COFF, no optional header or linked-image flags, at most 4,096 sections inside the 64 MiB object budget, bounded section and relocation ranges, readable executable `.text`/`.text$...`, valid relocation symbol indexes, and named symbols while admitting characteristic-backed storage-free `.bss`. A synthetic 154-section regression object covers the function-level section fanout observed after owned-text linkage without turning that observed count into a stable compiler-output promise. Import payloads, exact object symbols, COMDATs, relocation types, CodeView/PDB, and hosted evidence remain absent. |
| Windows x64 unwind structure and execution | `internal/pecoff` builds synthetic valid and malformed exception directories and applies the same parser to all nine local MSVC PE images. `.pdata` must contain sorted nonoverlapping 12-byte runtime-function entries whose code ranges are executable and whose DWORD-aligned unwind RVAs map to readable file-backed data. Version 1 prolog operations require bounded slot counts and descending offsets; version 2 short/long epilog descriptors and spare slots receive bounded structural handling after local MSVC 14.51 produced version 2 records. Handler and chained tails receive minimum RVA/range checks. A separate three-object probe runs at low and optimized profiles: a private `RaiseException` payload must reach its outer filter, `RtlVirtualUnwind` must walk raiser, termination, and catcher frames in order, and the real unwind must execute one abnormal `__finally`. Version-2 epilog offset semantics, language-specific handler payload parsing, recursive chain equivalence, Fairy/foreign-frame crossings, object-level unwind relocations, and hosted evidence remain absent. |
| C safety | Overflow, division, shifts, evaluation order, aliasing, alignment, inactive payload, and sanitizer-assisted checks |
| ABI | Target layout, calls, callbacks, raw bindings, and boundary shims for candidate target profiles |
| Runtime and stdlib | Zero-size normalization, allocation failure, reserve rollback, UTF-8 invariants, partial I/O, Linux close consumption, and retained-pointer wrappers |
| Compatibility | Exact pre-1 matching, cache misses, runtime mismatch, diagnostic identity, stale maps, target mismatch, and relation-specific completeness |
| Atlas schema | Deterministic identity, record compatibility, unknown edges, provenance, and whole-package rewrite |
| Atlas context | Budgets, stable ordering, omissions, stale source, text caps, and prompt-injection isolation |

## Diagnostic assertions

Compile-fail fixtures assert the stable diagnostic code, primary failure category, and required related spans. They do not freeze full prose unless wording itself is the contract. A single malformed input may produce recovery diagnostics, but the fixture names which diagnostic is primary.

Parser fail fixtures are stricter: they list every `E00xx` result, use half-open byte spans, and record whether recovery inserted, skipped, synchronized, or exited. Parser pass fixtures assert syntax only and may intentionally fail later name or type analysis. Operator fixtures compare normalized tree shape rather than only evaluation results.

Semantic fail fixtures list every independent diagnostic after cascade suppression, identify the owning resolver, type, or Flow phase, and use role-labelled related spans. Semantic pass fixtures may assert declaration coordinates, canonical types, and borrow origins, but never whole AST or HIR dumps or build-local IDs.

Resource fixtures sample complete-root state immediately before or after a source span. They assert that `Live` owns one cleanup obligation, `Dead` owns zero, and a source-visible `MaybeLive` join is rejected. MIR drop flags and executable drop order remain separate fixture contracts.

Region v1 resolves its lexical escape source through the explicit standard-library package instance and opaque fixture `Arena`; it does not pretend a user-declared type is the standard-library arena. Result fixtures retain parameter-driven cases to isolate handling, add the qualified ADR 0017 constructors and shallow matches, and exercise the ADR 0019 pending-obligation dataflow separately. Each manifest lists its deferred coverage explicitly.

MIR fixtures compare source-linked observations after verification. They do not compare textual dumps, block or local numbers, temporary names, or generated C. MIR-verifier fixtures separately require each named baseline to verify, apply one typed mutation, and return the complete one-code result; arbitrary malformed graph input remains deferred.

Package fixtures start from an already-resolved graph. They do not prove public manifest parsing, version solving, registry behavior, source acquisition, lockfile encoding, or a concrete package-instance hash.

C-emission fixtures compare structured unit facts and deterministic text hygiene without freezing the
entire generated translation unit. `internal/cbackendnative` separately requires Linux/amd64,
glibc, Clang, and GCC, then compiles and executes every emitted case at `-O0` and `-O2`. The command
fails rather than skips when a required toolchain component is absent. Its current semantic oracle compares
reserved process status and exact source-linked compiler-owned drop sequences through a closed
instrumentation profile. A distinct `address-undefined-o1` mode runs the uninstrumented production
unit under ASan+UBSan with both compilers and treats every diagnostic as failure; this is an
executed-path backend check, not another language oracle. The lane does not prove output, negative
or larger `i32` returns, user drop-body effects, layout, ABI, the complete product runtime, or target conformance. The
experimental build command shares this driver and scalar runtime but emits and links only the
uninstrumented single-file artifact path; it does not inherit the fixture observer or broaden target
evidence.

## Differential matrix

The first planned matrix targets `x86_64-linux-gnu`, uses the MIR interpreter plus generated C, and compares Clang with GCC at low and optimized profiles. Exact compiler and glibc baselines remain UNDECIDED. Fairy-to-C and C-to-Fairy fixture directions are both mandatory. A new target or backend cannot replace the interpreter and reference-C oracles merely because its outputs agree with itself.

The host-independent Linux C ABI fixtures cover scalar and pointer classification, fixed arrays,
register exhaustion, mixed integer and floating arguments, by-value aggregate plans, and hidden
return storage. The Windows fixtures separately cover shared ordinal slots, aggregate size gates,
indirect arguments, and hidden-result slot shifting. A separate-object Clang/GCC probe exercises all 14 Linux pass plans in both compiler
directions. A Fairy-source probe additionally crosses an 8-byte register struct result and a
20-byte hidden-memory struct result, while another crosses typed null pointers plus `usize` and `u8` in
both directions. Another four-way probe links the empty-input UTF-8 call against the actual
standard-library text-view C unit instead of a mock. A CStr probe additionally links the generated
caller against the actual owned-text C unit and supplies nonempty success and failure storage.
Buffer, Arena, and Linux File source probes supply non-null nested records, a self-pointer, one
pointer-output crossing, partial counts, and consuming state transitions, but native source fixtures still need independently padded
`@repr(C)` parameters, explicit C enums, narrow extension rules, mixed-class aggregate results,
and the permitted synchronous callback. Every excluded C ABI feature eventually needs a negative
source fixture as its semantic surface is implemented.

## Fuzzing and sanitizers

Lexer and parser fuzz entry points now exist, and their fixed seed corpus runs under the ordinary
test command on Windows and Linux CI. A timed or continuous fuzz campaign is not configured yet.
Type, Flow CFG, MIR verifier, and schema fuzzing remain later validation surfaces. Generated-C
sanitizer runs now cover every admitted emission case with Clang and GCC in the hosted native
command, but no hosted result has yet been observed. They are backend evidence, not proof of
language semantics. Fuzz duration, retained corpus policy, and coverage thresholds remain
uncommitted until reproducible campaign evidence exists.

## Review blockers

- A semantic rule has no positive or negative observable fixture class.
- A grammar, literal, precedence, or recovery change updates prose without changing `spec/grammar/stage0.ebnf` and the parser manifest in the same review.
- A resolution, typing, or Flow rule changes without a schema-compatible semantic pass or fail fixture and synchronized diagnostic meaning.
- A whole-root ownership rule changes without synchronized resource-state assertions, `E11xx` spans, and drop-elaboration review.
- A Region or Result rule changes without its dedicated manifest, deferred-coverage review, `E31xx` or `E41xx` spans, and MIR-lowering review.
- A MIR rule changes without synchronized source-linked observations, deferred-coverage review, verifier impact, and interpreter or enabled-backend impact.
- A MIR verifier rule changes without synchronized seed entities, one-mutation isolation, complete code coverage, and harness-failure review.
- A package identity or visibility rule changes without synchronized resolved-graph cases, module lookup review, compatibility review, and Atlas package-edge review.
- A backend test compares only generated text instead of executable behavior where execution is available.
- A diagnostic code changes meaning without the catalog and compile-fail fixtures changing together.
- A trap, move, or cleanup result differs by optimization profile.
- A foreign nonlocal-control test crosses a Fairy C frame instead of using a boundary shim.
- An Atlas test treats missing or opaque edges as a complete graph.
- A target is called conformant or supported while its declared fixture subset contains a skip.
- A runtime wrapper normalizes Linux-specific behavior and then documents it as portable POSIX behavior.
- Exact test counts, durations, or coverage numbers are promised without an executable harness and source of truth.
