# `.fymap` v1 Logical Schema

- Status: Proposed map encoding; the in-memory snapshot and provisional SymbolId producer exist
- Product source: `docs/product/02-spec.md`
- Decision source: `docs/adr/0004-atlas-semantic-snapshot-boundary.md`
- Snapshot source: `docs/atlas/semantic-snapshot-v0.md`

## Boundary

Analyzed typed HIR and verified MIR now produce the immutable in-memory `SemanticSnapshot` v0. The
future Atlas exporter converts that snapshot into `.fymap`. It does not reparse source or feed
decisions back into type checking. Snapshot implementation does not choose or imply the map's
encoding, checksum, file location, or command behavior. The separate
`fairyc.symbol-id/v1-sha256-128+256` implementation now fixes the provisional declaration join key
described in `docs/atlas/symbol-id-v1.md`.

The first implementation writes one deterministic map per package. Chunking, compression, and symbol-level incremental rewrites require measured evidence. The normative map encoding, byte order, and map-checksum algorithm remain UNDECIDED; this document fixes the logical records and trust behavior first. Package-instance references use the independent `PackageInstanceId` v1 contract. The map records the SymbolId algorithm independently rather than inheriting it from map schema.

## Identity

### `NodeId`

`NodeId` is a build-local typed arena identity. It is never serialized as a stable public identifier.

### `SymbolId`

`SymbolId` identifies a semantic declaration across deterministic rebuilds. The implemented
provisional v1 tuple includes package UID, module path, symbol kind, declaration path, generic arity,
and a canonical structured signature. Language revision belongs in the map header rather than the
identity input so a compatible revision does not renumber every declaration.

The external representation is an opaque 128-bit visible SHA-256 prefix accompanied in the map by
the 256-bit full identity digest. Exact encodings and text forms are fixed by
`docs/atlas/symbol-id-v1.md`. A visible collision with different full digests rejects the map;
random salt is forbidden. Regression coverage includes tuple boundaries, qualified-path rejection,
rename/signature sensitivity, version/body/span/build-number stability, and synthetic collision
handling.

`PackageUid` names a long-lived source project. `PackageInstanceId` identifies the resolved source, version, and feature instance in one lockfile. Cross-package edges use both package instance and symbol identity.

### `AnchorId`

`AnchorId` is reserved for a future human-maintained stable anchor. It is absent from v1 records unless a later schema revision defines syntax and uniqueness rules.

## Header

The logical header contains:

- format major and minor version;
- language revision;
- compiler identity;
- package UID, package instance, and the independent `fysrc1-sha256` package source digest;
- target identity when facts are target-dependent;
- identity-algorithm version;
- interface-hash algorithm version, currently the independent `fyif2-sha256` contract;
- deterministic feature flags;
- section directory and record counts;
- generation completeness and omission reasons.

Absolute local paths, timestamps that break reproducibility, source bodies, credentials, and machine-specific cache paths are excluded.

## Records

### File record

A file record contains a package-relative normalized path, content digest, source kind, and line-map identity. Source text is not duplicated into `.fymap`.

### Type record

A type record contains a stable type identity, kind, normalized structure, copy and drop properties, safe-borrow presence, target dependence, and references to related symbol identities. Private representation is included only when a query needs it and the export policy permits it.

### Symbol record

A symbol record contains `SymbolId`, kind, name, declaration span, visibility, normalized signature, owning module, type reference, unsafe or foreign markers, declared effects, and optional documentation reference. Free text is not stored in the compiler-fact field.

### Function summary

A function summary contains parameter and return type references, borrow-return origin class, resource moves, cleanup obligations, region use, possible `Result` errors, trap categories, allocation sites, unsafe sites, foreign calls, direct callees, indirect-call summaries, and related tests known from compiler facts.

### Edge record

An edge contains source symbol, relation kind, target description, certainty, source span, provenance, and optional reason for uncertainty.

Initial relation kinds may include `Calls`, `Reads`, `Writes`, `BorrowsShared`, `BorrowsMutable`, `Moves`, `Drops`, `Allocates`, `ReturnsError`, `MayTrap`, `UsesUnsafe`, `CallsForeign`, `Implements`, and `TestCalls`. This list is a proposed open enum until an encoded schema is implemented.

## Incomplete graph model

An edge target is one of:

- `Resolved(SymbolId)`;
- `External(package, symbol?)`;
- `CandidateSet(SymbolId...)`;
- `Unknown(reason)`.

Certainty is one of `Exact`, `Bounded`, `Partial`, or `Opaque`. Query results report relation-specific completeness and unknown-edge counts. Calls, ownership, unsafe reachability, layout, and test relations can have different statuses in the same result.

Completeness reasons include missing dependency map, unknown indirect call, newer optional schema, target mismatch, language mismatch, external FFI, incompatible schema, incompatible identity algorithm, stale source, and corruption. A stale source digest or corrupt map rejects the map. An incompatible dependency may preserve package-local names and spans while making transitive semantic joins unavailable. `.fymap` does not store eager transitive closure, points-to analysis results presented as certainty, or guessed edges for missing dependencies.

## Provenance and trust

Every record or text reference uses one provenance class:

- `CompilerObserved` for facts derived from analyzed IR;
- `CompilerDeclared` for explicit source declarations;
- `WorkspaceDocumentation` for local free text;
- `DependencyDocumentation` for dependency free text;
- `GeneratedDocumentation` for tool- or model-generated text.

Compiler facts and free text occupy separate sections. Documentation is excluded from queries by default unless requested. Documentation cannot replace a compiler fact, change completeness, or grant filesystem, process, network, credential, approval, or execution authority.

Text import performs size caps, control-character filtering, encoding validation, and source attribution. Atlas performs no URL fetch while building a map.

## Initial queries

The initial query surface is:

- `overview`: package structure, counts derived from the current map, and completeness;
- `show`: one symbol's signature, facts, relations, uncertainty, and optional attributed text;
- `around`: a bounded neighborhood with direction, relation filter, depth, and omissions.

`impact` and semantic `diff` are deferred until incomplete-edge behavior and identity stability are validated. Semantic diff eventually reports one of `Exact`, `Bounded`, `Partial`, `Heuristic`, or `Unavailable`; a heuristic rename is never applied as fact.

Before schema 1, readers require an exact schema and identity-algorithm match. Future compatible minor readers may skip an unknown optional section only while lowering every affected relation's completeness. A different major is not parsed as semantic evidence. Atlas incompatibility never fails compilation.

## Explicit exclusions

The v1 design excludes embeddings, model calls, vector databases, LLM summaries stored as compiler facts, source-body duplication, eager transitive closure, full points-to analysis, stable anchor syntax, structural patching, LSP or SCIP compatibility promises, MCP execution, agent-permission annotations, and automatic rename certainty.
