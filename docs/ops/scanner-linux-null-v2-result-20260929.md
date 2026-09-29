# Linux scanner timing null v2 result

Date: 2026-09-29. The first run of the [predeclared v2 diagnostic
protocol](scanner-linux-null-diagnostic-protocol-20260929.md) used Fairy C
source commit `09d77c7a8a24ab832a584a4e600b09a96e3a0991`. An external
tester ran the prebuilt Go 1.27.1 Linux/amd64 test binary on Debian 13.3,
glibc 2.41, GCC 14.2.0 and a KVM Intel Xeon Platinum 8573C guest. The
selected and pinned processor was CPU 0 of five allowed CPUs. This Windows
checkout did not execute the Linux binary.

Smoke and full each ran once, in that order. Both Go test processes exited 0
and reported `PASS`; their receipts say `SMOKE_ONLY` and **`FAIL`**,
respectively. GCC emitted no diagnostics, and the smoke and full generated C
sources and ELF images are byte-identical across the two runs. The full run
retained two sessions of 24 fresh-process C–C–C triads. All 48 PIDs are
distinct; all timed wall and process-CPU durations are positive; every warmup
and timed call has the expected 9,000,000 work checksum. No sample was
retried, removed or reweighted. Recomputing each ratio from the raw wall
durations gives the receipt's exact 7th and 18th order statistics:

| Session | Triads | Median | 7th lower | 18th upper | Below 0.98 | Above 1.02 | Verdict |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| 1 | 24 | 0.994905371 | 0.948945759 | 1.055512051 | 11 | 11 | FAIL |
| 2 | 24 | 0.985672948 | 0.962418368 | 1.028152648 | 10 | 8 | FAIL |

Both session intervals cross both limits of the predeclared `[0.98, 1.02]`
control band. The first session's minimum and maximum ratios were 0.69406 and
1.35277; the second's were 0.86060 and 1.29947. These extremes are retained
as diagnostics, not used as a separate rejection rule.

The v2 process-CPU diagnostic has almost the same median ratios: 0.994750
and 0.987216. The largest recorded wall-minus-CPU gap in one timed call was
1.916 ms in session one and 1.555 ms in session two. Involuntary context
switch counts summed to 103 and 57; voluntary counts were zero in both
sessions. The wall and CPU clocks do not have identical measurement bounds,
so even small negative wall-minus-CPU differences were preserved. These
readings do not show a large off-CPU gap that explains the wide ratios, but
they cannot identify CPU-frequency, hypervisor, cache, or workload effects.
No causal source of the variation is established.

The external evidence ZIP is not vendored into this
repository. Its filename is `fairyc-09d77c7a-linux-null-v2-evidence.zip` and
its SHA-256 is
`e38830af48b4b4939a7258e411bb5285317a1a5d05bfd778982f742e18c6ff4c`.
The ZIP contains both original receipts, all 48 triads, generated C and ELF,
GCC and Go output streams, execution journal, environment records, and a
read-only evidence audit. The full receipt SHA-256 is
`b54820ddab31d8e5fe21e91d193e02721a8884bcb082abff38782acb37590b4b`.
The ZIP SHA-256 and the separate receipt and stdout attachments were checked
against the ZIP contents locally. The order statistics, triad identities,
checksums and diagnostic sums above were recalculated from the attached full
receipt. The external audit reports no source edits, Go rebuilds, test
retries, or discarded triads.

The v1 `FAIL` remains historical evidence. This v2 `FAIL` prevents a matched
Fairy/C runtime comparison under this condition; no Fairy/C samples or
cumulative 8% runtime verdict were produced. The private Vec-loop source form
is not promoted. Another run in the same condition merely seeking a pass is
not justified; a new environment or measurement design needs a separately
predeclared protocol.
