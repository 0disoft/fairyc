# ADR 0213: Classic scalar `for` loop

- Status: Accepted
- Date: 2026-07-29
- Product source: `docs/product/02-spec.md`
- Grammar source: `spec/grammar/stage0.ebnf`
- Depends on: ADR 0024, ADR 0136

## Context

The C-shaped source cutover deliberately deferred `for` because spelling alone was not enough to
settle initializer scope, header side effects, cleanup, or `continue` behavior. Counter loops are a
high-frequency C idiom, but importing C's comma operator, increment operators, declaration lists,
and arbitrary header expressions would enlarge the language and make Flow joins less local.

Adding `for` also reserves a spelling that was an identifier in the immediately preceding pre-1
revision. Under the exact-pin compatibility policy, this is a language revision rather than a
compiler patch.

## Decision

The product compiler accepts the classic form:

```fy
for (var i32 index = 0; index < limit; index = index + 1) {
    body();
}
```

All three clauses are optional. An omitted condition is exactly `true`.

The initial implemented subset is intentionally closed:

- the initializer is one scalar `let` or `var` declaration, or one direct scalar-local assignment;
- the condition is a pure scalar expression with type `bool`;
- the post clause is one direct scalar-local assignment;
- admitted header expression trees contain literals, paths, grouping, explicit casts, prefix
  `+`, `-`, `!`, or `~`, and existing binary operators only;
- admitted header value types are `bool`, `i32`, `u8`, `u32`, and `usize`;
- calls, borrows, indexing, blocks, propagation, resource movement, and other effectful or
  ownership-relevant header behavior are rejected as unsupported before MIR.

The initializer owns one lexical loop scope. Its binding is visible to the condition, body, and
post clause and is not visible after the loop. The body retains its own nested lexical scope.

Execution order is initializer once, condition, body, post, then condition again. A false condition
exits without executing the body or post clause. `continue` first performs cleanup for scopes exited
from the body, then transfers to the post clause. `break` performs the required cleanup and transfers
directly to loop exit without executing the post clause. Labelled and value-producing loop control
remain absent.

Lossless CST nodes preserve separate initializer, condition, post, and body regions. AST and typed
HIR retain an explicit `ForLoop`. MIR lowers the construct into the existing verified CFG and emits
an eliminated-form fact for `classic-for`; no backend or interpreter receives a source-level `for`.
The existing ownership, Result, and borrow fixed-point analyses model the admitted subset as the
equivalent scalar `while`, because valid headers cannot change those states.

`for` is reserved only in the current lexer. The previous-revision lexer keeps it as an identifier
so the standalone migrator can parse old input; migration fails without writing when an old use of
that identifier cannot be represented after reservation.

This implemented cutover increments the exact language pin from `fairyc-language/0` to
`fairyc-language/1`. Current starter manifests, examples, SDK metadata, workspace fixtures, and
current cache inputs use `/1`; historical ADRs and historical-version fixtures keep their recorded
values. There is no released compiler version source in this prototype to bump.

## Rejected alternatives

- A range `for`, iterator protocol, or implicit desugaring is not bundled into this decision.
- `++`, `--`, the comma operator, multiple declarations, and assignment expressions are not added.
- Reusing `while` in AST/HIR would lose clause roles needed for diagnostics, source tools, and Atlas.
- Sending `continue` directly to the condition would silently skip the written post clause.
- Allowing arbitrary header calls or resource operations would require new effect and Flow
  fixed-point contracts and is not inferred from the scalar counter-loop feature.

## Evidence

- lexer tests distinguish the current keyword from the previous-revision identifier;
- parser and AST tests preserve complete and omitted clause shapes and exact reconstruction;
- semantic tests accept typed scalar counters and reject escaped initializer scope, calls, and
  non-scalar header state;
- MIR fixtures require verified elimination of `classic-for`;
- interpreter tests prove omitted-condition behavior, `continue` through post, and `break` skipping
  post;
- C-emission fixtures carry the same counter program through deterministic C11 generation.

## Consequences

Fairy C gains the familiar C counter-loop silhouette without inheriting C's header mini-language.
The price is a deliberately narrow first revision: programs needing calls or ownership-changing
work in the header must spell that work in the body with `while` until a separate contract proves a
safe extension.
