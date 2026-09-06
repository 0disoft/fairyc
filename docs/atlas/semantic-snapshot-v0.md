# SemanticSnapshot v0

- Status: Implemented in memory
- Decision: `docs/adr/0182-immutable-semantic-snapshot-v0.md`
- Implementation: `internal/semanticsnapshot/`
- Consumers: implemented provisional SymbolId derivation, per-package logical planning, transport codec, contained store, bounded package-local queries, compiler-fact Context Pack output, experimental query commands, and separate canonical exact retained-source/documentation attributed-text output

## Boundary

`SemanticSnapshot` is the one-way, immutable compiler-fact boundary between successful compilation
and Fairy Atlas. Its builder accepts only a semantic result completed through Flow, MIR that passes
the ordinary closed-world verifier, and an exact package metadata map. It performs no parsing,
filesystem discovery, source-text loading, cache access, backend work, toolchain resolution,
subprocess execution, network access, or map publication.

Snapshot failure is a downstream analysis failure. It cannot change source diagnostics, verified
MIR, generated code, a build artifact, or program behavior. Frontend, semantic, and MIR packages do
not import this package or an Atlas package.

## Identity removal

The detached Declaration model now has ordered TypeParameters and a binder-index TypeParameter
reference for generic signature preparation (ADR 0386). Accessors clone the parameter list.
GenericDeclarations converts compiler-retained original headers and visible nominal scope facts
to detached declarations. Binder names become positions; nominal types become package-coordinate
references. Declared effects are preserved, while unused-body validity is not claimed. These
optional in-memory facts do not change existing snapshot output or admit generic source through
Build yet: specialization joins and completeness remain pending. The v1 map planner/codec
explicitly reject nonempty TypeParameters. See ADR 0386 for the exact source-header boundary.

HIR `PackageID`, `SymbolID`, private `$pN` qualification, MIR block/local IDs, entity strings, and
absolute host paths are build-local and never appear in a snapshot. Construction replaces them
with existing independently versioned identities:

- exact package records use `PackageInstanceId` and `fysrc1-sha256`;
- file records use `fyfile1-sha256` content and `fylm1-sha256` line-map identities;
- nominal type references use source-independent `PackageCoordinateId`;
- source spans use exact package instance, package-relative `.fy` path, and half-open byte offsets;
- declarations use an exact-snapshot tuple of package instance, module, declaration kind, and
  source name.

That declaration tuple is a join key inside one snapshot, not an Atlas `SymbolId`.
`internal/symbolidentity` now converts declarations to the separately versioned provisional
`fairyc.symbol-id/v1-sha256-128+256` contract. `.fymap` transport is separately fixed by
`docs/atlas/fymap-transport-v1.md` and cannot be inferred from this model.

## Records

The v0 snapshot contains detached, canonically sorted package, file, module, declaration,
function-summary, and relation-completeness records. It also carries an in-memory-only declaration-
documentation sidecar containing exact declaration keys, declaration spans, and frontend-attached
documentation spans. Map planning deliberately does not consume that sidecar, so `.fymap` v1 and
SymbolId remain attachment-free. File records contain exact package instance,
canonical relative path, source/test kind, content digest, and byte line-map identity without source
text. Every file has exactly one typed module and every typed module has exactly one file.
The declaration set is closed to the current five Atlas families: structs, enums, resources,
ordinary functions, and foreign functions. Those records preserve source-declared visibility,
signatures, structured stable types, effects, C ABI/link names, aggregate fields and variants,
representation, and resource cleanup classification. Module constants accepted by ADR 0220 are
semantic compile-time facts and are deliberately absent from Semantic Snapshot v0, SymbolId v1,
and `.fymap` v1; no consumer may reinterpret them as locals or functions. Adding a constant
declaration kind requires a separate versioned Atlas transport decision. Compiler-created cleanup
functions remain private; their source resource owns the corresponding declared facts.

Verified function summaries currently retain:

- exact direct Fairy calls;
- opaque external C calls with their declared ABI and link name;
- trap categories;
- unsafe operation sites;
- complete-root move and executable drop sites;
- lexical Region placement sites;
- `Result` propagation sites; and
- semantic shared or mutable borrow sites and escape status.

Every fact carries either `compiler-declared` or `compiler-observed` provenance. Source bodies,
documentation bodies, ordinary comments, generated summaries, credentials, environment data, build
logs, and local paths are absent.

## Immutability and determinism

Construction deep-copies all retained slices. Accessors return new deep copies, so mutation of the
compiler input or one returned view cannot change the snapshot. Canonical ordering uses stable
package identities, module/declaration coordinates, source spans, and relation names rather than Go
map iteration or input discovery order.

Regression tests rebuild one two-package program with different disposable package numbers and
require identical records. They also mutate compiler inputs and accessor results, require the
snapshot to remain unchanged, canonicalize file input order, require complete file/module coverage,
reject unverified MIR, incomplete package maps, and overlapping documentation spans, and forbid
private package qualification or host-path leakage. Documentation-sidecar tests cover every
supported declaration family and require `.fymap` v1 bytes to remain identical when only that
in-memory sidecar is removed.

## Completeness

Completeness is relation-specific. Local direct calls, effects, traps, unsafe sites, moves, drops,
Region placement, Result propagation, and borrow facts are exact for the implemented Stage 0 MIR.
The presence of a foreign C call makes only `calls` partial and records `external-ffi`; its local
declaration and call site remain known, while behavior beyond the ABI boundary is opaque.

`reads`, `writes`, and `test-calls` are explicitly unavailable in v0. The builder does not guess
them or convert their absence into a complete graph. `test-calls` requires a later compiler-owned
test metadata handoff because the stable semantic test harness currently strips `@test` before
ordinary semantic analysis.

## Deferred

This model itself does not implement `.fymap` transport, current-source recomputation, dependency
map joins, stale-map validation, documentation text import, source excerpts, queries, Context Packs, map
rewrite measurements, anchors, `impact`, semantic diff, LSP, SCIP, MCP, embeddings, or structural
edits. Transport, current-source validation, and package-local queries are now separate downstream
Atlas boundaries; the remaining surfaces still require their own compatibility and failure contracts.
