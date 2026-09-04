# ADR 0380: Establish owner-bound state-borrow oracle semantics

- Status: Accepted
- Date: 2026-09-04

## Context

The transaction oracle copied every read across its boundary. That preserved alias safety, but the
compiler had no executable authority model for a future state-backed borrowed view. Adding only a
HIR or MIR borrow type would not prove invalidation, transaction escape, retry isolation, or exact
cell/map ownership.

## Decision

1. Add an internal `BorrowedValue` handle created only by `EncodedCell.Borrow` or
   `EncodedMap.Borrow`. The handle is owned by exactly one open transaction.
2. Record exact provenance as layout hash, starting snapshot version, source field index, derived
   storage key, and that key's transaction-local mutation generation. A private nonzero token also
   prevents authority transfer to another transaction attempt.
3. Read through the current transaction view, including prior overlay writes. Replacing or deleting
   the exact cell/map entry increments its generation and invalidates every earlier handle and
   nested view from that address. Mutating another map entry does not invalidate it.
4. Permit bounded nested byte views that retain the original provenance and generation. Reject
   negative or out-of-range views.
5. Return copied bytes from observation. The handle never exposes a pointer or mutable alias into
   the committed snapshot, write overlay, database adapter, or retained host buffer.
6. Reject observation after commit or rollback with the existing closed-transaction error. A new
   retry attempt receives a distinct private token even when it starts from the same store version.

## Consequences

The semantic oracle can now execute and test exact owner-bound lifetime and invalidation rules
without exposing raw storage. At this decision point, HIR/MIR and stable transport remained closed;
ADR 0381 subsequently admits that separate verified compiler slice. Source syntax, native lowering,
and production persistence remain closed.

## Evidence

- `internal/contractstate/borrow.go`
- `internal/contractstate/borrow_test.go`
- `internal/contractstate/transaction.go`
- `internal/contractstate/encoded.go`
