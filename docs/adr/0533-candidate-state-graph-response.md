# ADR 0533: Complete-bound Fairy graph response admission

- Status: Accepted for explicit private Windows graph transport
- Date: 2026-11-07

`module_state_graph_response::decode` accepts the actual retained transport
Prepared, its immutable compiled-owner Input, received bytes and caller-owned
structure/byte ledgers. It checks Input against the same Prepared's layout,
body and framed-program identities before any response admission. The input
encoder's immutable complete-request SHA256, including snapshot and limits,
binds every received response. A structural Packet, parsed header or caller
graph cannot substitute for these owners.

Three independently testable prerequisites separate untrusted wire offsets,
Verified-derived CFG/call-return facts and state observation/overlay evidence.
The framing parser preserves the existing 186-byte header, 21-byte event,
37-byte final-write frame and 64MiB response ceiling. Count/length arithmetic
is checked before slicing or allocation. These facts alone confer no result
or publication authority.

The path plan owns dense immutable operation/terminator sites from Verified,
with the existing global site numbering, independent operation/cost counters,
sorted trap IDs and at most 33 helper continuations. Only actual CFG successors
or the current helper return continuation may advance. An error can retain
the checked prefix or lose precisely one next already-charged event, matching
the existing native contract. Success requires a completed matching exit.

Observation admission derives physical map addresses from the original
canonical key bytes. Reads must equal the immutable snapshot or earlier
received overlay, including typed absent borrow values. Every mutation checks
key/value/write-count/current-byte limits; replacement and deletion adjust
the current overlay rather than hiding earlier violations. Final commit writes
must equal that overlay, sorted, unique and byte-for-byte. Reject/trap/error
return no writes, and accessors enforce that outcome before exposing a row.

Admitted is sealed and owns all returned read/write/trap bytes independently
of request/response storage. The decoder does not interpret SSA, re-evaluate
branch conditions or manufacture execution results. Dynamic event operands
retain the existing private compiled-owner assumption. Native executable
selection/build/run orchestration and one-attempt persistent CAS remain
separate mandatory steps; this admission API cannot reach storage publication.

Structure work is capped at 16MiB units and byte work at 1GiB units, with
separate histories to admit bounded large runtime values without inheriting
the source-literal ceiling. Charges precede processing and are never reset
after failed calls. Allocation failures retain a distinct error. The local
implementation uses the existing SDK buffer ABI for uniquely owned private
evidence storage; it adds no exported runtime or SDK symbol.

Windows evidence, failed preparation and remaining gates:
`docs/development/candidate-state-graph-response-20261107.md`.
