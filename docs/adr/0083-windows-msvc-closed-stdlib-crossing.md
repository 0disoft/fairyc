# ADR 0083: Windows MSVC closed standard-library crossing

- Status: Accepted
- Date: 2026-07-19
- Target: `x86_64-windows-msvc`
- Command: `fairyc experimental build-target`
- Symbol: `fy_std_text_utf8_valid_v0`

## Context

ADR 0082 selected the Windows target before C emission and produced a native PE, but deliberately
rejected every called foreign symbol. That proved target-correct scalar emission without proving
that a generated Fairy caller and a project-owned C implementation agree at a real Windows call
boundary. Opening arbitrary companion files, libraries, or linker arguments would turn one evidence
gap into an unbounded and unsafe command surface.

The existing allocation-free text-view component is ISO C11 and already exposes one closed internal
entry point whose pointer-width scalar signature is implemented and exercised on Linux. Its
empty-input path needs no platform API, allocator, filesystem, or retained pointer.

## Decision

1. The Windows MSVC standard-library contract admits exactly
   `fy_std_text_utf8_valid_v0(const uint8_t *, size_t) -> uint32_t`. Its generated Fairy declaration
   uses `(*u8, usize) -> u32` and an exact `@link_name`.
2. The target-specific component resolver maps that symbol only to `text-view`. It rejects an
   ordinary foreign symbol, an unknown `fy_std_` name, or any other admitted Stage 0 standard-library
   symbol before resolving MSVC.
3. The Windows support unit contains the minimal selected text-view C source followed by the scalar
   runtime. It contains no SystemHeap, Buffer, Arena, owned-text, Linux File, POSIX feature macro,
   downloaded input, or user-provided object.
4. `examples/windows_target/main.fy` retains the unused `c::long` declaration from ADR 0082, calls
   the UTF-8 predicate with null data and zero length, and returns 42 only when the C result is one.
5. The configured Windows smoke compiles the generated caller, support source, and runtime with the
   same locally resolved x64 MSVC toolchain at low and optimized profiles, inspects every
   intermediate COFF object and both linked PE/unwind structures, executes both images, and requires
   result 42 twice.
6. The command still accepts no companion source, object, library, search path, arbitrary linker
   option, SDK override, or WSL path. A future symbol or component needs its own portability,
   dependency-closure, negative, and native-execution evidence before joining the allowlist.
7. This one-way local crossing does not clear complete bidirectional ABI, complete runtime or
   standard-library, hosted Windows, toolchain baseline/provisioning, debugger, packaging,
   conformance, or support evidence.

## Consequences

- The first declared Windows product path now crosses from checked-in FY source into real
  project-owned C and back, rather than succeeding only inside generated code.
- Target-specific standard-library availability is explicit and fail-closed instead of inferred
  from a C file looking portable.
- The public experimental CLI stays bounded: users cannot smuggle arbitrary native build inputs
  through this feature.

## References

- `docs/adr/0048-source-callable-utf8-validation-transport.md`
- `docs/adr/0064-used-symbol-standard-library-auto-link.md`
- `docs/adr/0082-windows-msvc-target-c-emission.md`
- `docs/runtime/runtime-stdlib-v0.md`
- `docs/cli/command-contract.md`
