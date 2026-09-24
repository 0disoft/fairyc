# Private vector-loop lexer size comparison (2026-09-24)

One local Windows AMD64 MSVC 14.51.36231 run of
`fairyc_measure_vector_lexer_size` compared the checked-in lexer scanner
(`scanner.fy` SHA-256
`ee8ff097e70effa01565b0f96af8e365ae1be3482b300050d0cd35b964005e8a`)
with four compiler-private source variants. Each variant changed one or all
three read-only SDK vector loops; the original used explicit indices. All
variants passed source analysis, Flow and verified MIR. On empty input and a
cooked-number plus unterminated-comment input, low and optimized native runs
had identical console output. Storage totals matched the Go lexer observation,
and the allocator probe observed no live SDK allocation after return.

| Profile | Source | Generated C bytes | Functions | Foreign symbols | Specializations | `unit.obj` bytes | EXE bytes |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: |
| low | explicit index | 383,365 | 81 | 13 | 21 | 125,026 | 219,136 |
| low | token loop | 383,482 | 81 | 13 | 21 | 125,058 | 219,136 |
| low | cooked-buffer loop | 383,666 | 81 | 13 | 21 | 125,090 | 219,136 |
| low | diagnostic-entry loop | 383,666 | 81 | 13 | 21 | 125,090 | 219,136 |
| low | all three loops | 384,076 | 81 | 13 | 21 | 125,170 | 219,136 |
| optimized | explicit index | 383,365 | 81 | 13 | 21 | 87,518 | 144,896 |
| optimized | token loop | 383,482 | 81 | 13 | 21 | 87,534 | 144,896 |
| optimized | cooked-buffer loop | 383,666 | 81 | 13 | 21 | 86,193 | 144,896 |
| optimized | diagnostic-entry loop | 383,666 | 81 | 13 | 21 | 87,905 | 144,896 |
| optimized | all three loops | 384,076 | 81 | 13 | 21 | 87,542 | 144,896 |

The combined candidate added 711 generated-C bytes and 24 optimized object
bytes against the explicit-index Fairy source. The EXE size did not change in
either profile. These are one-run artifact observations, not timing or
independent-C results. The cumulative 8% C-relative runtime and EXE budget,
check/build time, normalized executable MIR and unused-feature overhead remain
open gates for source admission.
