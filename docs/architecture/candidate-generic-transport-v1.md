# Fairy candidate direct scalar generic transport v1

- Owner: [ADR 0494](../adr/0494-broader-scalar-generic-transport.md).
- Status: Bounded Windows prototype; native Linux remains deferred.
- Evidence: [source and native receipt](../ops/windows-candidate-generics-20261102.md)
  and [full CLI and persistent-cache receipt](../ops/windows-candidate-generics-cli-cache-20261103.md).

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
rejects without publishing a partial graph, except for the direct payload of a
root flat Option admitted by [ADR 0547](../adr/0547-flat-option-generic-transport.md).
References, pointers, views, arrays, nested Option/Result and storage generic
forms containing the extension retain their independent boundaries.

The flat Option follow-up uses existing kind 5 and scalar tags 32-38. Only a root
`Option<S>` with a direct named scalar payload receives the permission; recursive
type traversal resets it. Written scalar leaves and descriptor/interned generic
substitutions canonicalize to the same graph nodes. Generic arguments remain
scalar, so this does not admit `id<Option<S>>`. Both ordinary resolution and
specialization roll back an interned payload when wrapper publication fails.
The separate aggregate-enabled resolver retains its previous policy.

Some/None construction, inert copies, reassignment, ordinary and generic calls,
returns and exhaustive extraction use the existing checked IR and tagged C
emitter. No boxing, allocation, new ABI, scalar arithmetic or deferred Option
capture is introduced. See the [Fairy execution record](../development/candidate-flat-option-transport-20261107.md)
for graph/capacity, current-source artifact restoration, independent Go
diagnostics and Windows generated-C observations. The
[flat Option CLI/cache record](../development/candidate-flat-option-cli-cache-20261107.md)
adds both Windows profiles, actual product builds/run/native-test dispatch,
fresh-process disk hits, current packet observations, body invalidation and
repair. These are separate from the earlier direct-scalar receipts below.

`generic_transport_scalars.fy` owns the executable names and C spellings. Concrete
type lookup, early signature diagnostics, detached cleanup type validation and
public type spelling recognize the new direct values. Public interface hashes
and whole-project source restoration retain their existing identities and wire
format. The [2026-11-03 Windows receipt](../ops/windows-candidate-generics-cli-cache-20261103.md)
checks ten separate CLI processes per compiler profile through cold and locked
checks, development/release build and run, JSON test listing, and interpreter and
native tests. Its cross-package persistent-cache test runs a public `pick<T>` and
private `sink<T>` through fourteen fresh processes per profile, and both compiler
profiles passed. Built products and `run` return 42 with empty output, and each
test invocation reports one passing test.

The direct extern-C scalar lane stays unchanged. The native candidate probe uses
private C helpers calling generated ordinary functions, with an independent C
observer supplying signed boundaries, unsigned maxima and exact floating values.
It does not establish candidate aggregate-by-value FFI parity. The owning Go
fixture separately exercises its existing repr(C) observers.

The CLI fixture keeps private typed observer entries reachable through
literal-false guarded calls to recursive source-only seeds, which never execute.
It extracts their exact unique C signatures from a captured fresh `--packet-input`
rather than hard-coding function IDs or exporting public source symbols. A
test-only capture proxy saves those bytes and forwards arguments and streams to
the original C-only Go companion, then restores it during cleanup. A separate C
observer supplies real scalar arguments, calls the seven captured entries and
compares complete object bytes, including signed zero, `UINT64_MAX` and signed
negative values. This observer is compiled with the low C consumer profile only;
product builds continue to use development and release profiles. The seven
extension types remain transport-only: this evidence adds no scalar literal or
arithmetic admission.

The new fixture compiles C11. It does not add C17 evidence; the earlier Go
repr(C) C11/C17 receipt retains that separate result. Normal interpreter tests use
the original scalar constructors and do not establish seven-type literal,
arithmetic or complete interpreter-runtime execution.

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
- `internal/cli/candidate_generic_cli_test.go`,
  `internal/cli/candidate_generic_cli_observer_test.go` and
  `internal/cli/candidate_generic_cli_cache_test.go`.
