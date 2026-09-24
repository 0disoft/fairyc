# Private vector-iteration C comparison

Date: 2026-09-24. Compiler source: `6cdb8c3fde90e6150854d9391afe3dc7e30b8dfe`.
Windows AMD64, Go 1.26.4, MSVC 14.51.36231, `GOMAXPROCS=1`, optimized
C11 `/W4 /WX /MT /O2 /INCREMENTAL:NO`.

The compiler-private candidate changes only the four-element inner traversal
of the [indexed Fairy workload](c-budget-vector-read-index-20260924.md) to
`for (let ref u32 item in ref values)`. It constructs and checks the same SDK
`Vec<u32>`, computes the same recurrence for 50 million outer rounds, and
checks every result against the independent Go oracle. The independent C
source is unchanged. The candidate loader authenticates the embedded SDK;
source and Flow analysis reach verified MIR through the private lane. An
untimed preflight verifies that stable workspace checking still rejects the
candidate syntax.

One warmup pair preceded five alternating measured C/candidate pairs. All
ten boundary executions passed. The [raw report](evidence/c-budget-vector-read-candidate-20260924.json)
contains source hashes, build and kernel timings, generated-unit and PE sizes.
Allocation precedes the timed kernel; cleanup follows it. C keeps its own
checked vector in one translation unit, while Fairy retains its generated-C
and SDK support boundary. Ambient OS caches and CPU scheduling were not
controlled.

| Metric | C median | Candidate Fairy median | Fairy/C |
| --- | ---: | ---: | ---: |
| Timed kernel | 1,218.0377 ms | 1,657.8501 ms | 1.36108 |
| EXE | 168,448 B | 171,520 B | 1.01824 |
| Build | 539.1898 ms | 720.8163 ms | 1.33685 |

The five paired runtime ratios sorted were 1.34894, 1.35009, 1.37255,
1.38734 and 1.49480; their median was **1.37255**. All five exceed the
cumulative 1.08 runtime limit. The EXE is 3,072 bytes larger and remains
within the 1.08 size limit. Build time is an observation marker, not the
admission budget.

The candidate generated C is 29,187 bytes, 26 bytes larger than the earlier
indexed-workload report. Both reports recorded an 8,182-byte optimized
`unit.obj` and 171,520-byte Fairy EXE. These are separate runs and revisions:
their timing difference does not isolate the source-form cost. A directly
paired index/candidate run and generated-access inspection remain necessary
before attributing the bottleneck or optimizing it. This microkernel does not
establish full-lexer cost, Linux behavior, cold CLI startup or unused-feature
overhead. Stable source admission remains closed.
