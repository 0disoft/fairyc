# Compatibility and Versioning v0

- Status: Draft Stage 0 contract
- Product source: `docs/product/02-spec.md`
- Decision source: `docs/adr/0008-versioned-contract-boundaries.md`

## Compatibility vector

Build receipts, caches, emitted objects, diagnostics, and maps record the applicable members of this vector:

```text
compiler release and full build identity
language revision
internal MIR version
target schema
ABI profile
diagnostic catalog
diagnostic JSON schema
runtime ABI
standard-library version and package instance
manifest schema
lockfile schema
.fymap schema
SymbolId algorithm
interface-hash algorithm
```

No concrete current values exist because no compiler or released schema exists. Examples use families such as `L0.n`, `M0.n`, `RT0.n`, and algorithm IDs such as `SID1`; they are formats, not release claims.

## Compiler and language

The compiler release uses SemVer. A patch may fix crashes, performance, diagnostics wording, or implementation behavior that contradicted the current specification. It cannot intentionally add syntax, change overflow or ownership semantics, or make a valid program invalid without a language revision change.

Pre-1 language revisions are exact pins. A compiler may implement several revisions, but a newer pre-1 revision does not imply source compatibility.

Grammar spelling, keyword status, literal cooking, precedence, module-path identity, recovery acceptance, resolution, canonical typing, resource and Region Flow rejection, Result handling, declared effects, call-effect acceptance, and whether a diagnostic blocks HIR are language-revision semantics. Parser-, semantic-, resource-, Region-, and Result-fixture schemas version their expectation-file shapes; they do not make two different language revisions source-compatible.

For one function contract, removing a declared effect preserves every existing caller accepted by
the effect checker. Adding an effect or replacing it with another can invalidate callers and is a
source-compatibility change. Reordering or deduplicating the same set is not a semantic change
because typed HIR stores canonical lexical order; duplicate source declarations are rejected.

## MIR and cache

Internal MIR accepts an exact version only. It is mutable compiler state, not a plugin or interchange ABI. Serialized MIR and caches are regenerated from source rather than migrated. The source-based MIR fixture manifest versions repository observations separately and deliberately excludes serialized graph identity. A future stable backend exchange format, if needed, is a separate contract.

`fairyc.mir-verifier-fixtures/v0` versions named seed vocabulary and mutation recipes. It does not serialize arbitrary MIR, stabilize compiler-local IDs, authorize external graph input, or make old MIR migratable.

A cache key includes the full compiler build identity, language and MIR versions, target schema and ABI profile, toolchain fingerprint, backend and profile flags, runtime ABI, standard-library instance, source digest, dependency interface digests, feature set, and interface-hash algorithm. Incompatibility is a cache miss followed by lazy collection.

## Target and ABI

Target schema versions the structure of target descriptions. ABI profile versions the meaning of layout, calling convention, return classification, and target-visible representation. Fixing an implementation to match an unchanged profile changes the compiler build and invalidates cache; changing the specified meaning creates a new ABI profile major.

Local SDK or compiler paths never participate in target semantic identity. Toolchain family, baseline, and capability fingerprint do.

The prototype `fairyc.target-spec/v0` object is the first executable target-schema member. Its
identity, scalar and pointer layouts, call alignment, red-zone and shadow-space rules, classifier identities, object and symbol model,
unwind/debug contract, and C-layout feature vocabulary are compatibility data. Its maturity
evidence is current capability evidence, not an ABI promise; promoting a status requires the named
fixture and hosted receipts rather than only editing the booleans. The experimental CLI schema at
`spec/cli/experimental-target-spec-v0.schema.json` must remain synchronized with the Go model.
The target entry's `emissionSubset` is compatibility data: it names what the selected backend
profile may represent, not whether the target is conformant. The experimental `build-target`
command currently admits only `x86_64-windows-msvc`, a fixed low/optimized profile choice, one
positional FY entry plus a bounded explicit repeated-source closure, and one `.exe` output. Its
command spelling and text receipt remain unstable CLI surface;
the selected target identity, scalar layout, C dialect, and emitted static assertions still must
match the target-schema and ABI-profile contracts. Adding another called source-level FFI symbol or another target
requires a separate capability and compatibility decision rather than silently widening this path.
ADR 0083 admits exactly `fy_std_text_utf8_valid_v0` as the first Windows source-call linkage. That
symbol's signature, selected component, and fail-closed rejection of every other foreign symbol are
prototype compatibility data; this does not make the complete standard library portable or stable.
ADR 0084 adds exact SystemHeap allocate/deallocate linkage and a target-only 16-byte fundamental
alignment ceiling derived from the Windows x64 `malloc` contract. The ceiling, two signatures,
record layout, minimal component closure, and normalized release observations are prototype
compatibility data; allocator resize, forced OOM, over-aligned allocation, hosted CRT deployment,
and the public `std::alloc` API remain outside the admitted slice.
ADR 0085 adds exact Buffer initialize, zeroed-resize, and drop linkage. The nested 32-byte record,
Buffer-to-SystemHeap component closure, deterministic single emission, capacity/length
observations, and normalized drop state are prototype compatibility data. Zeroed byte observation,
append/reserve/truncate/clear, safe ownership, and the public `std::bytes` API remain outside the
Windows slice.
ADR 0086 adds exact Arena initialize, Region begin, zeroed allocation, Region reset, and Arena drop
linkage. The private 32/24/32-byte node/Arena/Region layouts, target-width generation transport,
Arena-to-SystemHeap closure, single emission, non-null allocation/node state, active flags, and
generation 1/2/3 transitions are prototype compatibility data. Returned-byte observation, copied
placement, simultaneous or stale Region failures, active-drop rejection, safe lexical lowering,
stable representation, and the public `std::alloc` API remain outside the Windows slice.
ADR 0087 adds exact CStr validation, String copy/view/append/drop, and CString from-text/view/drop
linkage. Private 16-byte views, 32-byte nested owned records, 8-byte result returns, the
owned-text-to-SystemHeap/Buffer/text-view closure, deterministic single emission, length/status/
pointer observations, and normalized drops are prototype compatibility data. Independent output-
byte and terminator observation, invalid-input and allocation-failure rollback, safe ownership,
source `Result` mapping, stable layouts, and the public `std::text` API remain outside the Windows
slice.
ADR 0088 adds `experimental run-target` over the same `x86_64-windows-msvc` compilation contract.
Its required target/profile flags, absence of an output path, target-labelled native-result receipt,
canonical product-trap mapping, zero-output rule, pre-execution PE/unwind validation, and temporary-
artifact cleanup are unstable CLI and validation policy, not a stable language or target ABI. The
command adds direct local evidence without changing `emittable`, `conformant`, or `supported`.
ADR 0091 adds the unstable `experimental exec-target` process contract and exact Windows-only
`fy_std_windows_stdout_write_all_v0` symbol. Required target/profile flags, separated bounded
streams, preserved 32-bit program status, the `{ status: u32, written: usize }` transport, status
meanings 0 through 4, and target-specific admission are prototype compatibility data. They do not
stabilize a public `std::io` API, console encoding, installation, or target maturity.
ADR 0117 adds `console` to the raw declaration, checked safe wrapper, and example entry contract.
Those annotations are language-level call-acceptance data, while the helper name, boolean result,
and example module paths remain experimental and do not become a standard-library compatibility
promise.
ADR 0092 adds `.len` to safe borrows of `str` and `[u8]`. The property name, `usize` result, byte-count
meaning, metadata-only behavior, and absence of scanning, allocation, calls, and traps are pre-1
language semantics. This does not stabilize private view layout, indexing, slicing, Unicode
counting, `CStr` operations, or a public string ABI.
ADR 0104 adds `.is_empty` to the same views as the exact zero-length `bool`, preserving the same
metadata-only and no-scan/no-call contract without stabilizing private representation.
ADR 0105 adds mandatory-bound, end-exclusive `&[u8]` slicing. The result type, left-to-right
single evaluation, `start <= end <= len` rule, `bounds` trap, and borrow-origin preservation are
pre-1 semantics; omitted bounds, text slicing, and private C layout remain unstabilized.
ADR 0106 adds `assert(condition);` as an always-checked statement. Exact `bool` input,
single evaluation, profile-independent execution, and the `assertion-failed` category are pre-1
semantics; messages, debug-only checks, compile-time assertions, and unsafe assumptions remain
separate and unstabilized.
ADR 0107 adds omitted start and end spellings to immutable byte-view slicing. Zero and retained
receiver length normalization, written-bound evaluation order, the unchanged checked MIR shape,
and the `bounds` outcome are pre-1 semantics; inclusive ranges, text slicing, and private C layout
remain unstabilized.
ADR 0108 assigns `E1013` to a well-formed integer literal outside its selected implemented integer
type. Radix-independent mathematical comparison, complete-literal spans, pre-MIR rejection, and the
four Stage 0 limits are pre-1 semantics; signed-literal folding, suffixes, wider integers, and C
scalar immediates remain unstabilized.
ADR 0109 admits the exact `i32` minimum magnitude only beneath direct unary minus. Equivalent radix
spellings, normalized literal HIR, unchanged runtime negation overflow, and `E1013` for larger or
independently grouped magnitudes are pre-1 semantics; broader constant folding remains unstabilized.
ADR 0110 admits fixed arrays over five copy-scalar types, nonempty list construction, internal
value transport, and checked read-only `usize` indexing. ADR 0111 adds exact `usize` `.len` while
preserving receiver evaluation and forbidding storage-based reflection. ADR 0112 adds the
contextual direct shared borrow from `[u8; N]` to immutable `&[u8]`, retaining the local origin and
whole-array loan. Element order, length, `bounds`, and this contextual conversion are pre-1
semantics. The current `1..=4096` element admission ceiling and private C wrappers are provisional
implementation guards rather than compatibility promises; empty/repeated literals, fields,
nesting, mutation, general slicing, and ABI use remain unstabilized or excluded. Exact whole-array
safe-borrow parameters are pre-1 internal-call semantics and do not stabilize a C ABI.
ADR 0125 adds the exact contextual mutable pair from a mutable `[u8; N]` place to `&mut [u8]`,
retaining one exclusive origin and canonical length through internal calls. The private writable C
descriptor and matching `*mut u8` extraction are pre-1 transport contracts. Safe mutable indexing,
assignment, slicing, resizing, temporary borrowing, retained foreign pointers, and a public slice
ABI remain excluded or unstabilized.
ADR 0126 adds checked mutable scalar-array and `&mut [u8]` indexed assignment, mutable-view reads,
bounds-before-value evaluation order, and the private `array-index-store`/`view-index-store` MIR
forms. These are pre-1 source and internal-IR additions; mutable slicing, public slice layout, foreign
safe references, and stable MIR serialization remain absent.
ADR 0127 adds the provisional `experimental check` spelling, explicit primary-plus-source closure,
and one human verified-MIR count line. These are prototype CLI contracts only. They do not stabilize
the future package-aware `fairyc check`, configuration, target selection, JSON, exit codes, or
function-count meaning.
ADR 0113 admits branch-aware safe-borrow reassignment with right-hand-side-first replacement,
loan-generation preservation across aliases, reachable `if`/`match` origin joins, and path-aware
loan liveness. Those rules and fail-closed multi-origin returns are pre-1 semantics. ADR 0114 adds compiler-internal fixed-point return-origin
summaries for exact formal borrow parameters, region parameters, and static sources. Positional
remapping, unselected call-bounded arguments, and fail-closed unknown recursive results are pre-1
semantics; the summary representation is not a public function-type or ABI promise.
ADR 0115 extends safe-borrow reassignment through `while` with canonical static loan-site IDs,
entry/fallthrough/`continue` header joins, separate `break` exits, and a zero-iteration path. Those
rules are pre-1 semantics; scalar predicate correlation and at-least-once loop proofs remain
unstabilized.
ADR 0093 admits exact `&str` and `&[u8]` parameters on ordinary internal functions. Exact source
types, non-owning borrow behavior, matching call operands, and the absence of implicit raw-pointer
conversion are pre-1 language semantics. Interpreter values and generated-C by-value descriptors
remain private implementation transport; foreign safe-borrow signatures, exported ABI, retained
borrows, and public `std::io` remain excluded.
ADR 0094 adds repeatable `--source <file.fy>` to the three unstable Windows target commands. The
positional entry rule, explicit-only closure, 64-file ceiling, per-file and aggregate limits,
duplicate/path/module rejection, and lack of discovery are unstable CLI and validation policy.
They do not define a package manifest, lockfile, module search path, dependency solver, registry,
foreign companion input, linker interface, or supported distribution contract.
ADR 0095 applies the same explicit-only closure to unstable `experimental run`. It preserves the
interpreter subset and result/trap contract while adding no backend, package, discovery, cache, I/O,
or host-process compatibility promise.
ADR 0096 applies the closure to `emit-c`, Linux `build`, and target-neutral Windows `build-host` and
`run-host`. The repeated flag admits only FY files; compiler, linker, foreign-input, target, output,
and artifact contracts remain unchanged and unstable.

`fairyc.layout-fixtures/v0` separately versions the repository input and expectation vocabulary for
the target-layout engine; each manifest binds one declared target. It is not a public
type-description format, native ABI profile, cache
artifact, or C probe result. Adding a layout family or fixture-visible failure code requires a new
fixture-contract decision; changing target sizes, alignments, field placement, or classifier
meaning additionally requires the applicable target-schema or ABI-profile decision.
Native probes derived from a manifest are versioned repository evidence consumers, not new target
identities. A local compiler pass does not promote maturity without the declared coverage set and
hosted receipt.

`fairyc.c-abi-fixtures/v0` versions fixed-signature type inputs and target-bound class, register,
stack, indirect, and hidden-return plans. The Linux manifest records SysV AMD64 eightbyte and
register-family behavior; the Windows manifest records Microsoft x64 ordinal slots and aggregate
indirection. The Windows pass subset additionally owns the inputs for the same-version local MSVC
caller/callee probe; changing that subset or its initialization/check semantics requires an
applicable fixture and probe-contract decision. It reuses the layout fixture's recursive type vocabulary
but is not a public binding format, stable native Fairy ABI, object contract, or hosted conformance
receipt. Changing class merge, exhaustion rollback, hidden-result, or register identities requires
a new fixture-contract and applicable target/ABI-profile decision.

The PE executable inspector is internal validation policy rather than a serialized compatibility
format. Tightening its admitted image structure requires synchronized synthetic rejection tests and
Windows smoke evidence, but does not create a stable PE, COFF object, debug, or unwind contract.
The raw COFF object inspector follows the same internal-policy rule. Its current section, symbol,
and relocation bounds are validation mechanics; no object byte identity, symbol ordering, COMDAT
selection, relocation-type set, or serialized receipt is a compatibility promise. Its 4,096-section
and 64 MiB object budgets bound parser work while admitting current MSVC function-level section
fanout; the synthetic 154-section regression case is evidence, not a stable output count.
Microsoft x64 unwind inspection is likewise an internal validation policy. Neither the observed
runtime-function count nor version-1/version-2 mix is stable. Future semantic epilog, handler,
or chain fixtures require an explicit unwind-contract decision before they can become target
evidence. ADR 0081 admits one closed native exception and virtual-walk probe as internal target
evidence; its case IDs, process result, and required ordering are test contracts, not a public Fairy
C exception ABI or serialized compatibility format.

## Diagnostics

The diagnostic catalog versions rule identities. Once a code appears in a released compiler, it is not reused for another rule. Wording, related notes, and repair hints may improve without changing the code; severity or rule meaning changes require a new compatibility decision.

Machine-readable diagnostic JSON has its own schema. Stable fields are code, severity, primary span, and rule identity. Exact prose and layout are not stable machine contracts.

The prototype in-memory span carries a build-local package discriminator separately from the
relative source path. That numeric value and compiler-private qualified entity spelling are not
stable diagnostic JSON fields, public package identity, or cross-build join keys. A future package-
aware diagnostic schema must use an explicitly versioned package reference and display policy.

Semantic, resource, Region, Result, MIR, MIR-verifier, and package fixture manifests are repository test contracts, not the released diagnostic JSON schema, a stable MIR exchange format, or a public package-manager format. Their related-span roles and canonical fact shapes may change only through a new fixture schema version plus synchronized cases. `fairyc.region-fixtures/v1` adds an optional resolved-package-graph reference while preserving v0 span and expectation shapes; v0 remains readable historical input.

The native differential report is also a repository evidence contract rather than released compiler
output. `fairyc.native-c-report/v2` supersedes v1 by adding the required
`dropTraceExecutions` count. V3 supersedes v2 by adding the required sorted `layoutCases` identity
list and `layoutProbeExecutions` count. V4 supersedes v3 by adding the exact `sanitizer` mode
identity and `sanitizerExecutions` count; the existing `cases` list owns sanitizer case identity.
V5 supersedes v4 by adding the exact sorted `abiCases`, exact `abiDirections`, and
`abiProbeExecutions` count. V6 supersedes v5 by adding `systemHeapExecutions` for the closed
Clang/GCC low/optimized provisional standard-library allocation probe. V7 supersedes v6 by adding
`bufferExecutions` for the failure-atomic overlap-safe Buffer probe. V8 supersedes v7 by adding
`arenaExecutions` for the single-active-Region Arena probe. V9 supersedes v8 by adding
`textViewExecutions` for canonical UTF-8 validation and boundary slicing. V10 supersedes v9 by
adding `ownedTextExecutions` for String ownership and CString termination. V11 adds
`linuxFileExecutions` for partial I/O and consuming close. V12 adds `foreignCallExecutions` for the
closed exact-name scalar Fairy-source caller to hand-written C callee crossings. V13 adds
`foreignPointerCallExecutions` for the typed-null raw-pointer plus `usize`/`u8` crossing. The runner
V14 adds `textValidationSourceCallExecutions` for a generated Fairy caller linked against the
actual provisional text-view C unit. V15 adds `textViewOutputSourceCallExecutions` for a generated
Fairy caller that passes a verified raw address to a flat `@repr(C)` output record, observes the
record mutation, and links the same real text-view C unit. ADR 0051 later strengthens the existing
source probes without changing report shape or counters: their Fairy declaration names now differ
from their emitted external `@link_name` symbols. V16 adds `foreignCIntCallExecutions` for the
exact `c::int` producer-result to typed-local to consumer-argument crossing. V17 adds
`foreignCSizeTCallExecutions` for the corresponding exact `<stddef.h>` `size_t` crossing and adds
target `c.size_t` to the independently versioned target data. V18 adds
`foreignCSSizeTCallExecutions`, target `c.ssize_t`, and optional scalar `header`/`availability`
metadata for the negative-preserving POSIX.1-2008 crossing. V19 adds
`foreignCLongCallExecutions` for the exact negative C `long` crossing without changing TargetSpec
or layout-fixture shape. V20 adds `foreignCScalarFamilyCallExecutions` for one generated Fairy caller
and one hand-written C companion crossing the remaining twelve exact TargetSpec C scalars in both
compiler directions and profiles. V21 adds `systemHeapSourceCallExecutions` for the generated Fairy
caller crossing the exact `void *`/`size_t` block into the real SystemHeap allocate and deallocate
helpers in both compiler directions and profiles. V22 adds `bufferSourceCallExecutions` for a
generated Fairy caller crossing the exact nested SystemHeap-block plus `size_t` Buffer record into
the real initialize, zeroed-resize, and drop helpers in both compiler directions and profiles. V23
adds `arenaSourceCallExecutions` for exact self-linked node, Arena, and Region records crossing real
initialization, begin, zeroed allocation, reset, and drop in both directions and profiles. V24 adds
`cStrValidationSourceCallExecutions` for an exact generated CStr view crossing the real bounded
validator with valid, interior-NUL, and unterminated storage in both directions and profiles. V25
adds `foreignCStructResultCallExecutions` for one generated Fairy caller receiving both an 8-byte
register result and a 20-byte hidden-memory result from C in both directions and profiles. V26 adds
`ownedTextSourceCallExecutions` for exact generated String/CString and result records crossing the
real owned-text lifecycle helpers in both directions and profiles. V27 adds
`linuxFileSourceCallExecutions` for exact generated result, File, and open-option records crossing
the real Linux open, write-all, sync, consuming-close, partial-read, EOF, and final-close lifecycle
in both directions and profiles. V28 adds `standardLibraryBuildSmokeExecutions` for an ordinary
product build that resolves a self-contained Fairy UTF-8 validator call through the shared
used-symbol standard-library contract under both compilers and profiles. V29 adds
`cStringLiteralSourceCallExecutions` for a generated Fairy caller whose compiler-owned terminated
storage is read by C and `cStringLiteralBuildSmokeExecutions` for the ordinary product-build path
carrying nonempty static `&CStr` storage, both under both compilers and profiles. V30 adds
`byteStringLiteralSourceCallExecutions` for a generated Fairy caller whose private `&[u8]` view
retains exact bytes and length before C reads its data, plus `byteStringLiteralBuildSmokeExecutions`
for the ordinary product-build path carrying that view under both compilers and profiles. V31 adds
`utf8StringLiteralSourceCallExecutions` for a generated Fairy caller whose distinct private `&str`
view retains exact valid UTF-8 bytes and byte length before C reads its data, plus
`utf8StringLiteralBuildSmokeExecutions` for the ordinary product-build path carrying that text view
under both compilers and profiles. The
runner emits one exact version and does not pretend the report shapes are interchangeable.

## Runtime and standard library

Pre-1 runtime ABI uses an exact match. Runtime helper symbols carry an ABI-major suffix. The driver diagnoses an incompatibility before invoking the linker.

The standard library has an independent version and declares required language revision, runtime ABI, target profile, and package instance. Language and standard-library numbers do not move in lockstep.

## Package formats

Manifest schema, lockfile schema, source archive schema, and any future binary artifact schema are independent. Stage 0 needs manifest and lockfile schemas only. A stable binary package format would falsely imply a stable native Fairy ABI and is deferred.

`fairyc.package-fixtures/v0` versions repository input and expectation data for an already-resolved graph. It is not the public manifest schema, lockfile schema, solver protocol, or concrete `PackageInstanceId` encoding, and compatibility with it does not imply compatibility with any future package-manager format.

## `.fymap` compatibility

The map header records schema, compiler build, language revision, package UID, package instance, source-graph digest, target scope, feature flags, `SymbolId` algorithm, and interface-hash algorithm.

Before schema 1, readers require an exact match. A future major or minor policy may skip unknown optional sections, but every skipped semantic section lowers relation-specific completeness. A source digest mismatch or corrupt checksum rejects the map. Atlas failure never fails compilation.

Dependency handling is relation-specific:

| Condition | Atlas behavior |
| --- | --- |
| Missing dependency map | Local facts remain; transitive relations are partial. |
| Newer optional schema in same compatible family | Known sections may be used; affected relations are partial. |
| Incompatible schema major | Dependency semantic facts are unavailable. |
| Incompatible language semantics | Names and spans may remain; ownership and effects are unavailable. |
| Incompatible `SymbolId` algorithm | Package-local display remains; cross-package joins are unavailable. |
| Target mismatch | Target-independent facts remain; layout and ABI facts are unavailable. |
| Stale source or corrupt map | Entire map is rejected. |

## `SymbolId`

Canonical identity includes package UID, module path, declaration kind, canonical declaration path, generic arity when it later exists, and a canonical signature fingerprint. It excludes absolute paths, source positions, compiler release, body digest, and language revision. Language compatibility is checked by the map header; a canonical type encoding change creates a new `SymbolId` algorithm.

Graph edges identify a package instance plus symbol. `PackageUid` names the long-lived project; `PackageInstanceId` distinguishes resolved source, version, and feature instances.

The provisional representation uses a compact 128-bit visible identifier plus a stored 256-bit full identity digest. Different full digests under the same visible identifier are a hard collision error. Random salt is forbidden because it would destroy deterministic identity.

Different `SymbolId` algorithms do not exact-join. Future cross-version matching may use a stable anchor, canonical declaration comparison, or explicitly heuristic similarity, never silent equality.

## Deferred stability

Language 1.0 compatibility, stable native ABI and mangling, stable MIR exchange, binary package ABI, stable `SymbolId`, stable anchors, exact diagnostic-span guarantees, stable debugger formats, and cross-version `.fymap` migration are not Stage 0 promises.
