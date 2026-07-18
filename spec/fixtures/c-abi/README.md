# C ABI fixtures v0

`manifest-v0.json` is the repository contract for host-independent SysV AMD64 eightbyte
classification and fixed-signature register assignment over the Stage 0 C-representable type
vocabulary. It consumes the same recursive type shape as the target-layout fixtures and records
only semantic class and register identities, not Go-host layout or local compiler output.

The pass cases cover integer and SSE scalars, pointers, function pointers, explicitly based C
enums, arrays, mixed and homogeneous aggregates, hidden memory returns, integer and SSE exhaustion,
and whole-aggregate rollback. Failure cases keep direct array signatures, layout rejection, and
invalid floating enum bases fail-closed. `internal/cabifixture` must execute every case and may expose pass cases to a separate
native probe only after the complete manifest is clean.

This schema is not a public FFI manifest, serialized compiler IR, stable native Fairy ABI, or proof
that Clang and GCC crossed the boundary. It excludes variadics, vectors, x87, unions, packed or
custom alignment, bit fields, atomics, TLS, foreign unwinding, exact stack-area offsets, and every
type rejected by `docs/abi/c-abi-v0.md`.
