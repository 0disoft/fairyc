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

## Exploratory timing diagnosis (2026-09-29)

The following calculations reuse the same 48 raw triads. They were made after
the FAIL verdict and are **diagnostics only**: they do not change the interval,
discard samples, or create another null-control result. Each timed call is one
250,000-round C scan; values in the first three columns are medians of the
individual position durations, not a replacement for paired ratios.

| Session | First C, ms | Middle C, ms | Last C, ms | Ratios below 0.98 | Ratios above 1.02 |
| --- | ---: | ---: | ---: | ---: | ---: |
| 1 | 160.846 | 163.544 | 163.632 | 6 / 24 | 5 / 24 |
| 2 | 160.316 | 158.222 | 159.300 | 12 / 24 | 6 / 24 |

The second session's lower-bound failure is not caused by one isolated low
ratio: 12 of its 24 ratios are below 0.98. Its minimum and maximum, 0.83232
and 1.20470, show large deviations in both directions. The first and last 12
triads have diagnostic ratio medians of 1.00771 then 0.99642 in session one,
and 0.98313 then 0.99150 in session two. These opposite within-session
directions do not establish a consistent monotonic time trend. The middle call
was fastest in 8 and 10 triads and slowest in 8 and 6 triads, respectively;
those counts do not identify a fixed middle-position penalty or advantage.

The evidence ZIP's aggregate cgroup counters show a four-CPU quota per
100 ms and one additional throttled period totaling 10,473 microseconds
between the before/after snapshots. Those snapshots cover smoke and full
together, so the throttling cannot be assigned to a particular triad. They
also do not measure hypervisor steal time, frequency changes, or per-call
scheduler delays. None of those mechanisms can be named as the cause from
this receipt.

Keep the original FAIL and the Fairy/C runtime gate open. Do not rerun the
same condition in search of a pass. A follow-up on a more controlled host or
with additional timing diagnostics needs its own predeclared condition and
fresh receipts; deterministic generated-code and size checks can continue
without treating them as runtime evidence.
