# ADR 0521: Revalidate session-local state-graph cache packets

- Status: Accepted for the existing private single-module graph family
- Date: 2026-11-07

`project_state_graph_memory_cache` retains owned exact-source graph envelopes,
indexed by SHA-256 of the source and the canonical nonzero package coordinate.
An opaque Session vector owns its configuration row and packet rows. Required
compiler interfaces and default command selection remain unchanged.

Cold misses use the common Complete graph factory. Only a fully successful
Complete, source-envelope encoding and bounded final publication may insert a
row. Hits call the real `restore_complete_state_graph` factory under the same
caller-owned 16 MiB ledger; hit is reported only after fresh source, coordinate,
body, interfaces and decoded-graph verification succeed. Reusing a packet does
not skip current-source checking and does not establish a latency improvement.

The session permits at most 512 entries and 8 MiB of retained packet/key bytes;
descriptor overhead is additionally bounded by the entry count. Both 32-byte
keys count toward retained bytes. Disabled/full sessions return cold results
without storing. Allocation, malformed input and work failures remain
1003/1001/1002, with no successful prefix inserted. The opaque rows may be moved
by callers, so each build verifies configuration, row shapes and byte totals.

This closes the explicit in-process packet cache only. Disk identity/envelopes,
filesystem persistence, normal workspace selection and native graph execution
remain separate. Windows evidence and failures:
`docs/development/candidate-state-graph-memory-cache-20261107.md`.
