# ADR 0078: Windows PE executable structure inspection

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-19
- Target source: `internal/target/`
- Inspector source: `internal/pecoff/`
- Local runner source: `cmd/fairyc-windows-host-smoke/`

## Context

The Windows host smoke previously accepted the retained product artifact after checking only its
size and two-byte `MZ` prefix. That distinguishes a likely PE file from arbitrary output but says
nothing about the COFF machine, PE32+ optional header, subsystem, alignment, section ranges, or
entry point. The local layout and ABI artifacts were executed without an independent structural
parse. Successful loading alone is not a reviewable target contract because the host loader does
not expose which structural assumptions were checked.

## Decision

1. `internal/pecoff` uses Go's bounded PE reader over in-memory bytes and adds the project-owned
   executable-image invariants required by the provisional Windows host profile.
2. An admitted image must be at most 64 MiB, start with `MZ`, contain an AMD64 COFF header, use a
   PE32+ optional header, be executable and large-address-aware but not a DLL, and select the Windows
   console subsystem.
3. Image base, file alignment, section alignment, header size, and image size must satisfy the
   closed power-of-two and range rules. Sections must have unique names, strictly increasing aligned
   virtual addresses, aligned and in-file raw ranges, and in-image virtual ranges.
4. A readable executable code section named `.text` must exist, and the nonzero entry-point RVA
   must land inside executable code. Import and exception-directory presence is recorded without
   claiming their contents are valid.
5. Host-independent tests build one minimal valid AMD64 PE32+ byte image and mutate its signature,
   machine, subsystem, image-base alignment, section characteristics, raw-size alignment, and raw
   range to prove closed rejection behavior.
6. The configured Windows smoke applies the same inspector to five local MSVC images before
   execution: one retained `build-host` product plus low and optimized layout and ABI probes.
7. This gate does not clear `pe-coff-object-fixtures`. It parses linked PE executable structure, not
   `.obj` symbols, COMDATs, relocations, imports as an API contract, `.pdata`/`.xdata` unwind
   semantics, CodeView/PDB records, signatures, installers, or hosted distribution behavior.

## Consequences

- A file with only an `MZ` prefix can no longer masquerade as a valid Windows smoke artifact.
- The same parser covers retained and temporary products, so probe executables cannot silently
  drift onto a different machine, subsystem, or image class.
- ADR 0079 later adds structural inspection of every live MSVC COFF object. Exact versioned object
  expectations and Windows unwind validation remain separate because linked image structure and
  generic bounds cannot prove their records or semantics.

## External basis

- Microsoft PE/COFF format specification:
  <https://learn.microsoft.com/en-us/windows/win32/debug/pe-format>
