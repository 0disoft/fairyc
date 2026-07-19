# Compiler-Owned C-String Static Storage

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-19
- Language source: `docs/language/0002-borrows-and-return-origins.md`
- Literal source: `docs/language/0006-lexing-parsing-and-recovery-v0.md`
- MIR source: `docs/mir/c-backend-safety-v0.md`

## Context

The lexer and parser already cooked `c"..."` into exact bytes with one trailing NUL and rejected an
interior NUL, but semantic analysis previously rejected both `CStr` and the literal. Generated Fairy
callers therefore had to obtain every nonempty pointer from companion C code. That left the product
unable to prove that source-owned bytes survived semantic analysis, MIR lowering, C emission, and a
native read.

Treating the literal as an ordinary escaped C source string would make C token spelling, host
execution encoding, adjacent-hex escapes, and embedded bytes part of the language implementation.
Treating it as a mutable array or an owned `CString` would also contradict its `Static`, immutable,
allocation-free contract.

## Decision

1. `CStr` is a compiler-known unsized type. The admitted literal type is immutable `&CStr`; bare
   `CStr`, `&mut CStr`, and raw `*CStr` type uses remain rejected.
2. HIR preserves both source spelling and lexer-cooked hexadecimal bytes. Semantic Flow records a
   C-string literal loan with hidden `Static` origin.
3. MIR uses a dedicated comparable `c-string` operand containing the exact decoded bytes, including
   the one trailing NUL. The verifier independently rejects empty storage, a missing terminator,
   an interior NUL, or a non-`&CStr` operand type.
4. The C backend emits the bytes as translation-unit `static const uint8_t[]` data, never as a C
   string token. Equal byte sequences are emitted once. Definitions are ordered by their full
   SHA-256-derived symbol; a digest collision with unequal bytes fails emission.
5. Extracting the byte address for raw FFI is an explicit unsafe conversion from immutable `&CStr`
   to immutable `*u8`. MIR represents the direct-literal executable form as its own
   `cstr-data-address` operation with source and unsafe-scope provenance. Conversion to `*mut u8`
   remains rejected.
6. `experimental build` accepts a program containing nonempty compiler-owned C-string storage. A
   separate native source probe passes the extracted address to hand-written C, which must observe
   every byte and the terminator under Clang/GCC and low/optimized profiles.
7. `fairyc.native-c-report/v29` adds `cStringLiteralSourceCallExecutions` and
   `cStringLiteralBuildSmokeExecutions`; each becomes four only after all four native executions
   return 42 without output.
8. This slice does not implement UTF-8 string literals, byte-string/slice layout, C-string length
   access, a public safe `std::text` wrapper, general raw-pointer casts, interpreter execution of
   view values, or self-hosting.

## Consequences

- A `.fy` source file can now own nonempty immutable native data instead of borrowing every pointer
  from a test companion.
- Static lifetime, byte identity, deduplication, and raw-address escape are distinct verifier and
  backend contracts rather than consequences of generated C spelling.
- The product build smoke proves the ordinary CLI can carry the storage into an executable; the
  source-call probe separately proves that native C can read the exact bytes.
- Hosted Linux execution is still required before either configured count is observed evidence.
