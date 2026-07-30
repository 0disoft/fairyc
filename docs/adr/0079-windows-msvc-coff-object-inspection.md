# ADR 0079: Windows MSVC COFF object inspection

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-19
- Target source: `internal/target/`
- Inspector source: `internal/pecoff/`
- Driver source: `internal/cdriver/`
- Local runner source: `cmd/fairyc-windows-host-smoke/`

## Context

ADR 0078 validates linked PE executable structure, but linking discards or merges facts from the
compiler's raw `.obj` files. A valid final image cannot show whether an input object used the wrong
machine, had malformed section or relocation bounds, or referenced a missing symbol. The Windows C
driver already owns a bounded temporary build root and can inspect each object before cleanup
without retaining build debris.

## Decision

1. `internal/pecoff` has a distinct raw-object entry point. An admitted object must be bounded AMD64
   COFF with no PE optional header and no executable-image or DLL characteristics.
2. Every section name must be present. Raw section ranges must lie after the COFF section table,
   remain inside the file, and not overlap. An uninitialized-data section such as `.bss` may have a
   nonzero logical size with a zero raw pointer only when its characteristic explicitly says so.
3. At least one readable executable code section named `.text` or `.text$...` must exist. The
   subsection form is required because MSVC emits function-level COFF sections before the linker
   merges them.
4. Relocation tables must be bounded, relocation-overflow encoding remains rejected, and every
   relocation symbol index must address the raw COFF symbol table. At least one named non-auxiliary
   symbol is required.
5. Host-independent tests build a minimal object with one relocation and symbol, admit an MSVC
   `.text$...` subsection and storage-free `.bss`, and mutate image identity, machine, linked flags,
   code characteristics, raw pointers, and symbol references for closed rejection.
6. `internal/cdriver` reads and inspects the exact `.obj` corresponding to every source passed to
   local x64 MSVC. The returned internal artifact receipt preserves dense object names and copied
   bytes until its caller finishes, while the temporary files are always removed.
7. The configured Windows smoke therefore gates `build-host`, all fourteen `run-host` outcomes, and
   both profiles of the layout and ABI probes on COFF object structure. Layout receipts must contain
   `probe.obj`; ABI receipts must contain `caller.obj` followed by `callee.obj`.
8. This decision alone does not clear `pe-coff-object-fixtures`. ADR 0231 later adds versioned local
   fixtures for exact ordered primary symbols, AMD64 relocation types, and ANY and associative
   COMDAT selection. ADR 0232 later proves one linked REL32 calculation, ADR 0233 separately proves
   one archive extraction plus pick-any COMDAT final owner, and ADR 0234 binds one ADDR64 plus one
   unwind `.pdata` ADDR32NB begin relocation to final-image values. Other relocation formulas,
   broader archive/COMDAT policies, remaining `.pdata`/`.xdata` relocations, CodeView/PDB, compiler baselines, and a fresh
   hosted same-commit object receipt remain separate gates.

## Consequences

- A malformed object can no longer hide behind a successful final link in the local Windows lane.
- MSVC-specific `.text$...` and `.bss` behavior is explicit instead of being mistaken for malformed
  PE image structure.
- Object bytes remain bounded and ephemeral; the repository and build directories gain no retained
  intermediates.

## External basis

- Microsoft PE/COFF format specification:
  <https://learn.microsoft.com/en-us/windows/win32/debug/pe-format>
