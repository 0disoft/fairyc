# Contract-state transaction fixtures

`manifest-v2.json` is the current versioned source of truth for the experimental
contract-state transaction execution corpus. `manifest-v0.json` remains the
historical five-case cell baseline and v1 remains the three-case map extension. The runner lowers each case from
source-independent HIR through the compiler's atomic ordinary/state MIR
publication boundary and executes the published transaction against a fresh
in-memory semantic oracle.

The v2 corpus adds cell-read branch/join and map-found branch execution with
explicit SSA edge arguments, block parameters, and step costs. Four mutation
cases reject cycles, join type drift, missing costs, and unavailable values before
atomic compiler publication. It does not admit source state-access syntax, loops,
native execution, or production persistence.
