# ADR 0442: Nested scalar field assignment

- Status: implemented; Windows native, Linux execution deferred
- Date: 2026-10-04
- Related: ADRs 0394, 0412, 0441

## Decision

Admit direct modification of scalar leaves inside the ADR 0441 acyclic
nonempty plain-struct graph, instead of reconstructing a copied nested record:

```fairy
var Token copy = view[0];
copy.position.start = 10;
copy.position.end += 1;
```

A target must be a chain of declared stored struct fields rooted in a named
mutable local or existing `ref mut` struct receiver. Grouped places are allowed.
Leaf types are bool/u8/i32/i64/u32/usize for `=`, and the existing exact integer
compound operators for compound assignment. Shared or immutable roots, active
root/field loans, boolean arithmetic, aggregate leaf replacement, calls/indexed
receivers, resource roots and traversal through references/pointers remain closed.
The older direct scalar-resource and one-usize-wrapper exceptions are unchanged.

## Places, order and ownership

Nested places are never lowered by mutating a copied intermediate record. MIR
`field-store` retains the root receiver, ancestor names in `Fields` and the leaf
in `Field`. The verifier resolves every component against actual declarations and
checks both receiver and value local types; fake annotations cannot grant writes.
Repeated field names along distinct ancestor types are allowed.

For a nested borrowed receiver, lowering captures the selected reference before
RHS effects, and source flow holds the original loan until the final store.
Rebinding a `var ref mut` on the RHS cannot redirect that store. Through a named
local, replacing the root value on the RHS still stores to the same local place.
Compound assignment captures the old scalar first, evaluates the RHS once, then
uses existing checked arithmetic and writes only after success. Copy mutation
remains independent of the shared vector owner.

The C backend emits the exact nested lvalue and the interpreter walks the live
nested struct cells. No mutable field address or disjoint-field loan is exposed.
The existing MIR codec, package identity and warm cache preserve path names.
SDK stage0.84, language /20, runtime /0, grammar and wire schemas are unchanged;
compiler executable identity separates cached generations.

## Native-discovered correction

A literal out-of-range shift produced unreachable invalid C shifts after its
runtime range guard, and MSVC C4293 became a build error. `reachableShiftCount`
substitutes zero only inside those unreachable value/overflow expressions for
statically invalid immediate counts. The original count remains in the checked
MIR guard and still traps. Valid and nonconstant counts retain the existing C
emission. No warning suppression, cache bypass or weakened trap expectation is
used. Signed division overflow retains its separate existing trap category.

## Verification and limits

Eleven nested native fixtures pass both Windows dev/release (22 outcomes): all
six scalar kinds and integer compound operators, copy isolation, RHS effects
once, reference rebinding, local root replacement, overflow/underflow, zero and
signed-overflow division, and immediate invalid unsigned/negative/i64 shifts.
Source rejection tests cover ownership and unsupported target forms; eleven MIR
mutations reject forged paths and types. Interpreter/codec roundtrip, warm all-hit
cache, existing direct field native regressions and public CLI token smoke pass.
The token analyzer rebases copied positions through a nested mutable receiver.
Related semantic/MIR/interpreter/C-backend/codec/package artifact tests and Go vet
pass. Raw failures, fixes, results and hashes are linked from
`docs/ops/evidence/nested-field-store-20261004.json`.

`TestI64HelpersStrictLinuxConsumers` is skipped on this Windows host; Linux
execution is deferred to the owner's batch. No whole-repository suite or
performance benchmark, runner, PS1, installation, CI workflow or push is added.
