# Scanner numeric work-borrow experiment

Date: 2026-09-25. Baseline child HEAD:
`7d6ea050ffdd687f1593cb7540e36adea81088f1`.
The lexer candidate's numeric scanner and decimal materializer now mutate a
shared `Work` reference. Their result records return only the end, kind and
status needed by the caller; the work count no longer travels through `Cook`
and `Scan` return aggregates. Every charge failure still returns status 5
before consuming or materializing further bytes. This changes example source,
not language, SDK or runtime behavior.

Configured source-work, 38-input C/Fairy lexical-parity and combined
work/token/diagnostic-limit tests passed. The optimized four-image smoke
checked reporter output, 1,000 completed rounds and checksum for each lane.
Its [receipt](evidence/scanner-numeric-work-borrow-smoke-20260925.json)
records image hashes, MSVC version, generated-source lengths and
specialization counts.

| Image | Before EXE bytes | After EXE bytes | After source bytes |
| --- | ---: | ---: | ---: |
| Up-front C | 184,320 | 184,320 | 52,164 |
| Geometric C | 185,344 | 185,344 | 52,245 |
| Explicit-index Fairy | 203,264 | 202,752 | 398,487 |
| Private vector-loop Fairy | 203,264 | 202,752 | 399,315 |

Each Fairy EXE shrinks by 512 bytes and generated C by 3,774 bytes; the
specialization count remains 21. The up-front C-relative EXE ratio is 1.10.
At 512-byte file alignment, at least 4,096 more Fairy EXE bytes must be
removed to reach the 8% size ceiling of 198,656 bytes. Generated-source
reduction is not a measured check/build-time improvement. The single
one-thousand-round QPC smoke is not stable runtime evidence. Complete C
scanner parity, paired check/build cost, unused-feature overhead and the
cumulative runtime gate remain open. The vector source form stays private.
