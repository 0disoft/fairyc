# ADR 0160: Preserve one editor-shaped session process observation

- Status: Accepted for prototype evidence
- Date: 2026-07-22
- Measurement source: `internal/buildmeasure/session_process.go`
- Configured owner: `fairyc_windows_host_smoke`
- Process boundary: separately built `fairyc.exe`

## Context

ADR 0159 proves that two checks can share one compiler process, but its smoke used only an unchanged
workspace and recorded no process-memory evidence. The in-process timing harnesses also exclude
workspace loading, NDJSON transport, and process residency. A useful developer-loop observation must
exercise real edits through the actual executable without pretending that one local run establishes
a reference machine, budget, percentile, or speedup.

## Decision

1. Add `fairyc.workspace-session-process-measurement/v0` to the configured Windows smoke.
2. Build a temporary `fairyc.exe`, copy the two-package quickstart, and send eight sequential
   requests through one foreground session: cold, no-op, private root edit, no-op, private dependency
   edit, no-op, public dependency edit, and no-op.
3. Mutations and fixture copying occur outside each timer. Raw duration covers caller request write
   through complete response-line read, including workspace reload and lock refresh.
4. Use no retries and preserve the first observation. Emit no average, ratio, percentile, threshold,
   preferred sample, or speedup decision.
5. Read Windows `GetProcessMemoryInfo` while the child remains alive. Record process-reported peak
   working-set and peak pagefile-use bytes. These are OS process counters, not Go live heap, private
   working set, or an after-GC measurement.
6. Require every cache path, MIR function count, retained entry/byte bound, successful response, and
   cumulative reuse transition to match the closed eight-step workload.
7. Keep the existing eight-entry, 32 MiB retained-byte, and 8 MiB artifact limits as harness inputs,
   not defaults or recommendations.

## Evidence

The configured Windows smoke passed on 2026-07-22. One local run observed `13,115,392` peak
working-set bytes and `50,089,984` peak pagefile-use bytes. Raw request/response durations ranged
from `11,619,200 ns` to `151,475,700 ns`; all four no-op requests were complete all-hits, the private
root edit reused one dependency, and both dependency edits invalidated two packages. This is one
uncontrolled local observation and supports no latency or memory budget.

## Consequences

- the repository now has actual process-memory evidence and an editor-shaped process workload;
- startup, filesystem and lock work remain visible in the first cold request rather than being mixed
  with the narrower compiler-phase receipts;
- no background process, watcher, polling sampler, profiler, dependency, path disclosure, or retained
  artifact is added;
- a declared reference machine, repeated distribution, hardware and load controls, private/live-heap
  breakdown, cancellation behavior, and an accepted budget remain open.

## Rejected alternatives

- Sampling Task Manager or PowerShell was rejected because the configured Go harness can query the
  child process directly and validate the report.
- Polling in a goroutine was rejected because Windows exposes process-maintained peak counters; one
  snapshot after every completed request preserves the peak without lifecycle or sampling races.
- Calling the local machine a reference host was rejected because no product decision defines its
  hardware or maintenance contract.
- Retrying zero or noisy timings was rejected because post-result selection would bias the evidence.
