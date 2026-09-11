# Complete own-source syntax: bounded candidate milestone

All thirteen current source files were passed unchanged to the candidate parser
under low and optimized Windows MSVC profiles. Every file returned status 0 and
zero syntax problems, with matching counts across profiles. No file was excluded,
no limit was raised, and no prefix probe was needed. The previous
[statement/resource snapshot](parser-coverage-statements-2026-09-11.md) admitted
eleven of these thirteen files.

| File | Source bytes | Declarations |
| --- | ---: | ---: |
| attributes.fy | 4136 | 13 |
| bodies.fy | 30070 | 30 |
| budget.fy | 574 | 4 |
| diagnostics.fy | 3982 | 14 |
| main.fy | 2916 | 9 |
| numbers.fy | 9859 | 12 |
| pages.fy | 3949 | 14 |
| records.fy | 3547 | 10 |
| scanner.fy | 20161 | 43 |
| strings.fy | 9355 | 15 |
| syntax.fy | 25755 | 65 |
| types.fy | 8149 | 18 |
| utf8.fy | 807 | 3 |

Counts include module/import declarations. The 32768-byte source limit remains
unchanged. This proves complete syntax recognition of these sources, not all
Stage 0 syntax, full tree equivalence, semantic validity or self-hosted compilation.
The native driver still uses the existing Go compiler to build the Fairy candidate.

## Delivered changes

- Contextual variant names have a distinct kind-28 node retaining the leading-dot
  span and name token. Existing call nodes retain `.ok(value)` and `.err(status)`.
  Variant resolution and contextual record constructors remain out of scope.
- The corpus test now requires complete syntax for the thirteen named sources.
  Diagnostics, exhaustion and missing files fail the test even when both profiles
  agree. New source files remain observations until explicitly admitted.
- Cleanup bodies reuse the existing statement parser without introducing a module
  cycle. The syntax facade coordinates fields, cleanup and the final resource
  delimiter; `cleanup_body` points into the shared statement arena. Absent and
  empty cleanup remain distinct. Entry ranges, rollback and budgets are shared
  with functions; destruction and ownership semantics are not implemented.
- Repetition arrays use kind 29 with one value and one integer/path count node.
  They are not expanded or evaluated. Arithmetic, calls, casts and grouping in
  counts remain unsupported; ordinary list arrays retain their previous shape.

## Verification

The final source passed eleven focused tests in both Windows profiles:

- `fairyc_go_test_parser_closure`: variants, cleanup bodies and repeated arrays;
  Go syntax acceptance, spans/tree shapes, malformed rollback, shared 4096-node
  boundaries and nesting limits.
- `fairyc_go_test_parser_operands`: existing generic calls, references, casts and
  array lists, including the updated missing-repeat-count rejection fixture.
- `fairyc_go_test_parser_surface`: diagnostics, postfix, structs and the complete
  thirteen-file regression gate above.

`fairyc_go_test_parser_completion` also passed assertions, discard, resources and
slices before the final repetition-array increment. Go formatting, diff whitespace
checks and child-scoped strict Mustflow checks passed. No configured test failed
during this delivery. Resource rejection of a now-supported nonempty cleanup was
replaced with malformed `assert()` cleanup; old repeat rejection `[x;n]` became
missing-count `[x;]`. New tests positively cover both newly admitted forms.

The large legacy integration probe and full repository/remote CI suites were not
rerun for this bounded local delivery. No child documentation validator is
configured; docs were reviewed against implementation and native observations.
Normative grammar, stable compiler API, SDK, runtime, DB and version sources were
unchanged. The parent gained a bounded focused test intent and exact commit scope,
not dependency, network or background-process authority. Nothing was pushed.
