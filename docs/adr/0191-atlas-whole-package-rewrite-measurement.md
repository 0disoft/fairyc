# Atlas whole-package rewrite measurement

- Status: Accepted
- Date: 2026-07-23
- Performance source: `docs/engineering/03-performance-budget.md`
- Transport prerequisite: `docs/adr/0186-bounded-canonical-fymap-transport-v1.md`

## Context

The first `.fymap` transport deliberately rewrites one canonical document per package. Chunking,
compression, and symbol-level updates add formats, indexes, recovery states, and invalidation
branches. None is justified until the current whole-package encoder has a repeatable denominator.

Compiler checking, snapshot extraction, map planning, serialization, filesystem publication, and
queries are different costs. Timing all of them as one number would not show whether whole-package
serialization itself is the pressure point.

## Decision

1. Add path-free report schema `fairyc.atlas-rewrite-measurement/v0` to `internal/buildmeasure` and
   execute it in the configured Windows host smoke.
2. Use the checked-in `workspace_quickstart` workload. Load, analyze, verify, hash the running
   executable, and prepare every package partition before timing.
3. Time only `atlasmap.Encode` across the complete fixed partition set. This includes transport
   validation, checksum calculation, and canonical JSON encoding; it excludes workspace analysis,
   map planning, queries, and filesystem publication.
4. Perform one untimed warmup and preserve nine raw monotonic-wall samples. Do not force garbage
   collection or discard samples. Mark observations below the measured local clock resolution.
5. Require every timed rewrite to reproduce the warmup bytes exactly and report the fixed package,
   source-file, source-byte, record, and encoded-byte denominators. Emit no paths or source text.
6. Use an explicit 64 MiB per-package measurement ceiling. This is a harness input, not a product
   default or an authorization for automatic map publication.
7. Draw no format, chunking, compression, incremental-update, latency, or regression-threshold
   conclusion from the control fixture. Representative repositories and a declared reference
   machine remain necessary before adopting any such policy.

## Evidence

Schema tests reject changed scope, policy conclusions, sample count, clock classification, workload
drift, encoded-size drift, missing denominators, and oversized bounds. The Windows-smoke integration
builds current partitions from the real quickstart workspace, repeats the whole-package rewrite nine
times, validates canonical byte identity, and emits one compact path-free report.
