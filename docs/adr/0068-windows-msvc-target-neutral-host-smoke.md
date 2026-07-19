# ADR 0068: Windows MSVC target-neutral host smoke

- Status: Accepted
- Date: 2026-07-19

## Context

The first declared Fairy C target and native differential lane are Linux/amd64 with glibc, Clang,
and GCC. The developer host is Windows/amd64 without WSL, so that lane cannot demonstrate the local
experience of compiling a `.fy` program through generated C into a real executable.

Pretending the existing Linux target description is a Windows ABI would be wrong. MSVC uses LLP64,
PE/COFF, the Windows x64 calling convention, and different C typedef providers. Those contracts do
not exist yet. A useful local smoke can still be honest if it admits only target-neutral generated-C
features and makes no Windows target, ABI, FFI, standard-library, or conformance claim.

## Decision

1. `fairyc experimental run-host <file.fy> --profile <low|optimized>` is a Windows/amd64-only,
   unstable host-smoke command. It compiles one verified program with locally installed x64 MSVC,
   executes a temporary PE image, reports the program's `i32` result, and removes all temporary C,
   object, PDB, and executable files.
2. The command discovers Visual Studio through the official local `vswhere.exe`, reads the selected
   MSVC tool version, selects the latest complete Windows 10/11 SDK, and constructs a closed
   `PATH`/`INCLUDE`/`LIB` overlay. It invokes `cl.exe` directly; it does not require a Developer
   Command Prompt, PowerShell profile, WSL, network access, or a batch-file environment mutation.
3. Builds use C11 mode, warnings as errors, static MSVC runtime linkage, and the selected low or
   optimized profile. Compiler and program execution are bounded by time, output, and artifact-size
   limits with closed stdin.
4. Host smoke rejects all foreign symbols, provisional standard-library linkage, `@repr(C)`, exact
   `c::*` scalars, and foreign C calls or results. Those surfaces require a declared Windows target
   and ABI contract. Private fixed-width scalar, control-flow, resource, enum, byte-string, C-string,
   and UTF-8-string transport may be exercised when the C backend already admits it.
5. Native trap exit codes map back to the seven canonical Stage 0 trap categories. Other exit bits
   are reported as the program's `i32` result. Program output is currently rejected because the
   admitted host-smoke subset has no console API.
6. `examples/windows_smoke/main.fy` is the checked-in executable smoke. The configured
   `fairyc_windows_host_smoke` intent builds and runs it through MSVC and must report result 42.
7. This does not add `x86_64-windows-msvc` to target inspection, does not validate Windows aggregate
   layout or calling convention, does not produce a distributable artifact, and is not self-hosting:
   the compiler remains the Go Stage 0 implementation.

## Consequences

- A Windows developer without WSL can now prove `.fy -> Go Stage 0 compiler -> verified MIR -> C ->
  MSVC PE -> native execution` on the closed host-smoke subset.
- The resolver fails closed when Visual Studio components or a complete SDK are absent instead of
  falling back to an undeclared compiler or inherited shell state.
- Formal Windows target work remains a separate future decision with its own TargetSpec, layout,
  ABI, runtime, standard-library, and conformance evidence.
