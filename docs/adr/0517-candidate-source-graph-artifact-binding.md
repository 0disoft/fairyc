# ADR 0517: Bind restored Fairy graphs to exact current source

- Status: Accepted for private source-bound graph transport
- Date: 2026-11-07
- Builds on the admitted source producer and detached `FYSGRPH0` codec.

`module_state_graph_source_artifact` introduces the separate `FYSGSRC0`
envelope containing exact retained source bytes and one canonical detached
graph artifact. Only an opaque source-produced owner can be encoded. Caller
supplied graphs, layouts or claimed source hashes are never encoding authority.

Restoration first validates framing and compares the complete saved source
with the actual current bytes. It independently produces and verifies the graph
from those current bytes, regenerates the canonical detached artifact, compares
every artifact byte and re-enters the detached decoder/verifier. A valid source
beside a different independently valid graph cannot become a restored owner.
Module-only edits invalidate the source envelope even when graph facts remain
equal. Comments and formatting also require a newly encoded envelope.

The opaque `Restored` resource retains both the independently produced current
source owner and independently decoded graph/layout owner. Accessors expose
immutable borrows. Caller buffer mutation cannot alter either retained proof.
There is no unchecked constructor, mutable getter, ordinary Complete accessor,
execution handle or implicit cache-hit result.

This initial source-bound lane inherits the producer's explicit single-module,
single-root, straight-line literal storage subset and 64 KiB source bound. The
existing 8 MiB envelope and 16 MiB work ceilings, semantic graph limits and
private errors 1001/1002/1003 remain unchanged. Wider source families, package
coordinates, ordinary Complete linking, authenticated cache selection, runtime
execution and final CLI/native acceptance remain required separate work.

Windows low/optimized exact-source, graph-splice, corruption, work-boundary and
immutable-owner results belong to
[the integration receipt](../development/candidate-state-graph-source-artifact-20261107.md).
Linux execution remains batched. Stable APIs, SDK/runtime ABI, DB/provider,
dependencies, ordinary compiler budgets, repository policy and runners are
unchanged. No push or dispatch is part of this slice.
