# ADR 0075: Windows MSVC native layout probe

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-19
- Target source: `internal/target/`
- Fixture source: `spec/fixtures/layout/manifest-windows-msvc-v0.json`
- Generator source: `internal/windowslayoutprobe/`
- Driver source: `internal/cdriver/windows_host.go`
- Configured consumer: `cmd/fairyc-windows-host-smoke/`

## Context

The Windows layout manifest executes the project-owned layout engine, but agreement between that
engine and its own expected values is not an independent C implementation oracle. The existing
Windows host smoke already discovers an installed x64 MSVC and Windows SDK without WSL, bounds
compiler output and execution time, and deletes temporary artifacts. Reusing the target-neutral
Fairy program as a layout oracle would still be wrong because that path deliberately rejects exact
C scalars and `@repr(C)`.

## Decision

1. `internal/windowslayoutprobe` consumes only the target-bound Windows manifest through
   `layoutfixture.NativeProbeCases`. It rejects any other target and rechecks every pass case against
   the declared layout engine before emitting C.
2. The generator emits one deterministic private ISO C11 translation unit with no POSIX headers or
   `ssize_t`. Each emitted type has `sizeof` and `_Alignof` assertions; record fields additionally
   have `offsetof`, field-size, and field-alignment assertions.
3. Fixture case IDs do not enter C identifiers or diagnostics. The generated type names are dense
   private IDs, keeping repository labels out of compiler output.
4. `internal/cdriver` adds an explicit single-source MSVC probe path instead of passing target
   evidence through the target-neutral product-host API. It keeps the same closed x64 toolchain,
   C11, warnings-as-errors, static runtime, timeout, output bound, and temporary-artifact rules.
5. The configured Windows smoke compiles and executes the generated unit at low and optimized
   profiles. Both executions must exit zero with no output.
6. These two local executions cover the manifest's eight pass cases only. They do not clear
   `c-layout-probes` or `hosted-windows-receipt`, because the full declared scalar set and hosted
   receipt remain absent. They also do not prove Microsoft x64 calls, PE/COFF metadata, unwind,
   target C emission, runtime, standard library, or distribution.

## Consequences

- A bad four-byte `long`, eight-byte `size_t`, pointer width, function-pointer width, or mixed LLP64
  record offset now fails in the real MSVC frontend before the smoke can pass.
- Linux POSIX probe generation and execution remain untouched.
- The next Windows layout step is full declared scalar coverage plus a hosted receipt; the next
  independent ABI step is a Microsoft x64 classifier and native crossing ledger.
