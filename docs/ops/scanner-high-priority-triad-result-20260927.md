# High-priority scanner triad result

Date: 2026-09-27. The opt-in priority harness was committed at child HEAD
`11b42662` before measurement. The [protocol](scanner-triad-protocol-20260927.md#highest-priority-follow-up)
kept the optimized MSVC profile, fixed scanner input, 250,000 rounds per call,
9,000,000 checksum, two complete 24-triad C-only sessions, 10% full-span and
2% null-bias limits. The calling thread was pinned, then checked at Windows
`THREAD_PRIORITY_HIGHEST` before warmup. No sample was excluded or retried.

The [C-only raw receipt](evidence/scanner-high-priority-identical-c-triads-20260927.json)
records Go 1.26.4, MSVC 14.51.36231, 10,000,000 QPC ticks/s, and logical
processor 1. Both sessions passed the predeclared null control:

| Session | Middle / surrounding C median | Relative full span | Result |
| --- | ---: | ---: | --- |
| 1 | 1.00166 | 7.47% | Pass |
| 2 | 1.00040 | 4.84% | Pass |

The matching-condition C/Fairy/C test then ran once. Its
[complete raw receipt](evidence/scanner-high-priority-same-image-c-triads-20260927.json)
retains one warmup and all 24 measured triads in each image. Every call returned
the required checksum. The toolchain, profile, rounds and QPC frequency match
the control, but the three processes selected different logical processors:

| Fairy form | Processor | Fairy / surrounding C median | Relative full span | 10% stability |
| --- | ---: | ---: | ---: | --- |
| Explicit index | 8 | 1.02374 | 21.86% | Fail |
| Private vector loop | 0 | 0.99762 | 32.62% | Fail |

Both comparison spans fail even before considering the processor mismatch.
The medians do not establish an 8% runtime result. The combined 148,992-byte
images are not the standalone EXE-size comparison. The previously observed
standalone size pass remains separate; stable runtime and general build-cost
admission remain open. Repeating this condition until it yields a favorable
run would select evidence, so this report makes no retry claim.
