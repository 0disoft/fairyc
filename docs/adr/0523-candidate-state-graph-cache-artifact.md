# ADR 0523: Recompute graph cache identity on Complete restoration

- Status: Accepted for the explicit private single-module graph family
- Date: 2026-11-07

`project_state_graph_cache_artifact` transports canonical graph-key bytes and
the existing exact-source graph envelope in the separate `FYSGCAC0` family.
Sections use u64 big-endian lengths; the complete payload remains bounded by
8 MiB and all operations share the existing 16 MiB work ledger.

Encoding derives the key and source envelope from the same sealed Complete.
Restoration bounds and exhausts the frame, calls the actual current-source graph
Complete factory with the selected package coordinate, recomputes the full key
from that Complete and compatibility facts, then compares canonical bytes.
Only after every step succeeds may Complete escape. A normal-looking key,
source envelope or digest is never sufficient admission authority.

This rejects source/environment/library/coordinate drift and splices of
individually valid keys and source packets. Same-coordinate owner rebinding is
valid. No retained caller slice or assembled proof-field prefix is returned.
Errors remain 1001 malformed/mismatched, 1002 bounds/work and 1003 allocation;
failure never restarts the shared ledger.

Explicit snapshot factories support deterministic transport comparison. The
production wrappers take the existing sealed current environment. Snapshot
labels alone do not establish runtime environment or workspace authority.

This closes a private in-memory transport/restoration boundary only. Disk file
publication/lookup/retention, normal command selection, native graph execution
and persistence remain separate. Windows evidence:
`docs/development/candidate-state-graph-cache-artifact-20261107.md`.
