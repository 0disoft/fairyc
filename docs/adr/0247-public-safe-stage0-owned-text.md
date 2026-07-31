# ADR 0247: Public safe Stage 0 owned text

- Status: Implemented local compiler, SDK, and native-smoke gate
- Date: 2026-07-31
- SDK source: `internal/sdkstdlib/sources/std/text.fy`
- Runtime source: `internal/cstdlib/stage0_owned_text.c`
- Symbol contract: `internal/cstdlibcontract/`
- Example: `examples/std_text_quickstart/`
- Native runner: `cmd/fairyc-windows-host-smoke/`
- Depends on: ADR 0044, ADR 0062, ADR 0087, ADR 0131, ADR 0132, ADR 0135, ADR 0147, ADR 0246

## Context

The runtime, raw generated-source probes, and an example-private wrapper already proved
Buffer-backed String and CString ownership, allocation rollback, interior-NUL rejection, exact
termination, cleanup, and owner-bound views. Users still had to copy the example module and thereby
inherit private foreign declarations. The compiler-bundled SDK exposed bytes and console I/O but no
owned UTF-8 or C-string resource.

## Decision

1. SDK `0.0.0-stage0.8` adds `std::text`. `String` and `CString` are move-only resources with
   private storage. Cleanup calls the existing opaque storage bridges, which accept canonical empty
   state and normalize successful drop.
2. The public String functions are `copy`, `append`, `string_len`, `string_is_empty`, and
   `string_as_str`. The public CString functions are `cstring_from_text`, `cstring_len`,
   `cstring_is_empty`, and `cstring_as_cstr`. `TextError`, `error_status`, and
   `error_allocation_status` preserve the two existing stable numeric status fields.
3. Construction stays explicit and fallible. There is no implicit literal conversion, overloaded
   concatenation, hidden formatting, locale conversion, or method syntax. String length is a UTF-8
   byte count. CString length excludes its one required trailing NUL.
4. `string_as_str` and `cstring_as_cstr` use the existing unsafe compiler-known raw-parts
   operations inside the SDK. Each safe result retains the exact shared owner borrow. Safe callers
   cannot mutate owned text bytes, move or append the owner while a view is live, expose capacity,
   or retain a view after cleanup.
5. String copy and append preserve valid UTF-8. CString construction rejects interior NUL and
   publishes ownership only after the bytes and final terminator are complete. Allocation or
   validation failure leaves an existing owner unchanged and releases unpublished replacement
   storage.
6. The language revision remains `fairyc-language/7` and the runtime ABI remains
   `fairyc-runtime/0`; this decision adds no syntax, MIR operation, runtime symbol, or C signature.
   The SDK version, source digest, package identity, public-interface hash, generated locks, and
   dependent cache keys change.

## Evidence and limits

The SDK loader and workspace tests require the fourth embedded module and compile it through
verified MIR. A qualified Windows C-emission regression compiles the checked-in quickstart and
requires the exact used owned-text storage symbols. A separate MIR regression freezes the bool type
on short-circuit materialization; the quickstart exposed and now guards the prior loss of that type
before unary `!` C emission.

The checked-in quickstart contains no raw pointer, foreign declaration, or `unsafe`. Windows
development and release builds execute String copy, append, immutable owner-bound view, empty
String, nonempty and empty CString, CStr view, interior-NUL error mapping, and implicit cleanup. The
configured MSVC 14.51 smoke inspects and executes both resulting PE images and finishes with 59
structurally inspected images.

This does not stabilize the private four-field storage as a foreign ABI, expose String capacity or
mutable bytes, add consuming Buffer-to-String conversion, validate arbitrary byte slices into text,
add text indexing or slicing, count Unicode scalar values or graphemes, normalize Unicode, format
values, perform locale conversion, prove hosted same-commit CI, or promote another target.
