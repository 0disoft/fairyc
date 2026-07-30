# Parser Fixtures v0

This directory is the executable-data companion to `spec/grammar/stage0.ebnf` and `docs/language/0006-lexing-parsing-and-recovery-v0.md`.

`manifest-v0.json` is the fixture index. `schema-v0.json` defines its machine-readable shape. Every case is a mini-package with exactly one `src/` source root so file-to-module identity is tested rather than mocked.

The Go runner lives in `internal/parserfixture`. It rejects unknown manifest fields and paths that
escape their declared roots, requires lossless source reconstruction for passing cases, lowers an
AST only after an error-free parse, and compares every listed diagnostic exactly.
The pass corpus also pins contextual `>>`: the raw token remains lossless, expression use remains a
right shift, and nested type arguments close without whitespace.
It also retains module constants plus contextual module-item `static_assert` while proving that
`static_assert` remains available as an ordinary non-item identifier.

## Result boundary

- `pass` means UTF-8 validation, lexing, full parsing, recovery accounting, and module-path validation produce no `E00xx` diagnostic.
- `fail` means the listed diagnostics are produced with exactly the listed canonical byte spans and repair kinds.
- Parser fixtures do not assert name resolution, typing, ownership, MIR, or runtime success.
- Unlisted lexer or parser diagnostics fail the case.

## Normalized expression notation

Successful precedence fixtures use a compact prefix tree:

```text
(operator left right)
(group expression)
(call callee argument...)
(index receiver index)
(array element...)
(array-repeat element count)
(array-empty)
(field receiver name)
(? expression)
```

This notation is a test oracle, not a public AST serialization format. Parentheses written by the user produce an explicit `group` node.

## Span and repair notation

Diagnostic spans are zero-based half-open byte ranges over the original source. All current fail fixtures are ASCII so byte and scalar counts happen to agree; future non-ASCII fixtures must still calculate bytes.

Repair values mean:

- `none`: the lexer recognized a complete invalid token and needs no parser repair;
- `insert`: a zero-width missing token was inserted;
- `skip`: a real unexpected token was consumed into an error node;
- `synchronize`: input was consumed to a grammar-owned boundary;
- `exit`: the current production ended without consuming an outer delimiter.

## Adding a case

Add the source under `cases/pass/<id>/src/` or `cases/fail/<id>/src/`, add one manifest entry, and keep the module declaration equal to the source-root-relative path. A grammar change is incomplete until affected positive, boundary, and counterexample cases change in the same review.
