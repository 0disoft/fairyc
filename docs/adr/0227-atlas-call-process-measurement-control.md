# ADR 0227: Atlas workspace-call process measurement control

- Status: Implemented local control
- Date: 2026-07-30
- Performance source: `docs/engineering/03-performance-budget.md`
- Command prerequisite: `docs/adr/0226-experimental-workspace-call-catalog-command.md`

## Context

The exact workspace-call catalog is useful only if its complete workspace compilation, map planning,
call join, selection, and encoding cost remains visible. Timing only the final JSON writer would hide
the expensive work. Timing it inside the already busy Windows smoke process would make its memory
peak meaningless because previous compiler, backend, and native-probe allocations share that process.

One small local fixture cannot establish a product budget or representative performance. It can
still freeze an honest process boundary, preserve raw observations, catch catastrophic regressions,
and show which evidence is still missing before workspace traversal or default limits are discussed.

## Decision

1. Add path-free report schema `fairyc.atlas-call-process-measurement/v0` to
   `internal/buildmeasure` and execute it in the configured Windows host smoke.
2. Build one standalone `fairyc.exe` outside the measurement. Use the checked-in two-package
   `workspace_quickstart`, exact root `main` SymbolId discovery, and one untimed locked warmup outside
   every sample.
3. Run nine fresh locked `fairyc experimental atlas-calls list` processes. Each sample measures
   caller monotonic wall time from command start through process exit. Compiler construction, symbol
   discovery, and warmup are excluded; process startup, workspace loading, semantic and MIR checking,
   map planning, exact-current joining, selection, encoding, output capture, and memory-probe caller
   overhead are included. OS filesystem and executable caches remain uncontrolled.
4. Record Windows `GetProcessMemoryInfo` peak working-set and peak pagefile bytes for each process.
   These are whole-process resident and committed-memory observations, not Go live-heap, retained
   index bytes, or an attribution of memory to Atlas alone.
5. Require every measured process to reproduce the warmup catalog bytes and the fixed two-package,
   two-source-file, one direct cross-package call, one selected-call shape. Preserve exact source-byte
   and output-byte denominators without paths or source text.
6. Use explicit measurement inputs of eight packages, eight calls, and 2 MiB final output. These are
   harness bounds selected above the tiny fixture shape, not product defaults or recommendations.
7. Preserve every first observation without retry or outlier deletion. Mark durations below the
   measured local clock resolution. Emit raw samples only; do not compute percentiles or a speedup.
8. Draw no reference-machine, latency, memory, default-budget, traversal, impact, or regression-
   threshold conclusion. Representative repositories with larger package and call distributions,
   plus a declared reference environment, remain required for those decisions.

## Evidence

Schema tests reject changed scope, retry selection, policy conclusions, incompatible catalog
identity, missing source shape, construction-bound overflow, missing samples, false clock
classification, absent process-memory peaks, and canonical output-size drift. The Windows smoke
builds the standalone compiler once, proves exact symbol discovery and warmup outside timing, starts
nine fresh locked processes, validates every exact cross-package catalog, records each process peak,
and emits one compact path-free report.
