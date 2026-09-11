# Candidate coverage after assertions, slices, discard and resources

All thirteen current sources were passed unchanged to the private candidate
parser under low and optimized Windows MSVC profiles. The rows matched between
profiles. Eleven sources are diagnostic-free, up from seven in the
[operand snapshot](parser-coverage-operands-2026-09-11.md). No source was excluded
and no resource limit was raised.

| File | Bytes | Full status | Declarations | Problems | First blocker |
| --- | ---: | ---: | ---: | ---: | --- |
| attributes.fy | 4136 | 0 | 13 | 0 | none |
| bodies.fy | 28620 | 0 | 30 | 0 | none |
| budget.fy | 574 | 0 | 4 | 0 | none |
| diagnostics.fy | 3982 | 0 | 14 | 0 | none |
| main.fy | 2916 | 0 | 9 | 0 | none |
| numbers.fy | 9859 | 0 | 12 | 0 | none |
| pages.fy | 3949 | 0 | 14 | 0 | none |
| records.fy | 4019 | 0 | 10 | 0 | none |
| scanner.fy | 20161 | 0 | 42 | 1 | contextual variant `.err`, dot at 17285..17286 |
| strings.fy | 9355 | 0 | 15 | 0 | none |
| syntax.fy | 25304 | 0 | 64 | 1 | contextual variant `.err`, dot at 10539..10540 |
| types.fy | 8149 | 0 | 18 | 0 | none |
| utf8.fy | 807 | 0 | 3 | 0 | none |

Ranges are half-open byte spans. Counts include module/import items and recovery
diagnostics, not independent unsupported features. Status 0 with problems means
an incomplete syntax tree, not compiler acceptance. No full-file budget exhaustion
or separate prefix probe occurred. The unchanged source-size cap is 32768 bytes.

## Delivered boundary

- Assert statements preserve a required parenthesized condition and terminating
  semicolon. Discard statements preserve a required expression and semicolon.
  Neither becomes a return entry, call or value-block tail; both share existing
  expression, statement and nesting budgets.
- Range slices preserve receiver, optional bounds, the `..` token and source
  spans. Omitted bounds remain absent IDs, not numeric defaults. Ordinary index
  shape stays unchanged and all four bound combinations are admitted.
- Non-generic resources reuse struct field/type storage and the global 1024-field
  bound. Declaration kind 6 distinguishes resources. Cleanup may be absent or
  explicitly empty; separate spans retain that distinction. Nonempty cleanup,
  cleanup attributes and generic resources remain unsupported. Failure rolls back
  fields, types and declaration attributes, including recovery to a resource.

No normative grammar, stable compiler API, SDK, runtime, DB or version source
changed. The candidate does not implement assertion execution, borrow checks,
must-use rules, destruction, semantic analysis or compiler promotion.

## Verification

Thirteen focused tests passed on the final implementation:

- `fairyc_go_test_parser_completion`: assertions, slices, discard and resources;
  Go syntax acceptance, exact spans/tree shape, omission states, malformed rollback,
  shared field/statement budgets, exact slice-node and depth boundaries.
- `fairyc_go_test_parser_syntax`: types, visibility, literals, record expressions
  and attributes under both profiles.
- `fairyc_go_test_parser_surface`: diagnostics, postfix, structs and this complete
  own-source measurement under both profiles.

The first surface run failed because its old negative fixtures still rejected
`a[1..2]` and `a[..]`. They were changed to malformed ranges `a[1..2..3]` and
`a[..,]`, while the new slice test verifies acceptance and structure for all four
valid forms. The surface rerun passed. Old blanket resource-rejection fixtures
now exercise unsupported generic resources; the new resource test covers simple
resource acceptance and unsupported cleanup rollback. Existing positive postfix
oracles and budget assertions were preserved.

Go formatting, diff whitespace checks and child-scoped strict Mustflow checks
passed. A focused completion test lane was added to the parent command contract;
no background runner, dependency or network permission was introduced. The large
legacy integration probe and full repository/remote CI suites were not rerun for
this bounded local change. No child documentation validator is configured; docs
were reviewed against source and the native measurement. Nothing was pushed.
