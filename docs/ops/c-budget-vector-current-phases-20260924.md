# Current mutable-vector build phases against C

Date: 2026-09-24. Fairy C source:
`d47979c7fcdfa7df6d4ce32607d1c06e9e1816dc`. Windows AMD64,
Go 1.26.4, MSVC 14.51.36231, `GOMAXPROCS=1`, optimized C11
`/W4 /WX /MT /O2 /INCREMENTAL:NO`.

The [earlier direct-address comparison](c-budget-vector-address-20260924.md)
put repeated checked `Vec<u32>.at_mut` inside the runtime and EXE size budget.
Subsequent compiler changes could have altered build costs, so this run fixes a
current-source baseline before another optimization. It uses the same separately
authored Fairy and C vector fixtures, independent Go result calculation, five
boundary inputs per lane, one build/run warmup per lane, and five alternating
pairs. No samples were discarded.

The [raw evidence](evidence/c-budget-vector-current-phases-20260924.json)
includes ten timed samples, ten boundary executions, fixture and emitted-source
hashes, and per-stage wall times. The kernel timer excludes input parsing,
process startup and printing. Build timing includes source/workspace load,
verified check, C emission and support assembly, MSVC compilation and link,
artifact reads, and final EXE write. Common warmed toolchain discovery,
harness startup, workspace setup and program execution are excluded. OS caches
were not controlled.

| Metric | C median | Fairy median | Fairy/C |
| --- | ---: | ---: | ---: |
| Kernel | 327.127 ms | 305.720 ms | 0.93456 |
| EXE | 167,936 B | 171,008 B | 1.01829 |
| Whole build | 543.816 ms | 732.107 ms | 1.34624 |

The median paired kernel ratio is 0.93496; all five pairs fall between 0.92686
and 0.93843. The EXE is 3,072 B larger (+1.829%). Runtime and EXE size remain
within the cumulative 1.08 workload budget here. The build observation does
not: the Fairy median is 188.291 ms higher than C. The 1.08 C threshold would
be 587.321 ms, 144.786 ms below the observed Fairy median. These medians are
from only five pairs and are not a cold-build or language-wide guarantee.

| Stage | C median | Fairy median |
| --- | ---: | ---: |
| Source/workspace load | 0.544 ms | 17.242 ms |
| Verified workspace check | — | 96.796 ms |
| C emission and support assembly | — | 18.037 ms |
| MSVC build and artifact read | 542.187 ms | 608.119 ms |
| Final EXE write | 1.077 ms | 1.099 ms |

Stage medians are calculated independently and need not sum to the whole-build
medians. On this fixture, Fairy emits 35,902 B of C versus 2,170 B of hand C.
The roughly 66 ms MSVC-stage difference alone exceeds the 43.505 ms permitted
by an 8% allowance on the C whole-build median. Thus eliminating one compiler
analysis pass cannot by itself establish build-budget compliance. The next work
needs two independent parts: reduce verified SDK/generic preparation without
changing source authority or diagnostics, and reduce generated-C/MSVC work
without changing checked vector behavior. The [earlier SDK reference-cache
measurement](c-budget-vector-sdk-reference-cache-20260924.md) and
[negative nominal-encoding experiment](c-budget-vector-nominal-encoding-20260924.md)
already rule out simply repeating those changes.
