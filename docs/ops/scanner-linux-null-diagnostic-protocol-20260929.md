# Linux scanner null control with per-call diagnostics

Date: 2026-09-29. Status: predeclared; no v2 smoke or full run has been
reported. Source baseline: `a5f4df55`. This protocol supersedes the
[v1 schedule](scanner-linux-null-protocol-20260928.md) **for new runs only**.
The [v1 full verdict](scanner-linux-null-result-20260928.md) remains `FAIL`.
Its second session had 12 of 24 ratios below 0.98; changing its 97.734%
median interval to a nominal 95% interval would not diagnose the variation.

The C source, GCC profile, scanner input, 250,000 full rounds, fixed allowed
CPU, `CLOCK_MONOTONIC_RAW` wall timer, fresh-process `C, C, C` triads, two
24-triad sessions and checksums remain as specified in v1. Run one
`FAIRYC_SCANNER_LINUX_NULL_MODE=smoke` first; it checks the build and capture
path but has no timing verdict. Run `full` once only after smoke succeeds.
Retain every triad, failed build diagnostic, executable and receipt in its
unique ignored evidence directory. Do not retry, drop or reweight a sample.

For **each** of the three timed C calls, capture these additional diagnostics
in the same process:

- `CLOCK_PROCESS_CPUTIME_ID` nanoseconds before and after the call;
- `getrusage(RUSAGE_SELF)` voluntary and involuntary context-switch counts
  before and after the call.

The v2 receipt and triads contain `cpu_nanos`, `voluntary_switches` and
`involuntary_switches` arrays beside the original wall `nanos` array. A
missing clock or resource-usage reading, nonpositive CPU duration, or
decreasing switch counter makes the run `INVALID`. The additional readings
bracket each call closely but not at precisely the same instant as its wall
timer. CPU time versus wall time can reveal off-CPU time; context switches
can support a scheduler-interruption hypothesis. Neither identifies a
specific cause on its own, and neither is used to remove or rescore triads.

The **verdict rule is unchanged**: each session sorts 24 ratios of middle wall
time to the geometric mean of the first and third wall times. Its 7th and
18th ratios form the predeclared median interval, conditional on independent
and identically distributed triads. Both intervals must lie wholly in
`[0.98, 1.02]` to pass. This remains a C-only control. Go test `PASS` means
the harness completed; only `receipt.json.verdict` is the timing decision.
If v2 fails, no Fairy/C runtime or cumulative 8% verdict follows. If it
passes, the separate candidate comparison still needs matched conditions,
its own controls and a predeclared schedule. A shared KVM guest does not
guarantee independent triads or exclude timing bias, even when the interval
is narrow. This limitation follows the distinction between interval coverage
under sampling assumptions and measurement bias in the [NIST confidence
interval guide](https://www.itl.nist.gov/div898/handbook/prc/section1/prc14.htm)
and [LLVM benchmarking guidance](https://www.llvm.org/docs/Benchmarking.html).
