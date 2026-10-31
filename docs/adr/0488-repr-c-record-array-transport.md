# ADR 0488: Repr(C) record-array transport

- Status: Accepted for prototype
- Date: 2026-11-01
- Contract: [C ABI v0](../abi/c-abi-v0.md)
- Extends: ADR 0477, ADR 0482, ADR 0483, ADR 0484 and ADR 0485 bounded array transport

## Decision

Admit fields of the form `Element[N]` in actual acyclic nonempty `@repr(C)` records when `Element`
is an actual acyclic nonempty `@repr(C)` record and `N` is in `1..=4096`. A record-array field may
occur directly in a `@repr(C)` record or recursively through a record-array element's record
fields. The containing record and every element definition must already be admitted by the target
layout and C ABI slices.

An element may contain existing ABI leaves, nested `@repr(C)` records, the existing bounded
integer, floating, exact C-scalar and explicitly based C enum arrays, or recursively admitted
record-array fields. Existing scalar, enum and pointer admission rules apply throughout the
elements. The selected target owns every C scalar size, alignment and offset and every pointer
target's admission; this decision adds no pointer form. Empty records, cycles, resource fields,
safe borrows, slices, owning tags and direct array-of-array element types remain closed.

The C member is a named array of the generated C structure type, `Element field[N]`, not a pointer
or a private transport structure. Its size, alignment and offsets are target-computed, and the
generated assertions must agree with the C compiler. C-produced record values support whole-field
reads, inferred and typed local array values, internal by-value array arguments and results, record
copies and reconstruction, direct whole-field replacement through a mutable record or record
reference, and nested record reads and copies.

Lowering uses private assignable array wrappers and copies one C structure element at a time. It
does not treat raw C array storage as a wrapper and must not implement transport by `memcpy`
reinterpretation. Nominal element identity, field definitions, and array length are retained through
HIR, MIR, the MIR codec, public interfaces, and cache identity. Two records with identical layout
remain distinct when their nominal identities or fields differ. The existing cache identity format
carries this information without a new serialized field.

Record arrays whose elements are already in the plain-scalar-field executable subset retain their
existing literals, indexing, length, views and iteration. Record elements newly admitted outside
that subset, including elements carrying floating fields, exact C scalars, enums or nonplain nested
record/array fields, are transport-only. Their element literals, indexing, stores, length,
views/iteration and other element operations remain closed. Source checking and independent forged
MIR checks enforce that boundary.

Bare foreign array parameters and results remain closed. No new direct private, resource or tagged
payload arrays with these nonplain record elements are admitted, and nested stores into the new
nonplain record shape remain closed. The existing integer, floating, exact C-scalar and C enum
array admissions and their element-operation boundaries are unchanged.

This adds no source grammar, MIR opcode, serialized format, runtime symbol, dependency, SDK export,
CLI option, CI workflow or target-maturity promotion.

## Evidence

Local Windows validation passes. The focused native command is:

```text
go test -mod=readonly ./tools/dev/fairyc-windows-host-smoke -run '^TestReprCRecordFieldNativeCrossing$' -count=1 -v
```

MSVC 14.51.36231 passes C11/C17 at low and optimized profiles in four separate
`caller.obj`/`callee.obj` links. Every execution exits 42 with empty stdout and stderr. Nine probe
cases run per profile and optimization combination: three record families with three rotations
each. The cases cover a four-byte record result; a mixed `c::long`, `f64` and nested-leaf record;
and a record-array element containing `c::int[2]`, `f32[2]`, an explicit C enum array with
`UINT_MAX` outside the declared variants, and a nested `@repr(C)` leaf array.

Focused HIR, semantic, MIR, MIR-codec, module-interface, package-artifact and C-backend tests pass.
Compiler tests selected by the `(Array|Struct|CEnum|Transport)` filter also pass. Native Linux
execution remains deferred and is not attested by this Windows result.
