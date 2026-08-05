# ADR 0262: Add allocation-free Stage 0 u32/usize/bool stdout output

- Status: Accepted
- Date: 2026-08-05
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Depends on: ADR 0215, ADR 0258

## Context

ADR 0215 added `std::io::write_i32` as the first allocation-free decimal output and left the
remaining integer widths and boolean output as later exact primitives. Programs still cannot print
a `u32`, a `usize`, or a `bool` without hand-building bytes or crossing an unsafe foreign boundary.

A `bool` argument cannot cross the existing unsafe `extern C` boundary without widening the
compiler: the frontend requires exact argument types and the C backend has no language-`bool`
foreign parameter mapping. Adding such a mapping would be a compiler/ABI change outside this
decision.

## Decision

1. The compiler-bundled SDK version moves from `0.0.0-stage0.11` to `0.0.0-stage0.12` because this
   adds public functions to `std::io`. Language remains `fairyc-language/8` and runtime ABI remains
   `fairyc-runtime/0`.
2. `std::io` adds:
   - `write_u32(u32 value) -> Result<usize, ConsoleError>`: canonical decimal ASCII, `0` through
     `4294967295`, at most 10 bytes.
   - `write_usize(usize value) -> Result<usize, ConsoleError>`: canonical decimal ASCII, `0`
     through `18446744073709551615` on the current LLP64 Windows x64 target, at most 20 bytes.
   - `write_bool(bool value) -> Result<usize, ConsoleError>`: exactly ASCII `true` or `false`.
   All three declare `@effects(console)`, write to stdout only, append no newline, allocate
   nothing, are locale-independent, and reuse the existing `ConsoleError` status/written and
   partial-prefix contract.
3. The private target support symbols are `fy_std_stdout_write_u32_v0` and
   `fy_std_stdout_write_usize_v0`. Each target adapter formats into a stack buffer sized to the
   exact maximum digit count of its declared width (10 for `u32`, 20 for pointer-width `usize` on
   this x86_64 target) using one shared private back-to-front unsigned-decimal helper, then
   delegates the resulting view to its existing stdout write-all implementation.
4. `write_bool` adds no C symbol and no `extern C` declaration. It is implemented inside the SDK as
   `if (value) { write("true") } else { write("false") }`, reusing the existing public `write` path
   and therefore the existing `fy_std_stdout_write_all_v0` target support symbol. A program that
   uses only `write_bool` links exactly that one existing symbol.
5. No general formatter, format string, locale, floating-point conversion, base selection, padding,
   grouping, stderr numeric variant, implicit newline, heap allocation, or public C ABI is added.
   `u8`, `u64`, and `i64` output remains absent because `u64`/`i64` are not public language types
   and `u8` has no demonstrated need beyond the existing byte-view APIs.

## Compatibility

The Fairy functions are source-additive. Exact Stage 0 SDK selection is intentionally not
lock-compatible: the SDK source digest, package instance, public-interface hash, dependent
identities, generated locks, and whole-package cache keys change. The runtime ABI remains
`fairyc-runtime/0`. The two new C symbols are private pre-1 target support compatibility data, not
a public C ABI or stable Fairy native ABI.

## Evidence

- Compiler integration tests run each function alone through package loading, semantic Flow,
  verified MIR, and entry-reachable C emission and freeze the exact `unit.ForeignSymbols`: one new
  symbol for `write_u32`, one new symbol for `write_usize`, the existing
  `fy_std_stdout_write_all_v0` for `write_bool`, and no new symbols for an unused program.
- Component-contract tests require the two new symbols to select exactly the target-owned console
  component on Linux and Windows and reject a `fy_std_stdout_write_bool_v0` raw symbol.
- The Linux native probe checks `u32` `0`/`1`/`UINT32_MAX` and `usize` `0`/`1`/`UINT64_MAX` with
  exact concatenated bytes and written counts. Execution of that configured probe remains
  hosted-Linux evidence.
- The WSL-free Windows smoke builds and executes `examples/std_io_scalar_quickstart` at low and
  optimized MSVC profiles and observes the exact stdout bytes
  `0\n1\n4294967295\n0\n1\n18446744073709551615\ntrue\nfalse\n` with empty stderr and exit 0:
  2 native executions across 2 inspected PE32+ images. The existing `write_i32` `INT32_MIN` proof
  and all prior host-smoke proofs remain unchanged.

## Consequences

Command-line programs can print the full unsigned and boolean scalar domain without an allocator
or an unsafe block. General formatting remains deliberately absent; stderr numeric output remains
available only as a combination of the existing byte/text APIs, and `u64`/`i64` output waits for
public language types rather than a formatting shortcut.
