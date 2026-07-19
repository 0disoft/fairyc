# ADR 0082: Windows MSVC target C emission

- Status: Accepted
- Date: 2026-07-19
- Target: `x86_64-windows-msvc`
- Command: `fairyc experimental build-target`
- Backend: `internal/cbackend/`

## Context

The Windows host smoke already proved that local x64 MSVC can compile and execute constrained C,
but `build-host` deliberately used the target-neutral Stage 0 host profile. Relabelling that output
as a Windows target artifact would have hidden LLP64 differences such as four-byte C `long` and the
absence of POSIX `ssize_t`. A real target build must select its target before C emission and must
fail closed when source requests a scalar the selected target does not provide.

## Decision

1. The constrained C backend owns a distinct Stage 0 Windows MSVC profile. It binds
   `x86_64-windows-msvc`, C11, no instrumentation, and the target's LLP64 scalar table before
   emitting declarations, static assertions, or operations.
2. Profile setup admits target scalars individually. A scalar absent from the target does not make
   an unrelated program fail, but any declaration or operation that actually uses it fails with a
   target-specific capability error. Windows therefore admits four-byte signed `c::long` and
   rejects POSIX `c::ssize_t`.
3. `experimental build-target` requires one FY source, explicit
   `--target x86_64-windows-msvc`, explicit `--profile low|optimized`, and an explicit `.exe`
   output path. Unknown, duplicate, missing, reordered, or unsupported arguments fail before
   compilation.
4. The first slice permits no called foreign symbol. Unused foreign declarations remain useful for
   checking target C scalar transport, but a source-level FFI call fails before toolchain
   resolution. Companion C sources, libraries, arbitrary linker arguments, and implicit SDK
   discovery flags are not command surface.
5. `examples/windows_target/main.fy` carries an unused `c::long` declaration and returns 42. The
   configured Windows smoke copies it into a bounded temporary source root, emits through the
   Windows profile, builds it with local x64 MSVC, inspects intermediate COFF and linked PE/unwind
   structure, executes it, and requires result 42.
6. The target schema reports emission subset `scalar-and-llp64-c-scalar-c11` and removes only the
   `target-c-emission-profile` missing-evidence marker. Emittable, conformant, and supported remain
   false. This local result is not a hosted Windows receipt.
7. No WSL or Linux compatibility layer participates. Linux-native parity remains a separate CI
   lane and is skipped on a Windows host without a native Linux toolchain.

## Consequences

- A Windows executable can now be traced from checked-in FY source through target selection,
  target-correct C, MSVC object/link steps, PE inspection, and native execution.
- The target-neutral `build-host` and declared-target `build-target` paths remain visibly distinct.
- Called source-level Windows FFI, bidirectional crossings, complete Windows scalar coverage,
  standard-library closure, debugger evidence, distribution packaging, and hosted conformance are
  still separate gates.
- The experimental command can change before release, but widening its target or linker surface
  requires synchronized target, CLI, fixture, schema, documentation, and compatibility review.

## References

- Microsoft x64 type and storage-layout conventions:
  <https://learn.microsoft.com/en-us/cpp/build/x64-software-conventions?view=msvc-170>
- `docs/abi/c-abi-v0.md`
- `docs/cli/command-contract.md`
- `docs/product/02-spec.md`
- `spec/cli/experimental-target-spec-v0.schema.json`
