# ADR 0073: Executable x86_64 Windows MSVC target specification

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-19
- Product source: `docs/product/02-spec.md`
- ABI source: `docs/abi/c-abi-v0.md`
- Compatibility source: `docs/compatibility/versioning-v0.md`
- Implementation source: `internal/target/`
- CLI schema: `spec/cli/experimental-target-spec-v0.schema.json`

## Context

The WSL-free Windows host smoke proves a deliberately target-neutral generated-C subset can pass
through the locally installed x64 MSVC compiler. It does not describe Windows target semantics.
Keeping Windows absent from `TargetSpec` forever would also be wrong: LLP64 changes C `long`, the
Microsoft x64 caller reserves shadow space and has no SysV red zone, and PE/COFF plus Windows unwind
metadata are different compatibility members.

The existing stack object could record call alignment and a red zone but could not represent the
required four-register shadow store. Adding a Windows target without that field would freeze an
incomplete calling contract.

## Decision

1. `internal/target` adds the known target ID `x86_64-windows-msvc` under the existing prototype
   `fairyc.target-spec/v0` schema. Lookup and `target.All()` return isolated validated copies in
   canonical ID order.
2. The target records little-endian x86-64, LLP64, eight-byte data and function pointers, the Stage
   0 language scalar table, and an MSVC C scalar table where `long` and `unsigned long` are four
   bytes while `size_t` is eight bytes. Plain C `char` follows MSVC's default signed profile; the
   compiler profile must not enable `/J`. POSIX `ssize_t` is absent rather than renamed or invented.
3. `StackRules` gains `shadowSpaceBytes`. Linux SysV records zero shadow space and its existing
   128-byte red zone. Microsoft x64 records 16-byte call alignment, zero red-zone bytes, and 32
   shadow-space bytes.
4. The Windows identities are `microsoft-x64`, `msvc-x64-llp64`, `coff-x86-64`, COFF symbols,
   `.pdata`/`.xdata` unwind metadata, and generated-C CodeView/PDB debugging. These are target
   identities, not evidence that the corresponding classifier, object writer, or debugger exists.
5. Go validation and the JSON Schema close the Linux and Windows identity combinations. A known ID
   cannot be paired with the other profile's OS, environment, data model, stack, classifier,
   object, unwind, or debug values.
6. Windows maturity remains fail-closed. Its emission subset is `none`; `emittable`, `conformant`,
   and `supported` stay false. Missing evidence names layout probes, a Microsoft x64 classifier,
   bidirectional calls, PE/COFF objects, unwind fixtures, runtime, standard library, hosted receipt,
   target C emission, and toolchain policy.
7. The existing `run-host` and `build-host` commands remain target-neutral developer smokes. Their
   success does not clear any Windows TargetSpec evidence marker or permit FFI, `@repr(C)`, exact
   C scalars, the standard library, or distributable-artifact claims.

## Consequences

- `fairyc targets` and `fairyc experimental target-spec x86_64-windows-msvc` can report the Windows
  contract without probing MSVC, the SDK, WSL, the registry, or the filesystem.
- The next host-independent gate can execute LLP64 layout fixtures against these facts. Microsoft
  x64 classification, native ABI crossings, PE/COFF inspection, and unwind validation remain
  separate later gates.
- Extending the stack schema changes every target document structurally even though the Linux
  shadow-space value is zero.

## External basis

- Microsoft x64 calling, shadow store, 16-byte stack alignment, aggregate passing, and unwind rules:
  <https://learn.microsoft.com/en-us/cpp/build/x64-calling-convention>
- Microsoft x64 scalar storage and alignment:
  <https://learn.microsoft.com/en-us/cpp/build/x64-software-conventions>
- Microsoft default signed `char` and `/J` override:
  <https://learn.microsoft.com/en-us/cpp/c-language/type-char>
- Microsoft PE/COFF format:
  <https://learn.microsoft.com/en-us/windows/win32/debug/pe-format>
