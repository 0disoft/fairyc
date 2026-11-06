# ADR 0512: Verify Fairy root state borrows

- Status: Accepted for the detached Fairy graph prerequisite
- Date: 2026-11-07
- Extends ADR 0511 with the existing Go rules in ADRs 0380-0381.

The private Fairy graph model retains explicit borrow input/result identities
and u32 view offset/length facts. Cell/map borrows, nested views and copies are
independently checked by `module_state_graph_borrows`. Borrow and ordinary value
IDs use separate namespaces, with nonzero globally unique u32 identities in
each. Handles are local to their graph block. Copies and found flags enter the
ordinary typed value table; only those ordinary values can cross typed edges.

Borrow provenance records the exact selected field and literal map key, using
spans into immutable graph data. A cell mutation invalidates every handle/view
from that cell. A literal map mutation invalidates only the same exact entry.
Dynamic map mutation with any same-field live handle refuses as ambiguous,
even if the handle is unused afterward. Root helper calls with verified
transitive write effects invalidate all handles; read-only calls preserve them.
Helpers cannot create or own state borrows.

Views require a live input and a nonempty u32 range, retaining the original
codec and provenance. Actual value width and offset/length validity remain
runtime checks, as in the Go graph verifier. This detached step does not turn
known view shapes into proof of runtime range validity or encode partial
scalar values as already canonical copies.

The internal mutable ledger is plain nonowning metadata with fixed scalar
arrays for at most 1,024 handles, bounded by the existing graph operation cap.
It neither changes the language's closed mutable-resource-field projection
rule nor exposes a pointer into state. Only the graph verifier creates its
fresh block ledger before sealing an immutable graph; callers cannot inject
an external ledger into that factory. Scans, key comparisons, definitions and
invalidations charge the existing 16 MiB work ledger before proceeding.

The common operation shape also carries a helper index and helper-return tag.
Standalone graph verification has no helper namespace and cannot seal a call.
The context-aware checker returns only a status code, never a proof; a separate
whole-program factory must first verify helper names, reachability, recursion,
combined operation bounds and exact transitive effects before publishing one.

Malformed/bound/allocation errors remain private 1001/1002/1003. Ordinary
five-codec declarations, FYSTATE0/FYSTATE1 layout bytes and stable artifacts are
unchanged. The test-only graph packet carries the additional fields without
becoming a product protocol. No source lowering/Complete, artifact/cache hit,
transaction execution or ordinary CLI authority is granted by these proofs.

Independent Go comparison and Windows low/optimized results are recorded in
[the borrow implementation receipt](../development/candidate-state-graph-borrows-20261107.md).
Native Linux and every final gate remain pending. Stable public APIs, SDK/runtime
ABI, database/provider, dependencies, default compiler budgets, repository rules
and CI/runners are unchanged.
