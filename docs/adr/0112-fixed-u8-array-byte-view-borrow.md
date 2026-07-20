# ADR 0112: Fixed `u8` array borrow as an immutable byte view

- Status: Accepted
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Language sources: `docs/language/0002-borrows-and-return-origins.md`, `docs/language/0005-surface-syntax-v0.md`
- MIR sources: `docs/mir/mir-v0.md`, `docs/mir/c-backend-safety-v0.md`

## Context

The executable subset can construct and read positive scalar fixed arrays, while the immutable
`&[u8]` view already supports length, emptiness, indexing, slicing, and internal calls. Without one
safe bridge, locally assembled bytes cannot use that view API and callers must choose between a
fixed-array-specific function or compiler-owned literal storage.

## Decision

1. A direct shared borrow of an admitted `[u8; N]` place is contextually converted to `&[u8]` only
   when the destination, argument, or return position already expects `&[u8]`. The source form is
   `&array`; no implicit conversion occurs for a bare array value.
2. The source must be one resolved place. Array literals, calls, indexing results, and other
   temporary expressions cannot be borrowed by this slice. `&mut`, element types other than `u8`,
   zero-length arrays, and shapes outside the provisional executable bound remain closed.
3. The view retains the source place's hidden origin and exact canonical `N`. The ordinary
   last-use loan prevents assignment, mutation, or movement of the array while the view is live.
   A view of a callee-local array or by-value parameter cannot escape and is rejected with `E2102`.
4. ADR 0113 permits safe-borrow reassignment outside loops after Flow has evaluated the right-hand
   side, replaced the target provenance, and joined reachable branch origins. Loop-carried
   reassignment remains closed rather than silently changing provenance across iterations.
5. Typed HIR uses `array-view-borrow`. MIR uses one explicit `array-borrow-view` operation with a
   local `[u8; N]` operand, canonical owner type, exact immutable byte-view result, and `&` marker.
   The verifier rejects every other shape.
6. The bounded interpreter snapshots the exact bytes for its read-only oracle. This is
   observationally equivalent because Flow forbids mutation during the loan. Generated C performs
   no copy or allocation: it creates the private byte-view descriptor from the array local's data
   address and canonical `N`.
7. The private generated-C descriptor is not a stable Fairy or C ABI. General array-to-slice
   coercion, mutable slices, returned local views, public FFI exposure, array slicing, and generic
   element views remain closed.

## Consequences

- Fixed bytes assembled at runtime can reuse the existing immutable byte-view API and internal
  functions without raw pointers or unsafe code.
- C execution remains zero-copy while the interpreter stays simple and deterministic.
- Contextual typing is intentional: `let bytes: &[u8] = &array` and `consume(&array)` work, while a
  bare `let borrow = &array` retains the distinct fixed-array borrow type and is outside executable
  byte-view lowering.

## Verification

- semantic tests distinguish contextual conversion, permitted outside-loop reassignment, and the
  closed loop-carried form;
- MIR tests preserve `array-borrow-view` and reject a mutated operator;
- interpreter and C-emission tests read the exact bytes through an internal `&[u8]` call;
- the shared fixed-array acceptance fixture exercises the bridge without changing fixture totals;
- the WSL-free Windows quickstart compiles and executes the borrowed local array through x64 MSVC.
