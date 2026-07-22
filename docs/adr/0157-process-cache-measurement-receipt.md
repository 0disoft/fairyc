# ADR 0157: Emit a process-cache control-flow measurement receipt

- Status: Accepted for prototype evidence
- Date: 2026-07-22
- Measurement source: `internal/buildmeasure/cache.go`
- Configured owner: `fairyc_windows_host_smoke`
- Workload: `examples/workspace_quickstart/`

## Context

ADR 0156 proves by test hook that a complete verified hit skips MIR lowering. That establishes
correct control flow but not a configured measurement receipt. Reusing the no-cache schema from ADR
0150 would be dishonest because it deliberately fixes `cacheImplementation` to `none`, includes
native C compilation and linking, and compares ordinary versus locked invocations in separate
compiler processes.

The first cache receipt must isolate one comparable compiler phase, retain raw observations, prove
the cache decision and program result, expose retained state, and refuse to calculate a speedup from
one cold and one warm sample.

## Decision

1. Add `fairyc.process-cache-measurement/v0` alongside the existing no-cache schema rather than
   changing the meaning of `fairyc.build-measurement/v0`.
2. The scope is exactly `semantic-through-verified-mir`. Workspace filesystem loading, lockfile
   handling, C emission, native compilation, linking, PE inspection, and execution time are outside
   the recorded duration.
3. One configured Windows smoke process loads the checked-in two-package quickstart, creates one
   `workspacecompiler.Session`, and records exactly two raw observations: `cold-fallback` followed
   by `warm-all-hit`.
4. Each sample records monotonic nanoseconds, cache outcome, package hit/miss/reject/store counts,
   whether MIR lowering or reassembly fallback occurred, MIR function count, interpreter result,
   retained entries and bytes, and cumulative verified hits.
5. The report records its cache limits. The current `8` entries, `33,554,432` bytes, and `8,388,608`
   artifact bytes are harness-local measurement inputs, not product defaults or recommended limits.
6. Validation requires cold complete fallback and population, warm complete verified reuse without
   lowering, positive durations, result `42`, retained state within bounds, and no rejection,
   skipped admission, or reassembly fallback. `decision` is fixed to
   `observed-only-no-speedup-claim`.

## Evidence

The configured Windows/MSVC smoke passed on 2026-07-22. Its raw local observations were:

- cold fallback: `1,579,600 ns`, two misses, two stores, MIR lowering performed;
- warm all-hit: `1,093,300 ns`, two verified hits, zero misses, MIR lowering skipped;
- both: two MIR functions, interpreter result `42`, two retained entries and `5,514` retained bytes.

These numbers are one host-local execution, not a distribution. No ratio, percentage, regression
threshold, or claim that the cache makes Fairy C builds faster is adopted from them.

## Consequences

- the configured Windows lane now emits independently valid no-cache native-build and process-cache
  compiler-phase reports;
- package reuse, lowering bypass, retained bytes, and result correctness appear in one path-free
  machine-readable receipt;
- future samples can keep phase scope stable without rewriting the no-cache baseline contract;
- ADR 0158 supplies the first repeated edit-class distributions and ADR 0159 supplies a bounded
  foreground product boundary; process memory, reference hardware, noise rules, and measurements
  through that real session workload remain required before a speedup or default cache policy;
- ordinary one-shot CLI invocations still do not share this process-local cache.

## Rejected alternatives

- Dividing the two observed durations was rejected because two points do not establish a stable
  speedup distribution.
- Including interpreter or native backend time was rejected because that would dilute the MIR cache
  effect with unrelated work.
- Reclassifying the locked no-cache build as warm cache was rejected because it runs in a separate
  compiler process and cannot read the Session.
- Treating harness limits as product defaults was rejected because no representative retention or
  workload distribution supports them.
