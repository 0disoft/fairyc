# Exact-byte chunked nominal digests on the vector build

Date: 2026-09-24. Baseline: [entry-reachable aggregate run](c-budget-vector-reachable-aggregates-20260924.md), using `6e0cfb1484ed7e3e0cadfa9ffc94ac72f45a6d73` plus that emitter candidate. Current source: `fabe9468241f584d13e6d97d03f04a815e7537c2` plus the four reviewed digest-source changes. Windows AMD64, Go 1.26.4, MSVC 14.51.36231, `GOMAXPROCS=1`, optimized C11 `/W4 /WX /MT /O2 /INCREMENTAL:NO`.

The private generic source path hashed `string(encoded) + "\n" + typeName` for every local nominal aggregate, or the closed-graph context followed by those bytes for a package nominal. This materialized a new string containing the entire module JSON for each type. `DigestChunks` now writes the same domain and total-length prefix followed by those byte segments directly to SHA-256. Closed-graph context hashing uses the same helper on the existing JSON bytes. The helper's test compares concatenated-text and chunked digests for empty, binary, Unicode, large-module, and package-prefix inputs. No digest version, declaration identity, cache key, source syntax, SDK, or executable transport was intentionally changed.

A local Go CPU profile of the preceding source ran the five-pair vector harness with ten passing boundary executions. It showed about 0.51 s of samples under `workspacecompiler.Check` across six calls, about 0.30 s under generic source preparation, and about 0.21 s under nominal type collection. Windows syscall waits dominated the whole-process profile, so whole-profile percentages are not interpreted as Go CPU shares. The profile identifies the path to investigate; it is not a before/after speed measurement. The opt-in profile output remains local under `.cache/`.

The ordinary unprofiled candidate run used the same independent Fairy and C fixtures, one build/run warmup per lane, five alternating C/Fairy pairs, and ten passing boundary executions. [Raw candidate evidence](evidence/c-budget-vector-digest-chunks-20260924.json) retains every sample and phase; [raw baseline evidence](evidence/c-budget-vector-reachable-aggregates-20260924.json) remains separate. The fixture hashes, 26,996-byte generated Fairy C and its SHA-256 `f225b9c956b8a4602f78106823224b6a5bbc51abd2f3442955898459437c4877`, support C, and 171,008-byte Fairy EXE match the baseline. The independent C unit and EXE sizes also match. Identical generated C and EXE sizes are artifact observations, not a proof of identical machine bytes.

| Metric | Baseline C | Baseline Fairy | Candidate C | Candidate Fairy |
| --- | ---: | ---: | ---: | ---: |
| Kernel median | 325.016 ms | 302.123 ms | 329.443 ms | 302.874 ms |
| EXE | 167,936 B | 171,008 B | 167,936 B | 171,008 B |
| Whole-build median | 518.216 ms | 700.442 ms | 528.527 ms | 682.341 ms |
| Verified check median | — | 95.024 ms | — | 68.831 ms |
| C emission/support median | — | 13.616 ms | — | 17.394 ms |
| MSVC build median | 517.109 ms | 575.817 ms | 527.441 ms | 579.973 ms |

The candidate's five verified checks range from 61.905 to 72.568 ms; the baseline range is 88.619 to 99.110 ms. Their separate medians differ by 26.193 ms (27.56%). This non-overlap and removal of large per-type concatenations support keeping the change, although ambient OS caches and separate runs prevent a precise causal speedup claim. C build medians also moved by 10.311 ms with unchanged source, illustrating that build timing is noisy. Phase medians are independent and need not sum to whole-build medians.

The candidate Fairy/C whole-build median ratio is 1.29102, versus 1.35164 in the earlier run; the candidate median of paired build ratios is 1.29313. The candidate kernel and EXE ratios are 0.91935 and 1.01829. Runtime and EXE remain within the cumulative 1.08 budget for this vector workload, but the build remains above its provisional 1.08 observation marker. This is one warm, in-process Windows vector workload, not a cold CLI, parser-scale, Linux, or language-wide result.

Focused generic, semantic, MIR, interpreter, C backend, compiler, generic session/cache, and low/optimized 18-source parser self-corpus checks passed. The next build work must examine both the remaining generic preparation cost and the roughly 50 ms MSVC-stage difference instead of attributing all of the build gap to digest construction.
