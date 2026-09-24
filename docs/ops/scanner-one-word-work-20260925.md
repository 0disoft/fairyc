# Scanner one-word work-budget experiment

Date: 2026-09-25. Baseline child HEAD:
`82c1919df175f7e5308dd9ed4faec77ca8bc4d7b`.
The lexer candidate's `Work` now stores only `remaining`. `charge` returns a
boolean and subtracts on success; each caller immediately takes its existing
work-exhaustion exit on failure. No failing call needs to carry an exhausted
flag into later recognition. The decimal expansion overflow path still returns
status 5 without attempting materialization. This is example-source code, not
a language or SDK change.

Configured source-work, 38-input C/Fairy lexical-parity, and combined
work/token/diagnostic-limit tests passed. The optimized four-image smoke
checked the reporter, 1,000 completed rounds and checksum for each lane. Its
[receipt](evidence/scanner-one-word-work-smoke-20260925.json) includes image
hashes, MSVC version, generated-source lengths and specialization counts.

| Image | Before EXE bytes | After EXE bytes | After source bytes |
| --- | ---: | ---: | ---: |
| Up-front C | 184,320 | 184,320 | 52,164 |
| Geometric C | 185,344 | 185,344 | 52,245 |
| Explicit-index Fairy | 203,776 | 203,264 | 402,261 |
| Private vector-loop Fairy | 203,776 | 203,264 | 403,089 |

Each Fairy EXE shrinks by 512 bytes and retains 21 specializations. Generated
C grows by 2,353 bytes per Fairy form, so this result does not establish a
check/build-time improvement. The up-front C-relative EXE ratio is 1.10278.
With 512-byte file alignment, 4,608 more Fairy EXE bytes must be removed to
reach the 8% size ceiling of 198,656 bytes. A single one-thousand-round QPC
smoke cannot decide the runtime gate. Complete C scanner parity, check/build
cost, unused-feature overhead and stable runtime evidence remain open; the
vector source form stays compiler-private.
