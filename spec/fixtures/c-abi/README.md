# C ABI fixtures v0

`manifest-v0.json` and `manifest-windows-msvc-v0.json` are the target-bound repository contracts for
host-independent C ABI classification and fixed-signature assignment over the Stage 0
C-representable type vocabulary. They consume the same recursive type shape as the target-layout
fixtures and record only semantic transport and register identities, not Go-host layout or local
compiler output.

The pass cases cover integer and SSE scalars, pointers, function pointers, explicitly based C
enums, arrays, mixed and homogeneous aggregates, hidden memory returns, integer and SSE exhaustion,
and whole-aggregate rollback. The failure cases exercise three rejections: direct array signatures
fail with `invalid-type` because fixed arrays at a function boundary require a containing
C-representable struct; unsupported
layouts such as the empty aggregate fail with `layout-rejected`; and floating C enum bases fail
with `invalid-c-enum-base`. `internal/cabifixture` must
execute every case and may expose pass cases to a separate native probe only after the complete
manifest is clean.

The Windows manifest covers Microsoft x64's four shared ordinal argument slots, integer versus SSE
register selection by slot, later stack arguments, one-slot hidden returns, 1/2/4/8-byte aggregate
integer transport, and indirect odd-size or larger aggregate transport. Its twelve pass plans feed
one generated caller and one generated callee C11 translation unit. The configured Windows host
smoke compiles, links, and executes that pair with local x64 MSVC at low and optimized profiles.
This same-compiler local crossing is not source-level Fairy FFI, independent-compiler or assembly
evidence, or a hosted receipt. Exact stack offsets, shadow-store observation, temporary alignment
and lifetime, variadics, vector exceptions, retained callbacks, and exports remain deferred.

This schema is not a public FFI manifest, serialized compiler IR, stable native Fairy ABI, or proof
that a target compiler crossed the boundary. It excludes variadics, vectors, x87, unions, packed or
custom alignment, bit fields, atomics, TLS, foreign unwinding, exact stack-area offsets, and every
type rejected by `docs/abi/c-abi-v0.md`.
