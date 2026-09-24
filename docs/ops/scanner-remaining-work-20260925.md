# Scanner two-field work-budget experiment

Date: 2026-09-25. Source HEAD before the experiment:
`5dc248ca413b74040eb30622dd19108f69fede2c`.
The Fairy-written lexer candidate changes `budget::Work` from `used`, `limit`,
and `exhausted` to `remaining` and `exhausted`. `charge` subtracts only after
checking `units <= remaining`; an already exhausted state remains exhausted.
The unrepresentable decimal path retains the remaining count while marking
exhaustion. This is example-source code, not a language or SDK change.

The configured Windows low/O2 38-input lexical comparison, work-boundary
suite, combined work/token/diagnostic-limit suite, and four-image optimized
smoke passed. Each of the thirteen measured four-image rotations also checked
the reporter, 250,000 completed rounds and checksum. No sample was retried or
excluded. The [raw report](evidence/scanner-remaining-work-four-image-20260925.json)
contains all image identities and QPC samples.

| Image | Before EXE bytes | After EXE bytes | After generated C bytes |
| --- | ---: | ---: | ---: |
| Up-front C | 184,320 | 184,320 | 52,164 |
| Geometric C | 185,344 | 185,344 | 52,245 |
| Explicit-index Fairy | 206,336 | 204,800 | 400,412 |
| Private vector-loop Fairy | 206,336 | 204,800 | 401,240 |

Both Fairy EXEs shrink by 1,536 bytes. Generated C is 1,251 bytes smaller
per Fairy form than the earlier keyword-length image. Against the unchanged
up-front C image, Fairy's EXE ratio is 1.11111, still above the cumulative
1.08 limit. At this 512-byte file alignment, at least 6,144 additional Fairy
EXE bytes must be removed to reach the size gate.

| Paired runtime ratio | Median | Relative full span |
| --- | ---: | ---: |
| Explicit-index Fairy / up-front C | 1.012575 | 13.7124% |
| Private vector-loop Fairy / up-front C | 1.033073 | 11.7452% |
| Private vector-loop / explicit-index Fairy | 1.009409 | 8.9395% |

The predeclared stability rule requires a relative full span at most 10%.
Both Fairy/C ratio series fail it, as does the up-front C lane, so their
low point estimates do not establish a runtime improvement or an 8% pass.
The vector/index series is stable for this fixed input but is not a complete
feature-cost result. Check/build time, unused-feature overhead, complete C
scanner parity and other workloads remain open. The vector source form stays
compiler-private.
