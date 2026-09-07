# ADR 0386: Generic declaration identity preparation

- Status: Accepted source identity, v2 snapshot/map/store/query/join and symbol/call/pack/text CLIs; generic Evidence Bundle pending
- Date: 2026-09-06

## Decision

Represent original generic function signatures with ordered display parameter names and positional
type-parameter references. A type parameter's identity is its zero-based binder position, not its
source spelling or one observed concrete specialization. Arity remains significant even when a
parameter is unused in the callable signature.

The existing SymbolId v1 tuple already encodes arity. FromDeclarations now derives it from the
detached declaration and validates binder references, including nested types. The existing type
encoding can encode the new `parameter` kind with the canonical decimal index in Name; other type
fields must be empty/zero. No existing non-generic identity/signature byte changes. Parameter-name,
ordinary argument-name, and visibility changes do not alter the join key; arity and binding/type
changes do.

Only ordinary module functions are admitted at this boundary. Invalid/duplicate names, unbound or
noncanonical indices, generic foreign functions, methods, and aggregate declarations fail closed.
This is an identity boundary, not independent proof that a caller-supplied declaration is a valid
source program. The source bridge must validate original headers against current compiler facts.

## Structural source-header bridge

The compiler's immutable GenericProvenance now retains original written parameter/result types
and attributes before expansion, checks original declaration header identity, and detaches all
mutable syntax nodes. GenericDeclarations converts that source-declared signature to positional
snapshot types without using observed concrete specializations. It includes unused declarations,
declared effects, scalar/builtin types, binder references, borrows, pointers, positive literal
fixed arrays, byte slices, Option, Result, grouping, and compiler-resolved nominal types. Source
spelling is not reparsed as a type.

Successful semantic analysis publishes deterministic visible nominal scope facts, including local
types and public imported types selected by the existing exact package-import resolver. They are
omitted from JSON and rejected analyses. Compiler provenance clones these facts; the Atlas adapter
replaces their build-local type qualification with PackageCoordinateId. Alias spelling and package
renumbering do not affect identity; a different provider package UID does. Private imported names
are unavailable, and no nominal lookup is inferred from source strings or concrete specializations.

This adapter does not claim that unused bodies have been checked. Unresolved nominal names,
non-literal array lengths, unsupported shapes, and malformed effects are explicit errors rather
than invented compiler facts. GenericDeclarations returns detached declarations; BuildCompiler
adds checked source-to-specialization joins and executable completeness as described below.
This identity adapter does not change stable language, SDK, runtime ABI, or stable CLI contracts.
Opt-in map, output, and experimental CLI versions are described below.

Borrow facts now retain a non-serialized build-local function discriminator. The executable
summary join uses both that identity and the original span, so two specializations sharing one
source borrow do not collide or receive each other's facts. Legacy caller-built facts without
the discriminator are accepted only when their span has exactly one owner.

## Compiler snapshot integration

BuildCompiler checks the original signature against every typed implementation after substituting
its exact positional bindings. It rejects missing provenance, orphan implementations, source-span
or signature contradictions, and missing executable summaries. Original declarations and their
documentation appear once; implementations retain separate summaries and concrete type arguments.
Calls retain the original declaration plus the exact specialization ID, including generic helper
calls. Accessors detach both type arguments and all summary slices.

The compiler marks semantic results requiring generic provenance without adding a serialized
field. The raw Build entry rejects that marker, even when all templates are unused. Ordinary
functions are not classified by a generated-looking name prefix. The workspace adapter now uses
BuildCompiler; ordinary snapshots and their existing SymbolIds remain unchanged.

The concrete specialization set has exact completeness. Original template bodies remain explicitly
unavailable as universal executable facts, whether or not a particular specialization was checked.

## V2 logical package plan

Package-aware generic analysis now retains immutable package interface facts in GenericProvenance.
Each record binds its semantic package ID and stable package coordinate to both the ordinary
interface hash with generated specializations excluded and the generic source-interface hash with
exported template closures included. PackageInterfaces returns detached records in semantic package
order. LowerAnalyzed and CheckGenericPackages preserve them only on verified results. Mutating the
public GenericAnalysis maps, the input coordinate map, or an accessor result cannot change retained
facts. Rejected analysis and coordinate-free explicit-file analysis publish no package interface
records. These compiler facts remain non-serialized and do not authorize stable cache reuse.
The v2 workspace adapter now binds the complete retained set to current semantic package IDs and
coordinates, rejects missing or contradictory facts, and records ordinary and generic interface
identities separately. CurrentWorkspaceExpectationsV2 uses the same original-source facts rather
than recomputing an interface from generated specializations. Compatibility checks require both
identities to match; callers omitting the generic expectation cannot accept a generic map.

BuildPlanV2 and PrepareWorkspaceV2 opt into `fairyc.fymap-logical/v2`. Each original declaration
and its existing SymbolId appears once. Each concrete specialization is assigned to its original
declaration's package, retains ordered type arguments and a separate executable summary, and is
counted separately from ordinary functions. Relation counts include both kinds of executable
summary. Calls retain their declaration and exact specialization target; generic body unavailability
and concrete specialization completeness remain separate ledger entries. Accessors detach type
parameter names, nested type arguments, calls, and all summary slices.

Specialization IDs are exact compiler-provenance join keys, not source SymbolIds or a new promise of
cross-build stability. The original source SymbolId remains the stable declaration join. No source
or documentation text is added to maps. The compiler build and package identity header still scopes
all executable facts. Ordinary package partitions differ only in their explicit v2 logical schema.

The legacy planner and workspace entry point retain v1 behavior. V2 plans cannot enter the v1
encoder, including a plan with only unused generic declarations. The v1 encoder also checks for
specialization payloads independently of header counts. Logical planning does not itself publish
maps. The separately opt-in v2 store now consumes the codec and current-source expectations;
package-local queries and workspace joins use separate v2 constructors. Symbol discovery has an
explicit v2 CLI path, as do Context Pack v2 and call catalog v1 through their own version flags. Other CLI
consumption still requires subsequent consumer integration.

## Transport boundary

.fymap v1 has no generic parameter fields. Its planner and encoder therefore reject generic
declarations explicitly. Silently serializing them as arity zero would erase information and make
rederived IDs contradict the original identity. The snapshot now preserves generic declarations,
specializations, and completeness together, as does the opt-in v2 logical plan.
The v1 encoder also rejects specialization calls rather than omitting their implementation target.
The explicitly versioned symbol discovery, Context Pack, and call catalog CLIs consume v2 maps; language/SDK/runtime
versions do not change.

EncodeV2 and DecodeV2 now implement the separate `fairyc.fymap/v2` transport described in
`docs/atlas/fymap-transport-v2.md`. The original Encode/Decode remain exact-v1 entry points.
Generic source declarations, positional arguments, concrete summaries, and specialization calls
round-trip with canonical bytes and checksums. Structural validation rederives source SymbolIds,
checks local implementation joins and scalar bindings, and preserves explicit template uncertainty.
Checksums detect corruption, not authorship or compiler provenance authenticity. Downstream command
consumers other than symbol discovery, Context Packs, and call catalogs still need their v2 CLI integration.

NewMapStoreV2 and NewMapStoreV2InRoot publish and load maps under the separate `fymap-v2` directory.
The existing root-relative publication, retention, cleanup, and current-source checks are shared,
with version-specific codec and checksum inputs selected when the store is opened. Version-root
links are rejected and the opened directory identity must match the inspected directory. No store
auto-migrates or searches another version's directory. See `docs/atlas/fymap-store-v2.md`.

NewQueryIndexV2 now admits exact-current v2 maps to package-local overview, catalog, show, and around
queries. Original generic nodes retain their signature but no universal Function facts; separate
specialization records carry concrete type arguments and observed relation counts. Relations retain
source and target implementation IDs, and context retains generic source-interface identity and
template-body unavailability. Existing byte/record/relation/depth budgets cover the added metadata.
Traversal remains a declaration neighborhood, not proof of a concrete execution path across
specializations. V1 queries and workspace joins do not silently admit v2 maps.
See `docs/atlas/fymap-query-v2.md`.

NewWorkspaceCallIndexV2 joins exact-current v2 partitions through the existing bounded direct-call
ledger. Joined calls preserve both implementation IDs alongside original declaration SymbolIds.
The target implementation must belong to the exact target declaration, including across packages;
a generic target without an implementation is rejected. Incoming/outgoing selection remains
declaration-based and returns separate concrete call records, not an execution-path traversal.
V1 construction remains unchanged. See `docs/atlas/workspace-call-join-v0.md`.

The internal workspace-call catalog EncodeV1 now preserves implementation endpoints in the separate
catalog v1 JSON/checksum contract. Canonical ordering includes both implementation IDs; exact record
deduplication does not collapse calls sharing a template source span. The v0 encoder rejects v2
indexes instead of erasing generic facts. Existing v0 call CLI output remains unchanged. Explicit
`atlas-calls list --catalog-version 1` connects the generic-aware workspace, v2 join, and v1 output;
its exact declaration selectors are discoverable through symbol catalog v2. Other output consumers
still require integration. See
`docs/atlas/workspace-call-catalog-v1.md`.

## Evidence

The existing `atlas-text source|documentation` commands now prepare generic-aware exact-current v2
queries. Their output remains attributed-text v0: exact original declaration spans already carry
all required identity and custody fields, without executable generic facts. Source and attached
documentation remain one record per original declaration, including unused templates, and fixed
`untrusted-text` authority. Tests cover ordinary v1/v2 byte equivalence, exact retained generic
bytes, snapshot/source mismatch, control and size omissions, both SymbolId forms, and CLI output.
No new text schema or flag is introduced. Generic Evidence Bundle composition remains separate.

Context Pack v2 encoders and `atlas-pack overview|show|around --pack-version 2` retain generic
source-interface identity, original declaration signatures, separate implementation observations,
concrete relation endpoints, and template uncertainty. V1 encoders reject v2 indexes. Default CLI
output stays v1, and the pack-version flag does not activate the shared Evidence Bundle parser.
Source/documentation arrays stay empty. See `docs/atlas/context-pack-v2.md`.

Symbol catalog EncodeV2 and `experimental atlas-symbols list --catalog-version 2` now expose
original generic declaration identities through the v2 map/query pipeline. Multiple concrete
implementations do not duplicate catalog records; unused templates remain discoverable. The
default and explicit catalog version 1 retain the existing ordinary compilation/output path.
V2 uses the shared generic-aware uncached workspace compiler, with the same exact lock, source
diagnostic, currentness, selection, and output limits. The symbol output has its own v2 checksum
domain/version inputs and rejects mixed map versions. See `docs/atlas/symbol-catalog-v2.md`.

Generic identity tests cover alpha-renaming, unused arity, changed binder positions, nested
references, malformed/out-of-range indices, and inadmissible declaration kinds. Existing ordinary
SymbolId vectors remain the compatibility oracle. The codec mutation corpus rejects generic data
instead of dropping it from v1 records.
