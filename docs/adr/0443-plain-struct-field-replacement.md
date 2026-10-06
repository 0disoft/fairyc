# ADR 0443: Plain struct field replacement

- Status: implemented; Windows native verified, Linux execution deferred
- Date: 2026-10-04
- Related: ADRs 0394, 0412, 0441, 0442

## Decision

Admit exact-type plain assignment of a complete nonempty plain struct into a
stored direct or nested field of the ADR 0441 acyclic scalar-leaf struct graph:

```fairy
copy.position = Span { .start = 10, .end = 20 };
copy.location.position = replacement;
```

Both the root and replacement type must satisfy that graph. A root is a named
mutable local or an existing `ref mut` struct receiver; grouping is allowed.
This is copied-value replacement, including construction, function/block results,
reads of another field and self-assignment. Aggregate compound assignment,
shared/immutable roots, active whole-root or field loans, array/call receivers,
resource roots, and reference/pointer traversal remain closed. Arrays, resources,
borrows, pointers, enums, Option/Result and empty/cyclic structs are not admitted
inside the new plain graph. Existing direct scalar/resource and one-usize-wrapper
store exceptions remain unchanged. No mutable field borrow or split loan is added.

## Selection and evaluation

Select the reference receiver before evaluating the RHS once. Retain its
original loan through the final store, even if the RHS rebinds a `var ref mut`.
For a named local the place remains that local: if the RHS replaces the root,
the final field store targets the current root in that same local. Evaluate the
whole replacement before writing; a failing RHS traps before the field store.
Copies do not alias the input value or a shared vector owner.

## Implementation and compatibility

Semantic mutability checking extends the nested leaf test. The MIR lowerer
captures direct aggregate reference receivers as well as nested receivers. The
existing `field-store` retains ancestor names in `Fields`, its leaf in `Field`,
and exact owner/value types. The verifier checks actual local types and every
declared field; a shared plain-graph predicate guards aggregate store admission.
The C backend assigns the actual nested lvalue. The interpreter clones the
replacement into the live nested cell. Codec and package cache reuse the existing
operation representation, with compiler executable identity separating generations.

No grammar, MIR opcode, wire schema, SDK export, runtime C symbol or target
maturity change is required. SDK stage0.84, language /20 and runtime /0 remain.
The compiler version is revision-derived; this compiler implementation extension
does not change the bundled SDK source/version.

## Evidence

`internal/compiler/testdata/plain_struct_field_store.fy` asserts mixed scalar
contents, >i32 values, RHS count/order, direct/nested/grouped stores, construction,
self-assignment, independent source/copies, root replacement, and direct/nested
reference reselection. Compiler tests execute fresh and codec-decoded MIR.
Source rejection tests protect mutability, loans, nominal exact types, receiver
shapes, resources and nonplain graphs; MIR mutations reject forged definitions,
locals and paths. Native tests execute the same successful program and a failing
RHS in dev/release; the public `examples/struct_vec_view` token analyzer replaces
its copied span with one aggregate assignment.

Review receipt: `docs/ops/evidence/plain-struct-field-store-20261004.json`.
Linux actual execution is deferred to the owner's batch. No performance experiment,
whole-repository test, installation, CI change, new runner or PS1 is introduced.

## Historical backend limitation (resolved by ADR 0444)

A RHS that deterministically overflows in its body before constructing a returned
struct passes `check` but MSVC release build fails on C4702 unreachable-code
warnings (CLI exit 6). The same aggregate-return function without any field store
reproduces on the prior compiler from ADR 0442; this predates replacement admission.
It was retained as an unresolved C-emission issue at ADR 0443. ADR 0444 resolves
this reproduction through bounded C reachability, without suppressing warnings.
The new native trap case uses a mutable local initialized from a function input
and passes dev/release; it validates runtime failure before final replacement.
Exact baseline/current sources, argv, output and hashes are retained in the review
receipt. Linux execution remains deferred.

## Fieldless value correction (2026-10-06)

After private fieldless construction became executable, field replacement still
rejected the same values. Field-store admission now includes private empty leaves
and structs containing them, through direct or nested plain-struct fields.
The shared acyclic graph walker has a separate stored-value entry point; array
and view element admission, field borrowing and dereference stores keep their
existing nonempty graph. Empty `@repr(C)` records remain rejected.

The selected receiver is retained before RHS effects. Sibling writes, root
replacement and early returns use the existing field-store execution paths.
The interpreter and C backend required no new representation or runtime code.
Focused compiler/MIR tests and Windows low/optimized native execution passed;
candidate empty-field checks now use Go-interpreter outcomes rather than a fixed
native expectation. Evidence: `docs/ops/evidence/empty-struct-field-store-20261006.json`.
SDK, language, grammar and wire versions are unchanged. Linux remains deferred.
