# Identical-C scanner three-call control result

Date: 2026-09-27. The [protocol](scanner-triad-protocol-20260927.md)
and test were committed at child HEAD `448e60fb` before this full run.
Windows AMD64, MSVC 14.51.36231, Go 1.26.4, optimized profile and
`GOMAXPROCS=1`. The [raw report](evidence/scanner-identical-c-triads-20260927.json)
retains both complete sessions of 24 triads, each with the QPC ticks from
three identical C scanner calls. One untimed triad preceded each session.
Every call checked the 9,000,000 Go-oracle work checksum for 250,000 rounds
of `let n=12345; /*open`. No pair was retried or removed.

The null ratio is the middle C time divided by the geometric mean of the two
surrounding C times. Both the relative full span of at most 10% and median
distance from 1 of at most 2% were fixed before measurement.

| Session | Median null ratio | Relative full span | Null control |
| --- | ---: | ---: | --- |
| 1 | 1.00903 | 14.27% | Fail |
| 2 | 1.00328 | 13.22% | Fail |

The median position bias is within 2% in each session, but both spans exceed
10%. The C-only null control therefore fails. A subsequent C–Fairy–C run
under this protocol would not produce an admissible runtime verdict on this
host, so none was run. The private `Vec<T>` iteration source remains closed;
neither the cumulative 8% runtime limit nor its incremental runtime cost is
proven by this control. Further local repeats under unchanged conditions
would select for a favorable run rather than resolve the host instability.
