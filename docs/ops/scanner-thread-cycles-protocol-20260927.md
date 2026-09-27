# Scanner locked-thread cycle control protocol

Date: 2026-09-27. This protocol is fixed before its first measurement. The
failed [process-wide cycle null](scanner-process-cycles-null-result-20260927.md)
remains failed; this experiment changes the measured thread and the batch
length, not that result or its threshold.

Use the checked-in indexed scanner and the private three-loop vector variant
with the same benchmark main and Windows AMD64 C11 emitter. Require
`GOMAXPROCS=1`, lock the measuring Go goroutine to one OS thread, and pin the
process to its lowest currently permitted logical processor. Restore affinity
and release the thread lock at the end. Force Go GC before every build, outside
the measured interval, and disable automatic GC during the run.
`QueryThreadCycleTime` brackets fresh source preparation, loading, analysis,
verified-MIR lowering and C emission on the locked thread. Record process CPU,
wall, check and emit time as diagnostics. Thread cycles omit work on other
threads and off-CPU waits; they are not elapsed build time or fixed-frequency
nanoseconds.

Run an identical-source null first. Both lanes use the indexed source and the
same compiler path. Pair zero is warmup; pairs 1 through 12 are measured.
Alternate `A-B-B-A` repeated four times and `B-A-A-B` repeated four times:
eight builds per lane per pair, 208 builds total. Retain every row without
exclusions, retries or adaptive stopping. Sum eight thread-cycle deltas per
lane for each measured pair, then divide clone by index and sort the 12
ratios. The null passes only if source and generated-C hashes, lengths,
function counts and specialization counts match across all builds, and the
4th and 9th sorted ratios lie in `[0.98, 1.02]`. Report the median and full
span even when the null fails. A failed null ends this protocol.

Only after a passing null, run the identical schedule with the private vector
source in lane B. Its incremental thread-cycle screen passes if the 9th sorted
vector/index ratio is at most `1.02`; source hashes must differ while both
lanes retain 80 emitted functions and 21 specializations. Preserve every raw
row and ratio. A passing screen would support only GC-excluded compiler work
on this locked thread. Normal check latency, native compilation and linking,
the independent-C runtime budget, and stable source admission remain separate
gates.
