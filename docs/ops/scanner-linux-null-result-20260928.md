# Linux scanner timing null result

Date: 2026-09-28. The first completed full run of the [predeclared Linux null
protocol](scanner-linux-null-protocol-20260928.md) used Fairy C source commit
`f0dd932ad1aa51e74c255926adf2bd968d1d3df8`. An external tester ran the
prebuilt Go 1.27.1 Linux/amd64 test binary on Debian 13.3, glibc 2.41, GCC
14.2.0 and a KVM AMD EPYC 9V74 guest. This repository's Windows host did not
execute the Linux binary.

Smoke and full each ran once. Both Go test processes exited 0 and reported
PASS; the smoke receipt says `SMOKE_ONLY` and the full receipt says **`FAIL`**.
The full run retained two consecutive sessions of 24 fresh-process C–C–C
triads. Every triad used CPU 0, 250,000 rounds, three warmup and three timed
work checksums of 9,000,000, and positive nanosecond durations. All 48 PIDs
were distinct. No sample was retried, removed or replaced. The 7th and 18th
sorted ratios recalculated from the raw durations match the receipt:

| Session | Triads | Median | Lower | Upper | Pass |
| --- | ---: | ---: | ---: | ---: | --- |
| 1 | 24 | 1.00711566 | 0.98389394 | 1.01798712 | yes |
| 2 | 24 | 0.98313221 | 0.96319750 | 1.01712117 | no |

The second session's lower endpoint is below the required 0.98. This is a
statistical null-control failure, not a Go test or GCC build failure. It does
not identify the source of timing variation, and the order-statistic interval
does not establish independent samples on a shared KVM host. The full run
lasted 47.71 seconds in the Go test log; that is not scanner execution time.

Both GCC builds emitted no diagnostics with `-std=c11 -pedantic-errors -Wall
-Wextra -Werror -fno-common -O2`. The smoke and full generated C sources share
SHA-256 `ea89b9ed5a2a1a15f54a3a07a161cdb1fbaac10abaa3fa20efbd1366ed2f054a`;
their ELF images share SHA-256
`323d06ecd30c7399e344fe9e9a481f232bf2f9fde507b8878e5177af528d1004`.
The source archive and test binary had SHA-256
`49597d2ab43583af84015e8f3d6ad81132b77af4173798c0bee9cf8e7ee99d1e`
and `5bcba0019b4cf4d15924f4f4fa743705492a60d0a2410e63202ec92e475cd443`.

The externally supplied evidence ZIP,
`fairyc-f0dd932a-null-20260928T145203Z.zip` (SHA-256
`bc9b374dc3bc6a975742e9b1966e10bfde1e8f8145b3bbe2111ed05006572dfd`),
contains the original receipts, all 48 raw triads, generated C and ELF, GCC
output, execution logs and environment records. Its contents and the separate
full receipt were inspected locally; the ZIP is not vendored into this source
tree.

The protocol forbids a Fairy/C timing comparison after this failed control.
No C–Fairy–C samples or 8% runtime verdict were produced. The private Vec
iteration candidate and its cumulative C-relative cost gate remain open;
this result does not rescore earlier controls or the Linux native-correctness
fixture.
