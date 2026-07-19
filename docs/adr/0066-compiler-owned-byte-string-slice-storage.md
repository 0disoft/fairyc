# ADR 0066: Compiler-owned byte-string slice storage

- Status: Accepted
- Date: 2026-07-19

## Context

The lexer and parser already decoded `b"..."` into exact bytes and the language contract assigned
the literal immutable `&[u8]` type with hidden `Static` origin. Semantic analysis still rejected the
type and literal, so no executable path preserved either its arbitrary bytes or its logical length.

Reusing the C-string representation would be wrong: a byte string permits embedded NUL, appends no
terminator, and is a bounded view rather than a sentinel-terminated pointer. Representing an empty
slice with a null pointer would also contradict Fairy C's non-null borrow contract.

## Decision

1. `[u8]` is a compiler-known unsized type. A byte-string literal has immutable `&[u8]` type; bare
   `[u8]`, `&mut [u8]`, and raw pointers to the unsized slice remain rejected.
2. HIR preserves the source spelling and lexer-cooked hexadecimal bytes. Semantic Flow records the
   literal borrow with hidden `Static` origin.
3. MIR uses a dedicated comparable `byte-string` operand containing the exact decoded bytes. Empty
   storage is valid, embedded NUL is ordinary data, and the operand's byte count is its logical
   length. The verifier rejects a non-`&[u8]` operand type.
4. The C backend lowers `&[u8]` to a private `fy_byte_slice` value with immutable `data` and
   target-width `length` fields. This is internal generated-C transport, not a public Fairy or C ABI.
5. Nonempty literals use exact translation-unit `static const uint8_t[]` storage. Empty literals
   use a one-byte immutable sentinel while retaining length zero, preserving a non-null address
   without inventing a logical byte. Equal byte sequences are emitted once in full-SHA-256 symbol
   order, and unequal data under one symbol fails emission.
6. Unsafe conversion from immutable `&[u8]` to immutable `*u8` may extract `.data` from a typed local
   or use the direct literal storage. Conversion to `*mut u8`, pointer arithmetic, dereference, and
   implicit byte-to-text conversion remain rejected. MIR records this as
   `byte-slice-data-address` with source and unsafe-scope provenance.
7. `fairyc.native-c-report/v30` adds `byteStringLiteralSourceCallExecutions` and
   `byteStringLiteralBuildSmokeExecutions`. Each becomes four only after the Clang/GCC and
   low/optimized executions return 42 without output. The source-call lane also requires generated C
   to carry length five and a separate C callee to read the five `fairy` bytes.
8. This slice does not implement UTF-8 string literals, indexing, slicing, source-level length
   access, a stable slice ABI, safe foreign slice parameters, interpreter execution of view values,
   additional targets, or self-hosting.

## Consequences

- Fairy source can now construct empty or nonempty immutable byte slices without allocation and
  carry them through the ordinary experimental C build path.
- Embedded bytes are emitted numerically, so C token spelling and execution encoding cannot change
  their meaning.
- Generated C exposes a concrete private view for execution while the source and MIR contracts stay
  target-independent.
- Hosted Linux success remains a separate evidence requirement; Windows Go tests prove frontend,
  MIR, verifier, and deterministic emission behavior but do not count the native executions.
