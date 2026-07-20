# ADR 0126: Checked mutable indexed assignment

- Status: Accepted
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Language sources: `docs/language/0001-values-and-resources.md`, `docs/language/0005-surface-syntax-v0.md`
- MIR sources: `docs/mir/mir-v0.md`, `docs/mir/c-backend-safety-v0.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`

## Context

ADR 0125 established an exclusive zero-copy `&mut [u8]` descriptor over mutable fixed-array storage,
but intentionally exposed only length and unsafe raw-pointer derivation. Safe FY code still could not
change an array element or observe a mutation through the view. Adding only source syntax would be
unsound evidence: the interpreter previously copied bytes when constructing a view, while generated C
aliased the array, and the MIR verifier recognized a checked load only when it immediately followed
its check block.

## Decision

1. An admitted mutable fixed scalar-array place supports `values[index] = value`. The array receiver
   must resolve to a mutable local or `mut` by-value parameter. The element and value types must match
   exactly and remain in the closed `bool`, `i32`, `u8`, `u32`, or `usize` fixed-array subset.
2. A value of type `&mut [u8]` supports both checked `u8` reads and `bytes[index] = value`. Mutating
   the referent does not require the borrow binding itself to be `var`; exclusivity belongs to the
   borrow type. A shared `&[u8]`, literal, immutable array place, or non-place expression cannot be a
   mutable target.
3. Indexed assignment evaluates the receiver and `usize` index once from left to right, freezes the
   view descriptor and scalar index needed by the later store, and performs the complete `bounds`
   check before evaluating the right-hand side. Failure traps without evaluating the value. Success
   evaluates the value once and then stores it. This is distinct from complete resource-root
   reassignment, whose existing transactional rule evaluates the new resource before dropping the old
   root.
4. Flow treats direct fixed-array mutation as a write to the whole owner and rejects it while a
   conflicting loan remains live. Mutation through `&mut [u8]` uses the already-proven exclusive loan;
   it does not replace or strengthen that loan and creates no subview.
5. MIR adds `array-index-store` and `view-index-store`. Each is a no-result side effect with receiver,
   frozen `usize` index, and value operands. The corresponding existing index-check marker and checked
   two-way terminator must dominate the store across any right-hand-side call or branch. The verifier
   matches the exact operands, owner, element, operator, trap category, success dominance, and
   non-returning trap edge.
6. The interpreter gives fixed arrays explicit backing identity. Ordinary array assignment, calls,
   parameters, and results preserve value semantics by copying the backing storage, while shared and
   mutable byte views retain the selected array backing. A mutable view store therefore changes its
   caller's array, but mutating an array passed by value does not change the caller's copy.
7. Generated C reuses the private fixed-array wrapper and
   `fy_mut_byte_slice { uint8_t *data; uintptr_t length; }`. It emits the bounds branch before the
   right-hand-side blocks and emits one exact typed `data[index] = value` only on the verified success
   path. No allocation, public ABI, `restrict`, retained pointer, or unchecked access is introduced.
8. Mutable slicing, nested place projections, resource elements, compound assignment, increment and
   decrement, atomics, and general user-defined indexing remain deferred.

## Consequences

- Safe File and other internal wrappers can now inspect or adjust bytes without dropping to unsafe
  code after acquiring a mutable byte view.
- Bounds failure has a deliberate side-effect boundary: a value-producing call is not run for an
  invalid place.
- The interpreter and generated-C path now agree on array value copies versus borrowed aliasing,
  removing a latent divergence that was observationally hidden while mutable views exposed only
  metadata.

## Verification

- semantic tests cover mutable fixed-array places, immutable rejection, mutable-view mutation through
  an immutable binding, shared-view rejection, and reads through both byte-view mutabilities;
- MIR tests cover store shapes, frozen place operands, check dominance across right-hand-side calls,
  and verifier mutations;
- interpreter tests and fixtures cover direct and borrowed mutation, by-value array isolation,
  `bounds` before right-hand-side effects, and exact return 42;
- constrained-C fixtures cover deterministic array and mutable-view stores plus the checked trap path;
- the checked-in Windows quickstart and host smoke compile `.fy` through the Go Stage 0 compiler,
  emit C, build with local x64 MSVC at low and optimized profiles, and execute without WSL;
- Linux Clang/GCC, glibc, ASan, and UBSan execution remains unobserved on this Windows-only host.
