# Stage 0 Grammar and Parser Contract

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-17
- Product source: `docs/product/02-spec.md`
- Planned successor: `docs/adr/0136-c-shaped-source-surface.md`

The grammar named by this ADR remains current until the next pre-1 revision completes the atomic
cutover required by ADR 0136.

## Context

The Stage 0 surface document fixed declarations, modules, ownership-visible syntax, and major exclusions, but it deliberately left source extension, literal spelling, operator grouping, file-to-module identity, and malformed-input recovery open. Starting a parser while those choices remain prose would turn implementation accidents into the language.

## Decision

1. Fairy C source files use `.fy`.
2. The normative prototype grammar is `spec/grammar/stage0.ebnf`. Explanatory prose is derived from that file when spelling or precedence is involved.
3. The first parser is a batch compiler parser over source bytes: UTF-8 decoding, trivia-preserving lexing, compact lossless CST construction, then separate AST lowering. Incremental reparsing remains deferred.
4. Canonical source spans are half-open byte offsets into the original source. Line, UTF-16, scalar, and display-column coordinates are derived views and are never mixed with byte offsets.
5. Items and statements use recursive descent. Expressions use a Pratt parser driven by the binding-power table in `docs/language/0006-lexing-parsing-and-recovery-v0.md`.
6. Prefix, postfix, arithmetic, shift, bitwise, comparison, and logical grouping are fixed. Comparisons are non-associative; chained comparisons are rejected instead of guessed. A lexed `>>` remains a shift in expression context and is consumed as two adjacent generic closers only in type-argument closing context; nested types do not require whitespace.
7. Stage 0 literals include boolean, null raw-pointer, integer, floating, UTF-8 string, byte-string, and C-string forms. Signs are prefix operators rather than literal text. There are no character, raw-string, hexadecimal-floating, octal, or numeric-suffix literals.
8. A package source root maps `path/to/name.fy` to `module path::to::name;`. The mapping is case-sensitive and contains no package coordinate. Stage 0 has one source root per package and no special `mod.fy`, `lib.fy`, or `main.fy` mapping.
9. Recovery may insert a declared zero-width missing token, skip a real unexpected token into an error node, synchronize at a grammar-owned boundary, or exit the production. Every recovery step must advance or consume a bounded repair action.
10. A recovered CST is diagnostic evidence only. Any lexer or parser error prevents AST-to-HIR handoff and executable output.
11. `spec/fixtures/parser/manifest-v0.json` and its source cases are the parser acceptance evidence. Fixture expectations use byte spans and stable diagnostic codes.

## Rationale

Recursive descent plus Pratt parsing keeps the grammar readable and makes recovery local without committing the language to a generated-parser tool. Exact byte spans preserve source truth across UTF-8 and future editor protocols. A direct file mapping avoids hidden module discovery and Windows-versus-Linux case behavior. Non-associative comparisons reject a common typo instead of manufacturing surprising trees.

The literal set covers low-level bytes, static UTF-8 text, raw-pointer nullability, and allocation-free C interoperability without adding a character type, implicit locale conversion, or a second numeric conversion system.

## Consequences

- `.fy` is no longer a proposal inside the Stage 0 prototype contract.
- Parser fixtures may be syntactically valid while intentionally failing later semantic analysis.
- Package coordinates and manifest syntax remain undecided; they select a source root but do not change file-to-module mapping within it.
- Exact parser resource limits remain uncommitted until an implementation can measure work, but the implementation must use one bounded parse budget before it accepts untrusted source.
- Formatter and incremental-parser work must preserve the same token, grouping, span, and recovery contracts rather than defining parallel syntax.

## Revisit conditions

Change a spelling or grouping rule only with an updated grammar, ADR, positive and negative fixtures, diagnostic compatibility review, and a new pre-1 language revision when releases exist.
