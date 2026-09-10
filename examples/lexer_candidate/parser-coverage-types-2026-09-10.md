# Coverage after declaration and type syntax

Measured with the public-function, written-type, ordinary-struct and generic-type
extensions in native low/optimized Windows MSVC profiles. Every checked-in
`src/*.fy` file was passed unchanged, and both profiles produced identical rows.
The inventory grew from nine to eleven files because type and field parsing were
split into `types.fy` and `records.fy`, keeping each file below 32768 bytes.

**Complete, diagnostic-free files: 1 of 11 (`utf8.fy`).** The earlier
[baseline](parser-coverage-2026-09-10.md) was 0 of 9; these are different source
inventories, not directly comparable coverage percentages. Clean syntax does not
imply type correctness, executable MIR or self-hosting completion.

| File | Bytes | Full-file result | First blocker |
| --- | ---: | --- | --- |
| budget.fy | 574 | Partial: 2 declarations, 2 problems | Record expression `{`, 210..211 |
| diagnostics.fy | 3982 | Partial: 6 declarations, 11 problems | `pub resource`, `pub` at 93..96 |
| main.fy | 2916 | Partial: 6 declarations, 6 problems | Attribute `@`, 178..179 |
| numbers.fy | 9859 | Partial: 8 declarations, 4 problems | `switch`, 288..294 |
| pages.fy | 3949 | Partial: 8 declarations, 12 problems | Attribute `@`, 261..262 |
| records.fy | 3139 | Partial: 8 declarations, 2 problems | Byte-string literal, 990..994 |
| scanner.fy | 20161 | Partial: 23 declarations, 26 problems | Byte-string literal, 1293..1296 |
| strings.fy | 9355 | Partial: 9 declarations, 6 problems | Record expression `{`, 318..319 |
| syntax.fy | 31017 | Status 4: no published tree | `pub resource`, `pub` at 1496..1499; prefix probe |
| types.fy | 5525 | Partial: 10 declarations, 5 problems | Byte-string literal, 1817..1821 |
| utf8.fy | 807 | Clean: 3 declarations, 0 problems | None |

Spans are half-open source-byte offsets. Totals include module/import declarations
and recovered syntax, not semantic feature coverage. Status 4 publishes no tables;
its transport zeroes mean unavailable counts, not zero problems. For `syntax.fy`,
the first blocker is measured separately on an original prefix ending at a
complete Go-CST top-level declaration boundary. The full-file result remains 4.

## Observed movement

- Visibility support moves `utf8.fy` from `pub` to its reference signature.
- Basic type syntax makes that file clean and moves `numbers.fy` to `switch`.
- Ordinary structs move `main.fy` and `pages.fy` to attributes.
- Generic type arguments move `scanner.fy` past `<` to byte-string expressions.

The next missing syntax families are literals, record expressions, attributes,
switch/pattern syntax and resource declarations. Supporting one does not imply
the others. Duplicate fields, access checks, ownership, layouts, generic
declarations/specialization and name/type resolution remain outside this parser.

## Verification

The configured `fairyc_go_test_parser_extensions` suite passed all seven focused
probes under both profiles: visibility, basic types, structs, generic types,
postfix expressions, diagnostics and complete self-source observations.
Type/field tests cover original spans and Go AST shape, `ref T[]` versus
`(ref T)[]`, fused `>>` versus spaced `> >`, unchanged expression shifts,
ordered generic arguments, failed-declaration rollback, and exact type-depth,
type-node and field limits. This report is not a timing or memory benchmark.

The larger existing parser proof and complete frontend/native CI are separate
validation lanes; their hosted outcome must be checked against the delivered SHA.
Re-run the source probe when syntax changes rather than treating these dated
counts as permanent limits or a required ceiling on improvement.
