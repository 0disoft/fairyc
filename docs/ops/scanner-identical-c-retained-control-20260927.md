# Retained identical-C scanner control

Date: 2026-09-27. Fairy C source HEAD `6142581a`; parent mustflow intent
commit `f1dab033`. Windows AMD64, MSVC 14.51.36231, Go 1.26.4, optimized
profile. The [complete raw receipt](evidence/scanner-identical-c-retained-control-20260927.json)
contains every QPC tick from the configured
`fairyc_measure_identical_c_scanner_control` run. Its local mustflow receipt
identity is `run-2026-09-26T21-29-37-566Z-24468-5a2d546d667f`.

The test built one C-only image and ran two consecutive 24-pair sessions.
Both calls in each pair invoked the same independent C scanner over
`let n=12345; /*open` for 250,000 rounds, with alternating first position.
One untimed call warmed each session. Every timed call checked the 9,000,000
Go-oracle work checksum. No pair was omitted or retried.

| Session | Median B/A QPC ratio | Relative full span | 10% rule |
| --- | ---: | ---: | --- |
| 1 | 0.99748 | 5.80% | Pass |
| 2 | 0.99256 | 8.56% | Pass |

This complete run passes the predeclared two-session control rule. The
[earlier control](scanner-identical-c-control-20260927.md) on the same host
retained a 14.22% span in its second session, which fails that rule. The
passing run does not erase that result or establish a consistently controlled
environment. It contains no Fairy call, so it does not measure the private
`Vec<T>` loop or the cumulative 8% Fairy/C runtime gate. Stable source
admission remains open.
