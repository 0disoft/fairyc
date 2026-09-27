# Robust scanner build-cost control and candidate protocol

Date: 2026-09-27. This protocol is fixed before its first measurement. The
earlier five-pair balanced identical-source control missed its `[0.98, 1.02]`
median band, even though its full span narrowed to 5.05%. Its verdict and all
earlier failed controls remain unchanged. This follow-up increases the number
of complete pairs and uses a central order interval to tolerate occasional
host scheduling noise without deleting observations. It is one new control,
not a retry of the five-pair protocol.

Lane A is the checked-in explicit-index scanner. Lane B uses identical source
and the same private compiler path for the null. Only if the null passes, a
separate candidate run replaces B with the compiler-private vector-loop
scanner. Each build uses fresh package preparation, Go check and verified-MIR
lowering, C emission, runtime-support assembly, optimized Windows MSVC
compile/link, and artifact validation. `GOMAXPROCS=1`; process affinity and
priority remain unchanged. No persistent compiler cache is requested.

Pair zero is a warmup excluded from decisions. Twenty-four measured pairs
follow, alternating `A-B-B-A` and `B-A-A-B`, for 100 complete builds per mode.
Every build retains its pair, position, lane, check/emit/support/native wall
times, specialization count, generated-C length and EXE length. Source hashes,
toolchain version and all observations stay in the raw receipt. For each
measured pair, sum the two complete build times in B and divide by the sum in
A. One complete build time is `check_ns + emit_ns + support_ns + native_ns`.
Sort all 24 ratios ascending; ranks below are one-based. No pair is removed,
trimmed or rerun to change a verdict. Record the full relative span as a
diagnostic even though it is not this protocol's decision rule.

The null requires identical lane source hashes and matching specialization,
generated-C and EXE lengths in every position. It passes only if both the 7th
and 18th measured ratios lie in `[0.98, 1.02]`. A failed null ends this
protocol without a candidate run. If the null passes, the candidate's
incremental total-build-cost screen passes only when its 18th ratio is at
most `1.02`. The same build phases, pair order and sample count apply. Retain
phase-specific ratios to reveal a compensating regression.

These ranks are a prespecified robust screen, not a confidence interval or a
proof that the unobserved tail is fast. A passing local screen does not prove
cold stable-CLI latency, a general host-independent build guarantee, the
independent-C 8% runtime budget, unused-feature overhead or release readiness.
The previous measurements cannot be reinterpreted under this new rule.
