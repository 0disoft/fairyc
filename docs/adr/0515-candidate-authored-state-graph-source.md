# ADR 0515: Produce Fairy state graphs from exact authored source

- Status: Accepted for the first private straight-line source subset
- Date: 2026-11-07
- Extends the seven-codec layouts and independently verified graph owner.

`module_state_graph_source::produce` owns and parses exact input bytes instead
of accepting caller-assembled Parsed/layout/graph facts. The first profile
admits one module, one private contract-UID state declaration and one private,
safe, nongeneric, parameterless `void transaction()` root in either declaration
order. Layouts, selected fields, literal bytes, effects, graph identities and
source spans are freshly derived and independently verified before `Produced`
escapes. All returned source/graph/layout borrows are immutable.

The six cell/map read/write/delete operations use literal operands. Discarded
reads retain distinct value/found identities. All seven storage codecs are
supported. u32/u64 source integers are unsigned decimal tokens with checked
4/8-byte canonical accumulation, including their exact maxima. Other numeric
spellings remain closed in this initial lane. One explicit final commit, reject
or canonical trap is required; subsequent statements fail.

The source lane has an explicit 64 KiB input ceiling, the unchanged 1,024
operation ceiling, and the existing metered verification/allocation errors.
Reversed parser statement links are materialized once in O(N) order. Scanner
and parser retain their independent internal bounds; the supplied project work
ledger meters admission, lowering, declaration derivation and verification.
Local/dynamic values, control joins, helpers and borrows remain closed here,
even though the detached graph verifier supports them. This is a bounded
producer expansion, not a claim of complete Go source parity.

The sealed owner retains original source, exact declaration/body/effect and
operation/exit spans, derived graph and independent layouts. Exact source
matching alone grants no package-coordinate, source-root, ordinary Complete,
cache-hit or execution authority. Wider lowering and those integration gates
remain required separate work.

Independent Go source/MIR comparisons pass 83 cases and 166 Windows native
processes across low/optimized, with five owner refusals. See
[the source receipt](../development/candidate-state-graph-source-20261107.md)
for boundaries and failed-attempt accounting. Linux and allocation-failure
injection remain unexecuted. Stable CLI/SDK/runtime ABI, DB/provider, dependencies,
ordinary budgets, repository policy and runners are unchanged. No push/dispatch.
