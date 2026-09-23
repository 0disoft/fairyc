# Checked vector-slot C comparison

Date: 2026-09-24. Compiler baseline: `3c15e4ac063070e6f30ee41ba375d180d0509334`
with the `vector_slot` fixture and opt-in harness change in this commit.
Windows AMD64, Go 1.26.4, MSVC 14.51.36231, `GOMAXPROCS=1`, optimized C11
`/W4 /WX /MT /O2 /INCREMENTAL:NO`.

## Question and method

Does a repeated checked `std::vec::at_mut<u32>` element update fit the
cumulative 8% runtime and EXE budget against an independently authored C
vector? Both programs allocate and initialize one element before timing, then
perform 50 million checked index-zero updates using the same checksum
recurrence. Both read seed and rounds from stdin, enforce `seed < 65521` and
`rounds <= 50000000`, time with the same QPC support, and return a result
checked against an independent Go calculation. Each lane passes the five
boundary inputs before timed runs. Allocation and cleanup are outside the
kernel timer; the final checked element read is inside it.

The C vector has a six-field descriptor and checks bounds, element size,
alignment and capacity before each slot access. Fairy uses the actual SDK
`Vec<u32>` and private storage transport, including its descriptor checks.
The C implementation is deliberately independent; it does not link Fairy's
runtime and does not implement every `Vec` operation or every private heap
invariant. This comparison therefore measures the cost of the current whole
paths against a small checked C vector, not a same-runtime language-only
comparison. Both use the same input and arithmetic, and the result is consumed
and checked.

After one build/run warmup per lane, five alternating pairs were recorded. The
[raw evidence](evidence/c-budget-vector-slot-20260924.json) includes 10 timed
samples, 10 boundary executions, fixture hashes, emitted-unit/support hashes,
build times and PE sizes. Build time includes source loading, Fairy checking
and MIR/C emission where applicable, MSVC compile/link, artifact reads and EXE
write. Common warmed toolchain discovery, harness startup and workspace setup
are excluded. Ambient OS caches were not controlled.

## Observations

| Metric | C median | Fairy median | Fairy/C |
| --- | ---: | ---: | ---: |
| Kernel | 326.706 ms | 473.629 ms | 1.44971 |
| EXE | 167,936 B | 171,008 B | 1.01829 |
| Build | 536.185 ms | 743.713 ms | 1.38704 |

The median of the five paired runtime ratios is 1.44743; individual pairs
range from 1.43708 to 1.48142. The EXE is 3,072 bytes larger (+1.829%).
Build pairs moved, so the build ratio is an observation rather than a stable
compiler-speed conclusion. The runtime result exceeds the current 1.08
workload budget on every pair; the EXE result is below it. No samples were
discarded or rerun for a favorable result.

MSVC can inline or eliminate checks in the small C translation unit, while
Fairy's SDK and private vector storage are emitted through separate C support.
That compilation boundary is part of the observed implementation cost. This
single-slot, fixed-index checksum is not a parser or container workload and
does not isolate `at_mut` from arithmetic. The result covers one Windows
optimized profile and does not establish Linux, debug, sustained, cold-start,
or whole-language performance. The local-scalar
[dereference comparison](c-budget-scalar-deref-20260924.md) passed its observed
budget; this vector result identifies a separate unresolved hot path.
