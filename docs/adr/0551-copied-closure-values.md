# ADR 0551: Bounded copied closure values

- Status: Accepted for staged prototype implementation
- Date: 2026-11-08
- Extends: ADRs 0496 and 0546; C callbacks remain separate

## Source type and lifetime

The first closure value type is `fn i32(i32)`. It is distinct from an unsafe
extern-C callback. Function-type syntax retains its structural result and
parameter types; executable admission initially requires exactly this direct
i32 result and one direct i32 parameter. Other function signatures remain
unsupported. The existing untyped direct-call closure family is preserved.

Construction uses an explicitly typed immutable binding at the outer function
body level:

```text
fn fn i32(i32) make(i32 offset) {
    let fn i32(i32) action = fn [i32 offset] i32(i32 x) { return offset+x; };
    return action;
}
fn i32 apply(fn i32(i32) action,i32 x) { return action(x); }
```

Captures remain explicit, copied i32 values, zero through four in written order.
They are evaluated once during construction. Existing pure checked arithmetic,
up to sixteen immutable i32 locals and sixty-four expression nodes remain the
closure-body boundary. Ordinary function arguments/results, immutable local
copies and fields of ordinary non-repr(C) structs may carry the value. Extract
a field to a local before calling it; the first invocation syntax accepts only
a named local or parameter. Different literal targets can share this type.

There is no reassignment, global/state storage, array/Option/Result/Vec wrapper,
generic argument transport, closure capture, resource/borrow/pointer capture,
FFI crossing, async or cross-thread contract in this slice. Structs containing
closures cannot gain repr(C), wrapper, raw-pointer or foreign transport through
this decision. Existing rejection paths must remain closed independently.

## Private representation and execution

A value contains a private invoke pointer and four inline i32 capture slots;
unused slots are zero. A literal-specific adapter invokes the checked lifted
helper. No environment points into its creator's stack frame, so returning and
copying a value remain safe after that frame exits. There is no allocation,
reference counting, new cleanup obligation or exposed size/field/ABI promise.

Evaluate the named callable value before the value argument, each once. Body
locals and checked traps execute on each invocation in original order. Creation,
copy and struct storage do not run the body. Equality, casts and pointer access
to this representation remain unsupported.

Verified execution must retain explicit construction and value-call operations.
Construction binds a real safe pure lifted target, capture count/types and
signature; invocation binds a fully initialized value of the exact closure type.
The existing call-successor and trap/ordering rules remain applicable. Closure
construction is a reachability edge even when the factory never invokes it.
Forged captures, targets, signatures, callable operands and body facts must fail
before backend execution.

## Identity, artifacts and delivery

The canonical internal leaf type is `fn i32(i32)` with a distinct HIR identity;
it must never alias a C callback or infer a type from arbitrary source text.
Public interfaces expose only this signature. Stable artifacts record helper
identity and capture values/operands, never an actual native pointer. Restore
rebinds and revalidates the target and source-owned body before execution.

New executable record families require deliberate exact private format gates.
Where possible, preserve existing no-closure bytes through separately selected
closure format versions; reject a closure record under the old version and
noncanonical version/feature combinations. Final format names belong to the
codec implementation review, not an accidental permissive decoder extension.

Delivery proceeds through syntax/structural type retention, verified Go
source/execution, Fairy source/execution, restored package/cache parity and
whole CLI acceptance. A syntax-only stage must still reject execution rather
than claim partial runtime support. Acceptance includes returned factories,
different helpers through one parameter, struct copies, snapshot/argument/trap
order, forbidden shapes, forged artifacts and fresh-process cache restoration.
The existing invalid copied-closure Go/Fairy status difference remains a gap to
close explicitly. Windows small commits precede batched Linux; remote actions
and the final self-build/distribution/CI gates remain separate.
