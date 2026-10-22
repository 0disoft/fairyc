# ADR 0477: Repr(C) integer array fields

- Status: Accepted for prototype
- Date: 2026-10-23
- ABI source: `docs/abi/c-abi-v0.md`
- Extends: ADR 0468's array storage, ADR 0476's foreign record arguments
- Supersedes: the rejection of repr(C) array fields in ADR 0468, for this subset only

## Decision

Admit fixed-array fields of `u8`, `i32`, `i64`, `u32` and `usize` in nonempty acyclic
`@repr(C)` records. Lengths are `1..=4096`, retaining the provisional by-value array limit.
Nested records may contain these fields. Bool, enum, floating, C-scalar, pointer, resource,
borrowed, nested-array and nominal-struct array elements remain outside this admission.
This adds no bare foreign array parameter or result.

The field is an actual named C array member, such as `uint32_t words[3]`, not a private array
wrapper or pointer. The target layout engine supplies size, alignment and every field offset;
generated assertions must agree with the C compiler. Record argument/result transport uses the
exact public record declaration and lets the C compiler choose registers or memory.

Source construction initializes every field explicitly. Whole-array field reads copy values,
including reads through a borrowed record. Whole-field replacement through a mutable record or
mutable record borrow, and nested whole-field replacement, preserve existing ownership and
evaluation order. Record copies and argument/result copies retain independent array contents.
The emitter bridges raw members and private array value wrappers with element copies; it must
not reinterpret raw array storage as a wrapper struct. Internal constructors and field stores
consume already-evaluated MIR locals, so copying never repeats source expressions.

References, views, slices, direct element stores and direct field iteration remain closed for
these raw members. Read indexing uses an array value copy instead of borrowing the C member.
Copying the whole field into a normal array local retains ordinary local-array operations.
MIR verification must reject a raw member borrow even if source checking is bypassed. This
boundary avoids introducing an incompatible wrapper pointer or changing existing array-reference
transport.

## Compatibility and validation

There is no new syntax, MIR opcode, serialized field, SDK export, runtime symbol or CLI option.
Existing interface array types and exact compiler identity distinguish the admission; no target
maturity flag is promoted. The interpreter keeps array value semantics independent of C storage.

Focused compiler tests round-trip MIR, execute copy/replacement checks in the interpreter and
emit both Linux and Windows profiles. Source tests reject unsupported element/length shapes and
raw-field views/stores; a MIR mutation test rejects a private wrapper borrow after its containing
record is changed to repr(C).

The existing shared foreign record probe now includes independent C array declarations and
observations for a 64-byte record and an 8-byte record. It checks layout, all array elements,
record arguments/results, caller-copy isolation and replacement after a copied read. The Windows
probe links the canonical scalar runtime with its companion C unit for checked indexing; the
Linux probe already links that runtime separately.

The focused MSVC 14.51.36231 test passed all four C11/C17 low/optimized combinations. The initial
full Windows smoke exposed the missing runtime link at this probe; after correction, only this
probe was rerun. Hosted Linux GCC/Clang crossings and the full repository Go suite were not rerun.
Previous F1 receipts do not attest arrays. Current commands and evidence limits are recorded in the
[F2-1 update](../development/phase2-target-scope.md#f2-1-integer-array-fields).
The later [hosted F2-5 receipt](../development/phase2-target-scope.md#hosted-f2-5-receipt-2026-10-23)
records GCC 13.3 and Clang 18.1 crossings in both directions and profiles at `f79e55c6`, with all
nine jobs passing.
