# ADR 0519: Explicit seven-codec graph interface encoding

- Status: Accepted for the private state-graph interface profile
- Date: 2026-11-07

State, ordinary public and generic source interface encoders expose explicit
`prefix_graph`, `encode_graph` and `hash_graph` paths. Only the selected layout
uses seven-codec validation; unrelated package layouts cannot alter that selected
package record. Existing entry points retain five-codec admission and identical
canonical bytes and work behavior. No command automatically selects this profile.

The bytes remain the existing Go public-interface v5 and generic public-interface
v0 encodings; there is no new incompatible identity domain. Source-derived bytes,
hashes, empty-state and legacy compatibility, malformed owner/field facts and
exact/one-short work controls pass 21 cases per Windows profile. A Complete
source owner must still bind these facts before cache or execution authority.

See `docs/development/candidate-state-graph-package-20261107.md`. Public CLI/SDK
ABI, database, dependencies, runner settings and default limits are unchanged.
