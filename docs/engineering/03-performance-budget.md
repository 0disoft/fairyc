# Performance and Build Measurement Contract

- Status: Draft
- Owner: UNASSIGNED
- Product contract: `docs/product/02-spec.md`
- Evidence status: Configured local Windows no-cache calibration, two-sample process-cache control-flow receipt, 36-sample edit-class raw distributions, one eight-step standalone session with process-memory peaks, nine paired cold/warm persistent-cache process observations, and one nine-process bounded retention staircase exist; no reference machine, cache speedup decision, default cache policy, or regression threshold exists

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

The configured `fairyc_windows_host_smoke` emits `fairyc.build-measurement/v0` for
`examples/workspace_quickstart`. It preserves four raw monotonic wall-time samples across ordinary
and locked-repeat builds at both Windows profiles, artifact bytes, lock states, environment identity,
and native result 42. `cacheImplementation: none` and `decision: baseline-only` are mandatory. This
calibrates the schema and workload; it is not cold/warm cache evidence, a timing distribution, a
cross-machine comparison, or a numeric budget.

The same configured smoke also emits `fairyc.process-cache-measurement/v1` for the semantic-through-
verified-MIR portion of that workload. One bounded Session records a cold fallback followed by one
complete verified hit. The receipt includes raw monotonic durations, package hit/miss/store counts,
whether lowering ran, MIR function count, interpreter result, retained entries and bytes, and the
harness-local bounds. It shares one bounded clock calibration with the edit measurement, preserves
zero-duration observations, and marks whether each sample is below the observed resolution.
Validation requires result `42`, rejects partial or rejected warm hits, and
fixes `decision` to `observed-only-no-speedup-claim`. The two observations prove the selected control
paths and comparable phase scope; they are not a distribution or a speedup conclusion.

The smoke additionally emits `fairyc.process-cache-edit-measurement/v0`. It records nine independent
fresh-Session observations for each of no-op warm, private root edit, private dependency edit, and
public dependency edit. One untimed request warms each Session; mutation and workspace loading stay
outside the timer. The report preserves the 36 raw durations, observes the smallest of 32 positive
monotonic transitions within a bounded ten-million-read calibration, and marks rather than replaces
samples below that resolution. It proves the current invalidation paths and result `42`; it does not define a speedup,
percentile, retry rule, reference-machine baseline, process-memory budget, or regression threshold.

`experimental check-workspace-session` now supplies the first actual foreground process boundary
that can reuse one verified-MIR Session across caller-triggered checks. ADR 0160 drives one
separately built `fairyc.exe` through an eight-request editor-shaped sequence and preserves raw
request-write-through-response-read durations plus Windows process-reported peak working-set and
pagefile-use bytes. Mutations stay outside the timer and retries are forbidden. This closes the first
process-memory and editor-workload evidence gaps, but one uncontrolled local sequence is not a
declared-machine distribution, memory budget, cancellation measurement, regression threshold, or
speedup claim. Its cache limits remain explicit harness inputs rather than measured defaults.

The configured smoke also emits `fairyc.persistent-cache-process-measurement/v0`. It prepares one
temporary locked workspace outside the timer, then runs nine cold/warm pairs. Every pair uses a
fresh disk cache and every sample starts a separate `fairyc.exe`. Caller-monotonic time spans child
creation through exit and bounded output capture, so executable hashing, workspace work, cache I/O,
and process overhead remain visible. Validation requires complete cold population followed by two
verified warm hits, zero lowering on warm samples, no rejection, eviction, skipped or failed store,
and one stable executable identity. Retries and derived speedup conclusions are forbidden. This is
a repeated local distribution, not a declared-machine comparator, default-policy study, regression
gate, or proof that warm is faster.

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

ADR 0170 added one control-workload retention staircase under an explicit six-entry harness ceiling.
ADR 0171 replaces its conflict-producing package format with package-scoped stable contexts. The
current v2 staircase reaches six entries through `2, 3, 4, 5, 6, 6, 6, 6, 6`, performs two
evictions, records zero quarantines, and ends with repeated verified all-hit reuse. A private
dependency edit now reuses the verified caller and rebuilds only the provider. The result remains
control evidence rather than a product-default justification.

## Review blockers

- An exact target has no owner, workload, baseline, configured measurement, or failure action.
- A composite score hides build, memory, storage, map, binary, and runtime tradeoffs.
- A safe check is removed without proof or an explicit unsafe boundary.
- A documentation-only change triggers machine-code regeneration without need.
- Cache or map growth has no ownership, inspection, and collection behavior.
- A development profile enables global production optimization by default.
