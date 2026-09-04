# ADR 0381: Verify owner-bound state-borrow transport

- Status: Accepted
- Date: 2026-09-04

## Context

ADR 0380 established executable transaction-oracle borrow provenance, but no compiler path could
transport that authority. Leaving borrows as an oracle-only API meant stable package bodies and
fixture publication could not prove that compiler-created views preserve exact cell/map ownership,
invalidation, or transaction confinement.

## Decision

1. Add compiler-private graph-only `cell-borrow`, `map-borrow`, `borrow-view`, and `borrow-copy`
   operations to source-independent HIR and contract-state MIR. Borrow identities are transaction-
   unique, cannot enter ordinary SSA values or block parameters, and cannot cross a block edge.
2. Require cell/map borrow origins to name an exact field and, for maps, an exact literal encoded
   key. Nested views retain that origin and value codec. `borrow-copy` is the only operation that
   produces an ordinary typed SSA value and always copies bytes through the ADR 0380 oracle.
3. Invalidate live handles after replacement or deletion of the exact cell/map address. A literal
   mutation of another map key remains valid. A dynamic mutation on the same map field fails closed
   while any borrow is live because the verifier cannot prove non-aliasing.
4. Keep borrow operations out of legacy linear bodies and helper bodies. Only bounded acyclic graph
   roots may contain them, and every handle must be consumed or abandoned inside its defining block.
5. Add `fairyc.contract-state-package-body-artifact/v4`. Its canonical payload and body hash include
   every borrow identity, view bound, and operand. v1 linear, v2 graph, and v3 helper payload bytes
   and their pre-borrow body hashes remain unchanged. Decode and fresh-owner rebind recreate private
   transaction capability and re-enter the verifier.
6. Make `manifest-v4.json` the current execution corpus. Two compiler-published positive cases prove
   nested cell views and map-borrow survival across an unrelated key mutation. Three isolated
   mutations prove exact-address invalidation, conservative dynamic-key alias rejection, and block-
   edge escape rejection.

## Consequences

The compiler can now carry owner-bound read-only state views through HIR, verified MIR, stable cache
payloads, fresh rebind, and atomic oracle execution without exposing pointers or mutable storage.
This does not add Fairy source state-access syntax, ordinary-HIR lowering, borrow transfer through
branches or helpers, mutable views, native lowering, production persistence, gas, or a deployable
contract target.

## Evidence

- `internal/hir/state.go`
- `internal/contractstatemir/borrow_verify.go`
- `internal/contractstatemir/borrow_execute.go`
- `internal/contractstatemir/borrow_mir_test.go`
- `internal/contractstatefixture/`
- `spec/fixtures/contract-state/manifest-v4.json`
- `spec/fixtures/contract-state/schema-v4.json`
