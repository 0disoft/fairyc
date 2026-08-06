# ADR 0303: Allocation-Free Scalar Stderr Output

- Status: Accepted
- Date: 2026-08-06

## Context

`std::io` already provides canonical allocation-free scalar output for stdout and exact text and
byte output for stderr. Command-line programs still have to manually choose literals or construct
decimal bytes to report scalar diagnostics on stderr. That duplicates policy and makes ordinary
error reporting less direct than stdout reporting.

## Decision

1. SDK `0.0.0-stage0.38` adds these target-neutral public operations:
   - `write_stderr_i32(i32 value) -> Result<usize, ConsoleError>`;
   - `write_stderr_u32(u32 value) -> Result<usize, ConsoleError>`;
   - `write_stderr_usize(usize value) -> Result<usize, ConsoleError>`;
   - `write_stderr_bool(bool value) -> Result<usize, ConsoleError>`.
2. Each operation declares `@effects(console)`, appends no newline, and returns the exact completed
   byte count on success. A failure preserves the raw status and completed prefix in `ConsoleError`.
3. Signed and unsigned values use the same locale-independent back-to-front stack formatters as
   stdout. Target adapters select stderr before calling their existing write-all loop, so the
   functions allocate nothing and preserve partial progress.
4. The private target-neutral numeric symbols are `fy_std_stderr_write_i32_v0`,
   `fy_std_stderr_write_u32_v0`, and `fy_std_stderr_write_usize_v0`. Both Linux GNU and Windows MSVC
   closed component contracts admit them.
5. `write_stderr_bool` reuses `write_stderr("true")` or `write_stderr("false")`. It adds no raw
   symbol; `fy_std_stderr_write_bool_v0` remains rejected.
6. The language stays `fairyc-language/10` and runtime ABI stays `fairyc-runtime/0`. The added raw
   symbols are private pre-1 target-support compatibility, not a stable public C ABI.

## Evidence

- SDK parsing tests require all four functions to be public.
- Closed C component tests require all three numeric symbols on both implemented targets and reject
  a bool-specific symbol.
- The Linux C probe captures stderr and checks exact `i32` zero/negative/min/max, `u32` zero/max,
  and `usize` zero/max bytes and written counts.
- Generated-C reachability tests require exactly one matching numeric symbol per used function,
  reuse only `fy_std_stderr_write_all_v0` for bool, and retain no scalar symbols when unused.
- The checked-in generated-Fairy quickstart runs at low and optimized Windows MSVC profiles and
  checks exact stdout and stderr bytes plus every returned byte count.

## Consequences

Fairy C programs can report the currently supported integer and boolean scalar domains directly to
stderr without allocation, general formatting, locale, implicit newlines, or a bool ABI entry.
Floating-point conversion, bases, padding, and additional integer widths remain separate decisions.
