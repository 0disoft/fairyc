# ADR 0302: Public allocation-free text byte view

- Status: Accepted
- Date: 2026-08-06
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- SDK source: `internal/sdkstdlib/sources/std/text.fy`
- Example: `examples/std_text_quickstart/`
- Depends on: ADR 0135, ADR 0247, ADR 0248

## Context

`std::text` can expose owned `String` storage as owner-bound `ref str`, and arbitrary immutable
bytes can be validated into `ref str`. Safe code still cannot pass already-valid text to APIs that
require immutable bytes, such as binary encoders or file writes, without copying through a Buffer
or entering unsafe pointer code.

A `ref str` already proves valid UTF-8, immutability, an exact byte length, and a borrow origin. The
compiler-known `byte_slice::from_raw_parts` operation can preserve that origin while changing only
the safe view type. No validation scan, allocation, copy, runtime helper, or public layout is needed.

## Decision

1. Move the compiler-bundled SDK from `0.0.0-stage0.36` to `0.0.0-stage0.37`. Keep language
   `fairyc-language/10` and runtime ABI `fairyc-runtime/0` unchanged.
2. Add effect-free public
   `std::text::as_bytes(ref str) -> ref u8[]`. Use an explicit free function; add no method syntax,
   implicit text/byte conversion, mutable result, allocation, copy, or failure variant.
3. Construct the result through the existing unsafe compiler-known
   `byte_slice::from_raw_parts(owner, data, length)` inside the SDK. Pass the input `ref str` as the
   owner, its immutable data address as `data`, and its exact UTF-8 byte length as `length`.
4. The result aliases exactly the same initialized bytes and retains the input borrow origin. A
   `String` reached through `string_as_str` therefore cannot be moved, appended, or dropped while a
   derived byte view has a later use. Static text remains static. Empty text produces an empty
   immutable view through the existing private sentinel normalization.
5. Expose no raw pointer, capacity, spare storage, mutable bytes, Unicode scalar indexing, public
   slice/string ABI, C termination, normalization, validation, or lifetime extension.
6. Extend the safe `std_text_quickstart` to check exact ASCII, multibyte UTF-8, appended, and empty
   byte views. Compile, inspect, and execute it under low and optimized Windows MSVC profiles.
7. Add package-aware negative evidence requiring `E2105` when an owning `String` is moved before a
   derived byte view's last use. Retain the existing raw-parts semantic/MIR/backend tests as the
   primitive authority.

## Consequences

- Text can feed immutable byte-oriented APIs without copying or exposing unsafe code.
- The operation is constant-time metadata construction and retains no additional owner or storage;
  its lifetime is exactly the input borrow's lifetime.
- Existing SDK source remains valid. The SDK source digest, public-interface hash, package instance,
  dependent identities, generated locks, and whole-package cache keys change.
- No target C symbol, component dependency, native ABI, JSON/schema, manifest, MIR format, or
  generated-C contract changes.
- Native evidence remains Windows x64 MSVC evidence and does not promote another target.

## Evidence

- `internal/sdkstdlib/sources/std/text.fy` owns the public wrapper.
- `internal/sdkstdlib/stdlib_test.go` owns embedded export and SDK-version evidence.
- `internal/workspaceloader/semantic_test.go` owns package-aware owner-loan rejection.
- `examples/std_text_quickstart/src/main.fy` owns exact source-level byte observations.
- `cmd/fairyc-windows-host-smoke/` owns low/optimized generated-Fairy execution and PE inspection.
