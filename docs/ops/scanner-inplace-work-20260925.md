# Scanner in-place work-budget experiment

Date: 2026-09-25. Baseline child HEAD:
`4708b1b75309fd09df8f27ea7a2f3b52cd244e31`.
The lexer candidate keeps the same `remaining` and `exhausted` state, but
`budget::charge` updates a mutable local instead of returning a new `Work`
aggregate at every charge. Call sites still check exhaustion immediately and
preserve their existing exit paths. This is example-source code, not a
language, SDK, or runtime change.

The configured C/Fairy source-work, 38-input lexical-parity, and combined
work/token/diagnostic-limit tests passed. The optimized four-image smoke
checked one reporter execution, 1,000 completed rounds, and checksum in each
lane. Its [receipt](evidence/scanner-inplace-work-smoke-20260925.json) records
the image hashes, MSVC version, source lengths, specialization counts, and
single-run QPC values.

| Image | Before EXE bytes | After EXE bytes | After source bytes |
| --- | ---: | ---: | ---: |
| Up-front C | 184,320 | 184,320 | 52,164 |
| Geometric C | 185,344 | 185,344 | 52,245 |
| Explicit-index Fairy | 204,800 | 203,776 | 399,908 |
| Private vector-loop Fairy | 204,800 | 203,776 | 400,736 |

Both Fairy images shrink by 1,024 bytes, with 21 specializations each. The
up-front C-relative EXE ratio is 1.10556. At the current 512-byte alignment,
at least 5,120 more Fairy EXE bytes must be removed to reach the 8% size
ceiling of 198,656 bytes. The one-thousand-round QPC observations are a smoke
check, not a stable runtime comparison. Check/build cost, complete C scanner
parity, unused-feature overhead, and the cumulative runtime gate remain open.
The vector source form stays compiler-private.
