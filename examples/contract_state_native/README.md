# Experimental native state transaction

`transaction.fy` writes the canonical u64 value 42, reads its pending value, and commits.
The explicit `compiler.CheckExperimentalState` entry point compiles these source operations into
verified state MIR. The ordinary released CLI does not yet accept the experimental storage calls.

`internal/contractstatenative` consumes this exact file in its source and Windows native tests.
The native test builds a small C host adapter around the emitted transaction and links the private
state runtime. It also derives delete, reject, and trap source variants and runs all six variants
at low and optimized MSVC profiles. Existing store values must survive rejected/trapped writes and
deletes. Every resulting probe exits with 42 on success and emits no stdout or stderr.

This example requires Windows amd64 and x64 MSVC for native execution. It does not provide durable
storage, a stable ABI, ordinary read expression values, or a deployable contract target.
