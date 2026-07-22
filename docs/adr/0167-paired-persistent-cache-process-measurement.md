# ADR 0167: Preserve paired persistent-cache process observations without a speedup claim

- Status: Accepted measurement contract
- Date: 2026-07-22
- Implementation: `internal/buildmeasure/persistent_cache_process.go`, `cmd/fairyc-windows-host-smoke/main.go`
- Command prerequisite: `docs/adr/0166-experimental-persistent-workspace-cache-command.md`

## Context

The one-shot persistent command proved that one separate compiler process could populate stable MIR
and another could verify and reuse it. Two calls were enough for control-flow evidence, but not a
distribution. They also did not freeze the measured interval, retry rule, pairing, or privacy shape.
Calling that evidence a speedup would hide process startup, executable hashing, workspace loading,
disk I/O, output capture, scheduler noise, and the absence of a declared reference machine.

## Decision

The configured Windows smoke emits `fairyc.persistent-cache-process-measurement/v0` with nine cold
and nine warm samples in canonical pairs. Each pair gets a new cache directory. Every sample starts
a new `fairyc.exe`; the warm sample is therefore cross-process reuse rather than an in-memory hit.
An untimed standalone `check-workspace` prepares the temporary fixture's lockfile before sampling.

The caller starts its monotonic timer immediately before child-process creation and stops it only
after process exit and bounded stdout/stderr capture. The interval therefore includes process
startup, executable hashing, workspace and lock loading, semantic analysis, cache reads or writes,
verified-MIR work, JSON output, and process teardown. The harness observes its clock resolution,
keeps every first observation, and performs no retry or preferred-sample selection.

Each raw sample carries its pair and condition, elapsed nanoseconds, clock-resolution classification,
package/MIR counts, cache outcome, hit/miss/rejection/store/failure decisions, lowering and reassembly
state, retained entries/bytes, raw record counters, evictions, and verified hits. The report carries
only logical workload and environment labels, the exact executable digest, explicit cache limits,
and raw samples. It omits source, absolute paths, cache directory names, arguments, and toolchain
paths.

Validation requires every cold sample to miss two packages, lower, and publish two records. Every
paired warm sample must read two records, produce two verified hits, skip lowering, and retain the
same bounded record count. Any rejection, eviction, skipped or failed store, partial hit, reassembly
fallback, identity drift, reordered sample, missing sample, retry policy, or speedup decision makes
the measurement invalid.

## Evidence

The configured Windows smoke passed on 2026-07-22 with all nine cold samples following complete
fallback and all nine warm samples following complete verified cross-process reuse. It emitted all
18 raw caller-observed durations and retained-state counters. This is one uncontrolled local run;
the contract intentionally derives no average, percentile, ratio, threshold, default, or speedup.

## Consequences and limits

- Fairy C now has configured repeated process-boundary disk-cache evidence rather than a two-call
  anecdote.
- The paired order controls cache state, not operating-system file cache, CPU scheduling, antivirus,
  thermal state, background load, or frequency scaling.
- The tiny two-package workload cannot select production cache limits or prove benefit on larger
  workspaces.
- A declared reference machine, interleaving policy, broader workspace distribution, retained-growth
  study, and accepted decision rule remain required before default enablement or a performance
  claim. ADR 0168 owns the separate crash and power-interruption safety boundary.
