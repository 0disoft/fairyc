# Scalar dereference C comparison

Date: 2026-09-24. Compiler source: `44c44c99a3e8d6e30d96b02e6720386c796b91b1`.
Windows AMD64, Go 1.26.4, MSVC 14.51.36231, `GOMAXPROCS=1`, optimized C11
`/W4 /WX /MT /O2 /INCREMENTAL:NO`.

## Question and method

Does Fairy C's plain `*slot = value` through `ref mut u32` fit the cumulative
8% runtime and EXE budget against an equivalent hand-written C pointer write?
The `deref` pair runs the checksum recurrence through a small pointer/reference
function. The existing `checksum` and `classifier` pairs run alongside it to
expose whole-run drift. Each lane reads seed and rounds from stdin, guards
`seed < 65521` and `rounds <= 50000000`, starts the same QPC timer after input,
and returns a result checked against an independent Go calculation. The C
arithmetic is proven in range for admitted inputs; Fairy retains its ordinary
checked arithmetic.

For each workload and lane, one build/run warmup precedes five alternating
measured pairs. The report includes 30 measured samples and 30 boundary
executions. Runtime is kernel-only; build time includes source load and Fairy
translation where applicable, C compilation/link, artifact reads, and EXE
write. Common warmed toolchain discovery, workspace setup, process startup,
input parsing, and output are outside the corresponding timed span. Raw
measurements and source hashes are in the [evidence JSON](evidence/c-budget-scalar-deref-20260924.json).

## Observations

| Workload | C runtime median | Fairy runtime median | Fairy/C ratio of medians | Median paired ratio | C / Fairy EXE | EXE delta |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| checksum | 326.804 ms | 303.221 ms | 0.92784 | 0.92680 | 167936 / 168960 B | +0.610% |
| classifier | 234.270 ms | 237.377 ms | 1.01326 | 1.00265 | 168448 / 168960 B | +0.304% |
| deref | 328.020 ms | 306.657 ms | 0.93487 | 0.92140 | 167936 / 168960 B | +0.610% |

All five `deref` paired runtime ratios are 0.91431–0.93946. The separate
median ratio and median paired ratio are both shown; they measure different
summaries and neither is substituted for the other. EXE bytes were identical
within each lane across all five repetitions. The `deref` build medians were
561.666 ms for C and 560.096 ms for Fairy (ratio 0.99721), but individual
build pairs moved in both directions, so this does not establish build-time
parity or improvement.

This run places all three **observed** runtime and EXE medians within the
current cumulative 8% budget. It does not certify the full language or the
Windows `Vec` implementation: the new workload mutates one local scalar,
not a vector slot. It covers one host, one optimized MSVC profile, five pairs,
and a fixed input; it excludes Linux, real application workloads, persistent
cache behavior, debug artifacts, and cold CLI startup. The previous
[constant-divisor comparison](constant-divisor-optimization.md) also warned
about phase drift. More stable and representative measurements are required
before claiming sustained 8% compliance.
