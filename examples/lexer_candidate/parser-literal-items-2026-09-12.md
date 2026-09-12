# Private parser literals and static assertions — 2026-09-12

This extends the [control-syntax milestone](parser-control-items-2026-09-12.md).
The stable grammar, parser fixture manifest, Go lexer/parser, SDK, runtime, API,
DB and version sources are unchanged. These changes extend private syntax
preservation, not numeric evaluation, constant evaluation or FFI execution.

## Literal preservation

Floating tokens (kind 6) now produce literal expression nodes. Native probes
compare raw spelling, normalized cooked spelling, token kind and exact byte spans
with Go. Cases cover zero, decimal/exponent forms, separators and large mantissas.
Independent Go AST comparisons cover unary minus, multiplication/addition and
postfix propagation around floats.

C-string tokens (kind 9) retain their distinction from string and byte-string
tokens through `literal_kind`. Tests compare raw source and hexadecimal cooked
bytes, including exactly the lexer-produced terminal NUL for empty, ASCII,
Unicode and escaped inputs. Interior NUL written as `\0`, `\x00` or `\u{0}`,
invalid escapes and unterminated strings remain lexical rejections.

An initial new float test incorrectly expected `1.2__3` to be rejected. Both
current lexers strip underscores before decimal normalization and accept it.
The test now records that existing compatibility behavior against Go rather than
changing the lexer or declaring a new grammar rule. Missing exponents and invalid
numeric suffixes remain negative cases, with reference rejection checked first.
Temporary diagnostic output used to locate this expectation error was removed.

## Contextual static assertions

Private declaration kind 8 retains module-item `static_assert(expression);`.
It has a required expression root and zero name spans, body and result type:
zero name spans mean no name was written, not a missing name repaired by the
parser. The verifier checks that special case without weakening named-declaration
checks. Declaration kind/order/full spans/absent name spans also participate in
the Go CST comparison.

Tests cover true and false conditions, expression precedence, references to
constants, attribute/public syntax, same-named functions and local variables,
malformed delimiters/operands/terminators and complete declaration rollback.
511 assertions plus the module fit the 512-declaration limit; one more exhausts
it. False conditions are intentionally parsed, not evaluated.

The contextual dispatch is confined to module items. The lexer does not reserve
`static_assert` globally, and existing function calls and identifiers keep their
meaning. Public syntax admission does not establish semantic visibility validity.

## Existing fixture measurement

All 22 unchanged parser fixtures were observed under low and optimized Windows
MSVC profiles. Complete native records agree between profiles.

| Classification | Previous | Current |
| --- | ---: | ---: |
| Supported valid syntax | 8 | 10 |
| Unsupported valid syntax | 4 | 2 |
| Matching rejection | 9 | 9 |
| File/module context gap | 1 | 1 |

`pass.literals` and `pass.compile-time-items` now parse completely and are
explicit support/removal regression gates alongside the earlier eight fixtures.
There was no budget exhaustion or unexplained false acceptance in this batch.
Matching rejection is not a claim of identical diagnostics.

Remaining valid fixture blockers are `pass.foreign-call` at bytes 51..57
(`unsafe extern`) and `pass.package-state` at 54..59 (`state`).
`fail.module-mismatch` still needs a relative filename that the byte-only
candidate interface does not receive.

## Verification boundary

`fairyc_go_format`, `fairyc_go_test_parser_literal_items`,
`fairyc_go_test_parser_evidence` and `fairyc_go_test_parser_surface` passed
on the final relevant sources. Native probes executed in both profiles. All
fifteen candidate sources parsed without problems and passed declaration
comparison and integrity checks. The changed files remain below the unchanged
32768-byte source cap: `bodies.fy` 31457 bytes, `syntax.fy` 27126 and
`integrity.fy` 8451. Node, item and recursion budgets were not raised.
Scoped strict mustflow and whitespace checks accompany delivery.

The new runner intent is a bounded local one-shot check with closed stdin, no
network and cache-only writes. The previously pending parent delivery settings
were committed once unrelated staging cleared; other work was preserved.
The full repository suite, remote CI, release checks and push are outside this
private candidate task. No child documentation validator is configured.
