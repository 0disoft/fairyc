# ADR 0242: AMD64 relocation-overflow link proof

- Status: Implemented local synthetic and native-smoke gate
- Date: 2026-07-31
- Object generator: `internal/pecoff/relocation_overflow_object.go`
- Inspector: `internal/pecoff/object.go`, `internal/pecoff/relocation_overflow_link.go`
- Probe: `internal/windowsrelocationprobe/`
- MSVC driver: `internal/cdriver/`
- Native runner: `cmd/fairyc-windows-host-smoke/`
- Depends on: ADR 0079, ADR 0232, ADR 0237, ADR 0241

## Context

The Go `debug/pe` object reader sizes a section relocation slice from the 16-bit
`NumberOfRelocations` header field. It does not normalize the COFF extended-relocation convention,
so an overflow section is truncated at the marker and its count sentinel is exposed as if it were a
semantic relocation. Rejecting the flag was safe, but left a valid Microsoft object boundary
untested.

Microsoft PE/COFF requires `IMAGE_SCN_LNK_NRELOC_OVFL`, a header count of `0xffff`, and the stored
record count in the first relocation's `VirtualAddress`. LLVM's current reader subtracts one from
that stored value and skips the first record; its writer stores the semantic count plus the
sentinel. The selected MSVC LINK accepts the same representation.

## Decision

1. Read each ordinary COFF section header directly from bounded object bytes. Normal and extended
   relocation tables enter one private normalized record path instead of trusting
   `debug/pe.Section.Relocs`.
2. For an extended table, require the flag and `0xffff` marker together, a stored count greater
   than `0xffff`, and zero symbol/type fields in the sentinel. Subtract and skip exactly one
   sentinel. Reject truncated ranges, arithmetic overflow, and more than 1,048,576 semantic
   relocations per object before allocating report records.
3. Generate the smallest LLVM-compatible overflow object: one readable executable `.text`
   section, one external leaf function returning 42, 65,534 semantic
   `IMAGE_REL_AMD64_ABSOLUTE` no-op records, one final `IMAGE_REL_AMD64_REL32` record that binds the
   function's RIP-relative load to its adjacent 42 constant, and one leading count sentinel whose
   value is 65,536. Ignoring or truncating the final semantic record changes the native result.
4. Preserve independent mutations for the header marker, below-threshold count, sentinel symbol,
   sentinel type, truncated range, and allocation budget.
5. Reuse the existing low and optimized relocation images. Require the exact source symbol to be
   owned by `overflow.obj` in the MSVC map, preserve its opcode, return, and data bytes in an
   executable PE section, recompute the final linked displacement, call it from the C consumer,
   and require result 42. No PE image or process execution is added.

## Evidence and limits

The configured local Windows smoke generated the exact 655,471-byte object, parsed all 65,535
semantic records, linked it with MSVC 14.51 at `/Od` and `/O2`, bound the final map owner and last
REL32 displacement, and executed both existing relocation probes successfully. The ordinary Go
suite also kills each malformed sentinel combination independently.

This is a compiler-private validation object, not a Fairy C object writer, public prebuilt-object
API, or promise that arbitrary large objects are cheap. It does not prove a versioned raw overflow
fixture, COMDAT section-definition overflow metadata, multiple overflow sections, bigobj,
incremental link, LTCG, non-MSVC linkers, another MSVC release, hosted same-commit evidence, or
non-AMD64 COFF. `pe-coff-object-fixtures` therefore remains fail-closed.

## External basis

- Microsoft PE/COFF section flags and relocation records:
  <https://learn.microsoft.com/en-us/windows/win32/debug/pe-format>
- LLVM COFF reader count normalization:
  <https://llvm.org/doxygen/COFFObjectFile_8cpp_source.html>
- LLVM WinCOFF writer sentinel emission:
  <https://codebrowser.dev/llvm/llvm/lib/MC/WinCOFFObjectWriter.cpp.html>
