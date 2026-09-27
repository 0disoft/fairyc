# Balanced scanner build-cost control and candidate protocol

Date: 2026-09-27. This protocol is fixed before the first balanced-mode run.
The earlier identical-source wall-time controls failed their 10% full-span
rule; affinity alone did not remove the spread. The new within-pair order
`A-B-B-A` alternates with `B-A-A-B` to cancel a roughly linear time trend
within each four-build pair. It does not remove random host load or Go GC.

Lane A is the checked-in explicit-index scanner. In the null control, lane B
is the same source and private compiler lane with a different label. Only if
that null passes, a separate candidate run replaces B with the compiler-private
vector-loop source. Both modes use the same scanner input, benchmark main,
fresh package preparation, Go check and verified-MIR lowering, C emission,
runtime-support assembly, optimized Windows MSVC build, and artifact reads.
The Go test runs with `GOMAXPROCS=1`; affinity and priority are unchanged.
Neither mode requests a persistent compiler cache.

Pair zero is a warmup excluded from decisions. Five measured pairs follow,
with 24 complete builds per mode. Every build records its pair, position,
lane, check and native phase times, specialization count, generated-C length
and EXE length; the receipt records lane source hashes and toolchain version
once. For each measured pair, sum
the two total build times per lane, where one total is `check_ns + emit_ns +
support_ns + native_ns`, then divide B's sum by A's sum. All observations,
including outliers, remain in the receipt. The null also requires equal
source hashes and matching specialization, generated-C and EXE lengths in
all four positions of every pair.

The null passes only if the median of its five pair ratios lies in
`[0.98, 1.02]` and `(maximum - minimum) / median <= 0.10`. A failed null ends
this protocol without a candidate run. If the null passes, the candidate run
uses the same sequence and passes the incremental total-build-cost screen only
if its five-ratio median is at most `1.02` and its full span is at most 10%.
Check, emit and native phase ratios are retained separately to expose a
compensating regression; a total pass alone does not establish each phase's
cost or cold CLI latency. Both runs are single attempts: no pair deletion,
adaptive stopping or rerun to select a favorable result.

Passing these local build screens would still leave independent-C runtime,
unused-feature overhead, general workload and release-pin decisions open.
The previous failed controls remain part of the evidence; this protocol does
not replace or retroactively validate them.
