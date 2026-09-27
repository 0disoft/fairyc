# Robust identical-source scanner build control result

Date: 2026-09-27. Child source HEAD: `ecc01855`. The
[protocol](scanner-robust-build-protocol-20260927.md) and opt-in harness were
committed before this first robust-mode run. The
[raw receipt](evidence/scanner-robust-build-null-20260927.json) retains all
100 optimized MSVC builds: one warmup and 24 measured `A-B-B-A` / `B-A-A-B`
pairs. Go was 1.26.4 with `GOMAXPROCS=1`; MSVC was 14.51.36231. Both lanes
used the same explicit-index scanner SHA-256 and produced 21 specializations,
369,719 generated-C bytes and 197,120-byte EXEs in every position.

| Sorted clone/index complete-build ratio | Value |
| --- | ---: |
| 7th (null lower bound) | 0.95499 |
| 18th (null upper bound) | 1.04519 |
| Median of all 24 pairs | 1.01874 |
| Minimum / maximum | 0.80267 / 1.35672 |
| Full relative span | 54.39% |

The predeclared null requires both the 7th and 18th ratios in `[0.98, 1.02]`.
Both fail. The median alone lies in the band but cannot override the order
limits. Pairs 7 and 14 are high at 1.35672 and 1.28660; pairs 18 and 24
are low at 0.80267 and 0.85613. Every pair remains in the receipt. Check,
C-emission and native wall times vary across those pairs; the record has no
host-load trace that would attribute this variation to a particular process.

This identical-source failure gives no vector-versus-indexed build-cost
verdict. The protocol forbids a candidate run after a failed null, so none
was made. It does not change the separate 8% runtime or EXE-size findings,
and read-only SDK `Vec` iteration remains compiler-private.
