# Scanner string state-borrow experiment

Date: 2026-09-25. Baseline child HEAD:
`8b631e63b9154bb1e81adc7b00b761d02a68e759`.
The lexer candidate's string scanner and nested escape scanner now mutate
their caller's work budget and diagnostic count through explicit references.
Their `Step` result contains only the end and status; a successful diagnostic
still increments the count exactly once, and each unsuccessful work charge
still exits before further source consumption. This changes example source,
not language, SDK or runtime behavior.

Configured source-work, 38-input C/Fairy lexical-parity and combined
work/token/diagnostic-limit tests passed. The optimized four-image smoke
checked reporter output, 1,000 completed rounds and checksum for each lane.
Its [receipt](evidence/scanner-string-state-borrow-smoke-20260925.json)
records image hashes, MSVC version, generated-source lengths and
specialization counts.

| Image | Before EXE bytes | After EXE bytes | After source bytes |
| --- | ---: | ---: | ---: |
| Up-front C | 184,320 | 184,320 | 52,164 |
| Geometric C | 185,344 | 185,344 | 52,245 |
| Explicit-index Fairy | 202,752 | 202,240 | 396,788 |
| Private vector-loop Fairy | 202,752 | 202,240 | 397,598 |

Each Fairy EXE shrinks by 512 bytes. Generated C shrinks by 1,699 bytes for
the indexed form and 1,717 bytes for the private vector form; both retain 21
specializations. The up-front C-relative EXE ratio is 1.09722. At 512-byte
file alignment, 3,584 more Fairy EXE bytes must be removed to reach the 8%
size ceiling of 198,656 bytes. Generated-source reduction is not measured
check/build-time improvement, and one thousand QPC rounds do not establish
stable runtime behavior. Complete C scanner parity, paired check/build cost,
unused-feature overhead and the cumulative runtime gate remain open. The
vector source form stays compiler-private.
