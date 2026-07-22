# Bounded current `.fymap` queries

- Status: Accepted
- Date: 2026-07-23
- Product source: `docs/product/02-spec.md`
- Query contract: `docs/atlas/fymap-query-v1.md`
- Store prerequisite: `docs/adr/0187-contained-fymap-store-and-current-source-validation.md`

## Context

Persisted maps are not useful to humans, IDEs, or LLM-facing tools until facts can be selected
without loading an unbounded graph or pretending incomplete dependency evidence is complete. A
query constructor that validates only transport bytes would also reopen the stale-span hole closed
by ADR 0187: a caller could decode an old map directly and bypass current workspace checks.

The first query slice needs a narrow semantic surface before a Context Pack or CLI freezes an output
schema. It must expose exact compiler facts, not import source text or documentation, and must make
record, relation, depth, and byte truncation visible.

## Decision

1. Add an internal immutable `QueryIndex` for exactly one package partition. Construction repeats
   complete transport validation and requires a valid `LoadExpectation`; source, file, line-map,
   compiler, and interface incompatibility rejects the index.
2. Accept only canonical visible or full SymbolId v1 selectors. Do not add name search, fuzzy match,
   path guessing, or rename inference.
3. Implement `overview`, `show`, and `around`. `around` is deterministic breadth-first traversal over
   relations selected by an explicit incoming, outgoing, or bidirectional request.
4. Expand every stored function-summary fact into one typed relation record. Only a local target
   SymbolId may extend traversal. Keep unresolved dependency targets as exact declaration
   descriptors and retain foreign ABI, link name, certainty, spans, provenance, and terminal fact
   detail where the snapshot contains them.
5. Require positive record and selection-byte limits plus nonnegative relation and depth limits.
   Count relations inside the total record budget as well as the relation budget. Treat the subject
   and current-map context as indivisible; fail if they do not fit.
6. Define selection bytes as compact JSON bytes per selected logical record plus one separator byte.
   Keep this accounting internal and explicitly separate from a future stable transport or model
   token count.
7. Stop at the first record that exceeds a deterministic selection bound. Report stable omissions
   for counts, bytes, depth, filters, unjoined dependency targets, and unavailable cross-package
   incoming edges. Never describe a bounded result as a complete workspace graph.
8. Keep the query layer read-only and downstream from compilation. It performs no source read,
   dependency discovery, process, network, filesystem, compiler, or authority-bearing action.

## Consequences

- A map already returned by `MapStore.LoadCurrent` is checked again when indexed. The repeated
  comparison is cheap relative to decoding and prevents two public internal entry paths with
  different freshness guarantees.
- `show` can report exact local incoming calls, but incoming calls from other packages are unknown.
  `around` can expose a dependency target descriptor without inventing the dependency SymbolId.
- Selection-byte accounting is deterministic and testable but not a public wire-compatibility
  promise. Context Pack encoding can later add its own versioned envelope without silently changing
  this query policy.
- No command, source excerpt, documentation field, dependency join, impact claim, or structural edit
  is enabled by this decision.

## Evidence

Tests build a real three-function Fairy C workspace, index its current partition, compare visible and
full SymbolId selection, require deterministic overview/show/around results, traverse a local call
graph, prove deep output detachment, enforce record/relation/depth/byte bounds, expose unjoined
dependency calls, reject stale compiler/source expectations, and fail closed on malformed maps,
selectors, directions, filters, relations, and budgets.
