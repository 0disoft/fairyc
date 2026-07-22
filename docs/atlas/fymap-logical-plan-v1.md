# `.fymap` logical package plan v1

- Status: Implemented internal pre-transport contract
- Model: `fairyc.fymap-logical/v1`
- Decision: `docs/adr/0185-fymap-logical-package-plan-v1.md`
- Implementation: `internal/atlasmap/`
- Input model: `docs/atlas/semantic-snapshot-v0.md`

## Purpose

The logical plan proves that a successful workspace analysis can be divided into complete,
deterministic package-owned inputs for future `.fymap` files. It deliberately stops before choosing
transport bytes. This keeps semantic ownership and compatibility review independent from codec,
storage, and query decisions.

`PrepareWorkspace` accepts one already validated workspace, its matching compiler result, and an
exact caller-provided compiler executable identity. It computes the complete public-interface-hash
set, constructs the immutable semantic snapshot, then builds package partitions. It performs no
source discovery, source reopening, process execution, network access, or map publication.

## Complete input rule

Planning succeeds only when:

- the snapshot model is exactly `fairyc.semantic-snapshot/v0`;
- the compiler build identity is nonzero and uses the exact executable identity contract;
- at least one package exists;
- every package instance, coordinate, and source digest has its canonical parseable form;
- every package has exactly one nonzero public-interface hash and no extra hash exists;
- every declared dependency names another package in the same plan;
- every file content and line-map identity parses canonically;
- every declaration has exactly one valid `SymbolId` pair; and
- relation completeness is canonically ordered and internally consistent.

Any violation rejects the entire plan. A planner never drops a bad package and emits a deceptively
complete remainder.

## Header

Each package header records these independent compatibility dimensions:

| Group | Fields |
| --- | --- |
| Models | logical schema, semantic snapshot model |
| Compiler | exact build identity and build-identity algorithm |
| Language | language revision |
| Package | UID, instance, coordinate, source digest, source kind, runtime ABI |
| Target | `target-independent` scope and empty target |
| Algorithms | package instance, package coordinate, package source, source file, line map, SymbolId, public interface |
| Graph | exact interface hash, canonical features, direct package instances |
| Completeness | generation status, sorted explicit omissions |
| Inventory | file, module, symbol, function, relation, and completeness counts |

An algorithm field never inherits its meaning from another schema version. Changing an algorithm
does not silently reinterpret old data.

## Package partition

One partition owns detached immutable views of:

- exact source-file metadata;
- modules;
- declaration-to-visible/full-`SymbolId` bindings;
- complete detached declaration records;
- complete detached function summaries; and
- the snapshot's relation-completeness ledger.

The plan counts semantic relation observations from calls, traps, unsafe sites, moves, drops,
allocations, returned errors, and borrows. It does not yet define transport records for those facts;
the snapshot remains their semantic source of truth.

## Completeness

The header is `exact` only when every snapshot relation is exact. Otherwise it is `partial`, and
each partial or unavailable relation contributes a sorted
`<relation>:<status>:<reason>` omission. Missing information is never converted into an empty edge
set or a false claim that a relation does not exist.

## Explicitly not implemented

This model does not define:

- encoded `.fymap` bytes, byte order, section offsets, alignment, compression, or checksum;
- file names, output directories, atomic replacement, retention, or cache policy;
- current-source recomputation and stale-map rejection;
- dependency-map loading or cross-map joins;
- `overview`, `show`, `around`, Context Pack, LSP, SCIP, or MCP output;
- source text, documentation text, embeddings, or execution authority; or
- rewrite-size, rewrite-time, or chunking performance claims.

Those require independently versioned contracts and their own executable evidence.
