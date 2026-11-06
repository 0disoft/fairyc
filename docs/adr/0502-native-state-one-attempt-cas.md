# ADR 0502: One-attempt native state execution and CAS publication

- Status: Accepted for experimental single-cell and literal multi-key Windows MSVC families
- Extends: ADRs 0376, 0382 and 0500
- Owner: [native persistence integration](../architecture/native-state-persistence-v1.md)

The subsequent [graph persistence](../architecture/native-state-graph-persistence-v1.md)
step also implements the existing graph/helper/borrow native family on Windows.
[ADR 0503](0503-experimental-source-state-graphs.md) and ADR 0500's explicit native
developer mode connect the currently admitted source graphs. The original
single-cell/literal decisions below retain their checkpoint scope; complete
source/Complete/artifact/cache/ordinary-CLI integration and final gates remain open.

Compile a sealed prepared native owner into an opaque executable before state
access. One execution takes one snapshot, runs generated C exactly once, checks
the complete bound response and submits at most one overlay through the existing
layout/version CAS authority. Never execute a Go transaction to obtain native
result bytes, replay a conflicting attempt or bypass the persistence port.

Only commit can compare-and-publish; rejection, trap, limit, malformed response
and process failure publish nothing. A clean commit still compares its snapshot
version with an empty overlay and preserves it. A dirty attempt at UINT64_MAX
fails without storage mutation. Keep exact key/value bytes, read traces and
rollback behavior equal to the independent Go transaction runner.

Allow caller context cancellation through the owned Windows execution job and
retain error causes. Check cancellation before CAS, but do not undo a successful
publication acknowledgement when cancellation arrives during publication. Report
an inconsistent success receipt as explicitly acknowledged/possibly committed,
with the receipt available to the caller; do not turn it into a retryable-looking
generic pre-publication failure.

Windows acceptance requires real generated-C execution against LocalPersistence,
close/reopen and Go comparisons, stale competing writers, backend/capacity/process
failure, and separate controlled-port protocol/cancellation tests. The native
single-cell family remains one straight-line `cell<u64>`. The separately bound
multi-key family admits straight-line literal cells/maps across the seven
canonical codecs, with its own framing and built owner. Both routes retain
the same persistence authority and acknowledged-publication error type.
Dynamic keys/values, graphs/helpers/borrows and source/CLI/cache
integration, Linux and all final acceptance gates remain required by the full
implementation queue. No production provider, public ABI or target promotion is
selected. Local durability retains ADR 0500's process-restart boundary.
