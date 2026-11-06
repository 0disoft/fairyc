# ADR 0514: Persist detached Fairy graph programs

- Status: Accepted for private detached graph transport
- Date: 2026-11-07
- Extends ADRs 0511-0513 without source, Complete or execution authority.

The private `module_state_graph_artifact` codec preserves an entire verified
root/helper program under the distinct eight-byte `FYSGRPH0` marker. Its framed
layout uses the existing seven-codec `FYSTATE1` format. Packed helper names,
every header/value/operand/operation/block/terminator fact, and literal/category
bytes are retained. Existing layout and ordinary program readers remain closed
to this marker. There is no unchecked decoded proof constructor.

Counts, section sums and spans are bounded before vector allocation. Decode
deducts combined operation counts across root and helpers, rejects high words
of u32 fields and exact-frame/trailing-data drift, then re-enters the complete
independent program verifier. A restored proof owns its own immutable layouts,
graphs and names. Neither caller wire mutation nor caller layout replacement
can change that proof.

Transport retains the existing 8 MiB payload ceiling and 16 MiB work ledger,
32 helpers, 64 blocks per function, 1,024 combined operations and 1 MiB graph
literal/name limits. Parsing, copying, encoding and revalidation are metered.
Errors remain private 1001 malformed, 1002 bound/work and 1003 allocation. A
valid detached graph is never a current-source ticket or a cache hit. Source
binding, Complete admission, command selection, execution and persistence are
separate required factories and acceptance gates.

Independent encoding comparisons, semantic corruptions and exact/one-short
budgets pass Windows low/optimized: 51 Go admissions, 416 native processes and
two sealed-owner refusals. Full details and failed-attempt accounting are in
[the implementation receipt](../development/candidate-state-graph-artifact-20261107.md).
Linux and allocation-failure injection remain unexecuted. Stable CLI/SDK/runtime
ABI, DB/provider, dependencies, ordinary budgets, repository rules and runners
are unchanged. No push or dispatch accompanies this slice.
