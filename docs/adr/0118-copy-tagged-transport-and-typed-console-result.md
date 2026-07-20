# ADR 0118: Copy tagged transport and typed console result

- Status: Accepted
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Backend source: `docs/mir/c-backend-safety-v0.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`

## Context

ADR 0034 admitted private generated-C transport for built-in tags only when at least one payload was
a direct resource. ADR 0117 then annotated the Windows console helper with the `console` effect but
left its safe text wrapper returning `bool`. That erased both completed progress and the runtime
status needed to distinguish invalid input, unavailable stdout, write failure, and impossible zero
progress.

Using a user enum directly as the error payload would pretend the C backend already has a target-
laid-out nested enum representation. It does not. The current user-enum C type is a private union
transport without a corresponding layout-engine type, and built-in tags containing user enums must
continue to fail closed.

## Decision

1. Private generated-C `Option` and `Result` transport additionally admits copy payloads from the
   canonical Fairy scalars `bool`, `u8`, `i32`, `u32`, and `usize`, plus admitted inert structs.
   Direct admitted resources remain supported under the existing resource-tagged rules.
2. Every generic tagged type is a private struct containing a `u32` tag followed by one field per
   possible payload. Its size, alignment, and field offsets come from the selected target layout
   engine and are guarded by generated `_Static_assert` checks. Inactive fields are zero-filled.
3. Copy-only tags are not classified as values needing drop. A tag needs drop only when one of its
   direct payloads is an admitted resource. Existing resource option/result feature identities and
   active-payload drop tracing remain unchanged.
4. The safe Windows text wrapper returns `Result<usize, ConsoleError>`. Success contains the exact
   written byte count. Failure contains the nonzero raw status and completed-prefix byte count from
   the private `@repr(C)` runtime record. Both the raw declaration and safe wrapper retain
   `@effects(console)`.
5. This does not stabilize a public tagged layout or C ABI. Arrays, raw pointers, C scalars, user
   enums, nested built-in tags, and resource-containing structs remain unsupported tagged payloads.
   `ConsoleError` is an example-local typed record, not the final portable `std::io` error taxonomy.

## Consequences

- Safe FY code no longer compresses a partial console failure to false; the typed result preserves
  the progress required by a later retry or diagnostic policy.
- The same copy-result transport can carry a future File error record without making file handles
  or user enums accidentally copyable.
- A symbolic `ConsoleErrorKind` enum still requires an explicit nested-enum layout decision rather
  than inheriting an accidental C union shape.

## Verification

- C-backend unit tests cover `Result<usize, inert-struct>`, target layout assertions, copy-result
  feature selection, and fail-closed `Result<i32, user-enum>` rejection;
- the C-emission manifest contains an interpreter-approved copy-result/struct-error case;
- the WSL-free Windows smoke compiles the checked-in two-module FY program through generated C and
  local x64 MSVC, forwards exactly `Fairy C says hi\n`, observes no stderr, and preserves status 42.
