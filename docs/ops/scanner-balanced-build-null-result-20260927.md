# Balanced identical-source scanner build control result

Date: 2026-09-27. Child source HEAD: `2fff4ab1`. The
[protocol](scanner-balanced-build-protocol-20260927.md) and harness were
committed before this first balanced-mode run. The
[raw receipt](evidence/scanner-balanced-build-null-20260927.json) retains all
24 optimized MSVC builds: one warmup and five measured `A-B-B-A` / `B-A-A-B`
pairs. Go was 1.26.4 with `GOMAXPROCS=1`; MSVC was 14.51.36231. Both lanes
used the same explicit-index scanner SHA-256 and produced 21 specializations,
369,719 generated-C bytes and 197,120-byte EXEs in every position.

| Measured pair | Clone/index summed-build ratio |
| --- | ---: |
| 1 | 0.95786 |
| 2 | 0.99604 |
| 3 | 0.96558 |
| 4 | 0.97489 |
| 5 | 1.00707 |

The median is **0.97489**, below the predeclared null band `[0.98, 1.02]`.
The relative full span is **5.05%**, within the 10% spread limit, but both
conditions are required. The identical-source null therefore **fails**. The
clone's lower median cannot represent a source-form improvement because both
lanes compiled identical source. The raw check, C-emission, support and native
times remain in the receipt; no pair was removed or rerun.

The protocol forbids a candidate run after a failed null, so no balanced
indexed/vector comparison was made. This result does not establish compiler
cost parity or change the separate 8% runtime and EXE-size findings. Stable
read-only `Vec` source admission remains closed.
