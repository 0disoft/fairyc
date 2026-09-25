# Indexed and vector-loop scanner check cost

Date: 2026-09-25. Child source baseline: `226b1784`.
The compiler-private [check-phase receipt](evidence/vector-scanner-check-cost-20260925.json)
compares the checked-in indexed scanner with the variant in which all three
eligible SDK `Vec<T>` loops use read-only reference iteration. Both use the
same lexer package graph, SDK and benchmark main. Each invocation creates
fresh temporary package files and reaches verified MIR. One warmup pair is
followed by eight measured pairs with alternating lane order, under
`GOMAXPROCS=1`. Both lanes produce 21 specializations on every invocation.

`check_ns` sums workspace load, package metadata extraction, semantic
analysis and verified-MIR lowering. Source-file preparation is recorded
separately. The test does not time C emission, MSVC compile/link, EXE writes,
stable CLI startup or a persistent-cache hit. Ambient filesystem caching,
process load and Go GC were not controlled.

| Check phase | Indexed median | Vector-loop median | Full span, indexed / vector |
| --- | ---: | ---: | ---: |
| Load | 44.16 ms | 46.32 ms | 42% / 59% |
| Analyze | 143.75 ms | 141.30 ms | 33% / 43% |
| Verified-MIR lowering | 250.35 ms | 247.56 ms | 32% / 32% |
| Combined `check_ns` | 446.12 ms | 443.40 ms | 26.8% / 37.0% |

The ratio of lane medians is 0.99390. The median of eight paired
vector/index ratios is 0.96047, with values from 0.85999 to 1.14260 and a
29.4% relative full span. Those spreads are too wide to claim that the source
form improves, matches or worsens check speed. This establishes a repeatable
measurement boundary and an inconclusive local observation, not a build-cost
gate result. Full check/build cost, unused-feature overhead, stable runtime
and the cumulative 8% C-relative EXE limit remain open.
