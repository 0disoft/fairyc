# Scanner byte matcher paired comparison

Date: 2026-09-25. Windows AMD64, MSVC 14.51.36231, Go 1.26.4,
`GOMAXPROCS=1`, optimized `/O2` profile. The
[raw receipt](evidence/scanner-byte-match-pairs-20260925.json) records the
current scanner source hashes, combined-image hash, and every timing pair.

The same instrumented image contains two private vector-loop scanner modules.
One substitutes the prior checked per-byte `matches` loop into the current
scanner; the other uses the bounded native byte matcher. Their other source
is identical. The comparison therefore isolates the matcher implementation
and its foreign declaration within this scanner shape. A fixed warmup pair
precedes twelve measured pairs, alternating which module runs first. Each
call scans `let n=12345; /*open` for 250,000 rounds and returns the
Go-oracle work checksum of 9,000,000. QPC and current-thread `GetThreadTimes`
samples bracket adjacent calls. No samples were retried or excluded.

| Current native matcher / prior per-byte loop | Paired median | Relative span |
| --- | ---: | ---: |
| QPC | 1.00283 | 38.39% |
| Thread CPU time | 1.01250 | 35.53% |

Relative span is `(maximum ratio - minimum ratio) / median ratio`, excluding
warmup. Both spans exceed the predeclared 10% stability limit. The near-parity
medians do not establish a speed gain or loss. The combined image is 145,408
bytes and generated C is 486,330 bytes; neither figure compares standalone
images or decides the cumulative 8% C-relative executable-size gate.
A separate current-source uninstrumented four-image `/O2` smoke checked the
reporter and 1,000-round checksum in each lane. The smaller C image was
184,320 bytes and both Fairy images were 201,728 bytes, 2,663 bytes above
the integer 8% ceiling. This smoke does not compare standalone prior-loop
and current-matcher Fairy images or establish a stable runtime result.

The native matcher is still candidate-only. This paired probe does not measure
full lexical output, all scanner limits, Linux execution, check/build cost,
or the performance of an independently written C scanner.
