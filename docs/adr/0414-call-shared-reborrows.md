# ADR 0414: Shared reborrows at calls

- Status: implemented
- Date: 2026-10-01
- Owner: [Borrow and return origins](../language/0002-borrows-and-return-origins.md)

The practical workloads need to pass an existing `ref mut T` to a reader
accepting `ref T`. Admit that exact referent conversion only at an internal
function argument or method receiver. Evaluate the original argument once and
create a shared view; do not move the owner or create another mutable owner.
This includes admitted scalar/aggregate references and exclusive byte views.

Earlier arguments stay held while later argument expressions execute. Reject
overlapping mutable access in either argument order, including calls and stores
inside later argument blocks. Multiple shared readers are compatible. A
call returning no borrow releases this shared access on completion.

A returned shared reference retains the actual argument's original origin and
root loans, together with the reference parent's shared loan. Copies and the
single safe-borrow leaf in Option/Result retain that proof. Mutating the parent
or original whole root before the returned reference's last use is E2101;
mutation after its last use is allowed. Parameters without a local root loan
retain their parameter origin and are protected by the same access check.
Returning a local owner remains E2102, and multiple possible origins retain
the existing E2103 rule. No source lifetime annotation is introduced.

Direct assignment, annotated binding, return, casts and tagged construction
do not gain an implicit mutable-to-shared conversion. They still require their
ordinary exact types; a call may explicitly return a valid shared reference.
Exclusive subview reborrowing, mutable field-address projection, disjoint field
loans and general raw-pointer conversion remain separate capability work.

Typed HIR and MIR carry `shared-reborrow`, with one exclusive local operand,
the exact shared destination type, exclusive `OwnerType`, and operator `share`.
The verifier derives operand types from local definitions and rejects altered
mutability, referent, missing operand and invalid operation shapes. The
interpreter preserves pointer/owner identity and copies a byte-view descriptor
with mutable access disabled. C transport copies the pointer or converts the
exclusive byte descriptor to the existing shared descriptor.

MIR version 1 remains the envelope: this additive opcode uses existing fields.
Old consumers reject an unknown operation rather than interpreting it as a
cast. Language /20, SDK stage0.73 and runtime ABI /0 remain unchanged because
this admits a previously unsupported subset without changing public SDK
symbols, source spelling or runtime representation.

Acceptance covers return escape, retained aliases, tagged payloads, parameter
references, scalar/byte/whole-aggregate stores, argument ordering and last-use
release. The [executable example](../../examples/shared_reborrow/README.md)
also covers method receivers, artifact round trips, malformed MIR rejection,
interpreter execution and low/optimized Windows MSVC execution. This change
makes no storage durability claim; physical power-loss testing stays deferred.
