# ADR 0239: Windows DbgHelp symbolication canary

- Status: Implemented local native-smoke gate
- Date: 2026-07-31
- Canary: `cmd/fairyc-windows-host-smoke/symbolication_windows.go`
- Receipt validator: `cmd/fairyc-windows-host-smoke/symbolication.go`
- Probe: `internal/windowsrelocationprobe/`
- Depends on: ADR 0232, ADR 0238

## Context

ADR 0238 binds a final PE RSDS identity to the matching PDB information stream and proves exact
object-level CodeView SECTION/SECREL pairs. Matching container identities still do not prove that a
real Windows symbol consumer can resolve a function or source line. A malformed or incomplete PDB
could pass structural identity checks while remaining useless during debugging.

Implementing a second full PDB decoder would share assumptions with the new structural parser and
would be a poor independent oracle. Windows already ships DbgHelp as the platform symbol-consumer
boundary used by many diagnostic tools.

## Decision

1. Reuse each low and optimized relocation-probe PE/PDB pair after all structural, relocation, and
   GUID/age checks pass.
2. Write the pair to one fresh private temporary directory as `program.exe` and `program.pdb`.
   Load `dbghelp.dll` only from the Windows system directory and initialize it with that directory
   as the closed symbol search path. Do not configure symbol servers or network fallback.
3. Load the exact PE with `SymLoadModuleExW`. Query the exact `fy_rel32_source` start RVA already
   derived by the independent object-to-image relocation inspector.
4. Require `SymFromAddr` to return the exact undecorated function name with zero displacement.
   Require `SymGetLineFromAddrW64` to return the generator-owned `SourceLine`, zero line
   displacement, and the exact `caller.c` basename. The generator test independently keeps that
   line number attached to the source declaration.
5. Bound symbol-name storage and UTF-16 source-path scanning. Keep the full source path only in the
   in-memory receipt; validation failures expose only its basename. Clean up DbgHelp state and the
   temporary PE/PDB directory after each profile.
6. Keep host-independent receipt mutations for wrong name, nonzero displacement, missing line,
   wrong nonzero line, and wrong source file. The actual DbgHelp calls remain the Windows native
   integration oracle.

## Evidence and limits

Local Windows smoke now resolves the exact known function and source line through system DbgHelp at
both low and optimized profiles. This independently proves that the matching ephemeral PDB is usable
for this narrow non-inlined C function canary; it does not add another linked image.

This does not create a Fairy C product debug mode, retain or publish PDBs, prove arbitrary Fairy C
source mappings, variables, types, lexical scopes, inline chains, optimized-out state, return-address
normalization, folded functions, public/private symbol separation, crash-dump loading, source
retrieval, another debugger, another Windows/DbgHelp version, hosted symbol-store retrieval, or a
release symbol lifecycle. Those remain separate admission gates.

## External basis

- DbgHelp initialization: <https://learn.microsoft.com/en-us/windows/win32/api/dbghelp/nf-dbghelp-syminitializew>
- Module loading: <https://learn.microsoft.com/en-us/windows/win32/api/dbghelp/nf-dbghelp-symloadmoduleexw>
- Symbol lookup: <https://learn.microsoft.com/en-us/windows/win32/api/dbghelp/nf-dbghelp-symfromaddr>
- Line lookup: <https://learn.microsoft.com/en-us/windows/win32/api/dbghelp/nf-dbghelp-symgetlinefromaddrw64>
