# Distinct `c::long` Source Transport

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-19
- Product source: `docs/product/02-spec.md`
- ABI source: `docs/abi/c-abi-v0.md`
- Target source: `internal/target/spec.go`

## Context

The selected `x86_64-linux-gnu` target is LP64, so C `long`, Fairy `i64`, and Fairy `isize` happen
to share an eight-byte layout. Treating them as aliases would still be wrong: a different target may
give them different widths, and generated foreign prototypes must preserve the C declaration rather
than a coincidental machine representation. Target layout and native layout-probe coverage for
`c.long` already exist, but source typing and call transport did not.

## Decision

1. Source `c::long` is a distinct canonical HIR and MIR type mapped to target key `c.long`. It is
   never canonicalized to `i64` or `isize`.
2. Generated C spells the type exactly `long` and emits target-derived size, alignment, and
   signedness assertions. The current target requires eight-byte size and alignment.
3. The executable slice is transport-only: a foreign producer may return a negative `long`, a
   typed MIR local retains it, and a later foreign consumer receives it. Fairy literals,
   arithmetic, comparison, casts, interpreter host bindings, raw C-scalar pointers, and aggregate
   fields remain closed.
4. The existing `pass.c-long` layout fixture remains the layout authority. No new layout case or
   layout-probe count is created merely because source transport now consumes that fact.
5. A hand-written C producer returns `-41L`; its consumer requires the negative value and returns
   `42`. `fairyc.native-c-report/v19` adds `foreignCLongCallExecutions`, expected to be four only
   after both compiler directions and both profiles execute successfully.
6. Plain C `char`, `_Bool`, floating C types, the remaining signed and unsigned keyword-width
   families, literals, operators, and conversions require separate contracts rather than inheriting
   this slice by width. ADR 0056 later implements their exact transport-only identities without
   opening those value semantics.

## Consequences

- Fairy source can transport a second exact C keyword integer without lying about LP64 identity.
- The native crossing is configured but remains unobserved on this Windows host.
- This change does not expose Linux File APIs, broaden `@repr(C)` fields, or promote target maturity.
