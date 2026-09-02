# ADR 0373: SDK-owned scalar text append for Buffer

- Status: Accepted
- Date: 2026-09-02
- Product issue: #51
- SDK source: `internal/sdkstdlib/sources/std/bytes.fy`
- Success example: `examples/std_bytes_quickstart/`
- Native failure proof: `cmd/fairyc-windows-host-smoke/scalar_formatting_failure_test.go`

## Context

Fairy C can write scalar values to console streams, but ordinary source cannot compose the same
canonical text inside a caller-owned byte buffer. Adding format-string syntax, planner IR, templates,
sink preflight, C varargs, reflection, or display dispatch before a concrete workload proves those
costs would turn a small SDK gap into a second formatting language.

The existing move-only `std::bytes::Buffer` already owns allocation, capacity growth, initialized
prefixes, typed allocation failure, and failure-atomic bulk append. The smallest useful slice is
therefore an additive set of ordinary SDK functions over that existing operation.

## Decision

1. Advance the compiler-bundled SDK to `0.0.0-stage0.55`. Language
   `fairyc-language/12`, runtime ABI `fairyc-runtime/0`, the seven Buffer runtime symbols, and
   existing source syntax remain unchanged.
2. Add these public `std::bytes` functions:

   - `append_i32(ref mut Buffer, i32) -> Result<usize, BufferError>`
   - `append_u32(ref mut Buffer, u32) -> Result<usize, BufferError>`
   - `append_usize(ref mut Buffer, usize) -> Result<usize, BufferError>`
   - `append_bool(ref mut Buffer, bool) -> Result<usize, BufferError>`

   Success returns the Buffer's new initialized length.
3. Integer output is canonical locale-independent decimal ASCII with no newline, grouping,
   leading zeroes, or positive sign. `i32` covers `-2147483648` through `2147483647`; `u32`
   covers `0` through `4294967295`; current 64-bit Stage 0 `usize` covers `0` through
   `18446744073709551615`. Boolean output is exactly `true` or `false`.
4. Each integer function encodes back-to-front into one fixed local `u8` array sized to the
   admitted type's maximum text length, then calls the existing bulk Buffer append exactly once.
   `i32` keeps the signed quotient and remainder throughout, so the minimum value is never
   negated and cannot overflow.
5. No new C runtime function, exported symbol, parser, allocation policy, syntax form, variadic
   path, reflection hook, or user-defined formatting protocol is introduced. Boolean formatting
   delegates directly to the same existing append operation with static byte literals.
6. The existing Buffer append contract remains the only commit boundary. Capacity growth either
   succeeds before the initialized length changes or returns `BufferError` with the original data,
   length, capacity, and ownership intact. Formatting performs no partial append.
7. `std_bytes_quickstart` verifies zeroes, signed and unsigned extrema, both booleans, exact byte
   order, ordinary capacity growth, and automatic cleanup in development and release MSVC builds.
   A separate generated-Fairy probe executes both the success path and a forced growth-allocation
   failure, then requires zero live allocations after each execution.

## Compatibility

This is an additive SDK API and changes the bundled standard-library source digest and package
identity. Existing Fairy source continues to compile unchanged. Runtime ABI, target profiles,
language grammar, public C ABI, lockfile schema, CLI contracts, and historical manifests do not
change.

## Deliberate exclusions

Format literals, format plans, interpolation, width and alignment, floating point, templates,
user-defined display, implicit Buffer creation, console shortcuts, partial-progress results, and
multi-sink preflight remain outside this slice.

## Validation

- `gofmt -w internal/sdkstdlib/*.go cmd/fairyc-windows-host-smoke/*.go`
- `go test -mod=readonly -count=1 ./internal/sdkstdlib`
- `go test -mod=readonly -count=1 ./cmd/fairyc-windows-host-smoke`
- `go test -mod=readonly -count=1 ./...`
- `go vet -mod=readonly ./...`
- `go run -mod=readonly ./cmd/fairyc-windows-host-smoke` on Windows
- `git diff --check`

The Windows test and host smoke compile generated Fairy separately at development and release
profiles rather than treating a C-only helper probe as language execution evidence.
