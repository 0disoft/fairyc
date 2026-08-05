# ADR 0284: Carry copy fixed arrays through built-in Option and Result

- Status: Accepted
- Date: 2026-08-06
- Product source: `docs/product/02-spec.md`
- Typing source: `docs/language/0007-name-resolution-and-typing-v0.md`
- MIR source: `docs/mir/c-backend-safety-v0.md`
- Depends on: ADR 0008, ADR 0029, ADR 0110, ADR 0174, ADR 0251

## Context

Fairy fixed arrays already provide bounded positive-length value semantics for the five copy scalar
elements, with typed HIR, verified MIR, interpreter execution, private assignable C wrappers, and
target-owned layout. Source typing nevertheless rejected that same value as a direct payload of
compiler-known `Option` or `Result`, and the C backend deliberately failed closed at the same
boundary.

This excluded useful small optional records and fallible fixed-size results despite requiring no
new ownership, allocation, drop, or raw C-array semantics. Opening only the backend would leave
source programs unsupported; opening only typing would leave native compilation incomplete.

## Decision

1. Cut the exact language revision from `fairyc-language/8` to `fairyc-language/9` and admit one
   positive bounded scalar fixed array as a direct payload of `Option` or either payload of
   `Result`.
2. Retain ADR 0110's exact element and length boundary: `bool`, `u8`, `i32`, `u32`, or `usize`, with
   length `1..=4096`. Nested arrays, resource or aggregate elements, and array fields remain outside
   this decision.
3. Reuse ordinary fixed-array value-copy semantics for constructors, parameters, returns,
   exhaustive switches, and compatible Result `?` propagation. No move-only or drop behavior is
   added.
4. Reuse the existing private assignable `fy_array_N` C wrapper inside generic tagged records.
   Compute the outer record from target-owned array layout and retain generated size, alignment,
   and offset assertions. No raw C array, public tagged layout, or foreign ABI is introduced.
5. Add one deterministic source-to-C fixture covering Option Some and None, Result Ok and Err,
   indexed payload reads, and both Result propagation paths with an interpreter result of 42.
6. Compile, inspect, link, and execute the same generated-Fairy program under low and optimized
   Windows MSVC profiles. Both executions must return 42.
7. Keep SDK source and public API at `0.0.0-stage0.25` and runtime ABI at `fairyc-runtime/0`.
   Update the SDK language requirement, starter and example manifests, current test inputs,
   package identities, locks when generated, and whole-package cache-key vector to `/9`.

## Consequences

- Small fixed-size collections can cross absence and fallible-result control flow without heap
  allocation or a user-defined wrapper.
- Existing `/8` source bytes remain valid, but exact manifests require a manual `/9` pin update.
- The SDK package instance and dependent cache or lock identities change even though SDK source
  bytes and version do not.
- Raw-pointer and `c.*` scalar tagged payloads, nested tags, nested fixed arrays, resource-bearing
  arrays, array fields, public tagged layout, and foreign tagged ABI remain rejected.

## Evidence

- `internal/semantic/resource_types.go` admits bounded fixed arrays inside compiler-known tags.
- `internal/cbackend/arrays.go` discovers tagged array payloads and owns their target layout input.
- `internal/cbackend/tagged_types.go` admits the private wrapper as a copy payload.
- `spec/fixtures/c-emission/cases/pass/copy-tagged-fixed-array/src/main.fy` owns interpreter and
  deterministic C-emission evidence.
- `spec/fixtures/package-cache/manifest-v5.json` fixes the `/9` cache-key invalidation vector.
- `fairyc_windows_host_smoke` executes both optimization profiles through real MSVC images.
