# Scanner GC-controlled process-CPU cost protocol

Date: 2026-09-27. Commit this protocol and its opt-in test mode before any
measurement. The earlier process-CPU null failed and keeps its verdict: all
100 durations were quantized to 15.625 ms, and the central ratios missed the
fixed 2% null band. This is a new control, not a rescore of those rows.

Use the current checked-in indexed scanner and its compiler-private three-loop
vector rewrite with the same benchmark main and Windows AMD64 C11 emitter.
`GOMAXPROCS=1` is required. `GetProcessTimes` user plus kernel CPU time encloses
fresh package preparation, load, semantic analysis, verified-MIR lowering and
C emission. MSVC compilation and linking are excluded. Emit source hashes,
generated-C hashes and byte lengths, function and specialization counts,
per-build CPU and wall durations, and check and emit phase wall durations.

For this mode, disable automatic Go GC, explicitly collect before each build,
and require `NumGC` not to change during its timed interval. A collection
outside that interval is not charged. This isolates GC-excluded compiler work;
it does not represent normal-GC check latency or complete build time. The
process-CPU clock remains quantized and includes other Go process threads.

Run an identical-source null first. Both lanes use the indexed source and the
same stable compiler path. Pair zero is warmup. The next 24 pairs alternate
`A-B-B-A` twice and `B-A-A-B` twice: four fresh builds per lane per pair,
200 builds total. Keep every row and pair, with no adaptive exclusions,
retries or threshold changes. For each measured pair, divide the sum of four
B CPU durations by the sum of four A durations. Sort the 24 ratios. The null
passes only if all source and generated-C identities, sizes, function counts
and specialization counts match and both the 7th and 18th ratios lie inside
`[0.98, 1.02]`. Report the median and full span as diagnostics. A failed null
ends this protocol; do not run the candidate mode.

Only after a passing null, run the same 200-build schedule with the private
vector source in B. Its incremental GC-excluded CPU screen passes when the
18th sorted vector/index ratio is at most `1.02`. The two scanner source hashes
must differ, while every build must emit 80 functions and 21 specializations.
Report all ratios and artifact identities. This screen alone cannot admit the
source form: complete native build cost, ordinary check latency, controlled
runtime stability, and the independent-C 8% budget remain separate gates.
