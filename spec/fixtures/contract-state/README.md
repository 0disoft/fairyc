# Contract-state transaction fixtures

`manifest-v1.json` is the current versioned source of truth for the experimental
contract-state transaction execution corpus. `manifest-v0.json` remains the
historical five-case cell baseline. The runner lowers each case from
source-independent HIR through the compiler's atomic ordinary/state MIR
publication boundary and executes the published transaction against a fresh
in-memory semantic oracle.

The v1 corpus retains commit, reject, trap, cell delete, and write-limit behavior
and adds map write/read, map delete, exact physical storage-key, and map-key-limit
evidence. It does not admit source syntax, native execution, or production
persistence.
