# ADR 0378: Data-dependent contract-state transaction graphs

- Status: Accepted
- Date: 2026-09-04
- Owners: Fairy C compiler and contract-state semantic oracle

## Context

The compiler-private contract-state MIR admitted only a static operation list and one static exit.
That proved layout identity, effect derivation, atomic overlays, stable transport, and cache
reassembly, but a value read from state could not select later work. Reusing ordinary MIR control
flow would hide state-specific capability, trap, cost, and rollback requirements.

## Decision

Add a second, mutually exclusive transaction representation made of bounded acyclic SSA blocks.
Legacy straight-line programs and their `fycsbody1` hash and v1 payload bytes remain unchanged.

1. A graph has at most 64 dense source-ordered blocks and 1024 storage operations. Block zero or
   another in-range declared entry has no parameters or predecessors; every block is reachable and
   cycles are rejected.
2. Storage reads define an exact codec-typed value and a `bool/v1` found value. A missing read keeps
   the public observation absent while the internal typed result receives the codec's canonical zero;
   control must inspect the found value when absence matters.
3. Values are SSA-local. Uses must follow their definition in the same block. Cross-block values
   travel only through explicit edge arguments and same-typed block parameters, which makes joins
   and dominance independently verifiable.
4. `branch` consumes one available `bool/v1`; `jump`, `commit`, `reject`, and `trap(category)` are
   the only other terminators. Every block has exactly one terminator and all exits stay explicit.
5. Every graph operation and terminator carries canonical cost `1`. Unknown or missing costs fail
   closed; loops, recursion, calls, ordinary trapping operators, and variable-cost operations remain
   outside this graph.
6. Every storage operation retains the program's opaque capability. Required `storage_read` and
   `storage_write` effects are derived across all blocks before the oracle begins.
7. Execution uses one existing `contractstate.Execute` transaction. Branches observe snapshot or
   read-your-write values, writes stay in its private overlay, and reject, trap, limits, or runtime
   errors publish nothing.
8. Graph bodies participate in the existing package body hash using a graph-domain encoding. Their
   canonical stable envelope is `fairyc.contract-state-package-body-artifact/v2`; decode and fresh
   rebind reconstruct private capability identity and re-enter the graph verifier. Legacy bodies
   continue to encode as v1 byte-for-byte. Because cache key v4 already consumes the exact body hash,
   any block, edge, value, cost, or terminator change invalidates the owning package.
9. `manifest-v2.json` is the current transaction corpus. It preserves historical v0/v1 coverage and
   adds cell-value branch/join and map-found branch execution plus cycle, join-type, missing-cost,
   and unknown-value verifier mutations through atomic compiler publication.

## Consequences

Useful compiler-private state programs can now let cell and map observations select later writes
without weakening atomicity or importing general MIR control flow. Stable cache reassembly covers
the same graph and creates fresh authority. The extra block/value structures are intentionally
small and canonical rather than a general SSA framework.

Source state access syntax, ordinary HIR conditional lowering, loops, arbitrary computation,
verified helper calls, owner-bound state borrows, native lowering, durable providers, gas schedules,
events, block context, external calls, reentrancy, and deployable targets remain separate gates.
Language `fairyc-language/13`, SDK `0.0.0-stage0.56`, public-interface v5, package-cache v4, stable
ordinary MIR v3, and runtime ABI `fairyc-runtime/0` do not change.

## Evidence

- `internal/hir/state.go`
- `internal/contractstatemir/graph_verify.go`
- `internal/contractstatemir/graph_execute.go`
- `internal/contractstatemir/graph_test.go`
- `internal/contractstatemir/stable_payload.go`
- `spec/fixtures/contract-state/manifest-v2.json`
- `spec/fixtures/contract-state/schema-v2.json`
- `internal/contractstatefixture/`
