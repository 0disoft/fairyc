# ADR 0216: Add checked statement-only compound assignment

- Status: Accepted
- Date: 2026-07-29
- Product source: `docs/product/02-spec.md`
- Grammar source: `spec/grammar/stage0.ebnf`
- Arithmetic source: `docs/language/0004-results-traps-and-control-flow.md`
- Depends on: ADR 0024, ADR 0100, ADR 0101, ADR 0104, ADR 0136, ADR 0213

## Context

Fairy C already has C-shaped declarations and classic counter loops, but a counter post clause still
requires `index = index + 1`. That is noisy and makes the familiar loop silhouette look unfinished.
Copying C's `++` and `--` would also copy their expression values, prefix/postfix distinction, and
sequencing traps. Compound assignment provides the useful shorthand without adding another
side-effecting expression class.

Adding accepted tokens and statement grammar is a language change under the exact pre-1 pin policy.
It cannot be slipped into `fairyc-language/1` as a compiler patch.

## Decision

1. The product language pin becomes `fairyc-language/2`. Current starter templates, examples,
   compiler-bundled SDK metadata, workspace fixtures, locks, cache inputs, and current documentation
   use `/2`. Historical records keep their original pins.
2. Assignment remains a statement and returns no value. In addition to `=`, mutable `i32`, `u8`,
   `u32`, and `usize` locals support `+=`, `-=`, `*=`, `/=`, `%=`, `<<=`, `>>=`, `&=`, `^=`, and
   `|=`. Mutable fixed-array scalar elements and mutable `u8` view elements support the same
   operators when their exact element type admits the underlying operation.
3. A compound assignment evaluates the target place once from left to right, captures its old scalar
   value, evaluates the right-hand side once, applies the corresponding existing checked binary
   operation, and stores only on success. Indexed targets perform their complete bounds check and
   old-value read before evaluating the right-hand side.
4. The right-hand side must have exactly the target type. Arithmetic overflow or unsigned underflow
   traps as `integer-overflow`; zero division or remainder and signed minimum divided by negative one
   keep their existing categories; invalid shifts and overflowing left shifts keep their existing
   order and categories. A trap leaves the destination unchanged.
5. Classic `for` initializer and post direct-scalar assignments admit compound operators when their
   right-hand side remains in the existing pure scalar header subset. Calls, borrows, indexing,
   resources, and other header side effects remain rejected.
6. CST, AST, and typed HIR retain the exact compound operator. MIR eliminates it into an old-value
   read, the existing verified checked-binary operations, and the existing assignment or indexed
   store. The interpreter and C backend receive no new operation kind.
7. Resource values, `bool`, fields, nested projections, assignment expressions, chained assignment,
   `++`, `--`, wrapping compound operators, implicit promotions, and overloaded assignment remain
   absent.
8. The immediately preceding `/1` parser accepts the C-shaped grammar including classic `for` but
   rejects compound assignment. Because every valid `/1` source file is already valid `/2` source,
   `fairyc-migrate-prev` validates and preserves source bytes instead of rewriting them; package
   manifests still require an explicit exact-pin update.

## Compatibility

The change is source-additive in grammar but intentionally uses a new exact language revision.
Runtime ABI, target schema, C transport dialect, diagnostic catalog, MIR version, SDK source bytes,
and SDK version remain unchanged. Changing the SDK's required language pin changes package and cache
identity inputs even though its embedded source digest is unchanged.

## Evidence

- Lexer tests require longest-match lossless tokens for all ten operators.
- Parser and AST tests retain exact operators, reconstruct source bytes, and prove `/1` rejection.
- Semantic tests admit every operator on exact integer types, admit direct `for` post use, and reject
  unsupported target types.
- MIR tests require checked binary operations plus one indexed old-value load and final store from
  one evaluated place; verifier success remains mandatory.
- Interpreter tests cover success plus overflow, division-by-zero, and invalid-shift traps.
- C-emission tests require reuse of checked arithmetic and indexed-store features.
- The WSL-free Windows host smoke builds and runs the checked-in workspace whose dependency computes
  42 through `result += 2`, including a separately built compiler.

## Consequences

Counter loops and small numeric state updates now look like ordinary C without importing C's
increment-expression baggage. The cost is an exact `/2` cutover and a deliberately closed scalar
place subset; richer projections must first gain their own single-evaluation place representation.
