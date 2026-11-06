# ADR 0513: Seal Fairy helper transaction graphs

- Status: Accepted for the detached Fairy whole-program graph prerequisite
- Date: 2026-11-07
- Extends ADRs 0511-0512 using the Go helper contract in ADR 0379.

`module_state_graph_program` owns one root graph, an ordered vector of helper
graphs and packed canonical module-qualified names. Its assembly function
creates unverified data. Only its private sealed factory may publish a
`Verified` resource, retaining immutable graph/name access and an independent
copy of the validated layouts. It grants no current-source or backend access.

The program admits at most 32 sorted, uniquely named helpers. The root plus all
helper bodies share the existing total bound of 1,024 operations. Every helper
must be reachable from the root, and the complete call graph must be acyclic.
All graphs use the same selected layout and nonzero transaction identity.
Call indexes resolve only within this exact helper namespace and carry no
storage, value, borrow or view operands. Every call and terminator has cost 1.

Each helper's declared storage effects must exactly equal its own operations
and the transitive effects of its called helpers. A helper-bearing root obeys
the same exact rule. Context checking alone publishes no proof: the factory
first verifies the entire namespace/call graph and then independently checks
all helper/control/storage bodies with the verified context. Only root graphs
may commit; only helper graphs may return. Helpers cannot own state borrows.

The fixed 33-node call graph uses bounded reachability and topological checks;
duplicate call sites do not inflate its adjacency/incoming counts. All scans
charge the existing 16 MiB work ledger. Name data retains the existing 1 MiB
arena limit, and each graph retains its own 1 MiB literal/category arena.
Input spans and combined counts are checked before additions and indexes.

Standalone graph factories retain an empty helper namespace. No caller can
turn a status-only contextual check or an arbitrary effect vector into a
sealed helper proof. Numeric identities/names remain detached metadata;
source integration must still rederive exact same-package internal no-argument
void owners and ordinary call projections before Complete publication.

The exact independent Go matrix, Windows evidence and unexecuted source/cache/
CLI/Linux/final gates belong to
[the helper implementation receipt](../development/candidate-state-graph-helpers-20261107.md).
Existing ordinary layout/artifact bytes, stable APIs, SDK/runtime ABI, database,
provider selection, dependencies, compiler limits, repository rules and CI/
runners are unchanged. No push or dispatch is part of this decision.
