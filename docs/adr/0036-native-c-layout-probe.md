# ADR 0036: Probe target layouts with native Clang and GCC

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-19
- Target source: `internal/target/`
- Layout source: `internal/layout/`
- Fixture source: `spec/fixtures/layout/manifest-v0.json`
- Implementation source: `internal/layoutfixture/` and `internal/cbackendnative/`

## Context

ADR 0031 made target layout computation executable and froze fourteen pass/fail cases, but every
result still came from the Go engine and target table. The C backend now relies on those results for
private resource and tagged structs. A shared arithmetic or mapping mistake could therefore satisfy
Go fixtures and generated `_Static_assert` expectations while disagreeing with both target C
compilers.

The five failure cases describe Fairy C input rejection—empty aggregates, duplicate fields, invalid
lengths, unknown scalars, and arithmetic overflow. Feeding malformed equivalents to a C compiler
would test a different language and could turn an expected Fairy rejection into implementation-
defined or extension behavior. Only the nine validated pass cases belong in a native layout probe.

## Decision

1. `internal/layoutfixture.NativeProbeCases` returns isolated copies of the nine validated pass
   cases only after the complete fourteen-case manifest is clean. Case order remains the manifest's
   strictly sorted canonical order.
2. The native runner generates one deterministic ISO C11 translation unit with private numeric
   typedefs. Fixture IDs, source type names, field names, absolute paths, and build-local identities
   do not enter C text.
3. Language fixed-width scalars use the corresponding `<stdint.h>` representation; Fairy `bool`
   uses `uint8_t`; `isize` and `usize` use `intptr_t` and `uintptr_t`; floats use `float` and
   `double`. C scalars use their exact C spelling. An explicitly based C enum probes its declared C
   scalar base rather than an implementation-selected plain C enum.
4. Pointers use `void *`, function pointers use `void (*)(void)`, arrays use positive constant
   bounds, and structs use declaration-order numeric fields. Nested aliases are emitted before the
   type that consumes them.
5. Every leaf and recursively generated compound type receives C11 `_Static_assert` checks for
   `sizeof` and `_Alignof`. Every struct field additionally checks `offsetof`, field `sizeof`, and
   field-type alignment. Together these observations cover the fixture's top-level padding without
   serializing padding bytes.
6. The same resolved Clang and GCC toolchains and `low`/`optimized` profiles used by the C
   differential compile, link, and execute the probe. Four zero-output executions must return zero;
   a compile-time layout mismatch fails before an executable exists.
7. The native report advances from `fairyc.native-c-report/v2` to
   `fairyc.native-c-report/v3`, adding the sorted `layoutCases` identity list and
   `layoutProbeExecutions`. The expected configured shape is nine named cases and four executions;
   a redundant case count is omitted so identity drift cannot hide behind the same number.
   ADR 0037 later advances the report to v4 for sanitizer identity and execution count.
8. The existing `c-layout-probes` and `hosted-linux-receipt` target missing-evidence markers remain
   until a clean hosted Linux report is observed. Implemented code and a configured job are not
   native evidence by themselves.
9. These probes do not implement source-level arrays, structs, enums, or FFI; prove SysV AMD64
   argument/return classification; inspect object metadata; test bidirectional calls; or establish
   public Fairy ABI compatibility.

## Consequences

- Target table drift against both C compilers now has a fail-closed execution path.
- Invalid Fairy layout inputs stay owned by the Go engine instead of being laundered through C
  extensions.
- The probe reuses the bounded driver and toolchain fingerprints without adding a second compiler
  resolver or retaining artifacts.
- ADR 0037 later adds the closed sanitizer mode. Hosted execution, call classification, ABI
  crossings, and product runtime coverage remain separate Phase 2 gates.

## Rejected alternatives

- Probe only the C backend's resource structs: misses the independent array, pointer, function
  pointer, C scalar, explicit-base enum, and nested aggregate target vocabulary.
- Emit fixture and field names into C: creates injection and path-leak surface with no evidence
  benefit.
- Send failure cases to Clang/GCC: tests C diagnostics or extensions instead of Fairy layout
  rejection.
- Treat successful compilation as hosted evidence before running CI: confuses a planned path with
  an observed target result.
