# Pinned identical-source scanner build control result

Date: 2026-09-27. Child source HEAD: `1bf20726`. The
[pinned protocol](scanner-build-affinity-null-protocol-20260927.md) was
committed before this first and only run. The
[raw receipt](evidence/scanner-build-affinity-null-20260927.json) retains the
warmup and five measured pairs. The test pinned its process to allowed logical
processor mask `0x1`, with `GOMAXPROCS=1` and unchanged normal priority.
Both lanes used the same explicit-index scanner hash and benchmark main,
producing 21 specializations, 369,719 generated-C bytes and 197,120-byte
optimized EXEs. Go was 1.26.4 and MSVC was 14.51.36231.

| Measured pair | Clone/index total-build ratio |
| --- | ---: |
| 1 | 1.14519 |
| 2 | 0.98936 |
| 3 | 1.00179 |
| 4 | 1.07844 |
| 5 | 1.01247 |

The median is **1.01247**, inside `[0.98, 1.02]`, but the relative full span
is **15.39%**, above the predeclared 10% limit. The pinned null control
therefore **fails**. In pair 1, the clone took longer in Fairy check, C
emission and native build, even though source and artifacts were the same;
pair 4 also shows variation across those phases. No pair was removed or
retried.

The earlier [unpinned null](scanner-build-null-result-20260927.md) failed at
14.33%. This separate pinned run does not repair or replace that result.
Restricting Go and its MSVC descendants to one logical processor did not
make this host's elapsed build times stable enough for the candidate-cost
gate. A future comparison needs a separately controlled environment or a
validated process-CPU/phase comparator with its own identical-source null.
Neither null run estimates the private vector-loop build cost, and stable
source admission remains closed.
