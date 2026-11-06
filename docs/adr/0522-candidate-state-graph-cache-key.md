# ADR 0522: Derive a separate state-graph cache identity

- Status: Accepted for the explicit private single-module graph family
- Date: 2026-11-07

`project_state_graph_cache_key` derives immutable canonical bytes and SHA-256
from an existing sealed graph Complete. Ordinary Complete is rejected. The
marker is `fairyc.state-graph-cache-key\0\1`; the digest domain is
`fysgraphcache1-sha256-`. Neither is an ordinary or generic cache identity.

The canonical preimage contains eight length-framed UTF-8 compatibility strings
in existing environment-field order, then six raw 32-byte hashes: standard
library, canonical package coordinate, exact retained source, portable state
body, public interface and source interface. Numeric owners, memory addresses
and cold/restored storage modes are excluded. Source comments invalidate the
exact-source member; same-coordinate owner rebinding preserves the identity.

`compute` admits an explicit compatibility snapshot for deterministic transport
comparison, not as authority about the running compiler. `derive` obtains that
snapshot and standard-library identity from the existing sealed current
environment. Both require an actual graph Complete and preserve the shared
16 MiB ledger and private 1001/1002/1003 error family. Input sizes are checked
before compatibility-sized allocation; callers' buffers are never mutated.

The admitted family has one module and no package imports. This key adds no
invented dependency closure or workspace ownership. Matching keys alone grant
neither cache admission nor execution authority. Current-source artifact
restoration and filesystem persistence remain separate. Windows evidence:
`docs/development/candidate-state-graph-cache-key-20261107.md`.
