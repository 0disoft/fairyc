# ADR 0158: Preserve raw process-cache edit distributions

- Status: Accepted for prototype evidence
- Date: 2026-07-22
- Measurement source: `internal/buildmeasure/cache_edit.go`
- Configured owner: `fairyc_windows_host_smoke`
- Workload: `examples/workspace_quickstart/`

## Context

ADR 0157 records one cold fallback and one immediate all-hit request. That receipt proves the
selected control-flow paths, but two observations cannot describe timing noise or show how source
edits invalidate package artifacts. A follow-up measurement must preserve repeated raw observations,
keep edit setup outside the timed region, and distinguish a cache decision from a speedup claim.

The Windows monotonic clock can also return the same timestamp for work shorter than its observable
resolution. Replacing a zero duration with an invented positive value or retrying until a preferred
sample appears would corrupt the distribution.

## Decision

1. Add `fairyc.process-cache-edit-measurement/v0` without changing either existing measurement
   schema. Its phase scope remains exactly `semantic-through-verified-mir`.
2. Record four conditions in canonical order: `no-op-warm`, `private-root-edit`,
   `private-dependency-edit`, and `public-dependency-edit`.
3. Record nine independent samples per condition. Every sample creates a fresh bounded Session,
   performs one untimed verified warmup, applies any edit and reloads the workspace outside the
   timer, and times exactly one subsequent `Session.Check` request.
4. Preserve all 36 raw observations. Do not emit a percentile, ratio, speedup, regression threshold,
   retry-selected value, or preferred summary.
5. Observe the clock by collecting 32 positive monotonic transitions within at most 10,000,000
   readings and retaining the smallest interval. The loop cooperatively yields to the scheduler
   every 4,096 unchanged observations so a fast busy loop cannot consume the entire observation
   bound before a coarse Windows clock advances. It does not sleep, retry a measured compiler
   sample, or alter an observation. Calibration fails closed if that bounded sample is incomplete.
   Record the interval as `clockResolutionNs`. A sample records `belowClockResolution: true` exactly
   when its nonnegative duration is smaller than that observed interval, including a zero-duration
   observation; the raw duration is not altered or discarded.
6. Every sample records the cache path, package hit/miss/reject/store counts, lowering and reassembly
   decisions, MIR function count, interpreter result, retained entries and bytes, and cumulative
   verified hits. Validation requires result `42`, bounds compliance, and no rejected entry,
   skipped admission, or reassembly fallback.
7. The harness-local limits remain eight entries, 33,554,432 retained bytes, and 8,388,608 bytes per
   artifact. They are measurement inputs, not product defaults.

## Invalidation contract

- `no-op-warm` is a complete all-hit request: two verified hits, zero misses, no lowering, and two
  retained entries.
- `private-root-edit` reuses the unchanged dependency and invalidates the root: one hit, one miss,
  normal lowering, and three retained entries.
- `private-dependency-edit` invalidates both packages: zero hits, two misses, normal lowering, and
  four retained entries. The root key currently contains the dependency's exact
  `PackageInstanceId`, so a private dependency source change conservatively invalidates its caller
  even when the public-interface hash is unchanged.
- `public-dependency-edit` also invalidates both packages and adds one public function to the
  verified program: zero hits, two misses, normal lowering, four retained entries, and three MIR
  functions rather than two.

## Evidence

Evidence on 2026-07-22 includes a complete configured Windows/MSVC smoke through the corrected
bounded 32-transition calibration. It preserved all 36 invalidation samples, observed a `333,500 ns`
minimum positive clock interval for that run, marked six shorter zero-duration samples without
altering them, produced result `42` for every sample, and matched every cache-path contract above.
The same run also completed the native and separately built compiler process checks. These raw local
values remain observations rather than a reference-machine baseline or speedup decision.

The same date's first hosted Windows run exposed a calibration defect before any compiler sample:
the runner consumed 1,000,000 tight-loop reads while observing only 11 of 32 coarse-clock
transitions. Cooperative scheduler yields improved that to 18, proving the original observation
bound still encoded host loop speed rather than clock adequacy. The ten-million-read fail-closed
bound leaves margin for that observed hosted clock without adding sleeps or an elapsed-time guess.
Linux was already green, and this change does not select, retry, or rewrite any measured compiler
duration.

This is controlled local invalidation and raw-distribution evidence. It is not a claim that warm
Fairy C compilation is faster, a representative-machine baseline, a process-memory measurement, a
product cache policy, or a regression gate.

## Consequences

- edit classes are now exercised by the same configured Windows path that owns the earlier cache
  receipt;
- fresh Sessions prevent one sample's retained state from contaminating another sample;
- source mutation and workspace loading cannot be mistaken for measured compiler-phase time;
- the conservative private-dependency invalidation is visible rather than silently described as
  interface-only reuse;
- ADR 0159 supplies a bounded foreground Session boundary, but reference-machine controls, process
  memory, noise/retry policy, measurements through that real command, and an explicit threshold
  decision remain open;
- ordinary one-shot CLI invocations still do not share this process-local cache.

## Rejected alternatives

- Reusing one Session across all 36 samples was rejected because cache history would make later
  observations depend on earlier condition order.
- Timing file mutation or workspace loading was rejected because the cache receipt owns only the
  semantic-through-verified-MIR phase.
- Repeating a zero-duration observation until it became positive was rejected because that would
  select measurements after seeing the result.
- Treating a private dependency edit as an immediate caller hit was rejected because the current v1
  cache key deliberately includes the exact dependency instance identity.
