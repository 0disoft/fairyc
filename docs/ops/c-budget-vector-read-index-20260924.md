# Indexed SDK vector-read C comparison

Date: 2026-09-24. Compiler source: `bd39c644b6cc573ea0f85f832888f731d5f2bd61`.
Windows AMD64, Go 1.26.4, MSVC 14.51.36231, `GOMAXPROCS=1`, optimized
C11 `/W4 /WX /MT /O2 /INCREMENTAL:NO`.

The stable Fairy workload constructs an SDK `Vec<u32>` containing 7, 11, 13
and 17. Each of 50 million outer rounds reads those four items through
`v::at<u32>` and updates the same bounded checksum. The independent C workload
constructs a checked six-field vector descriptor and performs the same four
reads and arithmetic. Both take seed and rounds from stdin, guard
`seed < 65521` and `rounds <= 50000000`, use the shared QPC timing support,
and check results against the independent Go recurrence. Allocation precedes
the timed kernel; cleanup follows it. C uses its own allocator and does not
link the Fairy SDK. MSVC can inline its checked C access within one translation
unit; Fairy retains its current generated-C and SDK support boundary.

One warmup pair preceded five alternating measured pairs. All ten boundary
executions passed. The [raw report](evidence/c-budget-vector-read-index-20260924.json)
contains fixture hashes, build and kernel times, generated-unit sizes and PE
sizes. Ambient OS caches and CPU scheduling were not controlled.

| Metric | C median | Indexed Fairy median | Fairy/C |
| --- | ---: | ---: | ---: |
| Timed kernel | 1,213.8334 ms | 1,678.1513 ms | 1.38252 |
| EXE | 168,448 B | 171,520 B | 1.01824 |
| Build | 531.2178 ms | 714.8468 ms | 1.34568 |

The five paired runtime ratios were 1.38544, 1.35253, 1.36547, 1.45849 and
1.38659; their median was **1.38544**. Every pair exceeded the cumulative
1.08 runtime limit. The EXE was 3,072 bytes larger but remained within the
1.08 size limit. Build time is an observation marker, not a release budget.

This is a failure of the **existing indexed Fairy vector-read path** on this
four-element microkernel. The compiler-private vector iteration syntax was not
measured here, so the gap cannot be attributed to that syntax. The result does
not cover the full lexer workload, Linux, cold CLI startup or the cost of
unused syntax. Measure the candidate and inspect the generated access path
before making an admission decision.
