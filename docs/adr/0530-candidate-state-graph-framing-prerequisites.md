# ADR 0530: Explicit graph framing facts and source input

- Status: Accepted for private graph transport prerequisites
- Date: 2026-11-07

Fairy prepares native scratch bounds from sealed Verified graphs using an
iterative memoized DFS. It evaluates each body once while adding its bound at
every call site, including mutually exclusive static paths. Saturation after
each operation matches Go's 64 MiB transport ceiling; it does not introduce a
smaller source/program limit. Literal bounds exclude SSA and trap bytes.
Root-layout cell prefixes are copied, sorted in physical big-endian order and
mapped to the native codec IDs. Maps are excluded from this known-cell table.

The existing exact C framing implementation is supplied through a separately
pinned distribution bundle. The Python tool extracts the complete Go raw
constant without trimming its initial/final newline. Size and SHA256 precede
offset use; Bundle owns independent immutable storage and its content view
cannot outlive that owner. Product code receives explicit bytes and performs
no source discovery, Python/Go invocation or runtime-selector mutation.

Both prerequisites retain 16 MiB work ledgers and private invalid/bound/
allocation errors. Neither detached plan nor raw C grants Complete, request,
execution or persistent publication authority. Windows evidence:
`docs/development/candidate-state-graph-framing-inputs-20261107.md`.
