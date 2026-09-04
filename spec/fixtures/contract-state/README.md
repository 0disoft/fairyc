# Contract-state transaction fixtures

`manifest-v4.json` is the current versioned source of truth for the experimental
contract-state transaction execution corpus. `manifest-v0.json` remains the
historical five-case cell baseline, v1 remains the three-case map extension, and v2 retains the
bounded data-dependent graph generation. v3 retains the helper-call generation. The runner lowers each case from
source-independent HIR through the compiler's atomic ordinary/state MIR
publication boundary and executes the published transaction against a fresh
in-memory semantic oracle.

The v4 generation adds two positive borrow programs and three mutation programs while leaving v0
through v3 immutable. It proves nested cell views, map-borrow survival across an unrelated literal
key mutation, exact-address invalidation, conservative dynamic-key alias rejection, and block-edge
escape rejection. It does not admit source state-access syntax, cross-block or helper borrow
transfer, mutable views, native execution, or production persistence.
