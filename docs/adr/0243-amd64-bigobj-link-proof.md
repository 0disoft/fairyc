# ADR 0243: AMD64 bigobj link proof

- Status: Implemented local synthetic and native-smoke gate
- Date: 2026-07-31
- Parser: `internal/pecoff/big_object.go`
- Object generator: `internal/pecoff/big_object_probe.go`
- Link inspector: `internal/pecoff/big_object_link.go`
- Probe: `internal/windowsrelocationprobe/`
- MSVC driver: `internal/cdriver/`
- Native runner: `cmd/fairyc-windows-host-smoke/`
- Depends on: ADR 0079, ADR 0232, ADR 0242

## Context

Microsoft `/bigobj` replaces the ordinary 20-byte COFF file header with a 56-byte anonymous
object header, expands section count and symbol section numbers to 32 bits, and expands symbol
records from 18 to 20 bytes. Go `debug/pe` does not recognize that header. Passing bigobj bytes to
the ordinary reader therefore cannot produce a trustworthy object receipt.

The format's theoretical section address space is not an appropriate parser budget. Microsoft
documents up to 2^32 addressable sections, but the Fairy C inspector already caps untrusted object
input at 64 MiB and 4,096 sections. Admitting the wire format does not justify removing those
bounds.

## Decision

1. Detect the anonymous-object `0, 0xffff` signature before `debug/pe`. Admit only version 2,
   AMD64, the exact bigobj class GUID, and sixteen zero reserved metadata bytes.
2. Keep the existing 4,096-section limit and add a 262,144-symbol limit. Validate the 56-byte
   header, section table, 20-byte symbol table, string table, raw ranges, relocation ranges,
   symbol section numbers, symbol targets, and executable `.text` characteristics with bounded
   unsigned arithmetic. Raw data, relocation, symbol, and string-table intervals must be mutually
   disjoint.
3. Preserve `ObjectSymbol.SectionNumber` as signed 32-bit data for both ordinary and bigobj
   receipts. The existing versioned fixture JSON remains numerically unchanged.
4. Keep the first admitted slice closed: zero symbol auxiliaries, no COMDAT sections, and short
   section names only. Unsupported anonymous class IDs, versions, reserved metadata, auxiliaries,
   COMDAT, long section names, excess counts, missing symbols, and unknown relocations fail closed.
5. Generate one exact 161-byte version-2 bigobj. Its external leaf function uses an internal
   `IMAGE_REL_AMD64_REL32` against a 20-byte static symbol record to load an adjacent 42 constant.
6. Reuse the low and optimized relocation images. Require `bigobj.obj` map ownership, recompute
   the final displacement, preserve opcode/data bytes, call the function, and require result 42.
   No PE image or process execution is added.

## Evidence and limits

The configured local Windows smoke parsed the exact object, linked it with MSVC 14.51 at `/Od` and
`/O2`, verified final displacement 1, and executed both existing relocation probes successfully.
Unit mutations independently kill version, machine, GUID, reserved fields, count budgets, table
ranges, string table, auxiliary records, symbol section, COMDAT, long section name, relocation
target/type, map owner, linked opcode, and final displacement.

This proves one compiler-private bigobj wire-format path through the selected linker. It does not
prove a high-section object beyond the single generated section, an MSVC `/bigobj`
compiler-produced corpus, auxiliary or
COMDAT record layouts, long section names, relocation overflow inside bigobj, archives containing
bigobj, versioned raw fixtures, another MSVC release, non-MSVC linkers, or hosted same-commit
evidence. It is not a public Fairy object writer or arbitrary prebuilt-object compatibility
promise. `pe-coff-object-fixtures` remains fail-closed.

## External basis

- Microsoft `/bigobj` option and address-capacity boundary:
  <https://learn.microsoft.com/en-us/cpp/build/reference/bigobj-increase-number-of-sections-in-dot-obj-file?view=msvc-170>
- LLVM bigobj header, class GUID, and record sizes:
  <https://sources.debian.org/src/intel-graphics-compiler2/2.28.4-4/llvm-project/llvm/include/llvm/BinaryFormat/COFF.h>
