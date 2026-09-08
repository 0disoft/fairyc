# Generic growth calibration v0

- Status: Bounded local measurement harness; no performance threshold or broader generic admission
- Requirement: issue 27, compile-time and code-size growth against planner budgets
- Harness: `internal/compiler/generic_growth_test.go`, `TestGenericGrowthMeasurementMSVC`

The configured `fairyc_generic_growth_measure` lane emits a JSON receipt in verbose test output.
Four source shapes instantiate one two-parameter identity function with 1, 2, 4, or 8 distinct
argument tuples, calling each twice. The value type is `i32`, `u32`, `usize`, or `u8`; the unused
marker is `bool` or `i32`. These are implemented HIR types, not the larger syntax vocabulary.
The 1-tuple shape is the scaling baseline, not a handwritten non-generic
performance comparator. Three repetitions alternate ascending/descending shape order. Each shape
compiles without compiler artifact caching and runs as 42 under both low and optimized MSVC. The
planner policy allows 8 total specializations, 8 per declaration, and depth 1. Counts must equal the
distinct tuple count, not the source call count. Total and per-declaration limits of 7 independently
reject the largest shape without publishing MIR.

The `fairyc.generic-growth-calibration/v0` receipt records the exact test executable identity, Go
and MSVC versions, OS/architecture, CPU count, GOMAXPROCS, source digests and bytes, policy, order,
zero warmups, three repetitions, and raw samples. No source text, local paths, environment dump,
binary contents, or tool arguments are published. It is an internal output-only receipt, not a
public interchange schema or compiler-release identity.

Monotonic nanosecond intervals are separate:

- `specialization_to_verified_mir_ns` starts after parsing and covers source specialization through
  ordinary semantic/Flow/MIR verification. It excludes source generation, parsing and tool discovery.
- `c_emission_ns` covers emission of the verified MIR to a C unit.
- `native_build_and_inspection_ns` includes temporary source I/O, compiler/linker execution, and
  artifact reads/inspection. It excludes executing the finished program.

The first two intervals are shared observations repeated on the two profile rows, not two independent
compiler samples. There are 12 compiler samples and 24 native builds. Do not sum duplicated intervals
or infer CPU time, full CLI latency, warm-cache throughput, or a percentile from this small sample.
Ambient OS/toolchain caches, scheduling, other host work, and processor frequency are uncontrolled.
No failed run is rerun to obtain favorable timing; its test failure invalidates the full calibration.

Size fields distinguish emitted C bytes, the complete generated `unit.obj`, the sum of raw sizes of
its COFF code sections, and the final PE file. The code-section total includes the entry function,
alignment and any compiler-generated code; it is not per-specialization instruction attribution.
The PE includes the scalar runtime and linked libraries. Optimization may fold or inline trivial
identities, so native size need not grow monotonically. No time or size threshold is a correctness
assertion. This synthetic calibration does not justify default-budget tuning, asymptotic complexity,
representative workload performance, resource-element containers, or target promotion.

## Local observation, 2026-09-07

One successful configured run on Windows/amd64, Go go1.26.4, MSVC 14.51.36231,
12 logical CPUs/GOMAXPROCS 12, produced the following three-sample ranges. The test executable was
`fairyc.compiler-executable-sha256/v1:c02bd2ded3414167741b839bcb8797cdf5f8237e6ef58ea9375747ec1a4d01cc`.

| Specializations | C unit bytes | Low code-section bytes | Optimized code-section bytes | Specialization-to-MIR ms, min–max |
| --- | --- | --- | --- | --- |
| 1 | 1413 | 238 | 21 | 0.760–1.698 |
| 2 | 2405 | 430 | 26 | 0.554–0.729 |
| 4 | 4487 | 846 | 38 | 0.608–1.618 |
| 8 | 8623 | 1742 | 60 | 1.109–2.105 |

All 24 executions returned 42 with empty stdout/stderr. Both independent over-budget cases rejected
publication. Code-section totals were identical across the three repetitions. Whole-object sizes
occasionally varied despite equal code-section sizes; object metadata is not instruction size.
Optimized PE files were 107,520 bytes at every shape, illustrating linker/runtime overhead rather
than zero generic growth. Some raw emission intervals were 0 ns: interpret zero as below the observed
clock resolution, never zero work. These samples do not establish latency bounds or justify a
performance regression threshold.

## Aggregate/manual comparison, 2026-09-08

`TestGenericAggregateGrowthComparisonMSVC` in
`internal/compiler/generic_aggregate_growth_test.go` adds the configured
`fairyc_aggregate_growth_measure` lane. It compares a generic nested `Cell<T>` /
move-only `Token<T>` factory-and-consumer chain with explicitly named concrete
declarations generated from the same small workload template. It selects 1, 3 or
5 types from `bool`, `u8`, `i32`, `u32`, `usize`, creates and consumes each twice,
checks payload equality, and returns 42. Each type contributes exactly four
specializations: struct, resource, factory and consumer. Both paths independently
pass ordinary semantic/Flow/MIR verification and interpreter execution, including
exactly two drops per type. Both MSVC profiles must execute as 42 without output.

Two repetitions reverse both shape and generic/manual order; profiles remain
low then optimized. There are 12 compiler observations and 24 native executions.
Parsing, source generation, package-coordinate construction and toolchain discovery
are outside the check interval. Generic checking includes specialization; manual
checking does not. C emission and native build/inspection have separate intervals.
Each compiler interval appears on both profile rows and must not be counted twice.
There are no warmups, artifact caches, parallel samples or timing thresholds;
ambient OS/toolchain caches and other host activity are uncontrolled. This is a
module-local compiler comparison, not full workspace/CLI latency.

The output-only `fairyc.aggregate-growth-comparison/v0` receipt carries source
digests, executable identity, tool versions, planner limits, exact counts, source/C/PE
byte sizes and raw timings. The successful local run used test executable identity
`fairyc.compiler-executable-sha256/v1:676f3a3d26d5ee791b1d419b9d527cda68c9cb0c6a24f0c991d98305612007d8`.
The observed sizes were identical across both repetitions:

| Types | Generic/manual source bytes | Generic/manual C bytes | Low PE bytes, both | Optimized PE bytes, both |
| --- | --- | --- | --- | --- |
| 1 | 424 / 395 | 3023 / 3027 | 108544 | 107520 |
| 3 | 600 / 1078 | 8331 / 8331 | 109056 | 107520 |
| 5 | 788 / 1770 | 13700 / 13702 | 110080 | 107520 |

This demonstrates source deduplication for this bounded aggregate pattern without
material whole-file size inflation relative to manual expansion. It does not
establish a latency bound, asymptotic complexity, or a performance regression gate.
The two source forms share the workload template and compiler frontend/backend;
they are not independent compiler lineages. Empty cleanup and constant payloads
permit aggressive optimization. PE size includes runtime/linker overhead and is
not per-specialization instruction size. Native exactly-once observable cleanup
is covered separately by ADR 0387's scalar-domain counter fixture, not by the empty
cleanup used here. Representative application workloads, vector growth, allocation
failure and resource-element containers remain separate admission evidence.
