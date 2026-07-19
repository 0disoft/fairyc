# ADR 0067: Compiler-owned UTF-8 string static storage

- Status: Accepted
- Date: 2026-07-19

## Context

The language contract already assigned `"..."` literals immutable `&str` type with hidden `Static`
origin, but semantic analysis still rejected the literal and the executable pipeline had no distinct
text operand or generated-C representation. Treating text as a byte string would erase the UTF-8
invariant; treating it as a C string would incorrectly append a terminator and reject embedded NUL.

Escape cooking also allowed individually valid `\xNN` escapes to combine into invalid UTF-8. Source
files are valid UTF-8, but decoded text storage needs its own post-cooking validation boundary.

## Decision

1. `str` is a compiler-known unsized valid-UTF-8 type. String literals have immutable `&str` type;
   bare `str`, `&mut str`, and raw pointers to unsized `str` remain rejected.
2. After escape cooking, the lexer validates the complete decoded byte sequence as UTF-8. Invalid
   decoded storage is one `E0005` diagnostic over the literal and has no automatic repair.
3. HIR preserves source spelling and cooked hexadecimal bytes. Semantic Flow records the literal
   borrow with hidden `Static` origin.
4. MIR uses a dedicated comparable `string` operand containing exact valid UTF-8 bytes. Its verifier
   rejects invalid UTF-8 or a non-`&str` operand type.
5. The C backend lowers `&str` to a private `fy_str` value with immutable `data` and target-width
   `length` fields. This representation is generated-C transport, not a public Fairy or C ABI.
6. Nonempty values use exact translation-unit `static const uint8_t[]` storage. Empty values use a
   one-byte immutable sentinel with logical length zero. Equal byte sequences are emitted once in
   full-SHA-256 symbol order, and unequal data under one symbol fails emission.
7. Unsafe conversion from immutable `&str` to immutable `*u8` may extract `.data` from a typed local
   or use direct literal storage. Mutable extraction, pointer arithmetic, dereference, and implicit
   text-to-C-string conversion remain rejected. MIR records `str-data-address` with source and
   unsafe-scope provenance.
8. `fairyc.native-c-report/v31` adds `utf8StringLiteralSourceCallExecutions` and
   `utf8StringLiteralBuildSmokeExecutions`. Each becomes four only after Clang/GCC low/optimized
   executions return 42 without output. The source-call lane separately proves the eight UTF-8 bytes
   of `Fairy λ` and length eight reach C through a typed local.
9. This slice does not implement indexing, slicing, source-level length access, a stable public
   string ABI, safe foreign string parameters, interpreter execution of view values, additional
   targets, or self-hosting.

## Consequences

- Fairy source can construct empty and nonempty static UTF-8 views without allocation.
- Text, arbitrary bytes, and terminated C bytes remain distinct through semantic typing, MIR, and C
  lowering even when their private C views share a two-field physical shape.
- Generated C emits numeric bytes, so execution encoding and C token spelling cannot change literal
  meaning.
- Hosted Linux success remains separate evidence. Windows Go tests prove frontend, MIR, verifier,
  and deterministic C emission but do not count native executions.
