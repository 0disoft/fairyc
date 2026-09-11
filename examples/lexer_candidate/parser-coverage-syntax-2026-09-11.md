# Candidate own-source coverage after literal, record and attribute support

All thirteen current `src/*.fy` files were passed unchanged to the candidate
parser in both low and optimized Windows MSVC native profiles. The rows matched
exactly. The configured `fairyc_go_test_parser_surface` lane measures this corpus;
`fairyc_go_test_parser_syntax` covers the new syntax and its storage boundaries.

Two files are diagnostic-free: `budget.fy` and `utf8.fy`. The previous
[declaration/type report](parser-coverage-types-2026-09-10.md) measured one of eleven.
Two extracted modules increased the inventory, so percentages are not a
like-for-like performance or coverage comparison.

| File | Bytes | Full status | Declarations | Problems | First blocker |
| --- | ---: | ---: | ---: | ---: | --- |
| attributes.fy | 4136 | 0 | 11 | 2 | `if`, 2375..2377 |
| budget.fy | 574 | 0 | 4 | 0 | none |
| diagnostics.fy | 3982 | 0 | 7 | 7 | `pub resource`, pub at 93..96 |
| expressions.fy | 8895 | 0 | 13 | 3 | `switch`, 2692..2698 |
| main.fy | 2916 | 0 | 6 | 3 | fixed-array length `1`, 229..230 |
| numbers.fy | 9859 | 0 | 8 | 4 | `switch`, 288..294 |
| pages.fy | 3949 | 0 | 8 | 6 | fixed-array length `1`, 312..313 |
| records.fy | 3139 | 0 | 8 | 2 | `if`, 2246..2248 |
| scanner.fy | 20161 | 0 | 24 | 19 | `if`, 1761..1763 |
| strings.fy | 9355 | 0 | 10 | 5 | `switch`, 456..462 |
| syntax.fy | 30418 | 4 | unavailable | unavailable | prefix probe: `pub resource`, pub at 1610..1613 |
| types.fy | 5525 | 0 | 12 | 3 | `switch`, 2635..2641 |
| utf8.fy | 807 | 0 | 3 | 0 | none |

Spans are half-open source-byte ranges. Declaration counts include module/import
items; problem counts reflect recovery, not independent unsupported features.
Status 0 with problems means a partial tree, not complete grammar acceptance.
Status 4 publishes no tree, so its zero transport counters are unavailable data.
For that row only, the first blocker comes from a complete-top-level source
prefix ending at an independent Go parser boundary, not a full-file diagnostic.

The expression extraction reduced `syntax.fy` from 31017 to 26373 bytes before
the additions. Its final 30418 bytes stay below the unchanged 32768-byte lexer
admission. Pratt parsing lives only in `expressions.fy`; the syntax facade owns
declaration rollback. No second expression recognition path was retained.

Focused native checks cover raw and canonical cooked string views, malformed
lexical rejection, record initializer order/spans, nesting/postfix use, exact
4096-node admission and exhaustion, Go attribute shape/span comparisons, rejected
prefix recovery, and exact 256-attribute/1024-argument limits. Existing postfix,
diagnostic, type, struct and visibility probes also pass in both profiles.
The first combined extension run hit its 180-second limit after the generic
probe passed; its unfinished type/visibility checks were completed separately.
No tests or assertions were removed to accommodate that timeout.

Remaining first-blocker families include fixed arrays, value-producing `if`,
`switch` and resource declarations. Generic records, type/layout validation,
effect interpretation and complete Stage 0/compiler admission remain excluded.
No lexer storage limit, stable compiler API, SDK or runtime contract changed.
