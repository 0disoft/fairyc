# Candidate coverage after arrays and value expressions

All thirteen current source files were passed unchanged to the candidate parser
under low and optimized Windows MSVC profiles. Every reported row matched.
`budget.fy` and `utf8.fy` remain the two diagnostic-free files: this delivery
moves the other first blockers forward, not the complete-file count.

| File | Bytes | Full status | Declarations | Problems | First blocker |
| --- | ---: | ---: | ---: | ---: | --- |
| attributes.fy | 4136 | 0 | 11 | 2 | generic call `>`, 2822..2823 |
| bodies.fy | 24584 | 0 | 27 | 3 | generic call `>`, 3144..3145 |
| budget.fy | 574 | 0 | 4 | 0 | none |
| diagnostics.fy | 3982 | 0 | 7 | 7 | `pub resource`, pub at 93..96 |
| main.fy | 2916 | 0 | 6 | 3 | array expression `[`, 239..240 |
| numbers.fy | 9859 | 0 | 9 | 3 | reference expression `ref`, 1283..1286 |
| pages.fy | 3949 | 0 | 8 | 6 | array expression `[`, 322..323 |
| records.fy | 3139 | 0 | 8 | 2 | generic call `>`, 2367..2368 |
| scanner.fy | 20161 | 0 | 25 | 18 | generic call `>`, 2162..2163 |
| strings.fy | 9355 | 0 | 12 | 3 | cast `as`, 635..637 |
| syntax.fy | 23475 | 4 | unavailable | unavailable | prefix probe: `pub resource`, pub at 1600..1603 |
| types.fy | 6405 | 0 | 13 | 2 | generic call `>`, 2735..2736 |
| utf8.fy | 807 | 0 | 3 | 0 | none |

Ranges are half-open source-byte spans. Generic-call classification was checked
against the written `v::push<T>(...)` sites; the private diagnostic itself reports
the unexpected closing token. Counts include module/import declarations and
recovery diagnostics, not independent unsupported features. Status 0 with
problems is only a partial tree. Status 4 publishes no tables; its transport zeros
must not be interpreted as empty successful output. Only that row uses a separate
complete-top-level prefix probe to identify a first blocker.

## Delivered boundaries

- Fixed-array types retain integer or qualified-path lengths without evaluation.
  Array, length and generic nodes share the existing type-node budget.
- `bodies.fy` replaces `expressions.fy` and owns mutually recursive expression,
  block and statement parsing without importing the syntax facade. The facade
  keeps declaration publication, rollback and read-only query compatibility.
- Value blocks retain ordered statements and a separate optional tail expression.
  Source-order entry IDs are reserved before nested initializers; storage order
  does not change externally observed entry order or the 1024-entry bound.
- Conditional values retain conditions and ordered then/else connections,
  including absent else and bounded else-if chains.
- Switch values retain ordered cases, leading-dot variants with zero or one
  identifier binding, and value-block bodies. Whole-value wildcards, record
  patterns, multiple positional bindings and semantic pattern checks stay out.

`syntax.fy` decreased from 30418 to 23475 bytes; `bodies.fy` is 24584 bytes.
Both remain below the unchanged 32768-byte lexer input guard. Storage and active
recursion counters remain shared across expression/body boundaries.

## Verification

The final source passed twelve focused tests through:

- `fairyc_go_test_parser_values`: value blocks, conditional values and switches,
  both native profiles; exact node/entry/depth boundaries and malformed rollback.
- `fairyc_go_test_parser_syntax`: types, visibility, literals, records and
  attributes, both profiles; Go type/attribute oracles and source-order queries.
- `fairyc_go_test_parser_surface`: diagnostics, postfix, structs and this
  thirteen-file measurement, both profiles.

One initial value-block probe duplicated the large existing body-tree oracle and
timed out at 180 seconds during optimized native compilation. The duplicate
invocation was removed from the new focused probe; the original oracle and its
dedicated integration test remain unchanged. That large integration test and the
full repository/remote CI suites were not rerun for this local delivery.

Earlier reports remain historical snapshots, including the
[literal/record/attribute baseline](parser-coverage-syntax-2026-09-11.md).
No source was excluded to raise the complete-file count. No stable compiler API,
SDK, runtime, grammar specification or package version changed. Complete Stage 0
syntax, name/type checking, effect interpretation and executable compiler
promotion remain outside this candidate.
