# Candidate parser own-source coverage

Measured against syntax implementation `ede975eb5826c7509a4b482e20650a661a44d821`
on Windows AMD64 in both low and optimized MSVC profiles. All nine checked-in
`src/*.fy` files were passed unchanged to the native Fairy parser. Both profiles
returned identical observations. No source file was excluded or rewritten into
the admitted subset.

**Complete, diagnostic-free files: 0 of 9.** A successful Result containing
problems is only a partial tree, not compiler acceptance or self-hosting proof.

| File | Bytes | Full-file outcome | First actual token | Byte span |
| --- | ---: | --- | --- | --- |
| budget.fy | 574 | Partial: 1 declaration, 5 problems | `pub` | 98..101 |
| diagnostics.fy | 3982 | Partial: 4 declarations, 17 problems | `pub` | 93..96 |
| main.fy | 2916 | Partial: 4 declarations, 8 problems | `struct` | 91..97 |
| numbers.fy | 9859 | Partial: 6 declarations, 7 problems | `ref` (expected scalar type) | 239..242 |
| pages.fy | 3949 | Partial: 5 declarations, 15 problems | `struct` | 127..133 |
| scanner.fy | 20161 | Status 4: no published tree | `struct`, prefix probe | 306..312 |
| strings.fy | 9355 | Partial: 6 declarations, 10 problems | `pub` | 148..151 |
| syntax.fy | 26477 | Status 4: no published tree | `pub`, prefix probe | 221..224 |
| utf8.fy | 807 | Partial: 2 declarations, 2 problems | `pub` | 148..151 |

Spans are half-open source-byte offsets. Declaration totals include module and
imports; problem totals describe recovery behavior, not independent missing
features. Status 4 discards all tables, so its zero transport counters are
**unavailable**, not evidence of zero problems.

For the two exhausted files, the first blocker comes from separate, original
source prefixes ending at complete top-level CST declaration boundaries supplied
by the Go parser. The native candidate identifies the first problem in those
prefixes. This is not a fabricated full-file diagnostic, and the full-file status
remains 4. Other rows use the full-file first problem directly.

## What the evidence suggests next

1. Recognize visibility and declaration headers without discarding their source
   meaning. `pub` is the first blocker in five files; merely ignoring it would not
   add support for the following declarations.
2. Add a bounded type-syntax layer for qualified, reference, slice and generic
   forms before trying to accept the helpers' full signatures.
3. Then address struct/resource declarations, record expressions and switch
   syntax. Adding while and postfix expressions alone cannot parse this corpus.

These are future syntax work, not new language/SDK admission. Name resolution,
type validation, ownership checking, MIR generation and complete Stage 0 syntax
remain outside this candidate.

## Reproduction and limits

`TestWindowsFairyParserSelfCorpus`, included by the configured
`fairyc_go_test_parser_extensions` intent, compiles a small private stdin probe
once per profile and checks both clean and malformed control programs. It reads
each source directly, rejects symlinks and oversized inputs, validates six bounded
observation fields, and requires exact profile parity. The probe accepts at most
32768 source bytes; it does not raise parser budgets or hide exhaustion.

This is a coverage observation, not a speed or memory benchmark. The lexer still
has separate whole-source exact token observations; lexing all source does not
imply parsing all source. Re-run the probe when candidate syntax changes; these
counts are a dated baseline rather than a permanent expected coverage ceiling.
