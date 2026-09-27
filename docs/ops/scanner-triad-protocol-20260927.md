# Scanner three-call timing protocol

Date: 2026-09-27. Status: predeclared candidate protocol, not an admission
result. The [current paired comparison](scanner-current-same-image-c-20260927.md)
showed a 32.66% QPC ratio span for the private vector-loop image even after
an identical-C pair control passed. This protocol tests whether two surrounding
C calls can expose and reduce time-local drift without discarding any sample.

The first implementation is a C-only null control. In one optimized Windows
AMD64 MSVC image, each of two consecutive sessions runs one untimed warmup
triad and then 24 measured `C, C, C` triads. Every call scans the same
`let n=12345; /*open` input for 250,000 rounds and must return the 9,000,000
Go-oracle work checksum. QPC brackets each call separately. For each triad,
the null ratio is `middle / sqrt(before * after)`. The full relative span is
`(max ratio - min ratio) / median ratio`. A session passes only when its span
is at most 10% **and** its median lies within 2% of 1. Both complete sessions
must pass. The 2% null-bias rule is fixed before measurement and leaves a
separate margin beneath the product's 8% Fairy/C limit; it is not an
allowance added to that limit. A one-triad smoke checks the harness but has
no timing verdict. A complete run retains all raw QPC ticks, calculated
ratios and unfavorable sessions; every call's checksum is verified without
retry or exclusion.

Only after the null control is checked should a separate candidate compare
`C, Fairy, C` in a combined image, once each for explicit-index and private
vector-loop Fairy source. It must use the same input, 250,000 rounds per call,
one warmup triad and 24 measured triads per image. Its paired Fairy/C ratio
uses the Fairy middle call divided by the geometric mean of the two C calls.
Each image must independently meet the existing 10% relative-full-span rule
before its median is compared with the cumulative 1.08 runtime limit. An
unstable image remains inconclusive even when its median is favorable; a
passing null control cannot rescue it. The two Fairy forms in separate images
are not a direct vector-versus-indexed timing pair.

The combined images cannot decide executable size. The existing independent
standalone C and Fairy images remain the size gate. The null control alone
does not execute Fairy, establish runtime parity or admit the `Vec<T>` source
form. This protocol changes no source syntax, compiler optimization or
runtime safety check.

The first [complete C-only control](scanner-identical-c-triad-result-20260927.md)
failed the predeclared full-span rule in both sessions. The protocol and its
thresholds remain unchanged; no C–Fairy–C result was collected from that run.
A private C–Fairy–C harness now builds both Fairy source forms and has passed
a 1,000-round smoke with one warmup and one measured triad per image. Each of
its six calls per image returned the Go-oracle checksum, and the vector
allocator was clean at exit. The smoke has no timing verdict. Full
24-triad measurements remain gated by a passing C-only null control in a
controlled environment.

## Fixed-affinity follow-up

The first complete C-only control above ran without processor affinity and
failed in both sessions. Before collecting another full result, the retained
control and the C/Fairy/C harness now pin their calling thread to its current
allowed Windows logical processor, using the same checked
`SetThreadGroupAffinity` helper already used by the scanner cycle probe.
The selected processor number is recorded in each result. Pinning occurs
before warmup and is the only measurement-condition change: input, optimized
MSVC profile, one warmup, two consecutive 24-triad C-only sessions, 250,000
rounds per call, checksum checks, complete-sample retention, and the 10%
full-span plus 2% bias criteria remain fixed. A failed affinity call aborts
the probe.

The follow-up C-only result must be retained whether it passes or fails.
Only a passing pinned null control can gate the pinned C/Fairy/C harness;
the old unpinned result cannot do so. A passing null result alone still
cannot establish the Fairy/C runtime or standalone image-size budget.

Each process pins to its own current logical processor, so pinning alone does
not ensure that the C-only control and both C/Fairy/C images use the same
processor. The configured gated intent requires all three recorded processor
numbers to match before marking a stable comparison eligible for the runtime
budget. It also checks the profile, toolchain, Go version, rounds, expected
checksum and QPC frequency. A processor mismatch is retained with reason
`processor_mismatch` and `runtime_gate_eligible = false`; it is not a
passing runtime result.

The [first complete fixed-affinity control](scanner-pinned-identical-c-triad-result-20260927.md)
failed the same 10% full-span rule in both sessions. No full pinned C/Fairy/C
result was collected.

## Highest-priority follow-up

The failed pinned control still shows short bursts within otherwise similar
adjacent C calls. A separate opt-in condition sets the pinned calling thread to
Windows `THREAD_PRIORITY_HIGHEST` before warmup in both the C-only control and
the C/Fairy/C harness. Failure to set or read back that priority aborts the
probe. The normal-priority mode remains unchanged; each receipt identifies the
priority condition and hashes its generated source.

The next run retains the same source input, optimized MSVC profile, 250,000
rounds, two complete 24-triad C-only sessions, checksum requirements, processor
matching, 10% full-span and 2% bias limits. No sample is retried or discarded.
Run the matching high-priority C/Fairy/C comparison only if the complete
high-priority C-only control passes. A passing control at a different priority
does not qualify. This is an environment experiment, not a change to Fairy
source semantics or an automatic admission of vector iteration.

The [first complete high-priority result](scanner-high-priority-triad-result-20260927.md)
passed both C-only null sessions but failed the C/Fairy/C stability condition
in both images. The three processes also selected different logical processors.
The retained medians are not an 8% runtime verdict.

## Deterministic processor follow-up

The high-priority control and two comparison images above pinned themselves
to their own current processors (1, 8, and 0), leaving the required matching
condition unmet. A new opt-in mode chooses the lowest numbered processor in
each process's allowed Windows group mask before warmup, then checks
`THREAD_PRIORITY_HIGHEST`. Selection or priority failure aborts the probe.
Each receipt records the resulting processor; matching is still checked across
all three processes, since their allowed masks or groups could differ.

This deterministic rule is fixed before the next complete run and does not
choose a processor using a favorable timing result. The 250,000 rounds, two
complete 24-triad C-only sessions, one warmup and 24 C/Fairy/C triads per image,
exact checksums, 10% full-span and 2% null-bias limits remain unchanged.
Run a complete comparison only after its matching C-only null control passes.
Retain failures and mismatches without retry or exclusion. The normal and
current-processor modes remain separate historical conditions.

The [first complete deterministic-processor C-only control](scanner-lowest-allowed-triad-result-20260927.md)
failed: its first session had a 17.35% full span, above the unchanged 10%
limit. The second session passed at 8.93%, but both are required. The matching
C/Fairy/C full comparison was not run, and the 8% runtime gate remains open.

## Eight-triad startup follow-up

The failed deterministic control's first session had unusually long calls in
its first three measured triads (for example, the first surrounding C calls
took 7,683,925 and 4,111,218 QPC ticks; later calls were near 3.7 million).
This suggests, but does not establish, an insufficient startup warmup. The
second session alone cannot be selected as a passing control.

Before another full run, a separate opt-in condition fixes **eight total
warmup triads** per session or image. The C-only control runs eight untimed
`C, C, C` triads before each of its two 24-triad sessions. Each C/Fairy/C image
runs seven untimed `C, Fairy, C` triads and retains its existing timed warmup
triad before 24 measured triads. Every warmup and measured call must return the
same Go-oracle checksum. The default one-warmup mode and all historical
receipts remain separate.

This changes no sample selection or admission limit: input, 250,000 rounds per
call, optimized MSVC profile, lowest-allowed processor, checked highest thread
priority, two complete C-only sessions, 10% full-span and 2% null-bias limits,
matched processor and toolchain, and the cumulative 8% Fairy/C runtime limit
remain fixed. Retain the first complete eight-warmup control whether it passes
or fails, without retry or exclusion. Run one matching C/Fairy/C comparison
only if both C-only sessions pass. Longer warmup tests one environmental
hypothesis; it does not itself admit vector iteration or decide standalone
EXE size.

The first full eight-warmup C-only execution hit the probe driver's 60-second
process timeout before returning a receipt. No timing samples or verdict were
retained. The harness now requests a bounded two-minute host-probe timeout;
ordinary native execution keeps its existing 60-second default. This changes
only the outer execution deadline, not the scanner calls, warmup count, sample
selection, or admission limits. The first complete run under this condition
will still be retained whether it passes or fails.
