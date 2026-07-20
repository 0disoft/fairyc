# ADR 0123: Preserve mutable parameters and opaque borrow casts

- Status: Accepted
- Date: 2026-07-21
- Grammar source: `spec/grammar/stage0.ebnf`
- Typing source: `docs/language/0007-name-resolution-and-typing-v0.md`
- C boundary source: `docs/mir/c-backend-safety-v0.md`

## Context

A consuming safe File close needs a by-value callee-owned resource whose inline state can be
normalized before the raw close operation. Existing ordinary parameters are immutable places, so a
callee cannot take `&mut file`. Copying the argument into a `var` is impossible for a move-only
resource and would create the wrong ownership story even for copyable values.

The raw Windows File component intentionally accepts opaque storage. The existing borrow-to-pointer
cast requires an exact pointee type, which prevents a checked wrapper from passing the address of
its private inline resource representation as `void *`. The existing CStr literal conversion also
does not handle a borrowed CStr parameter, so an open wrapper cannot pass its path without exposing
raw storage at the call site.

## Decision

1. An ordinary internal parameter may be written `mut name: Type`. The modifier belongs to the
   callee-local binding, not to the calling convention or argument type. Raw foreign declarations
   and explicit method receivers do not accept this modifier in Stage 0.
2. The lossless syntax node records `mut`, the surface AST and typed HIR retain `Mutable`, and name
   resolution grants assignment and mutable-borrow permission only to that symbol. An unmodified
   parameter remains an immutable place and `E1008` rejects assignment or `&mut`.
3. Parameter mutability does not make a type copyable, change by-value move behavior, weaken borrow
   conflicts, or disarm cleanup. A resource parameter begins live in the callee and retains its
   ordinary cleanup obligation until moved or dropped.
4. The compiler-private resource drop receiver is mutable so verified drop glue may take its raw
   address. User source still cannot name or invoke that private function.
5. Inside an explicit unsafe boundary, a direct shared borrow may cast to `*void` and a direct
   mutable borrow may cast to `*void` or `*mut void`. A shared borrow may not become `*mut void`.
   Exact-pointee borrow casts remain available; all other pointee-changing casts remain closed.
6. MIR `raw-address` retains the concrete owner type even when the result pointer element is
   `void`. The verifier and C emitter accept only an exact owner element or `void`, require matching
   unsafe provenance, and preserve the existing no-mutability-strengthening rule.
7. A borrowed CStr path parameter may expose its immutable data pointer through the existing
   unsafe CStr-data operation. MIR and C emission distinguish a local descriptor from static literal
   storage and load `.data` only for the local descriptor.
8. HIR-to-MIR lowering intentionally stops carrying the mutable-binding bit after type checking:
   MIR is already a mutable-local control-flow representation, while ownership and cleanup facts
   remain explicit and verified separately.

## Consequences

- A safe wrapper can own a move-only File parameter, mutate its inline state, and let ordinary drop
  observe the normalized closed representation.
- Private target runtime storage can cross an opaque `void *` boundary without admitting arbitrary
  typed pointer casts or hiding allocation.
- `mut` is not C++-style reference syntax and does not change the caller's value; it only controls
  the callee's local place.
- Mutable fields, mutable indexing or slices, arbitrary pointer arithmetic, raw dereference, foreign
  mutable parameters, and stable public resource layout remain excluded.

## Verification

- parser and AST tests require lossless reconstruction plus exact modifier and name-span retention;
- semantic tests and fixtures accept mutable-parameter assignment and reject the same assignment on
  an immutable parameter;
- semantic, MIR mutation, and C-emission tests cover `&mut T as *mut void`, reject mutability
  strengthening, and retain the concrete owner type;
- C emission tests require a borrowed CStr parameter to load its descriptor data pointer;
- the complete Go suite remains the cross-platform frontend and backend oracle.
