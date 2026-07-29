# ADR 0215: Add allocation-free Stage 0 i32 output

- Status: Accepted for prototype
- Date: 2026-07-29
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Effect source: `docs/language/0008-effects-and-capabilities-v0.md`
- Package source: `docs/adr/0147-compiler-bundled-stage0-standard-library.md`
- Depends on: ADR 0125, ADR 0126, ADR 0205, ADR 0210, ADR 0211
- Implementation: `internal/sdkstdlib/`, `internal/cstdlib/`, `internal/cstdlibcontract/`

## Context

Fairy C programs can write known text and byte views, but cannot print a computed integer without
hand-building decimal bytes or crossing an unsafe foreign boundary. Pulling in a general formatting
engine here would be the usual standard-library bloat trap: locale, width, allocation, format
parsing, and type dispatch would arrive before the language needs them.

The narrow useful primitive is exact signed 32-bit decimal output. Its behavior must cover the full
domain, especially `-2147483648`, without relying on signed negation that is undefined in C.

## Decision

1. The compiler-bundled SDK version becomes `0.0.0-stage0.6`. Its exact source digest, package
   instance, public interface hash, dependent identities, and generated lock bytes change. The SDK
   identity, language revision, runtime ABI identity, package UID, and embedded source paths remain
   unchanged.
2. `std::io` adds `write_i32(i32 value) -> Result<usize, ConsoleError>`. It declares
   `@effects(console)`, emits locale-independent decimal ASCII to stdout, appends no newline,
   allocates nothing, and returns the exact completed byte count or the existing numeric output
   error with its completed prefix.
3. The output is canonical: zero is `0`, negative values have one leading ASCII minus, positive
   values have no sign, and there are no leading zeroes. The result length is 1 through 11 bytes.
4. The private target-neutral support symbol is `fy_std_stdout_write_i32_v0`. Each target adapter
   formats into one 11-byte stack buffer from right to left and delegates the resulting view to its
   existing stdout write-all implementation. Negative magnitude is computed as unsigned subtraction
   from zero, so `INT32_MIN` never undergoes overflowing signed negation.
5. No general formatter, format string, locale, floating-point conversion, base selection, padding,
   grouping, stderr variant, implicit newline, heap allocation, or reusable text object is added.

## Compatibility

The Fairy function is source-additive. Exact Stage 0 SDK selection is intentionally not
lock-compatible: an old generated lock names the prior SDK version, digest, package instance, and
interface hash. The runtime ABI remains `fairyc-runtime/0`. The new C symbol is private pre-1 target
support compatibility data, not a public C ABI or stable Fairy native ABI.

## Evidence

- Compiler integration tests require the public call to reach verified MIR and reject a caller that
  omits the `console` effect.
- Component-contract tests require the symbol to select exactly the target-owned console component
  on Linux and Windows.
- The Linux native probe checks `0`, `-42`, `123456789`, `INT32_MIN`, and `INT32_MAX`, including exact
  concatenated bytes and lengths. Execution of that configured probe remains hosted-Linux evidence.
- The WSL-free Windows smoke compiles the safe checked-in Fairy workspace with MSVC and observes
  exact `-2147483648` output from both the in-process compiler and a separately built compiler.

## Consequences

Small command-line programs can print computed counters, statuses, and measurements without an
allocator or an unsafe block. General formatting remains deliberately absent; later numeric types
should earn similarly exact primitives or a separately designed bounded formatting layer.
