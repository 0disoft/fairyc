# Linux scanner timing null protocol

Date: 2026-09-28. Status: predeclared, no measurement result. This is a new
Linux x86_64/glibc/GCC experiment on the scanner sources at `deec6377`. It
does not rescore any earlier failed Windows or Linux null control. Its first
deliverable is a C-only control; Fairy/C timing is prohibited until that
control passes under the same host, CPU, toolchain and source condition.

The opt-in `TestPrivateVectorSDKCandidateLinuxScannerNull` in
`internal/compiler/vector_lexer_linux_null_test.go` now implements the C-only
schedule. Set `FAIRYC_SCANNER_LINUX_NULL_MODE=smoke` for one non-admission
triad or `FAIRYC_SCANNER_LINUX_NULL_MODE=full` for the two complete sessions.
It builds one GCC executable, then runs each triad in a fresh process. A unique
ignored `.cache/linux-scanner-null-*` directory retains the concatenated C
source, ELF, GCC output and `receipt.json`; the receipt is updated after every
measured triad. The Go test's PASS means execution and evidence capture
completed. Only `receipt.json`'s `verdict` decides whether the null control
passed. No Linux measurement has been run for this implementation yet.

## Fixed workload and build

Use the independent C scanner in `tools/benchmarks/c-budget/scanner_reference.c`
and `scanner_workload_reference.c` with the same candidate limit and diagnostic
layout defines as `scannerCBudgetDefines(false)`: the up-front token and
diagnostic storage lane. Use the literal input `let n=12345; /*open` and call
`fc_workload_round` 250,000 times per timed call with reporting disabled.
Every call must return success and the existing Go-oracle total **9,000,000**.
Build one control executable with GCC using `-std=c11 -pedantic-errors -Wall
-Wextra -Werror -fno-common -O2`. Keep the generated source, exact GCC argv,
GCC version, source and executable SHA-256, and the glibc and kernel versions.
Do not change optimization, warning flags, input or rounds after observing a
timing result. A one-process smoke may check the harness before the full run;
it has no timing verdict.

## Full control schedule

Select the lowest CPU in the process's allowed affinity mask once, before any
timed call. Pin every measured process to that CPU and verify the current CPU
before and after each call. Abort the full control on failed affinity or a
CPU mismatch. Use `CLOCK_MONOTONIC_RAW` around each call; no fallback clock is
allowed within this protocol.

Run two consecutive sessions, each with 24 measured triads. Each triad starts
in a **fresh process**, performs one untimed `C, C, C` warmup triad, then times
one `C, C, C` triad in that order. Process startup and the warmup are outside
the timed calls. A session has exactly 24 complete process records. Preserve
every positive raw nanosecond duration and every warmup and measured checksum;
never discard, retry, replace, winsorize or select a favorable process or
session. A failed call, checksum, clock or process invalidates that run rather
than silently reducing its sample count.

For each measured triad compute the paired ratio
`middle / sqrt(before * after)` from the three durations. Sort the 24 ratios
within each session. The two-sided interval for the session median is the
**7th through 18th** ratios, counting from one. Under independent and
identically distributed triads this interval has 97.734% coverage, exceeding
95%; process isolation reduces but cannot prove independence on a shared KVM
host. Report the full range and median as diagnostics, without using the
maximum or minimum as a pass/fail threshold.

The null control passes only if **both** session intervals lie entirely within
`[0.98, 1.02]`, all checksums and environment checks pass, and both sessions
are preserved. Otherwise its verdict is `FAIL`, and no Fairy/C time ratio or
8% runtime verdict may be inferred from that run. An inconclusive or failed
control is not converted to a pass by increasing samples, changing the CPU or
dropping an outlier. A new condition needs a new predeclared protocol and new
receipts; the failed receipt remains historical evidence.

## Candidate gate after a passing control

Only then build the explicit-index and private Vec-loop scanner forms with the
same optimized GCC profile and source revision. For each form, run two matched
sessions of 24 fresh-process `C, Fairy, C` triads with the same warmup, input,
rounds, checksums, CPU and clock. Preserve the full raw record. Apply the
same 7th-to-18th interval to the Fairy-middle ratio in each session. A form
passes the cumulative 8% runtime screen only if **both** intervals have upper
endpoint `<= 1.08`; it fails only if **both** lower endpoints are `> 1.08`;
every other combination is indeterminate. Candidate images also need their
own same-condition C-only controls; a passing control from a different binary
or CPU cannot be borrowed. Compare the two Fairy forms directly for the
incremental Vec cost rather than subtracting separate C-relative medians.

The independent C geometric-storage lane remains a separate comparator.
Before a final C-relative verdict, measure it with its own matched control and
apply the product's faster-equivalent-C rule. Standalone ELF file and loaded
section sizes, generated functions and helpers, ordinary check/build time,
unused-feature overhead, and lexer failure parity remain separate admission
evidence. This protocol neither relaxes the [ADR 0404](../adr/0404-read-only-sdk-vector-iteration-candidate.md)
8% budget nor admits the private Vec source form.
