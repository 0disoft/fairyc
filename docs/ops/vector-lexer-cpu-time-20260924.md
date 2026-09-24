# Real-scanner paired QPC and thread CPU time

Date: 2026-09-24. Fairy C source: `19c871ba19d0561601340f3a47c074df656ad008`.
Windows AMD64, optimized MSVC. The [raw report](evidence/vector-lexer-cpu-time-20260924.json)
contains the exact image, source, support and harness hashes and every sample.

The checked-in scanner's explicit-index and compiler-private `Vec<T>` loop forms
run in one image. Each lane performs 250,000 `collect`/`storage` calls on
`let n=12345; /*open` per pair. The token reporter runs outside the timed
section. The independently computed Go result is 36 per scan, or 9,000,000
per timed lane. Both lanes returned it and passed the native allocation cleanup
check. One warmup pair preceded twelve measured pairs with alternating order;
there were no retries, exclusions or adaptive stopping.

The harness brackets each lane with QPC and with [Windows `GetThreadTimes`](https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-getthreadtimes).
The CPU value sums the calling thread's user and kernel time in 100 ns units.
It excludes time descheduled from that thread, but does not control CPU
frequency, migration or shared-machine activity. The same optimized image was
reused throughout and is 142,848 bytes; containing both scanner forms makes
its size unsuitable for a standalone Fairy/C EXE comparison.

| Metric | Indexed median | Candidate median | Candidate/index paired median | Ratio of lane medians | Indexed / candidate / paired relative full span |
| --- | ---: | ---: | ---: | ---: | ---: |
| QPC | 673.86370 ms | 671.68555 ms | 0.99593 | 0.99677 | 1.50% / 3.13% / 2.84% |
| Thread CPU | 671.875 ms | 671.875 ms | 1.00000 | 1.00000 | 4.65% / 4.65% / 6.87% |

The candidate-first and index-first QPC paired-ratio medians are 0.99593 and
0.99948. Both QPC and CPU series meet the prior 10% relative-full-span
stability heuristic in this run. QPC supports near-parity for the two Fairy
spellings on the two timed storage loops. It does not show a meaningful speed
advantage for either form. The CPU readings take only three distinct values per
lane: 656.250, 671.875 and 687.500 ms. Their observed 15.625 ms steps are too
coarse to resolve a sub-percent difference here.

The earlier [same-image QPC run](vector-lexer-execution-20260924.md#same-image-paired-follow-up)
was unstable. This separate run is stable, but the evidence does not establish
that adding CPU-time observations caused the improvement; host conditions were
uncontrolled. Neither run times the token reporter's loop. No independent C
full-scanner lane, unused-feature check/build comparison, or standalone EXE
comparison is included, so the cumulative 8% C-relative admission gate remains
open.
