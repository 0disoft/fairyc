# ADR 0511: Independently verified Fairy state graph IR

- Status: Accepted for the private detached storage/control graph prerequisite
- Date: 2026-11-07
- Extends ADR 0508 without granting current-source or execution authority.

The Fairy compiler now has an owned transaction graph representation independent
of parsed source. Its header identifies one selected, independently validated
seven-codec layout and one nonzero transaction identity. Numeric identities are
not capabilities to access a backend. Only the private verifier constructs its
opaque `Verified` resource, which owns the graph and an independent copy of the
validated layouts. Callers obtain immutable borrows; no mutable proof accessor
or unchecked proof constructor is exported.

The initial family admits the six existing cell/map read/write/delete operations,
literal or block-local typed operands, read value/found results, typed edge
arguments and jump/branch/commit/reject/trap terminators. Canonical effect bits
are storage-read 1 and storage-write 2. Every operation and terminator has the
existing canonical cost 1. Values have nonzero, globally unique u32 identities,
while use sites must resolve in their own block. Values from another predecessor
must enter through that block's ordered parameters. Layout storage codecs remain
separate from ordinary executable scalar type IDs.

Blocks are dense and source ordered, parameters and operations occupy exact
packed spans, and edge argument ranges account for the entire argument vector.
The entry has neither parameters nor predecessors. Independently checked edge
types precede bounded reachability and topological checks. Cycles, unreachable
blocks, invalid fields, missing effects, noncanonical encodings, operand drift,
mixed literal/dynamic operands and implicit exits reject before publication.

This private step retains the existing 64-block and 1,024-operation graph bounds,
with a 1 MiB literal/category arena and the existing independent 16 MiB work
ledger. Span arithmetic uses subtraction after checking the start. Each scan,
definition and copied row is charged; exact work succeeds and one-short work
rejects without overflowing the ledger. Error classes are the existing private
1001 malformed, 1002 bound/work and 1003 allocation. Dynamic bytes are raw,
including present-empty; text uses strict UTF-8. Storage key/value execution
limits remain owned by the later transaction attempt.

Helpers, root borrows/views/copies and source lowering remain closed in this
initial family. No graph marker or proof constitutes a current-source Complete,
artifact/cache hit, native execution or ordinary CLI admission. Those are later
steps, along with Linux and final acceptance. Existing ordinary five-codec
source, wire and cache behavior is unchanged.

Independent Go verifier comparisons and Windows low/optimized execution belong
to [the implementation record](../development/candidate-state-graph-ir-20261107.md).
No stable CLI/SDK/runtime ABI, dependency, database/provider, default compiler
bound, repository policy or CI/runner change is introduced.
