# Scanner process-cycle control protocol

Date: 2026-09-27. This protocol and its opt-in harness precede the first
measurement. The earlier GC-controlled process-CPU null remains failed; this
experiment does not reclassify its 200 readings or change its thresholds.

Use the same checked-in indexed scanner, private three-loop vector rewrite,
benchmark main, and Windows AMD64 C11 emitter as the process-CPU control.
Require `GOMAXPROCS=1`. Force Go GC before each timed build and disable
automatic GC during the run. `QueryProcessCycleTime` brackets fresh source
preparation, load, semantic analysis, verified-MIR lowering and C emission.
It counts user and kernel cycles for **all threads of the Go process**, not
wall time, native MSVC work, or a fixed-frequency duration. Record the existing
process-CPU, wall, check and emit durations as diagnostics; retain the source
and generated-C hashes, byte lengths, function and specialization counts.

Run an identical-source null first. Both lanes use the indexed source and the
same compiler path. Pair zero is warmup. For the next 24 pairs, alternate
`A-B-B-A` twice and `B-A-A-B` twice: four builds per lane per pair, 200 builds
in total. Retain every row, with no exclusions or adaptive retries. Sum four
process-cycle deltas per lane before dividing B by A for each measured pair.
Sort the 24 ratios. The null passes only if all source and generated-C
identities, sizes, function counts and specialization counts match and the 7th
and 18th ratios lie in `[0.98, 1.02]`. Report the median and full span too.
A failure ends this protocol; do not run the candidate mode.

Only after a passing null, run the same schedule with the private vector source
in lane B. Its incremental cycle screen passes when the 18th sorted
vector/index ratio is at most `1.02`. The source hashes must differ, while each
build must emit 80 functions and 21 specializations. Report raw rows and every
ratio. Cycle counts can change with frequency and thread scheduling even for
identical work. Passing this local screen cannot establish elapsed check/build
latency, normal-GC cost, independent-C runtime, or stable source admission.
