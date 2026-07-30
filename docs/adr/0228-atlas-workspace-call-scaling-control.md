# ADR 0228: Atlas workspace-call scaling control

- Status: Implemented local scaling control
- Date: 2026-07-30
- Performance source: `docs/engineering/03-performance-budget.md`
- Join prerequisite: `docs/adr/0225-bounded-exact-current-workspace-call-join.md`
- Catalog prerequisite: `docs/adr/0226-experimental-workspace-call-catalog-command.md`
- Process-control companion: `docs/adr/0227-atlas-call-process-measurement-control.md`

## Context

ADR 0227 measures the complete catalog command in fresh processes, but its single two-package,
one-call quickstart cannot show whether workspace-call index construction and canonical output grow
sanely as package and call counts increase. Reusing that end-to-end process measurement for larger
workloads would mix compiler startup, loading, semantic checking, map planning, operating-system
caches, joining, and encoding into one number.

A generated multi-shape control can isolate the join-and-encode portion and expose raw allocation
movement. It still cannot prove an asymptotic complexity law, a representative repository budget,
or a product default.

## Decision

1. Add path-free report schema `fairyc.atlas-call-scale-measurement/v0` to
   `internal/buildmeasure` and execute it in the configured Windows host smoke.
2. Generate three deterministic star workspaces from schema-owned definitions:
   `small-p2-c8` has 2 packages and 8 direct cross-package calls, `medium-p8-c64` has 8 packages
   and 64 calls, and `large-p32-c512` has 32 packages and 512 calls. The root package calls one
   public scalar function across its direct library dependencies. These shapes are controls, not a
   model of a typical repository.
3. Generate, load, compile, verify, plan, and discover the exact root `main` SymbolId outside the
   timed region. Prepare fixed logical partitions and current-source expectations once per shape.
4. Warm each prepared shape once. Then preserve nine first-attempt samples of exact
   `WorkspaceCallIndex` construction plus outgoing workspace-call catalog encoding. Do not retry,
   discard outliers, or compute percentiles.
5. Record monotonic wall nanoseconds, canonical output bytes, and process-wide Go runtime
   `TotalAlloc` and `Mallocs` deltas. The allocation counters are cumulative process deltas with
   uncontrolled garbage collection; they are not retained heap, peak RSS, or Atlas-only ownership.
6. Require package, source-file, source-byte, logical-record, call, cross-package-call, selected-call,
   and output-byte denominators to remain complete and increasing across the fixed shapes. Every
   sample for one shape must reproduce the warmup bytes.
7. Use each exact shape as its package/call construction bound and 64 MiB as a fail-closed final
   output bound. The output limit is deliberately above the generated corpus and is not a CLI
   default, recommendation, or accepted product budget.
8. Draw no Big-O, speedup, latency budget, allocation budget, retained-memory, default, traversal,
   impact, representative-use, reference-machine, cross-machine, or regression-threshold conclusion.

## Evidence

Schema tests freeze the three definitions, detached definition access, report identity, measurement
scope, allocation label, sample count, compatibility identities, growing denominators, invariant
output, and absence of policy conclusions. Generator tests compile and measure every generated
shape. The Windows smoke emits the canonical path-free report after validation.

ADR 0229 adds deterministic chain, diamond, and fan-in correctness controls, but they remain
synthetic. Useful representative evidence still requires a declared reference environment and
checked-in or otherwise reproducible real repositories with broader dependency and call
distributions. Traversal, indirect calls, other relation families, retained-index profiling, and
product defaults remain separate decisions.
