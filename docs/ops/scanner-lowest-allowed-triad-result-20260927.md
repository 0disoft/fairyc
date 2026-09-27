# Deterministic-processor scanner triad result

Date: 2026-09-27. The opt-in lowest-allowed-processor mode was committed at
child HEAD `9fcb1d7d` before this complete run. The
[protocol](scanner-triad-protocol-20260927.md#deterministic-processor-follow-up)
requires two consecutive 24-triad C-only sessions to pass both the 10%
relative-full-span and 2% null-bias limits before a matched C/Fairy/C
comparison. The calling thread selected logical processor 0 and used Windows
`THREAD_PRIORITY_HIGHEST`. Input, optimized MSVC profile, 250,000 rounds per
call and expected checksum 9,000,000 remained fixed. No sample was excluded or
retried.

The [complete raw receipt](evidence/scanner-lowest-allowed-identical-c-triads-20260927.json)
records Go 1.26.4, MSVC 14.51.36231, 10,000,000 QPC ticks/s and all 48
measured triads:

| Session | Middle / surrounding C median | Relative full span | Null control |
| --- | ---: | ---: | --- |
| 1 | 0.99740 | 17.35% | Fail |
| 2 | 0.99638 | 8.93% | Pass |

The first session exceeds the predeclared 10% span limit. The deterministic
C-only control therefore fails as a whole, even though the second session
passes. The gated C/Fairy/C measurement was not run. Neither median establishes
the cumulative 8% Fairy/C runtime result. The previously observed standalone
EXE-size pass remains separate; stable runtime and general build-cost admission
remain open. This result does not admit the private vector-loop source form.
