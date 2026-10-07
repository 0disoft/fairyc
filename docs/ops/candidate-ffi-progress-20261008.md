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

Borrow casts, pointer-to-opaque-pointer output slots, nominal pointees, and C
scalars belong to subsequent units. Raw dereference, arithmetic, integer-pointer
conversion, and implicit mutability conversion remain outside the current public
contract. Linux execution is deferred to the user's combined validation phase.

Evidence: `evidence/candidate-resource-pointer-transport-20261008.json`.
