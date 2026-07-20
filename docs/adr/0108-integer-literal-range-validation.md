# ADR 0108: Integer literal range validation

- Status: Accepted
- Date: 2026-07-20
- Diagnostic: `E1013`
- Superseded in part: ADR 0109 admits the exact `i32` minimum magnitude under direct unary minus and replaces the semantic arbitrary-precision comparison implementation.

## Context

The lexer already preserves exact decimal, hexadecimal, and binary integer spelling, including
valid separators, and typed HIR selects `i32`, `u8`, `u32`, or `usize`. Range errors were still
deferred until MIR parsed the text into a machine value. That turned a source type error into a
lowering failure and also caused hexadecimal spellings containing `e` or `E` to be mistaken for
floating candidates by the semantic prototype.

## Decision

1. Semantic checking recognizes decimal, `0x` hexadecimal, and `0b` binary integers independently
   of letters used as hexadecimal digits. Lexer-approved underscores are removed only for numeric
   comparison; exact source text and spans remain unchanged.
2. The selected implemented integer type defines the admitted nonnegative literal range:
   `i32` through `2147483647`, `u8` through `255`, `u32` through `4294967295`, and Stage 0
   `usize` through `18446744073709551615`.
3. A value beyond that range is one `E1013` on the complete literal span during type checking.
   The expression becomes invalid so no MIR, interpreter, or backend path receives an unrepresentable
   immediate.
4. `E0004` remains the lexer diagnostic for malformed digits, separators, radix syntax, or excluded
   spellings. `E1013` is only for a well-formed mathematical value that does not fit its selected
   implemented type.
5. A leading sign remains a prefix operator, not part of a literal. Existing checked arithmetic can
   construct `i32` minimum as `-2147483647 - 1`; directly admitting the positive token `2147483648`
   under unary minus would require a separate signed-literal folding decision and remains excluded.
6. Parsing uses arbitrary-precision comparison in the semantic frontend, so extremely long but
   lexer-bounded values receive `E1013` rather than overflowing a host integer parser.
7. The semantic fixture contract grows from 22 to 23 cases. Interpreter and C-emission counts do
   not change. The WSL-free Windows quickstart uses hexadecimal and binary values and continues
   through generated C to local x64 MSVC execution.

## Consequences

- User range mistakes stop at the owning type phase with a stable source diagnostic.
- Valid radix spellings containing `e` or `E` no longer fall through as unsupported floats.
- MIR numeric parsers remain defensive checks, not the first source-validation boundary.

## References

- `docs/language/0006-lexing-parsing-and-recovery-v0.md`
- `docs/language/0007-name-resolution-and-typing-v0.md`
- `docs/diagnostics/catalog-v0.md`
- `spec/fixtures/semantic/manifest-v0.json`
