# Contract-state transaction fixtures

`manifest-v3.json` is the current versioned source of truth for the experimental
contract-state transaction execution corpus. `manifest-v0.json` remains the
historical five-case cell baseline, v1 remains the three-case map extension, and v2 retains the
bounded data-dependent graph generation. The runner lowers each case from
source-independent HIR through the compiler's atomic ordinary/state MIR
publication boundary and executes the published transaction against a fresh
in-memory semantic oracle.

The v3 generation adds two positive helper programs and two mutation programs while leaving v0
through v2 immutable. It records transitive call cost, executes pure and state-operation helpers in
one overlay, proves helper rejection rolls back root writes, and rejects recursion and missing
transitive effects before executable publication. It does not admit source state-access syntax,
helper parameters/results, native execution, or production persistence.
