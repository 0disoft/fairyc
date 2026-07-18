# Raw Pointer and Target Scalar Foreign Calls

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-19
- Language source: `spec/grammar/stage0.ebnf`
- ABI source: `docs/abi/c-abi-v0.md`
- Predecessor: `docs/adr/0046-scalar-source-foreign-c-call.md`

## Context

The first executable foreign-call slice transported only `i32`. That proved symbol identity and
unsafe provenance but could not express the pointer and size boundaries used by target standard
library helpers. The Stage 0 grammar and language contract already define fixed-width scalars,
`usize`, `*T`, `*mut T`, and context-typed `null`; the implementation discarded those types.

The exact source spelling and semantic mapping for the broader `c.*` scalar family are not yet
decided. Inventing them as part of this change would couple a useful fixed-width slice to an
unsettled language decision.

## Decision

1. Typed HIR admits `u8`, `u32`, `usize`, immutable `*T`, and mutable `*mut T`. `null` is accepted
   only when its expected type is one of those raw-pointer forms.
2. MIR keeps `u8`, `u32`, `usize`, and typed-null operands distinct. The verifier rejects an
   out-of-range narrow operand or a null operand without a raw-pointer type.
3. For `x86_64-linux-gnu`, constrained C maps `u8` to `uint8_t`, `u32` to `uint32_t`, `usize` to
   `uintptr_t`, `*T` to `const T *`, and `*mut T` to `T *`. The admitted pointees are `u8`, `i32`,
   `u32`, and `usize` only.
4. Integer literals take an expected admitted scalar type at calls and comparisons. Unconstrained
   integer literals remain `i32`.
5. Raw pointer dereference, pointer arithmetic, casts, non-null pointer construction, and safe
   borrow creation from a raw pointer remain unsupported. This slice transports only null or a
   pointer returned by an admitted foreign call.
6. The interpreter remains fail-closed before binding or evaluating a foreign call.
7. A second closed native probe crosses a generated Fairy caller and hand-written C callee in both
   Clang/GCC directions and both profiles. It checks two typed null pointers plus `usize` and `u8`.
   `fairyc.native-c-report/v13` adds four `foreignPointerCallExecutions` without changing the four
   earlier `foreignCallExecutions`.
8. `c.*` scalars, `@repr(C)` aggregates, callbacks, exports, arbitrary link inputs, and public
   standard-library wrappers remain separate gates.

## Consequences

- Fairy source can now describe the scalar and null-pointer half of a real C boundary without
  pretending a safe reference is nullable.
- SystemHeap and text-view helpers still need source-level C-layout output records before their
  full safe wrappers can be connected.
- The Stage 0 compiler remains implemented in Go; this change expands programs compiled by it.
