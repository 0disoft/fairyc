# Scanner process-CPU compiler-cost control

Date: 2026-09-27. Protocol fixed at child HEAD `f59995c2` before any
process-CPU result. Earlier wall-time nulls failed and retain their verdicts.
This new metric tests whether off-CPU scheduling explains some of their
variation; it does not replace the complete-build gate.

The Windows test loads the checked-in scanner package graph afresh, runs the
existing indexed or private vector analysis lane through Flow and verified
MIR, then emits one Windows AMD64 C11 unit. No MSVC process is started. The
benchmark main and scanner source hashes, Go version, generated-C hash and
length, function count and specialization count are retained for every build.
`GetProcessTimes` supplies cumulative user plus kernel CPU time for the whole
Go process in 100 ns units. A before/after difference encloses source
preparation, load, analysis, MIR lowering and C emission; it excludes report
serialization, hashing, native compile/link and process startup. Wall-clock
total and check/emit phases are recorded as diagnostics. `GOMAXPROCS=1` is
required, but Go GC and other process threads remain part of the CPU total.

Run the identical-source null first: A and B both use the explicit-index
scanner and the same stable compiler path. Pair zero is warmup. Twenty-four
measured pairs alternate `A-B-B-A` and `B-A-A-B`, for 100 fresh builds in all.
Sum the two B CPU times in each pair and divide by the two A CPU times. Keep
every row and pair. The null passes only if source hashes and all generated-C
hashes, lengths, function and specialization counts match and both the 7th and
18th sorted measured ratios fall within `[0.98, 1.02]`. The full relative
span is reported without excluding outliers. A failed null ends this protocol
without a candidate run.

Only after a passing null, run a separate 100-build candidate comparison with
the same order and metric, replacing B with the private three-loop vector
scanner. Its incremental Go-process CPU screen passes when the 18th sorted
vector/index ratio is at most `1.02`. Compare phase wall times and artifact
counts for context, but do not select a favorable phase to override the CPU
decision. This screen is not a confidence interval, a complete-build verdict,
an independent-C comparison, or permission to admit the source form. Native
MSVC time, stable runtime, and the existing 8% limits remain separate gates.
