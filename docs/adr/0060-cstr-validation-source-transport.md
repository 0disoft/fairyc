# CStr Validation Source Transport

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-19
- Product source: `docs/product/02-spec.md`
- ABI source: `docs/abi/c-abi-v0.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`

## Context

The owned-text C slice already enforces the CStr invariant, but its existing probe is authored
entirely in C. That proves the helper in isolation, not that a generated Fairy caller can transport
the exact view record, supply nonempty bounded storage, distinguish failure statuses, or observe the
helper's failure-atomic output behavior across independently compiled translation units.

The source language does not yet expose byte-array literals or safe `&CStr` construction. A bounded
test-only C producer can supply immutable storage without pretending that those missing language
features exist.

## Decision

1. The internal source probe declares the exact `@repr(C) fy_std_cstr_view_v0` record with immutable
   `*u8` data and exact `c::size_t` length fields.
2. Test-only C producers supply three static bounded byte sequences: valid trailing-NUL storage,
   storage containing an earlier NUL, and storage with no trailing NUL. They return immutable
   pointers and exact `size_t` lengths; they do not emulate the production validator.
3. The generated Fairy caller invokes the real `fy_std_cstr_validate_v0` from
   `OwnedTextSource()`. The major-suffixed symbol is admitted explicitly while the rest of the
   `fy_std_` namespace remains closed by default.
4. The valid case must return status 0, retain the exact storage identity, and publish a logical
   length excluding the terminator. A C observer additionally checks every logical byte.
5. The interior-NUL case must return status 4 and the unterminated case status 3. Both calls reuse
   the prior valid output record, and the observer must still see that valid record after each
   failure.
6. Generated C must assert the 16-byte view size, natural alignment, field offsets, exact
   `const uint8_t *` data field, and exact `size_t` length and function parameter types for
   `x86_64-linux-gnu`.
7. The caller and callee are separate objects. Clang-caller/GCC-callee and the reverse each run at
   low and optimized profiles and must exit 42 without output.
8. `fairyc.native-c-report/v24` adds `cStrValidationSourceCallExecutions`, expected to be four only
   after all directions and profiles execute successfully.
9. This probe does not add byte-array source literals, construct a public `&CStr`, establish borrow
   origins, connect owned String/CString operations, define source `Result` layout, automatically
   link the standard library, or establish target conformance.

## Consequences

- A real generated Fairy caller now crosses nonempty CStr success and rejection paths without a
  mock validator.
- The C producers and observer are bounded test infrastructure. Production status values and
  mutation behavior still come from the standard-library C unit.
- The existing C-only owned-text probe remains responsible for String/CString ownership, Buffer
  moves, append overlap, allocation rollback, drop, empty CString, and malformed-record coverage.
- Native execution remains configured but unobserved on this Windows host.
