# Contract-state transaction fixtures

`manifest-v0.json` is the versioned source of truth for the experimental
contract-state transaction execution corpus. The runner lowers each case from
source-independent HIR, verifies the compiler-private transaction MIR, and
executes it against a fresh in-memory semantic oracle.

The v0 corpus fixes commit, reject, trap, delete, and write-limit behavior. It
does not admit source syntax, native execution, or production persistence.
