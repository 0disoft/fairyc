# ADR 0093: Borrowed-view internal call transport

- Status: Accepted
- Date: 2026-07-20
- Language surface: internal parameters of `&str` and `&[u8]`

## Context

Borrowed UTF-8 and byte views already retain exact data and length metadata, and `.len` makes their
byte count source-visible. The Windows output example could therefore avoid a duplicated length,
but `main` still declared a raw foreign function, cast text to a pointer, and opened `unsafe`
directly. Fairy C needs a way to contain that narrow operation without inventing a public standard
library before package manifests and `std::*` modules exist.

## Decision

1. An ordinary internal function may accept `&str` and `&[u8]` parameters. The parameter is a
   non-owning borrow with the caller-proven origin and does not acquire release authority.
2. Typed HIR retains the exact safe-borrow type. Flow applies the existing call-bounded borrow rules.
   This decision does not widen borrow returns, stored borrows, multiple-origin returns, or foreign
   signatures.
3. MIR retains the exact parameter types and call operands. The verifier requires caller and callee
   types to agree. The interpreter passes the private view value without converting it to a raw
   pointer.
4. The constrained C backend passes the existing private `fy_str` or `fy_byte_slice` descriptor by
   value between generated internal functions. This is compiler-private transport, not a stable
   Fairy ABI, C ABI, exported symbol, or public layout.
5. A safe ordinary function may contain a lexically explicit `unsafe` block and expose a smaller
   checked result to its safe callers. The admitted Windows write contract is synchronous and does
   not retain its input pointer after returning, so the caller's borrow covers the complete foreign
   use. The wrapper must validate status and completed length before reporting success; a retaining,
   asynchronous, or otherwise uncontracted foreign function could not use this safe shape.
6. The checked-in Windows output example lists a separate support module whose safe helpers own text
   output, raw-contract edge checks, and byte-view length. Its entry `main` contains no raw pointer,
   foreign call, `unsafe` block, or manually synchronized byte count. ADR 0094 owns how that support
   module enters the explicit program source closure.
7. This decision does not create `std::io`, portable console behavior, a public error type, effect
   typing, formatting, stdin, stderr writes, retained borrows, callbacks, or external borrowed-view
   ABI.

## Consequences

- A user program can isolate target-specific unsafe code behind one auditable function while its
  ordinary control flow remains safe source.
- Both UTF-8 and byte-slice descriptors now have interpreter, deterministic C-emission, and local
  Windows/MSVC internal-call evidence.
- The later public standard-library wrapper can reuse the same safe parameter shape, but it still
  requires real package and module ownership rather than a compiler magic name.

## References

- `docs/adr/0091-windows-stdout-exec-target.md`
- `docs/adr/0092-borrowed-view-byte-length.md`
- `docs/language/0002-borrows-and-return-origins.md`
- `docs/mir/mir-v0.md`
- `docs/mir/c-backend-safety-v0.md`
