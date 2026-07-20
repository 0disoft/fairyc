# ADR 0110: Scalar fixed-array execution

- Status: Accepted for the Stage 0 executable subset
- Date: 2026-07-20
- Product source: `docs/product/02-spec.md`

## Context

The Stage 0 grammar already reserves `[T; N]`, list literals, repeated literals, and indexing, but
the executable frontend previously admitted indexing only for immutable byte views. Treating every
array spelling as executable at once would silently introduce resource-element ownership, nested
layout, repetition amplification, mutable places, and ABI decisions that the current verifier and
build contracts do not own.

## Decision

1. The first executable fixed-array subset admits only `[T; N]` values where `T` is `bool`, `i32`,
   `u8`, `u32`, or `usize` and `1 <= N <= 4096`. The element ceiling is a provisional Stage 0
   implementation guard for the initial by-value representation, not a stable language or ABI
   limit.
2. A nonempty list literal `[a, b, ...]` constructs one array. An expected array type contextually
   types every element; without one, the first element selects the element type and later elements
   must match it exactly. The canonical length is the written element count.
3. Elements evaluate exactly once from left to right before construction. Arrays are inert copy
   values and may cross internal Fairy function parameters and results. This is not a public Fairy
   ABI or C ABI promise.
4. `array[index]` requires `usize`, returns `T`, evaluates the receiver and index once from left to
   right, and traps with `bounds` when `index >= N`. Indexed assignment and slicing remain closed.
5. Empty `[]` and repeated `[value; N]` literals remain parser-valid but semantically unsupported.
   The repeated form cannot open until expansion work and storage are charged to an explicit
   compiler budget; the implementation must not amplify a short source token into unbounded host
   allocation or generated C.
6. Arrays are not admitted inside structs, resources, enums, `Option`, `Result`, raw pointers, or
   foreign signatures in this slice. Array fields, nested arrays, resource elements, mutable
   indexing, general array-to-slice conversion, and stable layout are deferred. ADR 0111 separately
   admits the representation-independent `.len` property, and ADR 0112 admits one contextual shared
   `[u8; N]`-to-`&[u8]` borrow.
7. MIR uses explicit array construction, bounds-check marker, checked switch, and index operations.
   The verifier requires the exact element count, scalar operand types, `usize` index, `bounds`
   trap edge, and a dominating matching check before access.
8. Generated C uses a private assignable struct containing one C array. The wrapper permits
   deterministic internal value calls and returns without exposing raw C array assignment rules.
   Its spelling and calling convention are implementation details. Semantic admission, MIR
   verification, interpretation, and C emission independently reject shapes outside the same
   bounded executable subset.

## Evidence

- semantic tests cover contextual typing, inference, length mismatch, element mismatch, and closed
  empty/repeated forms plus the provisional element ceiling;
- MIR tests cover construction, ordered operands, checked indexing, and verifier mutation;
- interpreter fixtures cover all five admitted element types, in-range calls, and an exact `bounds`
  outcome;
- constrained-C fixtures cover the same five element types, return, and trap outcomes;
- the Windows host smoke compiles and executes all five array element representations in the
  quickstart path plus both optimized and low-profile out-of-bounds cases through local x64 MSVC
  without WSL.
