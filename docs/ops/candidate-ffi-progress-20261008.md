# Fairy candidate FFI progress

The Go compiler remains the bootstrap. These units implement the existing C ABI
contract in the Fairy candidate; they do not introduce new public language rules.
Product typing, HIR/MIR verification, and C emission remain Fairy source.

## Raw pointer transport

`pointer_types.fy` gives fixed-width scalar and opaque `void` pointers a namespace
separate from safe references, arrays, enums, and Option/Result. Both `ptr T` and
`ptr mut T` preserve their exact type through declarations, parameters, returns,
local copies, fields, owned HIR, MIR, and generated C prototypes. `null` requires
a raw-pointer context; matching pointer types support only `==` and `!=`.

MIR operation 36 represents typed null and rejects nonempty operands or payloads.
The native API test checks forged null operands and C output preservation on
verification failure. The observer links an independent provider translation unit
and compares program values in development and release C builds.

Nominal pointees belong to subsequent units. The exact opaque-pointer output
slot and target C scalar transport are described below. Raw dereference, arithmetic, integer-pointer
conversion, and implicit mutability conversion remain outside the current public
contract. Linux execution is deferred to the user's combined validation phase.

Evidence: `evidence/candidate-resource-pointer-transport-20261008.json`.

## Explicit borrow casts

Exact shared/mutable scalar references may become a matching raw pointer inside
an unsafe function or lexical block, without strengthening mutability. Direct
local-place borrows and borrow variables may erase their pointee to opaque void.
Byte views and shared text views contribute their data pointer. The descriptor
is not reinterpreted as the data buffer.

Owned HIR pointer-cast kind 28 independently rechecks lexical unsafe ancestry;
MIR operation 37 preserves the exact source/target types and mandatory unsafe
marker. Stripping the source block authority or MIR marker is rejected. The
source borrow remains active through conversion evaluation. Its safe origin is
not attached to the resulting raw pointer: foreign lifetime/aliasing obligations
remain with the unsafe wrapper, as in the existing public ABI contract.

Evidence: `evidence/candidate-resource-pointer-casts-20261008.json`.

## Target C scalar transport

Fifteen TargetSpec C scalar families have distinct candidate identities and
preserve their exact C spelling through foreign results, locals, copies,
parameters, returns, and ordinary stored fields. Fairy `bool` remains separate
from `c::bool`. C scalars have no implicit literals, arithmetic, comparison, or
integer conversion; only explicit `i32` / `c::int` conversion is admitted.

`c_abi_contract.fy` emits size/alignment checks for used families and signedness
checks for integer families. The `c::int` cast additionally requires the exact
signed 32-bit range. `long` / `unsigned long` check LLP64 on Windows and LP64 on
Linux; the Linux branch has not been executed in this task. POSIX-only
`c::ssize_t` remains rejected until the candidate carries explicit target
selection. It is never silently substituted with `intptr_t`.

The independent provider returns and consumes every supported C scalar family.
Evidence: `evidence/candidate-resource-c-scalars-20261008.json`.

## Mutable opaque-pointer output slot

The candidate admits only the existing `ptr mut ptr mut void` output slot.
`pointer_identity.fy` owns the closed raw-pointer identities so the safe-reference
namespace can represent a borrow of `ptr mut void` without an import cycle.
An unsafe cast takes either the direct local mutable borrow or an exact local
reference variable. The resulting C call receives `void **`, writes the pointer,
and can clear it again; subsequent reads observe the original pointer storage.

Shared outer/inner pointers, other pointees, third-level nesting, shared-borrow
strengthening, and safe casts remain rejected. Option/Result cannot encode the
new pointer borrow: the bounded tag encoding explicitly rejects it rather than
overlapping an existing tag identity. Raw pointers still carry no safe origin.

Evidence: `evidence/candidate-resource-output-slot-20261008.json`. Both candidate
profiles, the independent two-profile provider, the foreign-call API tests, and
the pointer/cast/C-scalar regressions are recorded there. Linux remains deferred.
