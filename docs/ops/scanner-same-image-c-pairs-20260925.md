# Same-image C/Fairy scanner timing pairs

Date: 2026-09-25. Windows AMD64, MSVC 14.51.36231, Go 1.26.4,
`GOMAXPROCS=1`, optimized `/O2` profile. The
[raw report](evidence/vector-lexer-same-image-c-pairs-20260925.json) retains
the exact generated-image hashes, source hashes and every timing pair.

Each of two instrumented images links the independently written C scanner
beside one Fairy scanner form. One image has the explicit-index Fairy source;
the other has the compiler-private vector-loop source. A fixed warmup pair
precedes twelve measured pairs in each image. C and Fairy alternate first
position. Each call scans `let n=12345; /*open` for 250,000 rounds and returns
the Go-oracle work checksum of 9,000,000. QPC and current-thread
`GetThreadTimes` samples bracket adjacent calls; no samples were retried or
excluded.

| Fairy form / independent C | QPC paired median | QPC span | CPU paired median | CPU span |
| --- | ---: | ---: | ---: | ---: |
| Explicit index | 0.96254 | 17.77% | 1.00000 | 25.16% |
| Private vector loop | 1.00955 | 24.13% | 1.03647 | 18.88% |

`Span` is `(maximum ratio - minimum ratio) / median ratio`. Every span
exceeds the predeclared 10% stability limit, so these point estimates cannot
admit either form under the cumulative 8% runtime rule. The same-image method
removes separate process launches from each pair but does not eliminate host
scheduling or CPU-frequency variation. The two Fairy forms were measured in
different combined images, so their medians are not a directly paired
vector-versus-index comparison.

Both combined images are 152,576 bytes. They contain the C scanner, Fairy
scanner and measurement helpers together; their size cannot be compared with
a standalone C executable for the product-size gate. The uninstrumented
four-image `/O2` smoke still exceeds the 8% EXE ceiling, and check/build cost
is not established by this timing probe. The vector source form remains
compiler-private.
