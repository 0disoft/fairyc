# ADR 0088: Experimental Windows target run command

- Status: Accepted
- Date: 2026-07-20
- Target: `x86_64-windows-msvc`
- Command: `fairyc experimental run-target`

## Context

`experimental build-target` can retain a real Windows target PE, but a developer still has to run
that file separately and interpret Windows process exit bits. `run-host` already executes temporary
PE files, but it intentionally rejects target-sensitive C scalars, `@repr(C)`, FFI, and provisional
standard-library linkage. Reusing that target-neutral command would erase the distinction between a
host smoke and a declared target build.

The direct target command must not grow a second compiler path, accept arbitrary linker inputs, or
skip the PE and unwind checks applied by the Windows evidence lane.

## Decision

1. Add `fairyc experimental run-target <file.fy> --target x86_64-windows-msvc --profile
   <low|optimized>`. The command accepts no output path, compiler selector, companion source, object,
   library, search path, linker option, or ambient target.
2. `build-target` and `run-target` share one function for source loading, Windows target C emission,
   exact standard-library closure, MSVC discovery, compilation, COFF object inspection, linked PE
   inspection, and Microsoft x64 unwind inspection.
3. `run-target` writes the validated artifact to a driver-owned temporary directory, executes it
   with closed stdin and bounded combined output, and removes the directory. Any stdout or stderr is
   a build/execution failure until target I/O is implemented.
4. An ordinary result prints `native result: <i32> (x86_64-windows-msvc, msvc <version>,
   <profile>)`. Windows process bits are preserved as signed `i32`. Reserved product-trap statuses
   print `trap: <category>` and return CLI exit code 5.
5. The configured Windows smoke executes the checked-in target example once through `run-target` at
   the optimized profile, requires result 42, and includes its temporary image in the ten-image
   PE/unwind evidence count. No WSL or Linux compatibility layer participates.
6. This command is an unstable developer surface. It does not establish hosted execution, target
   conformance, support, installation, distribution, debugging, source-visible I/O, or self-hosting.

## Consequences

- A user can now run the implemented Windows target FY subset directly instead of manually launching
  a retained build artifact.
- Retained and temporary target paths cannot drift in lowering, support linkage, or structural
  validation without failing shared tests and the Windows smoke.
- `run-host` remains the deliberately target-neutral lane; `run-target` is explicitly target-bound.

## References

- `docs/adr/0068-windows-msvc-target-neutral-host-smoke.md`
- `docs/adr/0082-windows-msvc-target-c-emission.md`
- `docs/adr/0083-windows-msvc-closed-stdlib-crossing.md`
- `docs/cli/command-contract.md`
- `docs/cli/output-and-exit-codes.md`
