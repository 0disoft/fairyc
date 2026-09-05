# Experimental native state transaction

`transaction.fy` writes the canonical u64 value 42, reads its pending value, and commits.
The explicit `compiler.CheckExperimentalState` entry point compiles these source operations into
verified state MIR. From this directory on Windows amd64 with x64 MSVC installed, run:

```text
fairyc experimental run-state transaction.fy --profile low
fairyc experimental run-state transaction.fy --profile optimized
```

Both commands report `state exit: commit` and `state value: 000000000000002a`. Every invocation
starts an empty in-memory store and discards it on exit. A successful command returns zero even if
the transaction reports reject or trap; the receipt distinguishes the transaction outcome.
The stable workspace build/run commands continue to reject the experimental storage calls.

`internal/contractstatenative` consumes this exact file in its source and Windows native tests.
The native test builds a small C host adapter around the emitted transaction and links the private
state runtime. It also derives delete, reject, and trap source variants and runs all six variants
at low and optimized MSVC profiles. Existing store values must survive rejected/trapped writes and
deletes. Every resulting probe exits with 42 on success and emits no stdout or stderr.

This example requires Windows amd64 and x64 MSVC for native execution. It does not provide durable
storage, a stable ABI, ordinary read expression values, or a deployable contract target.

The versioned C-emission corpus lives in `spec/fixtures/contract-state-native/manifest-v0.json`.
It references this commit example and separate reject/trap sources. Ordinary Go tests on every host
check their compiler publication, runtime calls, exact exit mapping, and deterministic emitted units;
Windows tests additionally execute the native command under both profiles.
