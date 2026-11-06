# ADR 0527: Explicit Fairy graph runtime assets

- Status: Accepted for private Windows graph emission prerequisites
- Date: 2026-11-07

The distribution-time Python packager assembles the exact live cstdlib graph
declarations/runtime, including the existing overlay and map-key sources. It
validates canonical LF/final-newline input and refuses pin drift unless an
explicit `--update-pin` regeneration is requested. There is no new global
runtime selector or implicit source discovery by the product.

Fairy `module_state_graph_native_assets` takes explicit caller-supplied bytes.
It checks the 1 MiB input bound, exact size and SHA256 before using pinned
section offsets. A sealed immutable Bundle owns a copy; each returned section
also owns a copy. The phase uses the existing 16 MiB work ledger and private
1001/1002/1003 invalid/bound/allocation errors, including already overdrawn work.

Windows low/optimized compare entire emitted buffers against current Go
cstdlib APIs and compile/link/run real graph calls. Source/Complete/native
operation emission and persistent execution remain separate steps. Evidence:
`docs/development/candidate-state-graph-native-assets-20261107.md`.
