# ADR 0299: Append one byte to a public Buffer

- Status: Accepted
- Date: 2026-08-06
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Depends on: ADR 0246, ADR 0256

## Context

Public `std::bytes::Buffer` can append a borrowed byte slice, but constructing a slice solely to add
one byte adds ceremony to binary encoders, framed protocols, and incremental builders. A Rust-like
`push` name would obscure the operation's C-style byte-buffer role, while a new target primitive
would duplicate the existing overlap-safe, failure-atomic append implementation.

The current raw Buffer append already accepts an exact pointer and count, grows fallibly, commits
only after allocation succeeds, and preserves the owner on overflow or allocation failure. Its raw
C probe covers overlap, growth, overflow, and detailed rollback. The missing surface is therefore a
safe source-level single-byte adapter and native evidence that its generated failure arm executes.

## Decision

1. Move the compiler-bundled SDK from `0.0.0-stage0.35` to `0.0.0-stage0.36`. Keep language
   `fairyc-language/10` and runtime ABI `fairyc-runtime/0` unchanged.
2. Add effect-free public
   `std::bytes::append_byte(ref mut Buffer, u8) -> Result<usize, BufferError>`. Use the explicit
   `append_byte` name; do not introduce method syntax, implicit conversion, or a generic `push`
   convention.
3. Delegate directly to `fy_std_buffer_append_v0` with a pointer to the by-value byte and count one.
   Success returns the new initialized byte count. Failure returns the exact existing allocation
   status and preserves data, capacity, alignment, length, and ownership.
4. Add no raw symbol, target component, runtime ABI entry, allocation policy, capacity guarantee,
   implicit retry, or exception behavior. Bulk `append` remains the preferred multi-byte operation.
5. Extend `std_bytes_quickstart` so generated Fairy appends one scalar byte and verifies its exact
   position among bulk-appended and later mutated bytes.
6. Add a generated-Fairy forced-allocation-failure source that starts from a canonical zero-capacity
   Buffer, calls `append_byte`, requires allocation status `1`, verifies zero length and capacity,
   and then relies on automatic cleanup. Compile, inspect, and execute it under low and optimized
   Windows MSVC profiles, requiring zero live probe allocations.
7. Retain the raw C Buffer probe as the byte-level authority for pointer, alignment, content, and
   rollback preservation. The generated source proof complements rather than duplicates that matrix.

## Consequences

- Binary builders gain a direct, C-friendly single-byte operation without temporary slice syntax.
- The wrapper inherits the established Buffer growth and failure-atomicity contract, and adds no
  target/runtime compatibility surface.
- Appending may allocate and remains explicitly fallible. The API promises neither amortized growth
  factors nor a stable data pointer across success.
- The SDK source digest, public-interface hash, package instance, dependent identities, generated
  locks, and whole-package cache keys change because the public source surface changes.
- Native failure evidence remains Windows x64 MSVC evidence, not a portable backend support claim.

## Evidence

- `internal/sdkstdlib/sources/std/bytes.fy` owns the public wrapper.
- `internal/cstdlib/stage0_buffer.c` and `stage0_buffer_probe.c` own the reused primitive and raw
  preservation matrix.
- `examples/std_bytes_quickstart/src/main.fy` executes generated-Fairy success.
- `cmd/fairyc-windows-host-smoke/generated_failure_probes.go` executes forced generated-Fairy
  allocation failure and cleanup under both MSVC profiles.
