# Robust scanner triad result

Date: 2026-09-27. The opt-in order-statistic protocol and rank checks were
committed at `94619118` before this measurement. This is its first complete
C/C/C control and one matched C/Fairy/C comparison. No sample was discarded
or retried. Earlier failed controls retain their original verdicts.

The [C-only receipt](evidence/scanner-robust-identical-c-triads-20260927.json)
contains two consecutive 24-triad sessions. Each used 250,000 scanner rounds
per call, eight untimed warmup triads, the fixed unterminated-comment input,
and the 9,000,000 Go-oracle checksum. The optimized MSVC probe ran at checked
highest thread priority on the lowest allowed logical processor, 0. Go was
1.26.4, MSVC was 14.51.36231, and QPC frequency was 10,000,000 ticks/s.

| C-only session | 7th ratio | 18th ratio | Robust null | Full span | Historical null |
| --- | ---: | ---: | --- | ---: | --- |
| 1 | 0.99278 | 1.00080 | Pass | 3.28% | Pass |
| 2 | 0.99634 | 1.00386 | Pass | 6.35% | Pass |

Both entire central intervals fit the predeclared `[0.98, 1.02]` null band,
and both full spans also fit the earlier 10% stability rule. This allowed one
matched comparison under the new protocol.

The [C/Fairy/C receipt](evidence/scanner-robust-same-image-c-triads-20260927.json)
records 24 measured triads per image after seven untimed and one retained timed
warmup triad. Go/MSVC versions, processor, priority, QPC frequency, source
input, optimized profile, rounds and checksum match the control. Each call
returned the expected checksum.

| Fairy source | Median Fairy/C | 17th ratio upper bound | Robust 8% screen | Full span | Historical 10% stability |
| --- | ---: | ---: | --- | ---: | --- |
| Explicit index | 0.98740 | 0.99522 | Pass | 5.37% | Pass |
| All vector `for` | 1.00006 | 1.00908 | Pass | 13.41% | Fail |

The upper order statistic for each Fairy image is below the unchanged 1.08
ratio ceiling. The vector image still fails the old full-span stability rule;
the robust result does not rewrite that verdict. These are adjacent calls on
one host, so correlation can make the nominal order-statistic coverage
optimistic. This result is a local runtime screen under the separately
predeclared protocol, not a general speed guarantee or public admission.

Both combined probe images were 148,992 bytes, but their layout is not the
standalone scanner size budget. The separately measured standalone optimized
Fairy scanner was 197,120 bytes against 184,320 bytes for independent C,
below the integer 8% ceiling of 199,065 bytes. Compiler check/build-time and
remaining source/ownership acceptance gates must be decided separately before
read-only SDK vector iteration can become stable.
