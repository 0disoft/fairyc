# Sparse cooked-token storage

The lexer now keeps a compact copy-only token record and a separate vector of
owned Buffers for nonempty cooked values. A one-based index connects them; zero
means an empty view. Raw keyword values still borrow the original source.
Public Stream getters, byte spans, budgets and token/diagnostic ordering do not
change. A failed cooked-table or token-vector push publishes no Stream and drops
both vectors, including an already appended cooked value.

## Same-input memory comparison

Compare with [the prior heap baseline](heap-baseline-2026-09-10.md). The baseline
was also reproduced before this edit in the call-parser validation run. Both
measurements used Windows AMD64, Go 1.26.4, MSVC 14.51.36231, the same heap hooks,
one fresh process per scenario and both low/optimized profiles. Counts matched
between profiles. These are SDK requested bytes, not RSS or latency measurements.

| Input | Allocations before / after | Peak bytes before / after | Retained bytes before / after |
| --- | ---: | ---: | ---: |
| Empty | 1 / 1 | 448 / 256 | 448 / 256 |
| Small function, 38 bytes | 6 / 7 | 2726 / 1574 | 1838 / 1326 |
| Token-heavy, 16384 bytes | 1213 / 1221 | 1396488 / 823048 | 938688 / 578240 |
| String, 10002 bytes | 27 / 28 | 75538 / 75538 | 43218 / 43282 |
| Diagnostic, 13 bytes | 4 / 4 | 975 / 783 | 975 / 783 |

The target workload uses **41.06% fewer peak requested bytes** and **38.40% fewer
retained requested bytes**. Its allocation count rises by eight (0.66%). The
string-only case retains 64 additional bytes and makes one extra allocation;
the extra descriptor vector is a real tradeoff, not an across-the-board win.
No latency improvement is claimed. Instrumentation scans a pointer ledger and
must not be used to infer uninstrumented throughput.

The focused heap test requires at least 25% peak and retained reduction for the
token-heavy case, no peak regression for the other baseline shapes, and at most
2% additional allocations for the token-heavy case. Exact token observations,
including every cooked byte, remain covered by corpus and own-source comparison.
The tests also retain the original failure ordinals and extend failures to cooked
table insertion and populated token-vector growth. All observed successful
allocations are freed and final live blocks/bytes are zero; this is bounded
fixture evidence, not proof of every possible allocation failure path.

## Build identity

Compiler baseline: `ca4dda260d04117b6dc803cee1030ebccb1824b4`, plus this scanner
storage edit. The JSON heap log records exact generated/support C hashes. Normal
support C is unchanged from baseline:
`e42f39ab5e1f88cb6769398c9de5d54b7394770204a0ca80979dde58bf6d7226`.

Generated C SHA-256 (same source for both profiles):

- Empty: `7fc6548bbf19970f92519369f8824d8ecd9dea24b197f77f5f415c09d6681aad`
- Small: `05586c59f104c446a8941d118f031b7807f90bb4e7d262be0090bcc31d093d01`
- Token-heavy: `1d0589671684e5dc234121194fd01a72cb01a7cf142c0be70e1c9da4339dabad`
- String: `20ea0b40a7cc9ff88414c01c9b3c8bc052932ffd1e7c737f0f4447c4ef9972bb`
- Diagnostic: `bb341ef3057a55edc0e8288e52da4ba0110fae9d821143c104cecd85e0f9315e`
- Seven-byte failure input: `446e58d32d8b13a7abd1c77a9c9471ea4750219453541d8323cef590db523681`
