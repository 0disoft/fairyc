# Entry-reachable aggregate C declarations

Date: 2026-09-24. The baseline is [the current vector build-phase run](c-budget-vector-current-phases-20260924.md) at `d47979c7fcdfa7df6d4ce32607d1c06e9e1816dc`; the candidate is `6e0cfb1484ed7e3e0cadfa9ffc94ac72f45a6d73` plus the three reviewed C-emitter source changes. Windows AMD64, Go 1.26.4, MSVC 14.51.36231, `GOMAXPROCS=1`, optimized C11 `/W4 /WX /MT /O2 /INCREMENTAL:NO`.

The C emitter already omitted unreachable functions. It now omits private struct, enum, and resource declarations not needed by reachable functions or foreign signatures. It retains transitive field types and all explicit `repr(C)` struct declarations and layout assertions. Full MIR verification and aggregate layout preparation still run before this emission filter. This changes generated C, not Fairy source semantics or the public runtime ABI.

The Fairy fixture, independent C fixture, and support C have identical SHA-256 hashes in both runs. Each run has five alternating C/Fairy pairs after warmup and ten passing boundary executions. Kernel timing excludes startup, parsing, and output. Build timing includes workspace/source loading, checked compilation, C emission and support assembly, MSVC compilation and linking, artifact reads, and final EXE write. [Raw candidate evidence](evidence/c-budget-vector-reachable-aggregates-20260924.json) retains every sample, source hash, phase timing, and artifact size. The baseline has [its own raw evidence](evidence/c-budget-vector-current-phases-20260924.json). OS caches were uncontrolled between runs.

| Metric | Baseline C | Baseline Fairy | Candidate C | Candidate Fairy |
| --- | ---: | ---: | ---: | ---: |
| Kernel median | 327.127 ms | 305.720 ms | 325.016 ms | 302.123 ms |
| EXE | 167,936 B | 171,008 B | 167,936 B | 171,008 B |
| Whole-build median | 543.816 ms | 732.107 ms | 518.216 ms | 700.442 ms |
| Generated unit C | 2,170 B | 35,902 B | 2,170 B | 26,996 B |

The Fairy generated unit shrank by 8,906 B (24.806%). The candidate unit hash is `f225b9c956b8a4602f78106823224b6a5bbc51abd2f3442955898459437c4877`; support C is unchanged. Fairy kernel and EXE ratios to C are 0.92957 and 1.01829 in the candidate run, within the cumulative 1.08 workload budget. The median paired kernel ratio is 0.92806, with five pairs from 0.91616 to 0.93541.

| Build stage | Baseline C | Baseline Fairy | Candidate C | Candidate Fairy |
| --- | ---: | ---: | ---: | ---: |
| Load | 0.544 ms | 17.242 ms | 0.547 ms | 17.515 ms |
| Checked compilation | — | 96.796 ms | — | 95.024 ms |
| C emission/support | — | 18.037 ms | — | 13.616 ms |
| MSVC build/artifact read | 542.187 ms | 608.119 ms | 517.109 ms | 575.817 ms |

The candidate's whole-build Fairy/C ratio is 1.35164, versus 1.34624 in the separate baseline run; the paired candidate median is 1.34131. Build remains above the provisional 1.08 observation marker. Both C and Fairy MSVC medians moved across runs, so the absolute build-time change cannot be assigned to aggregate pruning. The reduction in emitted C and emitter stage is a narrower result. This one vector workload does not establish a language-wide runtime, size, or build guarantee; more build-cost work is needed before claiming the 8% build target.
