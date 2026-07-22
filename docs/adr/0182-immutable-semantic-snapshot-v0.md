# ADR 0182: Implement the immutable SemanticSnapshot v0 boundary

- Status: Accepted
- Date: 2026-07-23
- Product source: `docs/product/02-spec.md`
- Atlas source: `docs/atlas/semantic-snapshot-v0.md`
- Prior boundary: `docs/adr/0004-atlas-semantic-snapshot-boundary.md`
- Implementation: `internal/semanticsnapshot/`

## Context

Phase 4 requires compiler facts to cross into Atlas without making Atlas part of compilation. The
repository already has package instance, package coordinate, source digest, typed HIR, semantic
borrow facts, verified MIR, and explicit call/trap/ownership facts. It has no executable
`SemanticSnapshot`, `.fymap` encoding, or query implementation.

Jumping directly to a serialized map would force unresolved `SymbolId`, checksum, encoding, and
compatibility choices to compensate for an untested in-memory fact model. Exposing HIR or MIR
directly would leak disposable package/symbol/local IDs and let later queries depend on mutable
compiler representation.

## Decision

Implement `fairyc.semantic-snapshot/v0` as an in-memory, immutable, path-free compiler-fact model.

- Construction requires semantic completion through Flow, no source diagnostics or unsupported
  forms, independently reverified MIR, and complete exact package metadata.
- Build-local package and symbol numbers are used only to join producer facts during construction.
  Output uses `PackageInstanceId`, `PackageCoordinateId`, `fysrc1-sha256`, source-visible module and
  declaration coordinates, and package-relative spans.
- ADR 0184 extends the same model with exact `fyfile1-sha256` content and `fylm1-sha256` line-map
  records supplied by the validated workspace. Every file and typed module match one-to-one.
- Structured type references retain builtin, nominal, borrow, pointer, fixed-array, Option, and
  Result shape without exposing private `$pN` qualification or choosing a public type encoding.
- The model retains declared packages, files, modules, aggregate/resource/function/foreign declarations,
  signatures, visibility, effects, ABI/link names, direct calls, traps, unsafe sites, moves, drops,
  Region placements, Result propagation, and borrow sites.
- Compiler-created cleanup functions do not become source declarations. Their resource declaration
  retains cleanup kind and declared effects.
- Foreign calls are recorded as opaque external targets. They make call completeness partial but
  do not degrade unrelated local relation classes.
- `reads`, `writes`, and `test-calls` remain explicitly unavailable rather than guessed.
- All internal storage is detached, and every public accessor deep-copies nested mutable data.
  Stable identity and source coordinates own ordering; input order and Go map order do not.
- The model contains no source text or free-form documentation. Its provenance vocabulary is only
  `compiler-declared` and `compiler-observed`.

The exact-snapshot declaration key is not `SymbolId`. This ADR does not select an encoded map,
checksum, identity hash, command, file location, or query grammar.

## Evidence

- A two-package source closure is compiled twice with different build-local package numbers and
  produces identical package, file, module, declaration, function, and completeness records.
- The same case proves exact cross-package calls plus assertion trap identity without `$pN` or host
  path leakage.
- Mutation of compiler inputs and detached accessor results cannot mutate the built snapshot.
- A foreign C call is retained with ABI/link identity and opaque certainty while only call
  completeness becomes partial.
- Incomplete semantic input, incomplete package/file metadata, a non-bijective file/module set, and
  a mutated unverified call graph fail before publishing a snapshot.

## Consequences

Later Atlas slices can design `.fymap` against tested semantic, file-integrity, and SymbolId seams
instead of serializing compiler internals. They must still add encoded-schema validation,
current-source recomputation, stale-source rejection, dependency-map completeness, and proof that
exporter failure cannot affect `fairyc check`.
