# Scalar Fairy Source to Foreign C Call

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-19
- Language source: `spec/grammar/stage0.ebnf`
- ABI source: `docs/abi/c-abi-v0.md`
- MIR source: `docs/mir/mir-v0.md`

## Context

The parser has accepted individual `unsafe extern C fn` declarations and `unsafe {}` expressions
since the Stage 0 grammar was fixed, but AST lowering discarded the declarations and semantic
analysis rejected the unsafe expression. The native ABI probes therefore tested only C against C.
The standard-library C slices cannot become source-callable until a foreign-call identity and
unsafe provenance survive every compiler boundary.

## Decision

1. AST lowering preserves foreign declarations separately from Fairy function bodies and preserves
   whether a Fairy function itself is unsafe.
2. Name resolution places a foreign declaration in the module function namespace. Typed HIR
   records build-local symbol identity, exact external name, `C` ABI, ordered parameter types, and
   result type.
3. A foreign call outside `unsafe {}` or `unsafe fn` produces `E5101`. A safe borrow in a raw
   foreign signature produces `E5102`. The first executable slice otherwise admits only `i32`
   parameters and an `i32` result; broader already-specified ABI types remain backend capabilities,
   not silently reinterpreted values.
4. MIR stores foreign declarations at program scope. Every foreign call records its `C` ABI,
   source span, exact unsafe-scope span, and an explicit unsafe-operation provenance record. `V1701`
   rejects a foreign call when that record is removed or incomplete.
5. The interpreter does not bind host symbols. It returns the structured `foreign-call` capability
   error before attempting execution.
6. The C backend emits one exact-name `extern int32_t` declaration and ordinary call. Compatible
   declarations of the same external name across modules coalesce to one prototype; conflicting
   signatures fail before emission. It rejects C11 keywords, `main`, generated `fy_`/`FY_` names,
   and preamble-owned identifiers. A local `#ifdef`/`#undef` guard prevents an included header macro
   from rewriting an otherwise valid external symbol. The backend adds no purity, alias, no-unwind,
   retention, visibility, or interposition promise.
7. The Linux native runner compiles the generated Fairy caller and a closed hand-written C callee
   into separate objects. Clang caller/GCC callee and the reverse each run at low and optimized
   profiles and must return `42` with no output. `fairyc.native-c-report/v12` adds the resulting four
   `foreignCallExecutions`.
8. Configured native work is not a hosted receipt. The target remains unsupported and the
   interpreter remains the oracle only for programs without foreign calls.

## Consequences

- A real `.fy` source now reaches an external C symbol through the implemented compiler pipeline.
- The slice is deliberately too narrow to connect SystemHeap, Buffer, Arena, text, or File because
  those APIs require raw pointers, size types, tagged errors, and resource wrappers.
- ADR 0047 subsequently adds a bounded typed-null raw-pointer and target-scalar slice. `@link_name`,
  exports, non-null pointer operations, callbacks, `c.*` scalars, `@repr(C)` aggregates, header
  import, arbitrary product link inputs, and foreign nonlocal control remain later gates.
- Self-hosting is unaffected; the Stage 0 compiler remains Go.
