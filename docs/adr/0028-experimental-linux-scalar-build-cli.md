# ADR 0028: Experimental Linux scalar build CLI

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-19
- CLI source: `docs/cli/command-contract.md`
- Backend source: `docs/mir/c-backend-safety-v0.md`
- Implementation source: `internal/cli/`, `internal/cdriver/`, and `internal/cruntime/`

## Context

The Go Stage 0 compiler could interpret verified MIR and emit a constrained C11 translation unit,
while a separate Linux fixture command contained its own compiler and trap-runtime plumbing. A
developer still could not request a native executable from the canonical prototype. Copying the
fixture implementation into the CLI would create two compiler-option and runtime contracts, and a
default output beside the source would risk overwriting or tracking generated binaries.

The stable package build, target selection, cache, object, linker, and install contracts remain
unfinished. The first user-visible native build therefore needs to stay explicitly experimental,
single-file, scalar-payload-only, and fail closed outside the one declared host profile.

## Decision

1. `fairyc experimental build <file.fy> --cc <clang|gcc> --profile <low|optimized> --output <build/path>`
   is the sixth unstable developer command. Compiler, profile, and output have no implicit default.
2. The source file follows the same bounded source-root-relative loading, syntax, semantics, Flow,
   MIR-lowering, verifier, `main() -> i32`, and scalar-payload C-emission gates as
   `experimental emit-c`.
3. The command is host-native only. It requires Linux on amd64, glibc reported by `getconf`, and the
   explicitly selected `clang` or `gcc` executable from `PATH`. It does not claim cross-compilation.
4. `internal/cdriver` owns the shared compiler invocation used by both the CLI and native
   differential runner. It fixes ISO C11, pedantic errors, common warnings as errors, `-fno-common`,
   and exactly `-O0` for `low` or `-O2` for `optimized`. Tool commands have a 60-second timeout and
   64-KiB combined-output limit; linked executable input to the CLI is capped at 64 MiB.
5. `internal/cruntime` embeds the scalar C runtime source used by both paths. Product builds reserve
   process statuses `129` through `135` for the seven current trap categories by binding base `128`;
   an invalid internal trap code exits `255`. This is a provisional process observation, not a
   stable runtime ABI, unwinding contract, or general application exit-code policy.
6. Output must be a canonical forward-slash path below the ignored source-root `build/` directory.
   Absolute, drive-relative, UNC, parent, dot, mixed-separator, overlong, control-character,
   platform-reserved-character, trailing-dot or trailing-space, and Windows-device-name paths fail
   before toolchain invocation.
7. Artifact materialization uses a source-root capability, creates only directories below `build/`,
   writes a random same-directory staging file, flushes it, and promotes it with a no-overwrite hard
   link. A pre-existing output is never replaced. Staging state is removed on ordinary failure.
8. Success prints one relative-path receipt with target, compiler family, and profile. Source
   diagnostics keep their existing exits; toolchain, link, runtime, or artifact failures use
   experimental exit `6` and bounded stderr. No object, C file, cache entry, manifest, network
   request, package graph, or installation side effect is produced.
9. This command does not make `x86_64-linux-gnu` conformant or supported, does not implement stable
   `fairyc build`, and does not mean the Go bootstrap compiler is self-hosted.

## Consequences

- A scalar-payload `.fy` program can now reach an actual native executable through the canonical Go Stage 0
  compiler on an eligible Linux host.
- Native differential fixtures and user builds cannot silently drift to different compiler flags or
  trap-runtime implementations.
- ADR 0029 admits `Option<i32>` and `Result<i32, i32>`; resources, resource-bearing or nested tags,
  Region, unsafe, FFI, standard-library, and broader runtime programs still fail at the C-emission
  capability boundary.
- The current Windows development host proves parsing, path safety, artifact promotion, and
  platform rejection, but cannot supply native build or execution evidence. The Linux CI lane owns
  that evidence until a compatible local toolchain is available.

## Rejected alternatives

- Default to whichever C compiler is installed: makes build identity depend on host accident.
- Default to optimized output: hides a semantic test dimension and makes reproducing failures harder.
- Write beside the source: mixes generated binaries with source ownership and risks collisions.
- Permit arbitrary output paths with lexical prefix checks: enables traversal, symlink, drive, and
  repository-metadata mistakes.
- Let the native fixture runner keep a separate C runtime file: allows the tested path and product
  path to diverge.
- Call the target supported after one executable builds: ignores runtime, ABI, standard-library,
  differential, sanitizer, and clean-CI gates.
