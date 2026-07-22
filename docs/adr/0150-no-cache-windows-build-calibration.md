# ADR 0150: Record a no-cache Windows workspace build calibration

- Status: Accepted for prototype evidence
- Date: 2026-07-22
- Measurement source: `internal/buildmeasure/`
- Configured owner: `fairyc_windows_host_smoke`
- Workload: `examples/workspace_quickstart/`
- Cache decision: `docs/adr/0146-defer-persistent-package-cache.md`

## Context

The package-aware Windows command provides a real workload, but one local duration printed in prose
would be junk evidence. A useful first calibration must preserve raw samples, exact units, target and
toolchain identity, profile, lock condition, output bytes, and correctness without leaking host paths,
source, arguments, or environment variables.

There is still no compiler cache. Calling the second invocation “warm cache” would lie: only the
workspace lock, operating-system state, and external toolchain process state can be warmer.

## Decision

1. `internal/buildmeasure` defines `fairyc.build-measurement/v0` and validates a path-free report.
2. The configured Windows smoke records four monotonic wall-time samples over the checked-in
   quickstart: ordinary and locked-repeat builds at both `low` and `optimized` profiles.
3. Every sample records nanoseconds, output bytes, observed lock state, and native result 42. Every
   produced PE still crosses PE and unwind inspection before execution.
4. The report fixes `cacheImplementation` to `none` and `decision` to `baseline-only`. Validation
   rejects missing pairs, duplicate conditions, zero duration or bytes, non-42 results, incomplete
   environment identity, and any attempted cache or speedup claim.
5. The four raw observations are calibration evidence only. They are not a distribution, cold-cache
   experiment, regression threshold, cross-machine comparison, or justification for cache policy.

## Consequences

- Configured evidence can now prove the measurement schema and workload are executable on this
  Windows/MSVC host without pretending that a cache hit occurred.
- The smoke now builds and inspects four package-workspace PEs, increasing its total inspected PE
  count from 33 to 35.
- ADR 0146 remains closed until a real reusable artifact produces hits and a dedicated comparable
  measurement covers distributions, invalidation, retained bytes, corruption, concurrency, and
  cleanup.

## Verification

- unit tests freeze deterministic JSON and reject misleading or incomplete reports;
- the configured Windows smoke validates and emits the report after all four native results equal
  42;
- the complete Go test, build, vet, PE, COFF, and unwind checks remain required.
