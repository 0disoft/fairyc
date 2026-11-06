# ADR 0529: Complete-bound Fairy graph operation C

- Status: Accepted for explicit private Windows graph emission
- Date: 2026-11-07

`module_state_graph_native::emit` requires the actual common Core Complete and
the separately pinned runtime Bundle. It refuses ordinary Complete and every
target except `x86_64-windows-msvc`. Caller-paired graph/source/hash data and
detached Verified/Plan do not authorize an Artifact.

The emitter independently writes C for the currently admitted structured graph
family: all seven codecs, cell/map reads/writes/deletes, literal/dynamic
operands, branch/join arguments, helpers, root borrows/views/copies, and exact
commit/reject/trap/return exits. Layout-local fields and graph IDs remain zero
based; helper body IDs are one based. Native codec ordering is explicitly
mapped from declaration codecs. Literal bytes and prefixes retain exact Go C
spelling rather than relying on semantically similar formatting.

Both observer modes preserve Go operation/terminator sites, call/step ordering,
cost accounting and transaction abort/commit rules. Temporary graph arena marks
are restored on every generated body exit. Immutable artifact identities bind
target, layout, portable body, canonical trap JSON, observer mode and complete
source through the existing native entry hash domain. The mode is represented
by its different emitted C, as in Go.

The existing 4096-slot preparation, 8 MiB C output and shared 16 MiB phase ledger
remain bounded. A sticky error prevents further writes; partially emitted C
never becomes an Artifact. All returned bytes remain owned after destruction
of input, Complete and runtime owners. Public CLI/SDK ABI, existing default
limits and runtime kernels are unchanged.

Windows tests compare complete C and receipt identities against live Go APIs
and compile/run the actual Fairy output. The audited test driver is the
existing Go framing program/decoder; it is explicitly a test driver, not a
Fairy-authored transport or persistent publication implementation. Those steps,
ordinary workspace binding, wider families and final acceptance remain open.
Evidence: `docs/development/candidate-state-graph-native-20261107.md`.
