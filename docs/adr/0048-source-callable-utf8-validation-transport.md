# Source-callable UTF-8 Validation Transport

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-19
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- FFI source: `docs/adr/0047-raw-pointer-target-scalar-foreign-call.md`
- Implementation source: `internal/cstdlib/stage0_text_view.c`

## Context

The provisional text-view C slice already owns canonical UTF-8 validation, but only other C helpers
could call its private validator. ADR 0047 made a Fairy caller capable of passing typed null raw
pointers and `usize`, so the empty-input validation path can cross the real standard-library C unit
without waiting for source-level C-layout output records.

This is an internal transport milestone, not the public `std::text` wrapper. A safe `&[u8] ->
Result<&str, ...>` surface still needs slices, borrow-origin construction, and error types.

## Decision

1. `fy_std_text_utf8_valid_v0(const uint8_t *, size_t) -> uint32_t` becomes a non-static,
   ABI-major-suffixed internal standard-library symbol. It returns exactly zero or one and keeps the
   same canonical UTF-8 algorithm used by text view, String, and CString helpers.
2. The C backend keeps its generated `fy_` namespace closed. It admits only this exact `fy_std_`
   symbol rather than opening the whole prefix.
3. A Fairy source fixture declares the symbol as `unsafe extern C fn(*u8, usize) -> u32`, passes
   `null` and zero, and returns `42` only when validation returns one.
4. The native lane links that generated caller against the actual `TextViewSource()` unit with
   Clang/GCC in both directions and both profiles. It does not substitute a hand-written mock.
5. `fairyc.native-c-report/v14` adds four `textValidationSourceCallExecutions`.
6. The interpreter remains fail-closed, no public safe API is claimed, and nonempty input remains
   blocked until Fairy source can construct or receive a non-null bounded pointer safely.

## Consequences

- A real `.fy` program now reaches real project-owned standard-library C code through the compiler
  pipeline and linker.
- Full text-view construction still requires `@repr(C)` output records or an equivalent decided
  binding representation.
- ADR 0049 subsequently supplies that narrow flat-record and direct-local raw-address transport;
  this ADR's predicate call and v14 counter remain independently meaningful.
- The target evidence remains configured but unobserved until the Linux lane succeeds.
