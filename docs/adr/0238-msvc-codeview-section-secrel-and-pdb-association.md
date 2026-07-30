# ADR 0238: MSVC CodeView SECTION/SECREL and PDB association

- Status: Implemented local synthetic and native-smoke gate
- Date: 2026-07-31
- Object inspector: `internal/pecoff/codeview_relocations.go`
- PE/PDB inspector: `internal/pecoff/program_database.go`
- MSVC driver: `internal/cdriver/`
- Native runner: `cmd/fairyc-windows-host-smoke/`
- Depends on: ADR 0079, ADR 0231, ADR 0232, and ADR 0237

## Context

AMD64 `SECTION` and `SECREL` relocations usually describe a section number plus an offset inside
that section for CodeView data. LINK consumes the object debug sections and normally publishes the
result in a separate PDB; there is no honest general formula that reads those resolved values back
from the final executable as if they were ordinary code relocations.

The previous object inspector admitted the relocation type numbers but did not prove their pair
shape, raw addends, or exact symbol targets. The Windows smoke also discarded any generated PDB and
did not bind it to the final PE. Treating successful `/DEBUG` linking or a file named `program.pdb`
as sufficient would allow mismatched or malformed debug artifacts to pass.

## Decision

1. Compile only the existing two relocation-probe C inputs with `/Z7`. Link those inputs plus the
   exact REL32-variant object with `/DEBUG:FULL`, a contained `program.pdb` path, and the existing
   non-incremental low and optimized profiles. No product build or public Fairy C debug mode changes.
2. Inspect every `.debug$S` AMD64 `SECREL` relocation and require exactly one same-symbol `SECTION`
   relocation four bytes later. Reject missing, duplicate, reused, out-of-range, undefined-target,
   and arithmetic-overflow pairs.
3. Read the raw 32-bit `SECREL` and 16-bit `SECTION` addends independently. Record the object-level
   formulas `symbol.Value + offsetAddend` and `symbol.SectionNumber + sectionAddend`; require exact
   pairs for the relocation source, ordinary target, and generated-object target functions in the
   compiler-emitted caller and target objects.
4. Correct the generic COMDAT validator using actual `/O2 /Z7` evidence. A non-associative COMDAT
   still requires a following key symbol, but a direct associative child such as MSVC `.debug$S`
   may legally contain only its static section definition and association auxiliary record.
5. Preserve the generated PDB bytes before temporary cleanup. Parse the final PE debug directory,
   require one bounded RSDS CodeView record, a contained `program.pdb` basename, consistent debug
   RVA/file offsets, and no trailing path bytes.
6. Parse the bounded MSF 7 superblock, stream directory, and PDB information stream. Require the
   PDB GUID and age to exactly match the final image RSDS identity. Reject escaped blocks, truncated
   directories, unsupported multi-block directory maps, malformed streams, and identity drift.
7. Keep synthetic mutation tests separate for CodeView pair structure/formulas, the associative
   COMDAT rule, PE debug-directory mapping, MSF container bounds, path containment, and GUID/age
   association. Execute both final images and retain every previous relocation oracle.

## Evidence and limits

The two existing relocation-probe executions now prove that local MSVC 14.51 emits exact function
`SECREL`/`SECTION` pairs accepted by the bounded object inspector, LINK consumes those `/Z7` objects
at low and optimized profiles, and each final PE names the exact generated MSF 7 PDB identity. The
PDB remains in memory only for validation and adds no distributable artifact or PE image count.

This is not complete CodeView or PDB support. It does not decode symbol, type, module, line, inline,
source, named-stream, DBI, TPI, IPI, global/public-symbol, or frame-data records; prove that a
specific object pair became a particular PDB record; retain or publish symbols; provide source-link
or path remapping; exercise FASTLINK, incremental linking, LTCG, ICF debug attribution, another
toolchain, or hosted same-commit retrieval; or define a stable public debug-artifact format. A real
ADR 0239 adds one independent system-DbgHelp function-and-line canary. Broader debugger,
symbol-store, source-retrieval, and product debugging claims remain gated.

## External basis

- Microsoft PE/COFF format and AMD64 relocations: <https://learn.microsoft.com/en-us/windows/win32/debug/pe-format>
- MSVC `/Z7`, `/Zi`, and `/ZI`: <https://learn.microsoft.com/en-us/cpp/build/reference/z7-zi-zi-debug-information-format?view=msvc-170>
- LINK `/DEBUG`: <https://learn.microsoft.com/en-us/cpp/build/reference/debug-generate-debug-info?view=msvc-170>
- LINK `/PDB`: <https://learn.microsoft.com/en-us/cpp/build/reference/pdb-use-program-database?view=msvc-170>
