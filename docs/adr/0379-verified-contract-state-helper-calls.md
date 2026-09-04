# ADR 0379: Verify bounded contract-state helper calls

- Status: Accepted
- Date: 2026-09-04

## Context

ADR 0357 correctly rejected every call while transaction MIR had no call graph, transitive effect,
body identity, or rollback model. ADR 0378 later supplied bounded state-side control flow, but useful
transaction programs still had to duplicate reusable storage work in each root owner.

## Decision

1. Admit only statically resolved same-package internal helpers whose current HIR signature is
   exactly `void fn()` in this slice. Calls are represented explicitly in the HIR state side table
   and must match the current ordinary-HIR call sequence.
2. Keep indirect, foreign, cross-package, argument-bearing, value-returning, receiver-bound,
   resource-owning, unsafe, raw-pointer, recursive, and dynamically dispatched calls closed.
3. Store helpers by canonical module-qualified name in sorted order. A root may reach at most 32
   helpers, and the root plus helper bodies may contain at most 1024 operations. Every helper must be
   reachable and the complete helper call graph must be acyclic.
4. Every call uses the root's opaque transaction capability and has canonical cost `1`. Each owner
   declares the complete transitive storage-effect set; missing or widened effects fail before
   execution with `CSM1005`, while malformed call graphs fail with `CSM1009`.
5. Execute helper storage operations in the caller's existing snapshot and private overlay. Helper
   `return` resumes its caller; helper `reject` rejects the root; helper or root trap rolls the whole
   overlay back. Only the root may commit.
6. Include helper order, names, transitive effects, calls, bodies, exits, and costs in the package
   body hash. Helper-bearing payloads use
   `fairyc.contract-state-package-body-artifact/v3`; v1 legacy and v2 graph payload bytes and hashes
   remain unchanged. Decode creates fresh shared capability authority and stable rebind rechecks the
   current root/helper call identities.
7. Make `fairyc.contract-state-fixtures/v3` the current fixture generation. It compiles and executes
   a pure-to-state helper chain and helper rejection rollback, then rejects recursion and missing
   transitive effects during atomic compiler publication.

## Consequences

Transaction logic can share a bounded no-argument internal helper without creating a second state
semantics or weakening atomic publication. The first surface is intentionally narrow: helper
parameters, return values, state borrows, recursion, external calls, native lowering, persistence,
graph-block call sites, and reentrancy remain separate gates.

## Evidence

- `internal/contractstatemir/helpers.go`
- `internal/contractstatemir/helper_execute.go`
- `internal/contractstatemir/helper_test.go`
- `spec/fixtures/contract-state/manifest-v3.json`
- `spec/fixtures/contract-state/schema-v3.json`
