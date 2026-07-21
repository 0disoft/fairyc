# Performance and Build Measurement Contract

- Status: Draft
- Owner: UNASSIGNED
- Product contract: `docs/product/02-spec.md`
- Evidence status: No compiler, benchmark corpus, reference machine, or measurement command exists

Stage 0 has no numeric performance, storage, or Atlas budget. Proposed millisecond, percentage, MiB, GiB, map-size, and context-size thresholds are removed until a maintained harness produces a baseline. This document fixes what must be measured and how a later budget becomes enforceable.

## Build-loop metrics

Measure these dimensions independently:

- process startup and manifest loading;
- lex, parse, resolution, type, Flow CFG, MIR, and verifier time;
- parser decoded bytes, tokens, nodes, maximum depth, repairs, diagnostics, work units, and peak retained source/CST bytes;
- `check` latency without backend work;
- generated-C emission, C compilation, and link time;
- clean, no-op, private-edit, public-interface-edit, and dependency-edit paths;
- cold and warm cache behavior;
- peak resident memory;
- project-local retained state and shared-cache growth;
- invalidated and reused package counts;
- requested final artifact size and regenerable intermediate size.

The C transport adds a second frontend and cannot be advertised as intrinsically faster than C. Function-level caching begins only after whole-package traces demonstrate a measured bottleneck.

## Runtime metrics

Correct safe semantics are the first gate. Runtime comparisons then measure:

- wall time and distribution;
- allocations and peak memory;
- instruction count and branch behavior where available;
- binary and mapped-code size;
- cost of retained bounds, overflow, shift, and division checks;
- optimization effects that can be attributed to proven ownership, alias, non-null, or escape facts.

No claim of C parity or superiority is valid without an equivalent algorithm, allocator, safety contract, optimization profile, target CPU, data set, and toolchain baseline. Expert intrinsic or assembly C is reported as a separate ceiling comparison.

## Atlas metrics

Measure map generation separately from compiler checking and queries:

- fact extraction and serialization time;
- whole-package rewrite time;
- `.fymap` size by section and relative to source and analyzed IR;
- overview, show, and bounded-neighborhood query latency;
- peak memory;
- context-pack encoded size, record count, relation depth, and omission count;
- cost caused by missing, stale, or incompatible dependency maps.

Chunking, compression, and fine-grained map updates require a measured failure of the whole-package design.

## Benchmark protocol

- Declare hardware, operating system, target, toolchain and libc versions, flags, input size, sample count, cache state, CPU affinity policy, and thermal controls.
- Report cold and warm distributions rather than one best run.
- Keep frontend, C transport, link, map generation, binary size, memory, and runtime as separate metrics.
- Record rejected samples and the rejection rule before looking at results.
- Inspect generated code and hardware counters before attributing a difference to the language.
- Emit optimizer promises only when verified semantic facts prove them.

## Adopting a numeric budget

A number becomes a regression gate only when:

1. an owner and representative workload are named;
2. the reference machine or normalized environment is reproducible;
3. a configured one-shot measurement command exists;
4. a baseline distribution is stored without raw private source;
5. noise and retry rules are documented;
6. the threshold has a failure action and review policy;
7. related CLI, CI, and reporting surfaces are synchronized.

Until then, exact values remain research notes outside the normative contract.

## Storage policy

High-retention candidates are module interfaces, source fingerprints, dependency graphs, typed signatures, stable non-generic MIR, and recent development objects.

Low-retention candidates are generated C, assembly, full-debug objects, optimized bitcode, old compiler-version entries, verbose MIR dumps, and cheap-to-recompute intermediates. Atlas maps are replaceable outputs with independent compatibility, inspection, and collection rules.

The executable v1 memory store requires caller-supplied positive entry and byte bounds and performs
LRU collection under both. It establishes bounded behavior but does not choose product defaults.
Persistent cache limits, age policy, and high/low-retention quotas remain UNDECIDED until the
benchmark protocol above produces representative growth and reuse evidence.

## Review blockers

- An exact target has no owner, workload, baseline, configured measurement, or failure action.
- A composite score hides build, memory, storage, map, binary, and runtime tradeoffs.
- A safe check is removed without proof or an explicit unsafe boundary.
- A documentation-only change triggers machine-code regeneration without need.
- Cache or map growth has no ownership, inspection, and collection behavior.
- A development profile enables global production optimization by default.
