# Aggregate-aware `.fymap` transport v3

- Status: Implemented opt-in planner, codec, store, queries, workspace join and catalog CLI
- Logical model: `fairyc.fymap-logical/v3`
- Transport: `fairyc.fymap/v3`
- Encoding: `fairyc.fymap-json/v3`
- Decision: `docs/adr/0387-generic-aggregate-source-preparation.md`
- Implementation: `internal/atlasmap/codec_aggregate*.go`

## Version boundary

`BuildPlanV3` and `PrepareWorkspaceV3` retain generic struct/resource declarations
and concrete implementations. `EncodeV3` and `DecodeV3` are explicit entry points.
Older planners and codecs reject aggregate records; no decoder guesses or upgrades
the format. Ordinary and function-only v1/v2 encodings remain unchanged.

The existing caller-selected byte bound, strict known JSON fields, exact schema
and encoding, checksum, and byte-for-byte canonical reencoding still apply. The
checksum algorithm and digest prefix remain v1 because the checksum already binds
the full schema and encoding. It proves integrity, not compiler authorship.

## Records

A specialization contains exactly one `function` or `aggregate` record. Aggregate
records use the original declaration key, compiler-observed provenance, concrete
self type and ordered substituted fields, without template binders or generated
names. Nominal type references may contain recursively bounded type arguments.
Top-level specialization arguments retain the existing scalar-only admission.

Resource specializations with non-empty cleanup also contain a `cleanup` function
summary owned by the original resource. Empty cleanup has no executable summary.
Struct specializations cannot carry cleanup metadata. Calls retain exact helper
specialization IDs; borrow observations remain in the owning cleanup, not an
ordinary source function. Relation counts include cleanup observations, while
ordinary function counts continue to exclude all specializations.

Concrete aggregate headers must agree with the source template after substitution.
Fields must retain source names, order and contained spans. Cleanup observations
must lie within the source resource. Inconsistent kinds, ownership, provenance,
arity, field types, drop classification, call targets and counts are rejected even
when the document has a valid recomputed checksum. Existing SymbolId, source
interface, package reference and template-body uncertainty checks remain active.

## Evidence and remaining integration

Tests cover used and unused templates, nested structs, two concrete resource
specializations, empty cleanup, helper calls, borrows, detached getters, canonical
round trips, byte bounds, version isolation, unchanged function-only v2 bytes,
corruption, unknown fields and resealed contradictions.

This codec does not publish maps, read source files or create query indexes. Its
explicit store consumer and current-source reader are documented in
`docs/atlas/fymap-store-v3.md`. Local queries are documented in
`docs/atlas/fymap-query-v3.md`. Workspace joins, symbol catalog 3 and call catalog 2
now explicitly consume v3; their CLI paths use in-memory maps. Stable aggregate
source admission is not promoted.
