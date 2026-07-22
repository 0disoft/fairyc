# Provisional SymbolId v1

- Status: Accepted
- Date: 2026-07-23
- Product source: `docs/product/02-spec.md`
- Atlas source: `docs/atlas/fymap-v1.md`

## Context

ADR 0182 introduced an immutable, path-free compiler-fact snapshot but deliberately stopped before
choosing `SymbolId` or `.fymap` encoding. A map exporter cannot safely freeze records or graph joins
while its declaration identity remains unspecified. Conversely, binding identity to exact package
instances, source positions, bodies, or compiler-private numbers would make ordinary edits renumber
the graph and destroy useful semantic continuity.

## Decision

1. Implement `fairyc.symbol-id/v1-sha256-128+256` as a separate provisional Atlas identity
   algorithm before selecting `.fymap` transport.
2. Hash a tagged, length-delimited tuple of package UID, module, declaration kind, declaration path,
   generic arity, and a canonical structured signature.
3. Normalize nominal signature types from snapshot `PackageCoordinateId` values to package UID and
   logical type path. Keep exact package instance, source digest, language, compiler, and target in
   future map compatibility data rather than the symbol hash.
4. Exclude visibility, parameter names, source spans, bodies, comments, compiler build, and
   build-local IDs. Include typed shape, unsafe state, effects, foreign ABI/link name, aggregate
   shape, representation, and resource cleanup semantics.
5. Store a 128-bit visible prefix and the complete 256-bit SHA-256 digest. Require the visible bytes
   to match the full prefix and reject different full digests sharing one visible value.
6. Reject invalid UTF-8, controls, invalid qualified paths, unknown declaration kinds, empty
   signatures, malformed text forms, pair mismatches, and collisions. Never repair identity with a
   salt or build-order suffix.
7. Keep snapshot production independent: `internal/symbolidentity` consumes
   `SemanticSnapshot`; frontend, semantic analysis, HIR, MIR, and snapshot construction do not
   import SymbolId code.

## Consequences

- Body, whitespace, span, package-version, source-instance, and build-number changes retain an ID
  when the semantic declaration contract is unchanged.
- Package UID, module, declaration path or kind, and normalized signature changes produce a new ID.
- Visibility can be compared as a changing fact under one ID rather than masquerading as a rename.
- A future map stores package instance beside SymbolId for exact graph edges and records this
  algorithm independently from map schema and interface-hash versions.
- `.fymap` bytes, checksums, exporter policy, query output, dependency joins, anchors, and released
  cross-version stability remain undecided.

## Evidence

`internal/symbolidentity` tests freeze the byte encoding and digest derivation, parser strictness,
component boundaries, validation reasons, visible/full integrity, synthetic collision rejection,
snapshot-derived stability and sensitivity, package-UID nominal normalization, and canonical output
ordering. The ordinary full Go suite exercises the same compiler pipeline that creates the input
snapshot.
