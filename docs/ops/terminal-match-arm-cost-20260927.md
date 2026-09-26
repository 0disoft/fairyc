# Terminal assertion match-arm size comparison

Date: 2026-09-27. Fairy C child HEAD: `67d57da3` (`fairyc-language/20`,
SDK `0.0.0-stage0.70`). Windows AMD64, MSVC 14.51.36231, Go 1.26.4.
This is a same-program comparison of two source spellings, not an independent
C-relative compiler or runtime benchmark.

The checked-in `nestedTokenProgram` in
[`nested_resource_test.go`](../../internal/compiler/nested_resource_test.go)
uses no-tail `assert(false);` in one Result error arm and one Option none arm.
The legacy variant restores only the unreachable `bytes::empty()` and
`make(-1)` expressions after those assertions. Both variants entered the same
private vector-storage package compiler and verified MIR path, used the same
entry and support C, and ran successfully in both MSVC profiles: exit 42,
empty stdout and stderr. A temporary focused Go probe collected the data
below and was removed after the run.

| Variant | Profile | Generated C bytes | C SHA-256 | Functions | `unit.obj` bytes | EXE bytes |
| --- | --- | ---: | --- | ---: | ---: | ---: |
| Legacy fabricated values | low | 48,396 | `f13f261b2784aa17269a7aaee71a496a9f57397cf8cd091d7f4e069e09bf76db` | 24 | 15,048 | 125,952 |
| Terminal assertions | low | 48,396 | `cbd852224a667e0802e114bede7acfaa411c6e7c2a14983c15f375ecb8b31ee4` | 24 | 15,048 | 125,952 |
| Legacy fabricated values | optimized | 48,396 | `f13f261b2784aa17269a7aaee71a496a9f57397cf8cd091d7f4e069e09bf76db` | 24 | 14,846 | 115,200 |
| Terminal assertions | optimized | 48,396 | `cbd852224a667e0802e114bede7acfaa411c6e7c2a14983c15f375ecb8b31ee4` | 24 | 14,846 | 115,200 |

The generated C is not byte-identical: the first difference is at zero-based
byte offset 15,502 in function-prototype text, including generated function
and local names.
This probe establishes equal source length, object size and linked image size
for this workload, not semantic equivalence of arbitrary generated C. Native
execution plus the separate semantic, MIR-interpreter and ownership tests
cover observable behavior. No runtime or build-time comparison was timed.
The cumulative 8% independent-C runtime gate remains open.
