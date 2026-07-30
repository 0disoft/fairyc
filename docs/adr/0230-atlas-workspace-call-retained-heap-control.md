# ADR 0230: Atlas workspace-call retained-heap control

- Status: Implemented local retained-heap control
- Date: 2026-07-30
- Join prerequisite: `docs/adr/0225-bounded-exact-current-workspace-call-join.md`
- Scaling companion: `docs/adr/0228-atlas-workspace-call-scaling-control.md`
- Topology companion: `docs/adr/0229-atlas-workspace-call-topology-corpus.md`

## Context

ADR 0228 records allocation churn while constructing and encoding a workspace call catalog. Its
`TotalAlloc` and `Mallocs` deltas cannot show what remains reachable after construction. ADR 0227's
fresh-process working-set and pagefile peaks are broader still. Neither observation can check whether
the call-only index keeps complete package-local query indexes alive after the join.

The implementation now retains only the joined call ledger, detached call-completeness context, and
visible/full SymbolId canonicalization state. A forced-GC live-heap control can observe the reachable
batch that remains, while preserving prepared partitions and current-source expectations across the
baseline and sample. It still cannot attribute every process byte to Atlas or establish a product
budget.

## Decision

1. Add path-free report schema `fairyc.atlas-call-retained-heap-measurement/v0` to
   `internal/buildmeasure` and execute it in the configured Windows host smoke.
2. Reuse ADR 0228's deterministic 2-package/8-call, 8-package/64-call, and
   32-package/512-call star workspaces. Generate, load, compile, verify, plan, and warm each shape
   before sampling.
3. Keep the prepared partitions and expectations live. Before every sample, force a Go garbage
   collection and record process `runtime.MemStats.HeapAlloc` and `HeapObjects`.
4. Construct and retain a batch of 16 complete `WorkspaceCallIndex` values, force another garbage
   collection, then record positive process-wide live-heap and live-object deltas. Preserve five
   first observations per shape. Do not divide the batch into a claimed per-index cost.
5. Require the exact schema, shape order, package/call/map-record denominators, batch size, sample
   count, compiler identity, logical-map identity, and positive raw deltas. Require no monotonic
   cross-shape growth because garbage collection and runtime background state remain process-wide.
6. Draw no Atlas-only ownership, exact retained size, peak-memory, per-index budget, default,
   representative-use, cross-machine, regression-threshold, traversal, or impact conclusion.

## Evidence

Schema tests freeze canonical path-free output and reject changed scope, Atlas-only ownership,
per-index budgets, shape drift, batch drift, missing samples, and empty heap observations. The
Windows harness test compiles every generated shape through the ordinary pipeline and requires five
positive 16-index post-GC batches per shape. The configured host smoke emits the same validated
report.

This closes only the missing live-heap observation class for the synthetic control. Prepared inputs,
the Go runtime, allocator state, garbage-collector metadata, and other process state remain part of
the observation. Representative repositories, a declared reference environment, independent heap
profiles, other relation families, indirect calls, traversal, impact policy, and product defaults
remain separate gates.
