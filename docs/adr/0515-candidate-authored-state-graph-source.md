# ADR 0515: Produce Fairy state graphs from exact authored source

- Status: Accepted for the private authored whole-program source subset
- Date: 2026-11-07
- Extends the seven-codec layouts and independently verified graph owner.

`module_state_graph_source::produce` owns and parses exact input bytes instead
of accepting caller-assembled Parsed/layout/graph facts. The current profile
admits one module, one private contract-UID state declaration and one private,
safe, nongeneric, parameterless `void transaction()` root, plus up to 32 private safe nongeneric parameterless void helpers, in
either declaration order. Layouts, selected fields, literal bytes, effects, graph identities and
source spans are freshly derived and independently verified before `Produced`
escapes. All returned source/graph/layout borrows are immutable.

The six cell/map read/write/delete operations accept literals or codec-matched
ordinary local operands. Discarded
reads retain distinct value/found identities. All seven storage codecs are
supported. u32/u64 source integers are unsigned decimal tokens with checked
4/8-byte canonical accumulation, including their exact maxima. Other numeric
spellings remain closed in this initial lane. Every root path must end in an explicit commit, reject or canonical trap;
helpers may return, reject or trap. Subsequent statements on a closed path fail.

The source lane has an explicit 64 KiB input ceiling, the unchanged 1,024
operation ceiling, and the existing metered verification/allocation errors.
Reversed parser statement links are materialized once in O(N) order. Scanner
and parser retain their independent internal bounds; the supplied project work
ledger meters admission, lowering, declaration derivation and verification.
The producer admits immutable locals, typed aliases, found flags, dynamic
operands, root borrow/view/copy and structured if/else with joins. Graphs have
at most 64 blocks each; root and helpers share the 1,024-operation allowance.
Independent whole-program verification checks helper reachability, recursion,
transitive effects and borrow invalidation. Borrow handles cannot cross edges;
ordinary copies and found flags can. This remains a bounded private producer,
not complete Go source parity.

The sealed owner retains original source, exact declaration/body/effect and
root operation/exit spans, the verified root/helper program and independent layouts.
`program` exposes an immutable whole-program proof; `graph` remains the root accessor. Exact source
matching alone grants no package-coordinate, source-root, ordinary Complete,
cache-hit or execution authority. Remaining source families and those integration gates
remain required separate work.

The previous literal-focused receipt passed 83 cases and 166 Windows native
processes across low/optimized, with five owner refusals. Current expanded
comparisons pass 68 Go cases and 136 Windows native processes; retained source
and artifact regressions bring this slice to 374 native processes and eight
ownership refusals. Fresh bootstrap check and locked check pass 716 source
files / 8,917 MIR functions. See
[the source receipt](../development/candidate-state-graph-source-20261107.md)
for boundaries and failed-attempt accounting. Linux and allocation-failure
injection remain unexecuted. Stable CLI/SDK/runtime ABI, DB/provider, dependencies,
ordinary budgets, repository policy and runners are unchanged. No push/dispatch.
