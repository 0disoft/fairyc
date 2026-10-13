# Checked package state-body identities for cache inputs

The new fact extractor accepts only the opaque `project_compiler_core::Complete`
owner. It requires that package's completed interface, derives its exact
coordinate from retained checked source provenance, and reads the revalidated
state layout from the same immutable IR. Ordinary compilation admits state
declarations but no transaction operations; the resulting hash is the exact
zero-transaction member of Go's state-body contract, including the package
coordinate, layout presence and actual layout digest. There is no constant empty
hash, caller-supplied replacement digest or separate unverified layout input.

Eight authored ordinary source cases complete through Go Flow with no state
transactions. The existing Go `ComputeHIRStateBodyHashes` supplies the independent
hash oracle; reconstructed preimages must first hash to those values. Four
authored tests match every preimage byte/hash in a directly executed Go-native
program and candidate C development/release. They cover absent/empty/populated
state, layout changes, coordinate changes, source/owner reorder, different local
numbers, unrelated package state and detached parser/source owners. Invalid or
unknown owners, malformed binary identities and exact/exhausted/invalid shared
16MiB phase work reject without publishing a result.

The candidate emits the 277-source test closure in 101.78s. Development execution
takes 1.76s and release 0.36s; release image construction takes 136.77s. These are
private API observations, not product timing gates. Candidate API images reserve
16MiB stacks; the Go-native program runs with its ordinary default stack. Root
and locked checks pass 455 files / 5871 MIR functions.

The maintained Go oracle is regenerated with `tools/run-go.ts run
tools/verify-candidate-cache-body.go --output <fixture>` and passed through the
maintained private test verifier. An initial oracle path did not match its
declared module and correctly failed E0002; canonical module-relative paths fix
the oracle, without changing the compiler or input source bytes.

Receipt: [evidence](evidence/candidate-cache-body-identity-20261014.json).
Next: package-local emitted specialization facts, then exact workspace/SDK/source
and execution-environment inputs, followed by verified artifact reuse. This API
does not authorize a cache hit and cannot hash future transaction-bearing or
experimental-state IR. Current full fixed point, product defaults/performance,
Linux/full suite and same-head CI remain open. Public CLI/SDK ABI and product
stack/timeouts are unchanged; no push.
