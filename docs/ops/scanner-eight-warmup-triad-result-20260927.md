# Eight-warmup scanner triad result

Date: 2026-09-27. The separate eight-warmup protocol was committed at
`876872af` before measurement. The first full execution hit the host driver's
60-second process limit and produced no timing receipt. Commit `deb34a18`
raised only this bounded host-probe deadline to two minutes. The first complete
run after that change is retained here without a retry or sample exclusion.

The [raw C-only receipt](evidence/scanner-eight-warmup-identical-c-triads-20260927.json)
records two consecutive 24-triad sessions on logical processor 0 at checked
highest thread priority. Eight untimed C/C/C triads preceded each session.
Each call used the fixed input, 250,000 rounds, optimized MSVC profile and
9,000,000 Go-oracle checksum. The receipt records Go 1.26.4, MSVC
14.51.36231, 10,000,000 QPC ticks/s and all 48 measured triads.

| Session | Middle / surrounding C median | Relative full span | Null control |
| --- | ---: | ---: | --- |
| 1 | 0.99733 | 12.48% | Fail |
| 2 | 0.99981 | 26.02% | Fail |

Both sessions exceed the predeclared 10% full-span limit despite medians
within the 2% null-bias limit. Extra startup warmup did not establish a stable
host under this condition. The gated C/Fairy/C full comparison was not run.
These C-only medians do not measure Fairy/C runtime or decide the cumulative
8% limit. The earlier standalone EXE-size pass remains separate, and the
private vector-loop source form remains closed. Further warmup counts or
threshold changes should not be selected from these failures; resolving the
runtime gate needs a demonstrably stable measurement environment or a
different predeclared control method.
