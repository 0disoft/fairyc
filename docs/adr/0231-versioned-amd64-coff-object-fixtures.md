# ADR 0231: Versioned exact AMD64 COFF object fixtures

- Status: Implemented local fixture and native-smoke gate
- Date: 2026-07-30
- Target source: `internal/target/`
- Inspector source: `internal/pecoff/`
- Fixture source: `spec/fixtures/coff-object/manifest-v0.json`
- Fixture runner: `internal/pecofffixture/`
- Native runner: `cmd/fairyc-windows-host-smoke/`
- Depends on: ADR 0078 and ADR 0079

## Context

ADR 0079 made every temporary MSVC object pass bounded raw-COFF structural inspection, but it did
not preserve exact primary-symbol, relocation-type, or COMDAT-selection meaning. A parser could
silently stop distinguishing an auxiliary symbol from a legal relocation target, accept a made-up
AMD64 relocation type, or lose an associative COMDAT parent while the final executable still ran.

Freezing complete compiler-produced `.obj` bytes would be the wrong fix. Timestamps, section order,
debug records, string-table layout, compiler revision, and optimization details can change without
changing the object contract. The stable test oracle must be a small raw object plus the structural
records that the linker consumes.

## Decision

1. Extend `ObjectReport` with ordered primary-symbol and relocation receipts. A primary symbol
   records its raw table index, name, value, section number, type, storage class, and auxiliary count.
   A relocation records its section, raw COFF virtual-address field, primary-symbol index and name,
   and AMD64 relocation type. The receipt does not relabel that field as a section-relative offset.
2. Reject relocation indexes that name auxiliary symbol records. Continue rejecting indexes beyond
   the raw table.
3. Accept only the closed Microsoft AMD64 relocation-type interval `0x0000` through `0x0010` and
   preserve the exact numeric type in the receipt. This validates format identity; it does not prove
   the linker's relocation formula or final displacement.
4. For every section carrying `IMAGE_SCN_LNK_COMDAT`, require its first same-section primary symbol
   to be the exact static section definition with exactly one auxiliary format-5 record, with the
   next same-section primary being a COMDAT symbol for that section. The auxiliary size, relocation count, and line count
   must equal the section header. Accept only selection values 1 through 6.
5. An associative COMDAT must name a different COMDAT parent with an exact definition. Association
   chains may pass through other associative COMDATs, but must be acyclic and terminate at a
   non-associative COMDAT. Non-associative selections must not carry a parent. A non-COMDAT section
   definition must have both selection and parent zero.
6. Add strict manifest `fairyc.coff-object-fixtures/v0` with two exact lowercase-hex raw AMD64 COFF
   objects. One fixes `IMAGE_COMDAT_SELECT_ANY` plus `IMAGE_REL_AMD64_REL32`; the other fixes an
   associative `.rdata$x` child of `.text$mn`. Expected sections, symbols, and relocations are
   complete and ordered.
7. Keep fixture decoding separate from object inspection. The runner rejects unknown, missing, or
   null required fields, trailing data, identity drift, invalid case IDs, unsorted or duplicate case
   IDs, non-lowercase or incomplete hex, and any section, symbol, or relocation receipt mismatch.
8. Apply the same strengthened inspector to every low- and optimized-profile MSVC object before the
   existing Windows smoke can execute or report success.
9. Do not clear the Windows TargetSpec `pe-coff-object-fixtures` evidence marker until a fresh hosted
   Windows job succeeds on the exact commit containing this fixture. Local MSVC execution and a
   historical hosted receipt do not satisfy that same-commit gate.

## Evidence and limits

Unit tests cover ordinary `.text`, MSVC `.text$...`, storage-free `.bss`, exact primary-symbol and
REL32 receipts, ANY and associative COMDATs, exact auxiliary counts and header agreement, following
COMDAT symbols, non-COMDAT parents, association cycles, missing definitions, invalid selections and
parents, auxiliary relocation targets, unknown relocation types, malformed ranges, and section-count
bounds. The manifest runner fixes two complete raw-object vectors, requires every schema-required
key at runtime, and kills independent section, symbol, and relocation expectation mutations. The
configured local Windows smoke applies the strengthened inspector to every compiler-generated object
under both profiles.

This is host-side structural evidence, not an object writer, linker implementation, stable public
Fairy ABI, or serialized product format. Whole-object MSVC identity, relocation formulas and linked
displacements, archive extraction, final-link COMDAT winner selection, weak externals, bigobj,
relocation overflow, imports/exports, CodeView/PDB, unwind relocation semantics, independent
producer crossings, and hosted same-commit evidence remain separate gates.

## External basis

- Microsoft PE/COFF format: <https://learn.microsoft.com/en-us/windows/win32/debug/pe-format>
