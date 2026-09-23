# Byte-view loop C-relative measurement (2026-09-24)

This local Windows AMD64 MSVC comparison uses three independently written
versions of the same four-byte accumulation algorithm: `byte_view.fy`,
`byte_index.fy`, and `byte_loop.c` under `tools/benchmarks/c-budget/`.
The two Fairy sources differ only in the inner-loop spelling and share the C
reference. Each execution reads the same seed and 50,000,000-round count, checks
five boundary inputs before timing, and returns the same result. Kernel time is
measured with QPC between the shared `begin` and `finish` calls; startup, stdin
parsing, and printing are outside the interval. Each lane has one uncounted
warmup, then five alternating paired samples. `GOMAXPROCS=1` was used for the
measurement harness. OS cache and CPU scheduling were not controlled.

| MSVC profile | Fairy loop | Paired runtime ratios (Fairy/C) | Median | EXE ratio | Median build ratio |
| --- | --- | --- | ---: | ---: | ---: |
| `/O2` | byte view | 0.9383, 0.9311, 0.9294, 0.9400, 0.9360 | 0.9360 | 1.0030 | 0.9997 |
| `/O2` | explicit index | 0.9321, 0.9380, 0.9354, 0.9402, 0.9410 | 0.9380 | 1.0030 | 1.0071 |
| `/Od` | byte view | 1.2045, 1.2141, 1.2133, 1.2336, 1.2269 | 1.2141 | 1.0061 | 1.0003 |
| `/Od` | explicit index | 1.2262, 1.2869, 1.2101, 1.2283, 1.2271 | 1.2271 | 1.0061 | 1.0238 |

The optimized byte-view kernel is within the cumulative 1.08 runtime and EXE
limits for this one workload. The low-profile byte-view runtime exceeds 1.08 in
all five pairs, as does the explicit-index version. Build 1.08 is an observation
marker rather than a pass/fail threshold. The loops have the same Fairy EXE
size in each profile; the optimized `unit.obj` size was 2495 bytes for both.
The low-profile result is a baseline compiler/backend cost issue, not evidence
that byte-view syntax caused a regression. This result does not establish a
language-wide performance claim or release readiness; the real lexer/parser
workload, low-profile check-cost isolation, and full-suite evidence remain open.

Low-profile source and object inspection narrows the likely cost. The generated
Fairy C keeps a bounds check before every byte load, checked hidden and outer
index increments, checked multiplication and additions, and divide-by-zero
guards around constant divisors. The independently written C loop uses the same
input guard and arithmetic, but has no explicit runtime checks because its
fixed four-element array and value ranges make those operations safe. MSVC
`/Od` retains substantially more of the generated control flow; `/O2` removes
or folds enough work for the measured kernel to pass. This source inspection
does not isolate the cost of any single check. The next compiler work is a
bounded proof that can eliminate only checks shown redundant by the loop bound
and numeric ranges, then a new paired measurement without changing safe-mode
trap behavior.

Fixture SHA-256 values: `byte_loop.c`
`34330de4572da9ed49d6e9aed63e2781eb9cd493eacd1cfa2e4ebab31a6413a3`,
`byte_view.fy`
`c482c0b2a3bb4bade4a78aece9b34511672d3736b988f34f9902a87da2d0dae2`,
`byte_index.fy`
`aea6abf6fd3ddbacf24941caff9af7b28f62bb4aeca47cdbd025685cecbc25f7`.
