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

The implementation is local commit `e5b03f3f`; the final 4,096-element probe and this evidence
change are prepared for the following local commit. The current
[Windows receipt](../ops/windows-record-array-transport-20261101.md) records the complete command
and rerun history.

The entire `internal/foreignstructprobe` package passes in 19.670s with only the user-deferred
Linux-native test skipped. Windows and POSIX emission inventories are 133 and 139 record shapes,
with 428 and 448 generated functions. Four MSVC 14.51.36231 C11/C17 low/optimized executions pass
through separate `caller.obj`/`callee.obj` links, each carrying 133 records x 3 value sets = 399
record-case entries. The fixture C bridge enters actual generated Fairy bodies, and two C-to-Fairy
assertion-trap executions pass with status 134, trap-channel control byte 6 and empty streams.

The forward record-array test passes in 5.524s. Four executions each run 12 record-array cases per
compilation profile, including lengths 1, 3 and 4,096. The four families cover small, mixed
`c::long`/`f64`/nested-leaf, exact C-scalar/float/C-enum/nested-array fields, and the 4,096-element
case. The large case exercises native structure-array stack transport and copies. It adds no source
export or general aggregate callback.

Full tests pass for the seven HIR, semantic, MIR, MIR-codec, module-interface, package-artifact and
C-backend packages. Compiler-package coverage is limited to the
`(Array|Struct|CEnum|Transport)|^TestCRecordArray` filter. MIR mutations reject `non-repr(C)`
elements, empty elements, cycles and element operations. Interface hashes distinguish nominal
element type, length and field shape; the cache-context rebind changes package identity from 3 to
19. Scoped `go vet` and `git diff --check` pass.

The first inventory run after implementation failed because stale tests rejected record arrays and
expected 116/122 shapes. Changing the rejection to a `non-repr(C)` element, admitting `@repr(C)`
element/container records, and setting the interim count to 129/135 made the limited rerun pass;
the 4,096-element family then produced the final 133/139 inventory and clean full package pass.
Native Linux execution, WSL, a full `go test ./...`, hosted CI, push and publication remain
deferred or unexecuted.
