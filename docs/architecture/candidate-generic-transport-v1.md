# Fairy candidate direct scalar generic transport v1

- Owner: [ADR 0494](../adr/0494-broader-scalar-generic-transport.md).
- Status: Bounded Windows prototype; native Linux remains deferred.
- Evidence: [Windows receipt](../ops/windows-candidate-generics-20261102.md).

The authored compiler admits direct `i8`, `i16`, `isize`, `u16`, `u64`, `f32`
and `f64` generic parameters/results alongside the original six scalars. Explicit
type arguments and forwarded parameters retain the existing bounded planner,
specialization identity and ordinary semantic/Flow checks. Identity/forwarding,
local copies and reassignment, and scalar arguments captured by `defer` use their
actual C types. No boxing, inference, implicit conversion or arithmetic is added.

The private type graph assigns the extension tags 32 through 38 in that order.
Existing void (7), callable, literal, slice and C ABI tags keep their meanings.
Source argument handles 4097 through 4103 remain outside the 4096-node graph;
direct substitution materializes a canonical value node. A nested substitution
rejects without publishing a partial graph. References, pointers, views, arrays,
Option/Result and storage generic forms containing the extension remain closed.

`generic_transport_scalars.fy` owns the executable names and C spellings. Concrete
type lookup, early signature diagnostics, detached cleanup type validation and
public type spelling recognize the new direct values. Public interface hashes
and whole-project source restoration retain their existing identities and wire
format; the native fixture restores the complete artifact against current sources
before emission. This does not claim a new whole-package disk-cache acceptance run.

The direct extern-C scalar lane stays unchanged. The native candidate probe uses
private C helpers calling generated ordinary functions, with an independent C
observer supplying signed boundaries, unsigned maxima and exact floating values.
It does not establish candidate aggregate-by-value FFI parity. The owning Go
fixture separately exercises its existing repr(C) observers.

Implementation and acceptance owners:

- `examples/lexer_candidate/src/generic_transport_scalars.fy`, `module_types.fy`,
  `generic_concrete_types.fy`, `generic_signature_diagnostics.fy` and
  `module_unit_generics.fy`.
- `module_unit_cleanup_mir_types.fy`, `module_public_type_text.fy`, `c_text.fy`,
  `module_unit_c_aggregate.fy`, `module_unit_value_terminal.fy` and
  `module_unit_cfg_defer.fy` in the same source directory.
- `examples/lexer_candidate/tests/generic_scalar_arguments.fy`,
  `generic_concrete_types.fy`, `generic_signature_diagnostics.fy` and
  `module_generic_transport.fy`.
- `internal/cli/candidate_generic_transport_test.go` and
  `internal/cli/candidate_generic_transport_pipeline_test.go`.
