# Owned Text Source Transport

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-19
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- ABI source: `docs/abi/c-abi-v0.md`

## Context

The String and CString C implementation already owns allocation, validation, rollback, termination,
and drop behavior. ADR 0061 made its exact two-`u32` operation result callable from Fairy source,
but no generated caller yet crosses the nested records or invokes the production lifecycle helpers.

Using integer wrapper shims would hide the real return ABI and could let the source records drift
from the C unit. The probe must use the production signatures directly while remaining honest that
these raw records are not the public safe resource representation.

## Decision

1. The source probe declares exact `@repr(C)` records for SystemHeap block, Buffer, text view,
   String, CStr view, CString, and owned-text operation result. Nested definitions and target layout
   assertions use the existing dependency-ordered record transport.
2. A bounded C producer supplies immutable `Fairy` bytes plus exact `size_t` length. It does not
   implement any String/CString operation.
3. The generated Fairy caller invokes the real `fy_std_string_copy_v0`,
   `fy_std_string_as_view_v0`, `fy_std_string_append_v0`, and `fy_std_string_drop_v0`.
4. A C observer requires the first view to equal `Fairy`, the post-append view to equal
   `FairyFairy`, and the dropped String to contain the canonical empty Buffer state.
5. The same caller invokes the real `fy_std_cstring_from_text_v0`,
   `fy_std_cstring_as_view_v0`, and `fy_std_cstring_drop_v0`.
6. A C observer requires the CStr logical view to equal `Fairy`, the next byte to be exactly one NUL
   terminator, and the dropped CString to contain the canonical empty Buffer state.
7. Every returned high-level status and allocation status must be zero. The source probe covers the
   successful lifecycle; the existing C-only owned-text probe retains invalid UTF-8, failed move,
   self-overlap, interior-NUL, empty-value, rollback, and malformed-state negatives.
8. Only the seven exercised major-suffixed String/CString symbols join the closed internal
   standard-library allowlist. The `fy_std_` namespace does not become generally callable.
9. Caller and callee are separate objects. Clang-caller/GCC-callee and the reverse each run at low
   and optimized profiles and must exit 42 without output.
10. `fairyc.native-c-report/v26` adds `ownedTextSourceCallExecutions`, expected to be four only after
    every compiler direction and profile succeeds.
11. This transport does not add public safe String/CString resources, source `Result` error
    resources, borrow origins, automatic standard-library linkage, allocator selection, or target
    conformance.

## Consequences

- Generated Fairy source now reaches real owned-text allocation, mutation, view, termination, and
  release without wrapper functions replacing production behavior.
- The two-`u32` operation result is ABI transport only. Mapping its statuses into a source `Result`
  remains a separate ownership and diagnostics decision.
- Native execution remains configured but unobserved on this Windows host.
