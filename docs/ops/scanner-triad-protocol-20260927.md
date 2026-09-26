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
