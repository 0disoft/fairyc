# ADR 0525: Route graph disk candidates before compilation

- Status: Accepted for the existing private single-module graph family
- Date: 2026-11-07

`project_state_graph_cache_lookup` owns canonical routing bytes and a
`fylookup0-sha256-` route accepted by the existing byte store. The distinct
`fairyc.state-graph-cache-lookup\0\1` preimage marker separates it from ordinary
and aggregate lookup families, despite sharing the transport route syntax.

The preimage contains eight existing-order u64-BE framed compatibility strings,
then raw standard-library, selected canonical package-coordinate and exact-source
SHA-256 members. Numeric owners, paths, timestamps and Complete facts are absent.
`compute` can route untrusted empty/binary source within 64 KiB; it does not
parse/admit it or return a cache hit. Namespace checks retain the existing
selected-owner scope, including deliberate tolerance of unrelated malformed rows.

`derive` uses the retained sealed current environment. `for_complete` instead
obtains source and canonical coordinate together from an actual graph Complete,
then uses the same framing. This save-side adapter rejects ordinary Complete
and prevents pairing Complete A with source/coordinate B. Rebinding the same
coordinate to another numeric owner preserves routes.

Output sizes precede compatibility-sized allocation. Both adapters retain the
same 16 MiB work history and private 1001/1002/1003 errors. Returned identities
own their bytes beyond caller or Complete lifetime. Routing alone provides no
semantic or workspace authority. Windows evidence:
`docs/development/candidate-state-graph-cache-lookup-20261107.md`.
