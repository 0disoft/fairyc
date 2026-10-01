# Repeated scalar arrays

`[value; count]` evaluates `value` once and copies the result into each array
element. This example checks one-time effects, the six supported scalar types,
a computed global count, internal by-value transport and independent edits to
a 4096-byte buffer.

Counts must be integer literals or evaluated global integer constants in
`1..=4096`. Empty arrays, runtime counts, resources, borrows and nested arrays
remain unsupported. See [ADR 0416](../../docs/adr/0416-repeated-scalar-array-initialization.md).

Repository validation uses `fairyc_repeat_check` for semantic and artifact
coverage, and `fairyc_repeat_native` for Windows MSVC low/optimized execution
with normal, initializer-trap and index-bounds cases. Results and executable
hashes are summarized in the [evidence](../../docs/ops/evidence/repeated-array-20261001.json).
