# SDK heap observation, 2026-09-10

`fairyc_go_measure_lexer_heap` runs `TestWindowsFairyLexerStreamHeap` on
Windows AMD64 (Go 1.26.4, MSVC 14.51.36231) with low and optimized profiles. Each scenario is one fresh
process, one source copy, one collection, a retained-Stream snapshot, and complete
scope destruction. This is deterministic allocation accounting, not a timing
benchmark: no warmup, percentile, speed comparison or regression threshold.

The test wraps the existing SDK system-heap malloc/free hooks. `Allocations` counts
successful calls; `Attempts = Allocations + Failures`. `PeakBytes` is the maximum
sum of simultaneously live **requested SDK heap bytes**, including vector and
Buffer capacity and transient overlap during growth. `RetainedBytes` is that sum
after collection, before Stream destruction. CRT metadata, allocation rounding,
stack, generated executable, static instrumentation table, Go compiler and process
RSS are excluded. The bounded 32768-slot static pointer ledger performs no tracked
allocations and aborts on exhaustion, unknown frees or live-byte overflow. Its scan
overhead makes these builds unsuitable for latency comparisons.

Both profiles produced the same rows below. All final live-byte and live-block
counts were zero, and all successful allocations were freed. The failure rows
inject exactly one failed request at the stated ordinal; they do not prove every
allocation failure path. A Stream with lexical diagnostics is still a successful
collection.

| Scenario | Source bytes | Attempts | Allocations / frees | Failures | Peak bytes | Retained bytes | Status |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Empty | 0 | 1 | 1 / 1 | 0 | 448 | 448 | 0 |
| Small function | 38 | 6 | 6 / 6 | 0 | 2726 | 1838 | 0 |
| Token-heavy | 16384 | 1213 | 1213 / 1213 | 0 | 1396488 | 938688 | 0 |
| 10000-byte string body | 10002 | 27 | 27 / 27 | 0 | 75538 | 43218 | 0 |
| Unterminated comment | 13 | 4 | 4 / 4 | 0 | 975 | 975 | 0 |
| Fail request 1 | 7 | 1 | 0 / 0 | 1 | 0 | 0 | 3 |
| Fail request 2 | 7 | 2 | 1 / 1 | 1 | 7 | 0 | 3 |
| Fail request 3 | 7 | 3 | 2 / 2 | 1 | 455 | 0 | 3 |

The token-heavy input is the same shape as the earlier cost baseline, but the
small function is 38 bytes rather than 41. Do not compare those small cases as an
optimization. The token-heavy Stream's 938688 retained requested bytes demonstrate
why its earlier 17584 logical payload bytes were not a heap estimate. This result
motivates investigating token-record capacity before changing representation; no
optimization or universal leak-freedom claim is made.

## Exact compiled input identities

Compiler baseline: `32c2bbaebc581237bde30404844c3cb15469e7fc`, plus the test-only
instrumentation in `internal/cli/fairy_lexer_heap_test.go`. The versioned JSON log
includes Go/MSVC/OS/architecture, each profile, raw counts and generated/support C
SHA-256. It emits no paths, heap contents, source text or environment dump.

Generated C SHA-256 by scenario (same for both profiles):

- Empty: `00cab56befcd20570f727c01d196d571fff47cb3d34591baf76aa559f60356dd`
- Small: `208eab486936f8863960e6853664e20d2e368cf5e06ec1cb0e593e90ea7d59df`
- Token-heavy: `070bf923273ff62cccd1d923c8ba02222cb9255d9d7e3b06b33cd54e5b24dc01`
- String: `31ad4d0345a5518a5e461dcdf2ac36caefadc86118e6f1328251164cfcf458d0`
- Diagnostic: `c73f2fa72a040cf1e6e81ef5eab838c7eb61da2136c675b5fb404bdd49d99c73`
- Failure input: `53763d3de12e0b68eb6a7b0ca055718efb1365785df274cec5a572a35b1908b2`

Support C SHA-256:

- Normal: `e42f39ab5e1f88cb6769398c9de5d54b7394770204a0ca80979dde58bf6d7226`
- Failure 1: `f39e7e6f550e01129e234b1959412c359c1babf31e3f96c3fcfa34a5a5312f39`
- Failure 2: `eaf830e786ded8a0f8938da5b43bf19a9e94d96d8a7c1a8c65930c175b66a776`
- Failure 3: `e2b7a2d8402f4d8d234b7b48c4aad62c05bd9a6ea0756c5c9d04081de87dbb3b`
