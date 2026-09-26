# Fixed-affinity identical-C scanner triad result

Date: 2026-09-27. The [protocol](scanner-triad-protocol-20260927.md#fixed-affinity-follow-up)
and both pinned harnesses were committed at child HEAD `77b63bef` before
this full run. The configured `fairyc_measure_identical_c_scanner_triads`
intent passed its execution and report-integrity checks. Its local receipt is
`.mustflow/state/runs/run-2026-09-26T22-45-48-291Z-36128-2918a8ef2ce7/receipt.json`;
the [complete report](evidence/scanner-pinned-identical-c-triads-20260927.json)
retains every QPC tick.

Windows AMD64, MSVC 14.51.36231, Go 1.26.4, optimized profile and
`GOMAXPROCS=1`. The calling thread was pinned to logical processor 9 before
warmup. The C source SHA-256 was
`e81dc241dab8f9362ada134e9defc47f7284cccfbfc4bef36d62ff56bf211e7c`.
Each session had one untimed warmup triad and 24 measured `C, C, C` triads;
every call verified the 9,000,000 work checksum over 250,000 scanner rounds.
No sample was retried or excluded.

| Session | Median middle / surrounding C | Relative full span | Null control |
| --- | ---: | ---: | --- |
| 1 | 0.99856 | 10.13% | Fail |
| 2 | 0.99760 | 14.95% | Fail |

Both median position biases are within the predeclared 2% limit, but both
relative full spans exceed the 10% limit. Processor pinning did not establish
a stable timing environment for this workload. The first span exceeds the
limit even before rounding. The prior unpinned result remains a separate
failed observation; these two conditions do not support a causal speed claim.

The matched C–Fairy–C full measurement was not run because its required
C-only null control failed. This result establishes neither the 8% Fairy/C
runtime budget nor incremental `Vec<T>` iteration cost. The source form
remains compiler-private. Repeating the same setup until it passes would
select a favorable run rather than repair the measurement condition.
