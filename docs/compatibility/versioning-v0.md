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

Grammar spelling, keyword status, literal cooking, precedence, module-path identity, recovery acceptance, resolution, canonical typing, resource and Region Flow rejection, Result handling, and whether a diagnostic blocks HIR are language-revision semantics. Parser-, semantic-, resource-, Region-, and Result-fixture schemas version their expectation-file shapes; they do not make two different language revisions source-compatible.

## MIR and cache

Internal MIR accepts an exact version only. It is mutable compiler state, not a plugin or interchange ABI. Serialized MIR and caches are regenerated from source rather than migrated. The source-based MIR fixture manifest versions repository observations separately and deliberately excludes serialized graph identity. A future stable backend exchange format, if needed, is a separate contract.

`fairyc.mir-verifier-fixtures/v0` versions named seed vocabulary and mutation recipes. It does not serialize arbitrary MIR, stabilize compiler-local IDs, authorize external graph input, or make old MIR migratable.

A cache key includes the full compiler build identity, language and MIR versions, target schema and ABI profile, toolchain fingerprint, backend and profile flags, runtime ABI, standard-library instance, source digest, dependency interface digests, feature set, and interface-hash algorithm. Incompatibility is a cache miss followed by lazy collection.

## Target and ABI

Target schema versions the structure of target descriptions. ABI profile versions the meaning of layout, calling convention, return classification, and target-visible representation. Fixing an implementation to match an unchanged profile changes the compiler build and invalidates cache; changing the specified meaning creates a new ABI profile major.

Local SDK or compiler paths never participate in target semantic identity. Toolchain family, baseline, and capability fingerprint do.

The prototype `fairyc.target-spec/v0` object is the first executable target-schema member. Its
identity, scalar and pointer layouts, stack rules, classifier identities, object and symbol model,
unwind/debug contract, and C-layout feature vocabulary are compatibility data. Its maturity
evidence is current capability evidence, not an ABI promise; promoting a status requires the named
fixture and hosted receipts rather than only editing the booleans. The experimental CLI schema at
`spec/cli/experimental-target-spec-v0.schema.json` must remain synchronized with the Go model.

`fairyc.layout-fixtures/v0` separately versions the repository input and expectation vocabulary for
the target-layout engine. It is not a public type-description format, native ABI profile, cache
artifact, or C probe result. Adding a layout family or fixture-visible failure code requires a new
fixture-contract decision; changing target sizes, alignments, field placement, or classifier
meaning additionally requires the applicable target-schema or ABI-profile decision.

`fairyc.c-abi-fixtures/v0` versions fixed-signature type inputs and expected SysV AMD64 class,
register, stack, and hidden-return plans. It reuses the layout fixture's recursive type vocabulary
but is not a public binding format, stable native Fairy ABI, object contract, or hosted conformance
receipt. Changing class merge, exhaustion rollback, hidden-result, or register identities requires
a new fixture-contract and applicable target/ABI-profile decision.

## Diagnostics

The diagnostic catalog versions rule identities. Once a code appears in a released compiler, it is not reused for another rule. Wording, related notes, and repair hints may improve without changing the code; severity or rule meaning changes require a new compatibility decision.

Machine-readable diagnostic JSON has its own schema. Stable fields are code, severity, primary span, and rule identity. Exact prose and layout are not stable machine contracts.

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
the real initialize, zeroed-resize, and drop helpers in both compiler directions and profiles. The
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
