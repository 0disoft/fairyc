# ADR 0261: Stable Run Program Argument Forwarding

- Status: Accepted
- Date: 2026-08-05

## Context

ADR 0177 and ADR 0212 stabilized `fairyc run [--release] [--locked]` over a temporary Windows image
with caller stdin forwarding, bounded program output, and preserved program status, but explicitly
left program arguments out of the v3 contract. The command contract already stated that `run`
separates compiler arguments from program arguments with `--`, yet no implementation or SDK surface
existed, so a generated Fairy program could not read its own process arguments on Windows.

The Windows UCRT narrow argv (`__p___argv`) is encoded in the active ANSI code page, not UTF-8, so
any argument forwarding must not assume narrow argv is UTF-8. The wide argv (`__p___wargv`) is
UTF-16 and can be converted strictly. The `mainCRTStartup` narrow policy under the unchanged
`int main(void)` entry populates `__argc` and narrow `__argv` but leaves `*__p___wargv()` NULL; the
host-smoke decode probe observed that state (probe return 10) before the lazy initialization below.

## Decision

1. Stable `fairyc run [--release] [--locked] -- <program-args...>` forwards every argument after
   the first `--` as the temporary program's process arguments. The delimiter itself is excluded.
   Program arguments belong to `run` only: `fairyc build` and the other stable commands do not
   accept the delimiter and keep rejecting unknown arguments.
2. Arguments before the delimiter remain compiler options. `run -- --locked` passes `--locked` to
   the program; `run --locked -- x` keeps `--locked` as a compiler option and passes `x`; an
   unknown flag before the delimiter is the existing invocation error. An embedded NUL byte in any
   program argument is a stable `run` invocation error before compilation, because Windows command
   lines cannot carry NUL bytes.
3. The generated C entry stays `int main(void)`; no backend, COFF, or emission contract changes for
   argv. The new compiler-bundled `std::process` module reads process-startup data through two
   Windows-only standard-library symbols, `fy_std_process_arg_count_v0` and
   `fy_std_process_arg_v0`, in a new `windows-process-args` cstdlib component.
4. The C implementation reads process arguments through `__p___argc()` and `__p___wargv()`. Because
   the narrow `mainCRTStartup` policy leaves `*__p___wargv()` NULL, `fy_std_process_arg_v0`
   materializes the wide argv lazily with `_configure_wide_argv(_crt_argv_unexpanded_arguments)` —
   the same unexpanded mode the `wmain` startup would use, so no wildcard expansion is introduced.
   If the call fails or the wide argv is still NULL, the function fails closed with status 2 and
   `needed == 0` without touching the caller buffer; later calls reuse the populated wide argv, and
   `arg_count` reports the same startup argument count from `__argc`. UTF-16 to UTF-8 conversion
   uses
   `WideCharToMultiByte(CP_UTF8, WC_ERR_INVALID_CHARS, ...)` in a strict two-pass form: the first
   pass computes the exact UTF-8 byte length, every validation completes before the second pass,
   and the second pass writes only the exact required bytes without a terminator. Every failure
   leaves the caller buffer byte-for-byte unchanged. The second pass reuses the same immutable wide
   string and the exact first-pass size, so it cannot fail or write a different byte count; a
   defensive check verifies the exact written count and is documented as unreachable so it never
   becomes a public failure path that has already touched the buffer.
5. `arg_count() -> usize` returns the argument count including the executable name at index 0.
   `arg(usize index, ref mut u8[] buffer) -> Result<usize, ProcessArgError>` copies one argument's
   exact UTF-8 bytes and returns the byte count; an empty argument succeeds with zero. Status 1 is
   index out of range, status 2 is invalid encoding (including an unpaired UTF-16 surrogate) or
   argv that could not be materialized as valid UTF-16, and status 3 is buffer too small with
   `needed` carrying the full UTF-8 byte length. `needed` doubles as the length query; there is no
   separate `arg_len` API. Both functions are effect-free and read only the process-startup data
   captured when the program launched.
6. The SDK version moves from `0.0.0-stage0.10` to `0.0.0-stage0.11` because this adds a public
   module. Language remains `fairyc-language/8` and runtime ABI remains `fairyc-runtime/0`.
7. `WideCharToMultiByte` is provided by `kernel32.lib` in the MSVC linker's default library set, so
   no new link flag or `#pragma comment(lib, ...)` is added; `_configure_wide_argv` is a public
   UCRT function linked from the static `libucrt` under `/MT`, and shell32 is not used. The native
   decode probe proves at both MSVC profiles that narrow startup leaves `*__p___wargv()` NULL,
   `_configure_wide_argv(_crt_argv_unexpanded_arguments)` succeeds, the wide argv becomes non-NULL,
   and Unicode, quote, trailing-backslash, and empty arguments round-trip. There is no ANSI argv,
   environment side channel, or `wmain` fallback.

## Consequences

- The checked-in `examples/process_args_quickstart` returns 42 only when argv[0] is present and
  nonempty, the count and ordering match the harness's five program arguments, an empty argument,
  a flag-looking argument, a space/quote/trailing-backslash argument, and a non-ASCII UTF-8
  argument round-trip byte-for-byte, out-of-range returns status 1, and a three-byte buffer rejects
  the five-byte argument with status 3, exact `needed == 5`, and unchanged sentinel bytes.
- Native evidence: development and release build artifacts execute directly with the argument set
  (2 images, 2 executions), stable `run` and `run --release --locked` forward the same set (2
  executions), and a host-smoke-only C decode probe at low and optimized profiles proves the
  narrow-startup NULL wide argv, successful lazy `_configure_wide_argv` materialization, and exact
  UTF-8 round-trips for `Fairy`, `a b"c\d\`, `한글λ`, and an empty argument, plus status 2 with an
  unchanged buffer for an unpaired surrogate (2 images, 4 executions). The CLI proofs reject
  embedded NUL, an unknown flag before the delimiter, and a `build` delimiter before any
  compilation.
- `unit.ForeignSymbols` for a program that imports `std::process` and calls both functions is
  exactly `fy_std_process_arg_count_v0` and `fy_std_process_arg_v0`; a program that never imports
  the module keeps the process symbols out of its emission and link closure. The host-smoke unit
  test freezes both sets.
- Existing generated-Fairy Region, Buffer, String, and CString proofs, stable build/run stdin and
  stream semantics, trap/exit meanings, and the raw C probe evidence boundaries are unchanged.
  `arg_count`/`arg` are effect-free; the SDK does not expose environment variables, the current
  directory, `wmain`, ANSI argv, or any console effect.
- Native proof remains Windows x64 MSVC evidence, not a portable backend or support claim. The
  process-args C component is a product implementation, not a test-only adapter; its low-level
  decode probe is host-smoke evidence only and is not checked in.
