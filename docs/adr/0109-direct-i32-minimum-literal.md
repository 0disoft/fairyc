# ADR 0109: Direct i32 minimum literal

- Status: Accepted
- Date: 2026-07-20
- Language surface: `-2147483648`, `-0x8000_0000`, and the equivalent binary spelling

## Context

Fairy C lexes a leading minus as a prefix operator, so the positive token inside
`-2147483648` is one greater than `i32` maximum. Rejecting that token before considering its direct
prefix made the most common spelling of `i32` minimum unavailable, even though MIR, the interpreter,
and generated C already represent the value exactly. Separately, semantic range checking should not
allocate a second arbitrary-precision integer proportional to an already source-bounded token.

## Decision

1. A unary minus directly applied to an integer token whose mathematical magnitude is exactly
   `2147483648` folds during type checking to the exact `i32` constant `-2147483648`.
2. Decimal, hexadecimal, and binary spellings with lexer-approved separators are equivalent. The
   complete prefix expression has type `i32`; the positive child token is not independently typed
   as a representable `i32` expression.
3. Parenthesized magnitudes do not receive this exception. `-(2147483648)` remains `E1013` because
   the out-of-range expression exists independently inside the group.
4. A magnitude above `2147483648`, with or without direct unary minus, remains one `E1013` on the
   positive literal token. Unary plus receives no signed-minimum exception.
5. The folded HIR is one normalized `i32` literal. MIR therefore carries one exact `OperandI32`,
   the interpreter returns the value without executing checked negation, and C emission uses
   `INT32_MIN` without relying on a nonportable positive C literal.
6. Semantic range comparison scans validated digits, ignores separators and leading zeroes, and
   compares digit count plus digit order against a small canonical limit. It allocates no second
   input-sized big integer; the existing source-byte and lexer-work budgets still bound the token.
7. The interpreter fixture contract grows from 87 to 88 cases. C emission remains 29 cases and the
   derived Linux matrix remains 116 compiler/profile executions and 58 sanitizer executions. The
   Windows quickstart executes the hexadecimal spelling through generated C and local x64 MSVC
   without WSL.

## Consequences

- All mathematical `i32` values now have a direct conventional source spelling.
- Runtime checked negation remains unchanged: negating an already materialized `i32` minimum still
  traps with `integer-overflow`.
- Very long well-formed literals still receive deterministic `E1013` without a second proportional
  semantic big-integer allocation.

## References

- `docs/adr/0108-integer-literal-range-validation.md`
- `docs/language/0006-lexing-parsing-and-recovery-v0.md`
- `docs/language/0007-name-resolution-and-typing-v0.md`
- `docs/mir/mir-v0.md`
- `docs/mir/c-backend-safety-v0.md`
