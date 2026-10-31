# Windows generated-C debug retention and lookup - 2026-11-01

Status: Windows local verification for the named debugging scope. Linux remains deferred.

## Supported boundary

This is a private developer evidence path for an exact MSVC PE/PDB pair. It supports exact
non-inlined function-start lookup, function name, generated-C source basename, source line,
retained line text and artifact identity. The fixture compiles actual Fairy through checked MIR
and the Windows C backend, with an independently compiled C runtime/boundary object. Generated
entry `main` and `fairyc_debug_boundary` are the named functions. A Fairy input snapshot is
retained for provenance, but is not advertised as a Fairy-source debugger mapping.

The existing private debug builder uses `/Z7`, `/DEBUG:FULL`, `/INCREMENTAL:NO` and contained
`program.pdb`; optimized builds preserve `/O2`, `/OPT:REF` and `/OPT:ICF`. This change does not
alter ordinary product build flags or add product CLI debug switches. Named functions remain
distinct in this corpus; general attribution under folding or inlining is not established.

## Storage and lookup contract

`debug_store.go` (storage commit `a5e5a7c0`) stages an entire directory and renames it under the
PE SHA-256 key. An existing identical bundle is reusable; altered metadata, corrupt contents or
a different PDB cannot silently replace an existing entry. The fixed eight files are:

- `program.exe`, `program.pdb`, `program.map`;
- `main.fy`, `unit.c`, `runtime.c`;
- `unit.obj`, `runtime.obj`.

`manifest.json` uses private schema `fairyc.windows-debug-bundle/v1`. It records source revision,
profile, toolchain, all eight file SHA-256 values and exact raw GUID bytes/age. It is not a stable
distribution format or a Microsoft SymStore layout. The source revision identifies the checkout;
the snapshots/hashes identify the actual fixture inputs. This does not certify a clean release.

Lookup requires a full nonzero lowercase image SHA-256, loads every file through `os.Root`, bounds
the manifest to 64 KiB and each file to 64 MiB, verifies hashes, and independently rechecks the PE
RSDS record against PDB stream 1 and the manifest GUID/age. Missing PDBs, wrong images, malformed
manifests, changed files and path-like/short keys fail. No alternative symbol path is tried by the
store. The root is a trusted local developer directory; hashes are integrity checks, not signed
authenticity. Staged publication is not a power-loss durability guarantee. There is no automatic
eviction, online upload, retention service or crash-recovery promise.

Private developer mode `--debug-lookup` in `tools/dev/fairyc-windows-host-smoke` reads the bundle
and exits after one lookup; it does not execute the full smoke or rebuild the program. Its JSON
`fairyc.windows-debug-lookup/v1` includes the image/PDB/source hashes, source revision, profile,
RVA, exact symbol, retained C basename, line number and line text. A zero/out-of-range/non-code
RVA, a displaced function or missing exact line evidence fails. Return-address normalization
and arbitrary interior PCs are outside this contract.

DbgHelp loads only from the system directory, after PE/PDB identity verification. Calls in this
consumer are serialized and use a private nonzero session key, an explicit fresh local symbol
search directory, exact-symbol matching and no prompts. Global options are restored after
cleanup. Both the PE and PDB are copied from validated retained bytes into the fresh session;
no compiler-build directory or PDB consumer session is reused. Source text comes from the hashed
snapshot, not the deleted original build path. The default host smoke still retains no PDB;
retention is opt-in through the native verification command below.

## CodeView correction and observed cases

The existing pair inspector rejected actual optimized generated-Fairy code because `$LN4` was a
no-type COFF LABEL (storage class 6) exactly at `.text$mn` size `0x3f`. Commit `6383494f` admits
only exact code-section-end labels with no auxiliary records or offset/section addends, and
records `SectionEndLabel`. External/function symbols, non-code labels, nonzero addends and
offsets beyond the section remain rejected. Five independent boundary mutations fail. This is
an observed debug boundary, not an executable byte or a decoder for all CodeView records.

| MSVC profile | `unit.obj` pairs | `runtime.obj` pairs | Fresh-process function/line lookups |
| --- | ---: | ---: | --- |
| low | 8 | 15 | `main` at `unit.c:44`; `fairyc_debug_boundary` at `runtime.c:68` |
| optimized | 12 | 42 | Same two exact function names and source lines |

Every pair is inspected, and each named function has an exact paired SECTION/SECREL reference.
These 77 object pairs do not claim a one-to-one decoding of final PDB records. Independent
DbgHelp queries demonstrate usability for the four named lookups.

Both generated-Fairy executables return 42 with empty output. Each lookup runs the separately
built developer tool from an unrelated working directory after builder cleanup; alternate
symbol environment paths point to that unrelated empty directory. Four interior-PC lookups,
two cross-profile PDB substitutions, four RSDS GUID/age changes, two absent PDBs and four
header/outside-image lookups fail. Existing name/displacement/file/line receipt mutations remain
covered. Storage tests separately run two generated-Fairy images, validate idempotent retention,
and reject metadata replacement, six file corruptions, five missing manifest fields, missing PDB
and four malformed keys per profile. Existing relocation/function/line canaries are rerun too.

## Reproduction and retained evidence

From the repository root in Windows PowerShell:

```powershell
$debugStoreRoot = Join-Path $PWD '.cache/windows-debug-retained-final-20261101'
$env:FAIRYC_RETAINED_DEBUG_STORE = $debugStoreRoot
go test -mod=readonly ./tools/dev/fairyc-windows-host-smoke -run '^TestWindowsRetainedDebugLookup$' -count=1 -v
Remove-Item Env:FAIRYC_RETAINED_DEBUG_STORE
$debugLookupRows = Get-Content (Join-Path $debugStoreRoot 'lookup-results.json') -Raw | ConvertFrom-Json
go run -mod=readonly ./tools/dev/fairyc-windows-host-smoke --debug-lookup --store $debugStoreRoot --image-sha $debugLookupRows[0].imageSHA256 --rva $debugLookupRows[0].rva
```

Omitting the environment variable uses a temporary test store which is removed. Setting it
retains the bundles and four `lookup-results.json` observations locally; later native runs may
add new image keys because MSVC identities change. Reusing this lookup command does not rebuild
the fixture, replace the retained PDB or require the earlier test process. It only builds/runs
the developer lookup utility. The retained artifacts stay ignored, outside Git and CI uploads.

Passed focused Windows checks:

```powershell
go test -mod=readonly ./tools/dev/fairyc-windows-host-smoke -run '^TestWindowsDebugStore$' -count=1 -v
go test -mod=readonly ./tools/dev/fairyc-windows-host-smoke -run '^(TestWindowsRetainedDebug(Lookup|ExistingCanary)|TestRetainedDebugLookupRejectsInvalidArguments|TestValidateWindowsSymbolication)' -count=1 -v
go test -mod=readonly ./internal/pecoff -count=1
go vet -mod=readonly ./internal/pecoff ./tools/dev/fairyc-windows-host-smoke
git diff --check
```

Linux/WSL, full `go test ./...`, full host smoke, hosted CI and clean-machine transfer were not
run. No push, workflow dispatch, release or deployment was performed. No public language/SDK API,
DB, dependency, production compiler flag, repository hygiene rule, CI workflow or runner changed.
The ABI evidence and ordered-work documents now name this bounded scope. CodeView/PDB types,
variables, lexical/inline frames, arbitrary optimized state, Fairy-source stepping, symbol servers,
PDB signing/access control and hosted retention remain separate work; target maturity is unchanged.

Platform references: [COFF storage classes](https://learn.microsoft.com/en-us/windows/win32/debug/pe-format#storage-class),
[DbgHelp options](https://learn.microsoft.com/en-us/windows/win32/api/dbghelp/nf-dbghelp-symsetoptions),
[DbgHelp session identity and search path](https://learn.microsoft.com/en-us/windows/win32/api/dbghelp/nf-dbghelp-syminitialize).
