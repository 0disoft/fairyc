# Byte-view loops in the real lexer candidate (2026-09-24)

This local Windows AMD64 comparison rebuilds the checked-in lexer candidate
with its three adopted byte-view loops, then with only those loops restored to
their former explicit-index spelling. The loops are in `diagnostics.fy`,
`pages.fy`, and `scanner.fy`. Each variant has an isolated workspace and its
own generated lockfile. The test uses MSVC 14.51.36231 under `/Od` and `/O2`.

One driver first runs the candidate's page protocol on a malformed source and
a 10,002-byte cooked string, then calls `scanner::report_work` on a fixed source
with a cooked integer and a nonempty diagnostic token. The test checks the
page prefix against the Go lexer oracle and requires the complete output to
match between variants in both profiles. The long input executes the page text
loop beyond its 4,096-value output window; the scanner call exercises the
numeric and diagnostic text loops. Both variants passed all output checks.

The source hashes below cover the three changed Fairy files, including their
relative names and byte lengths. All other program source is the same in the
two variants. One build per variant/profile was timed. After both images were
built, one uncounted warmup pair and five alternating process pairs used the
same 10,002-byte frame. Timings include process startup, input parsing, lexer
work, console output and collection of stdout; they are **not** kernel times.
OS scheduling and cache state were not controlled.

| Profile | Explicit-index / byte-view EXE | Explicit-index / byte-view `unit.obj` | Single build, ms | Paired process ratios (byte-view / explicit-index) | Median paired ratio |
| --- | ---: | ---: | ---: | --- | ---: |
| `/Od` | 226304 / 226304 B | 133818 / 133898 B | 1384.2 / 1325.2 | 0.7763, 0.9956, 0.9677, 0.7204, 1.0046 | 0.9677 |
| `/O2` | 147456 / 147456 B | 93778 / 93790 B | 2297.1 / 2278.8 | 0.7217, 0.9026, 0.9512, 1.0496, 0.7117 | 0.9026 |

The raw process durations, in nanoseconds and paired by array position, are:

| Profile | Explicit-index | Byte-view |
| --- | --- | --- |
| `/Od` | 89885600, 67503900, 71156000, 94489300, 67244100 | 69780000, 67204200, 68858600, 68070500, 67552900 |
| `/O2` | 76513000, 61326900, 58398500, 55095100, 77801600 | 55220200, 55354100, 55546900, 57829100, 55368300 |

The relative full span of paired ratios is about 29% under `/Od` and 37%
under `/O2`. Those distributions are too unstable for a runtime verdict; the
single build times likewise do not establish a compile-speed difference. The
identical EXE sizes and matching output are direct observations. The object
sizes include source-identity effects and do not isolate the loop instructions.
No PDB size was measured. This comparison is Fairy versus Fairy; the separate
[independent C microkernel](byte-view-c-budget-20260924.md) is the available
C-relative evidence and does not certify this full application against C.

Explicit-index source SHA-256:
`88e6087703c0f26548304cf6563feb5cfaaf7710566b6451c5224e6d45bc1f83`.
Byte-view source SHA-256:
`a0815c46e90299a6c2e5fb4703f681d1014203dd28b3c53e9d18d9a9f8942454`.
The configured proof is `fairyc_go_test_byte_view_real_workload`, implemented
in `internal/cli/byte_view_workload_comparison_test.go`.
