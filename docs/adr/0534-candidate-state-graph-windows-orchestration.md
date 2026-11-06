# ADR 0534: Fairy-owned Windows graph build and execution

- Status: Accepted for explicit private Windows graph execution
- Date: 2026-11-07

`module_state_graph_windows_build::build` consumes the sealed transport Prepared
and a pinned runtime Bundle. The same caller-owned 16MiB preparation ledger
charges both source copies before host I/O. Options and exhausted/overdrawn
work refuse before temporary-directory creation or compiler launch. Callers
cannot substitute raw C, a structural graph, a caller-provided image or a path.

Fairy writes entry/runtime C into an exclusively owned scratch directory,
compiles two actual COFF objects with MSVC `/W4 /WX /MT /Brepro`, validates
each object, links and validates the PE image. Low/optimized retain `/Od`/`/O2`
and the existing timeout/stack policies. The sealed Built owns the exact
consumed Prepared and resulting PE bytes; the object-count accessor reports
the two validated objects, not retained object-file ownership. Compile/link
failures retain their stage, status and unit while scrubbing the private
working-directory prefix. Scratch is removed on successful and failed builds;
a cleanup failure never replaces the original compiler error.

`module_state_graph_windows_run::run` borrows Built and an immutable Snapshot.
It encodes a request against that Built's Prepared, writes the actual owned
image/request into private scratch and launches that image with replay-file
stdin. Nonzero process status, a trap or any stderr refuses before response
admission. Only received stdout reaches ADR 0533's same-Prepared/Input decoder.
The returned sealed Admitted owns its observations and overlay independently
of scratch, input and capture buffers. Request/response structure and byte
ledgers remain separate, cumulative and visible after failure. Stage-specific
errors distinguish options, request, scratch, file, child, process, response
and successful-run cleanup failures.

Existing Windows capture entry points retain their shared 64KiB stdout/stderr
limit. New explicit `run_input_limit`/`run_environment_limit` entry points admit
1..64MiB before process launch; only graph execution opts into 64MiB. Overflow
still drains the child's output before refusal. A successful pipe read uses a
zero-time process poll; an idle round retains the existing one-millisecond
wait. Every round still checks the caller deadline, retains per-stream bounded
reads and the separate trap-control channel, and terminates the process job.

This is execution/response admission only. It confers no state-provider or CAS
authority, retry policy, ordinary CLI selection, Linux execution or final
self-host/distribution acceptance. One-attempt persistent publication must be
connected separately without replaying snapshot, execution or CAS on conflict.

Evidence and exact exclusions:
`docs/development/candidate-state-graph-native-orchestration-20261107.md`.
