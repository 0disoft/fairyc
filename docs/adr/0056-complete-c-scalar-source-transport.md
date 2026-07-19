# Complete Target C-Scalar Source Transport

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-19
- Product source: `docs/product/02-spec.md`
- ABI source: `docs/abi/c-abi-v0.md`
- Target source: `internal/target/spec.go`

## Context

ADR 0052 through ADR 0055 established exact, transport-only source identities for `c::int`,
`c::size_t`, target-specific `c::ssize_t`, and `c::long`. The same source-to-MIR-to-C path remained
closed for the other twelve scalars already fixed by the selected TargetSpec. Opening them by
copying one switch and one native counter per type would create manual drift. Opening their literals
or operators would instead require C promotion, conversion, plain-`char`, floating-point, NaN,
rounding, and `_Bool` normalization rules that this transport slice does not need.

## Decision

1. The closed source C-scalar vocabulary is the complete sixteen-entry TargetSpec family:
   `c::bool`, `c::char`, `c::double`, `c::float`, `c::int`, `c::long`, `c::longlong`, `c::schar`,
   `c::short`, `c::size_t`, `c::ssize_t`, `c::uchar`, `c::uint`, `c::ulong`, `c::ulonglong`, and
   `c::ushort`.
2. HIR owns that closed canonical vocabulary. Semantic analysis accepts each exact type identity;
   MIR retains it on foreign results, locals, and matching foreign arguments. No type aliases to
   Fairy fixed-width, pointer-width, boolean, or floating types are introduced.
3. The C backend owns one ordered transport table mapping every source type to its TargetSpec key,
   exact C spelling, emitted feature, and optional provider. Emission fails closed if the table and
   HIR vocabulary differ, a target scalar is absent, or provider metadata differs.
4. Generated C emits target-derived size and alignment assertions for every used C scalar. Signed
   and unsigned integer families also receive signedness assertions. `_Bool`, `float`, and `double`
   have no signedness assertion. Plain `char` uses the selected target's signed `c.char` fact.
5. This remains transport-only. Fairy literals, arithmetic, comparison, casts, promotions,
   interpreter host bindings, raw pointers to C scalars, aggregate fields, and by-value aggregate
   ABI do not become available. The native floating values are exactly representable finite values;
   the probe makes no NaN, infinity, signed-zero, rounding, or exception-mode claim.
6. One semantic family fixture covers the twelve newly opened identities, raising the semantic v0
   total to twenty-three. Existing individual `c::int`, `c::long`, `c::size_t`, and `c::ssize_t`
   fixtures remain independent stage evidence.
7. One Fairy caller and one hand-written C companion cross all twelve new scalars and return `42`
   only when every consumer observes its expected value. `fairyc.native-c-report/v20` adds
   `foreignCScalarFamilyCallExecutions`, expected to be four after both compiler directions and
   both profiles execute successfully. Existing layout, C-emission, and individual scalar counts
   do not change.

## Consequences

- Every C scalar already declared by the first target can cross an imported call without losing its
  exact C identity.
- Later C-scalar work is about value construction, operations, pointers, aggregates, callbacks, and
  safe wrappers rather than adding more transport-only names.
- Native execution remains configured but unobserved on this Windows host, so this does not promote
  target maturity or claim floating-point conformance.
