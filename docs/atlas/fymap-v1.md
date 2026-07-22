# `.fymap` v1 Logical Schema

- Status: Canonical transport and contained current-source-validated store implemented; commands and queries remain future work
- Product source: `docs/product/02-spec.md`
- Decision source: `docs/adr/0004-atlas-semantic-snapshot-boundary.md`
- Snapshot source: `docs/atlas/semantic-snapshot-v0.md`
- Logical-plan source: `docs/atlas/fymap-logical-plan-v1.md`
- Transport source: `docs/atlas/fymap-transport-v1.md`
- Store source: `docs/atlas/fymap-store-v1.md`

## Boundary

Analyzed typed HIR and verified MIR now produce the immutable in-memory `SemanticSnapshot` v0.
`internal/atlasmap` validates the complete package/interface set and converts that snapshot into one
immutable logical partition per package. It does not reparse source or feed decisions back into type
checking. The logical plan itself does not choose transport, while ADR 0186 separately fixes the
bounded canonical JSON encoding and checksum. ADR 0187 separately fixes an internal opt-in
contained store and current-source acceptance without choosing a default path or command. The separate
`fairyc.symbol-id/v1-sha256-128+256` implementation now fixes the provisional declaration join key
described in `docs/atlas/symbol-id-v1.md`.

The first implementation plans, encodes, and optionally publishes one deterministic map per package.
Chunking, compression, and symbol-level incremental rewrites require measured evidence. Transport
v1 is compact canonical JSON with a domain-separated SHA-256 checksum; another encoding requires a
new explicit identifier. Package-instance references use the independent `PackageInstanceId` v1
contract. The logical header records every identity algorithm independently rather than inheriting
them from the map schema.

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

The implemented `fairyc.fymap-logical/v1` header contains:

- logical schema and semantic snapshot model;
- language revision;
- exact compiler executable identity and its algorithm;
- package UID, instance, coordinate, source digest, source kind, and runtime ABI;
- the explicit `target-independent` scope and an empty target value;
- package-instance, package-coordinate, package-source, source-file, line-map, SymbolId, and
  interface-hash algorithms;
- the exact package public-interface hash, deterministic features, and direct package instances;
- canonical UID/instance/coordinate namespace references for self and direct dependencies;
- logical record counts;
- generation completeness and omission reasons.

Transport v1 has no section directory: its exact JSON object contains header, file, module, symbol,
declaration, function, and completeness arrays. A future target-dependent plan must use a new
explicit target scope and value rather than silently reinterpreting this header.

Absolute local paths, timestamps that break reproducibility, source bodies, credentials, and machine-specific cache paths are excluded.

## Records

### File record

A file record contains a package-relative normalized path, content digest, source kind, and line-map
identity. The implemented snapshot prerequisite uses `fairyc.source-file/v1` and
`fairyc.source-line-map/v1` as defined in `docs/package/source-file-identities-v1.md`. Source text is
not duplicated into `.fymap`; the store reader recomputes current package and per-file identities
through the validated workspace boundary before returning stored spans.

### Type record

A type record contains a stable type identity, kind, normalized structure, copy and drop properties, safe-borrow presence, target dependence, and references to related symbol identities. Private representation is included only when a query needs it and the export policy permits it.

### Symbol record

A symbol record contains `SymbolId`, kind, name, declaration span, visibility, normalized signature, owning module, type reference, unsafe or foreign markers, declared effects, and optional documentation reference. Free text is not stored in the compiler-fact field.

### Function summary

A function summary contains parameter and return type references, borrow-return origin class, resource moves, cleanup obligations, region use, possible `Result` errors, trap categories, allocation sites, unsafe sites, foreign calls, direct callees, indirect-call summaries, and related tests known from compiler facts.

### Edge record

An edge contains source symbol, relation kind, target description, certainty, source span, provenance, and optional reason for uncertainty.

Initial relation kinds may include `Calls`, `Reads`, `Writes`, `BorrowsShared`, `BorrowsMutable`, `Moves`, `Drops`, `Allocates`, `ReturnsError`, `MayTrap`, `UsesUnsafe`, `CallsForeign`, `Implements`, and `TestCalls`. Transport v1 encodes only the relations currently present in function summaries and the completeness ledger; adding a record kind requires an explicit schema decision.

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

Transport v1 readers require exact schema, encoding, checksum, logical-model, and identity-algorithm
matches and reject unknown fields. A future compatible reader may introduce a separately versioned
policy, but v1 never skips an unknown section. A different version is not parsed as semantic
evidence. Atlas incompatibility never fails compilation.

## Explicit exclusions

The v1 design excludes embeddings, model calls, vector databases, LLM summaries stored as compiler facts, source-body duplication, eager transitive closure, full points-to analysis, stable anchor syntax, structural patching, LSP or SCIP compatibility promises, MCP execution, agent-permission annotations, and automatic rename certainty.
