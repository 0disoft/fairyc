# Candidate coverage after generic calls, references, casts and array lists

All thirteen current source files were passed unchanged to the candidate parser
under low and optimized Windows MSVC profiles. Every reported row matched.
Seven files are now diagnostic-free, compared with two in the
[previous value-expression snapshot](parser-coverage-values-2026-09-11.md).
No files were excluded and no resource limits were raised.

| File | Bytes | Full status | Declarations | Problems | First blocker |
| --- | ---: | ---: | ---: | ---: | --- |
| attributes.fy | 4136 | 0 | 13 | 0 | none |
| bodies.fy | 27028 | 0 | 30 | 0 | none |
| budget.fy | 574 | 0 | 4 | 0 | none |
| diagnostics.fy | 3982 | 0 | 11 | 3 | `pub resource`, pub at 93..96 |
| main.fy | 2916 | 0 | 9 | 0 | none |
| numbers.fy | 9859 | 0 | 11 | 1 | range slice `..`, 9604..9606 |
| pages.fy | 3949 | 0 | 13 | 1 | `assert`, 1118..1124 |
| records.fy | 3139 | 0 | 10 | 0 | none |
| scanner.fy | 20161 | 0 | 34 | 9 | range slice `..`, 4122..4124 |
| strings.fy | 9355 | 0 | 14 | 1 | `assert`, 839..845 |
| syntax.fy | 24817 | 0 | 42 | 23 | `pub resource`, pub at 1600..1603 |
| types.fy | 8149 | 0 | 18 | 0 | none |
| utf8.fy | 807 | 0 | 3 | 0 | none |

Ranges are half-open source-byte spans. Counts include module/import declarations
and recovery diagnostics, not independent unsupported features. Status 0 with
problems means a partial tree, not compiler acceptance. No full-file exhaustion
occurred in this measurement, so no separate first-blocker prefix probe was used.
The prior syntax.fy status-4 result remains historical evidence, not a current
result. The lexer input guard remains 32768 bytes.

## Delivered boundaries

- Explicit generic calls use read-only lookahead capped at 256 significant tokens,
  followed by the shared type parser. Written type arguments retain source order;
  nested closing angles do not change ordinary shift/comparison token handling.
- `ref` and `ref mut` retain distinct nodes and their operand. Postfix chains bind
  inside a reference; borrow legality and lifetimes remain semantic concerns.
- `as` preserves its operand and written type root, associates left, and binds
  below prefixes but above multiplication. Conversion validity is not checked.
- Array lists preserve empty lists, ordered elements, nested lists, trailing
  commas and source spans. Repetition `[value; count]` is intentionally excluded.
- New nodes use the existing expression/type arenas, declaration rollback and
  shared depth limits. No stable compiler, SDK, runtime, normative grammar or
  package version was changed; this remains the private syntax candidate.

## Verification

The final source passed thirteen focused tests through these configured intents:

- `fairyc_go_test_parser_operands`: generic calls, references, casts and arrays,
  under both native profiles. Includes Go syntax acceptance checks, exact tree
  shape/source-order assertions, malformed rollback and lookahead/depth/node
  boundaries, including the exact 4096-node limit.
- `fairyc_go_test_parser_syntax`: types, visibility, literals, record expressions
  and attributes under both profiles.
- `fairyc_go_test_parser_surface`: diagnostics, postfix, structs and this
  thirteen-file measurement under both profiles.

Go formatting and the child-scoped strict Mustflow check also passed. One format
attempt was rejected while its newly edited command manifest was still being
sealed; waiting for that exact sync and rerunning succeeded. No source-test
assertions were removed or weakened. The large legacy integration probe and full
repository/remote CI suites were not rerun for this bounded local change. A
child-scoped documentation validator is not configured; documentation was
reviewed against the implementation and measurements. Nothing was pushed.

Full Stage 0 syntax, semantic checking, executable compiler promotion and release
readiness are not claimed. Resource declarations, range slices and assertions
remain visible first blockers in the current own-source corpus.
