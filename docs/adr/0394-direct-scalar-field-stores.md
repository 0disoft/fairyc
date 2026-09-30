# ADR 0394: Direct scalar field stores

- Status: Accepted
- Owning contract: `docs/language/0002-borrows-and-return-origins.md`

## Context

Typed field assignments already reach semantic analysis, but executable lowering
previously rejected every field store. Owner-qualified mutable methods therefore
could not update even scalar state. Interpreter aggregate copies also need to
remain independent once that state can change.

## Decision

Execute plain `=` assignments to direct `i32`, `u8`, `u32`, `usize`, and `bool` fields of
mutable local structs/resources and existing `ref mut` aggregate receivers.
The receiver is a direct path (optionally grouped); evaluate the right-hand side
once, require its exact field type, and retain resource visibility and moved-owner
checks. Active loans protect the whole root, including other fields of that root.

A private MIR `field-store` operation has no destination and carries the owner
type, declared field, exact scalar type, receiver and value. Verification checks
operand liveness and concrete types against the aggregate declaration. The native
consumer independently validates the shape and types. Decoded private artifacts
are reverified; this is not a public MIR compatibility promise.

The interpreter writes scalar state in the referent. Inert by-value aggregate
copies, including constructor payloads, have detached storage; borrowed values
retain their referents and resources retain linear ownership identity. Boolean
negation retains its boolean runtime type when used as a stored or asserted value.

## Boundaries

Nested field paths and general aggregate-field replacement remain unsupported.
ADR 0412 later admits checked direct integer field compound assignments using
the existing read/arithmetic/store operations. A later bounded extension admits direct plain
replacement of a non-`repr(C)` struct containing exactly one `usize` field from
a typed local. The interpreter detaches the copied value; MIR verification and
the C backend require the same shape. Multi-field structs, resources and
`repr(C)` wrappers remain excluded. This does not introduce mutable
field-address projection, field-disjoint borrowing, temporary owners, public
aggregate ABI or new source syntax.

Language revision 14, SDK `0.0.0-stage0.59` and runtime revision 0 stay unchanged:
this completes a bounded executable subset of existing assignment semantics,
without changing grammar, SDK APIs or the runtime ABI.

## Validation

Regressions cover mutable receivers, value-copy independence, inert state stored
inside a resource, last-use loan release, conflicting root loans, malformed MIR,
artifact round trips and the unsupported boundaries. Windows stable CLI tests
exercise both default and release native profiles. These tests establish behavior,
not performance or non-Windows native execution evidence.

The unsigned extension covers zero and maximum `u8`/`u32` values, local and
mutable-reference receivers, owned resource fields and detached copies. Out-of-range
source literals, mismatched unsigned types and malformed MIR immediates remain
rejected rather than truncated or promoted.
