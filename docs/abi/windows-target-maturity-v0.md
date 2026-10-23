# Windows x86-64 MSVC target maturity matrix v0

## Purpose

This file is the checkable evidence ledger for issue #49. It deliberately separates evidence already owned by the repository from gaps that still block a complete Windows target-maturity or distribution claim.

No row may be promoted by prose alone. A mandatory row is complete only when its owner path contains a direct fixture or receipt and the normal Windows hosted lane executes it from generated Fairy source where the claim is about Fairy behavior.

## Status vocabulary

`proven` means the repository already has direct bounded evidence for the stated claim.

`partial` means useful evidence exists but the issue's broader claim is not yet established.

`blocked` means the required hosted, distribution, or cross-boundary evidence does not yet exist.

## Evidence rows

| Area | Status | Owner | Current evidence | Exit condition |
| --- | --- | --- | --- | --- |
| Same-commit hosted LLP64 baseline | partial | `.github/workflows/frontend.yml`, `tools/dev/fairyc-windows-host-smoke/` | Windows CI executes the native smoke on the PR commit | Publish one compact receipt that records compiler SHA, target ID, development/release profiles, and every mandatory sub-receipt consumed by maturity promotion |
| Scalar Fairy to C and C to Fairy calls | proven | `docs/abi/c-abi-v0.md`, native ABI fixtures | Existing independently compiled crossings cover the admitted scalar ABI | Keep in mandatory smoke and prevent fallback to same-translation-unit evidence |
| Aggregate Fairy to C and C to Fairy calls | partial | `docs/abi/c-abi-v0.md`, `tools/dev/fairyc-windows-host-smoke/` | Bounded aggregate crossings exist | Broaden same-commit crossings to every aggregate shape claimed by the promoted target profile; public tagged aggregates remain tracked by #47 |
| COFF relocation families used by emitted/runtime objects | partial | `internal/windowsrelocationprobe/`, Windows host smoke, relocation ADRs | Multiple AMD64 relocation formulas and linked-image observations exist | Enumerate every relocation emitted by current Fairy/runtime objects and require an object plus linked-image assertion for each mandatory family |
| Weak externals and fallback chains | partial | Windows relocation/link probes and associated ADRs | Bounded weak-external alias and search-mode behavior has dedicated probes | Add weak chains and cycles used by production objects and require selected plus discarded-target evidence |
| Default-library resolution | partial | Windows linker/native smoke | Current MSVC link paths resolve required runtime libraries | Record exact default-library inputs and prove expected selection/failure behavior without relying on developer-machine accident |
| Archives and COMDAT selection | partial | COMDAT/archive probes, `internal/pecoff/generated_selection.go`, `internal/pecoff/generated_unwind_selection.go`, and ADRs under `docs/adr/` | Associative archive probes and actual generated-unit/runtime parents and unwind children have direct retention/removal evidence | Cover every archive/COMDAT form emitted by the promoted production path and library inputs in the hosted receipt |
| Basic AMD64 unwind tables | proven | unwind probes, PE inspection, Windows host smoke | Generated/runtime frames have bounded `.pdata`/`.xdata` evidence at both profiles | Keep mandatory in same-commit hosted receipt |
| Unwind v2, handlers, and chained unwind | partial | Windows unwind probes and ADRs | Selected handler/relocation forms are already inspected | Complete v2/handler/chain cases that can be emitted by the admitted target and execute unwind through those exact frames |
| Fairy to foreign mixed-frame inspection | partial | `internal/windowsunwindprobe/mixed.go`, Windows native smoke | Live metadata walk across generated Fairy and independent C frames at both profiles | Retain the mixed-stack proof in the same-commit hosted receipt; broader unwind requirements remain separate |
| Foreign to Fairy callback inspection and trap boundary | partial | `tools/dev/fairyc-windows-host-smoke/mixed_stack_probe.go` | Callback frames occur in the live walk; C catches its own exception and Fairy traps terminate without unwinding | Retain both callback outcomes and exact trap mapping in the same-commit hosted receipt |
| CodeView object records | partial | ADR 0238 and CodeView probes | Function-targeted `SECREL`/`SECTION` and RSDS/PDB association are checked | Expand only the records required by the promoted debug contract; do not claim full CodeView support |
| PDB association | partial | ADR 0238, Windows host smoke | Ephemeral MSF 7 PDB identity is associated with final PE RSDS data | Add hosted same-commit artifact retention/retrieval evidence before any distribution/debug-package claim |
| DbgHelp symbolication | partial | ADR 0239, `tools/dev/fairyc-windows-host-smoke/symbolication_windows.go`, `internal/compiler/vector_lexer_debug_map_test.go` | The source-line canary is joined by exact DbgHelp names for three Fairy functions in two optimized scanner variants and rejected PE/PDB identity mismatches | Retain these generated-source checks in the same-commit hosted receipt; bounded symbol lookup does not promote the whole debugger surface |
| UCRT dependency closure | partial | runtime/link smoke, PE import inspection and `tools/windows-portability/` | The transfer fixture records `/MT` program builds and direct/delay-load DLL dependencies in a hashed deployment manifest | Execute the declared files on a clean Windows host and account for runtime-loaded dependencies; local import inspection does not prove closure |
| VCRuntime dependency closure | partial | runtime/link smoke, PE import inspection and `tools/windows-portability/` | The transfer fixture includes development and release artifacts with independent hashes | Same clean-machine manifest/execution requirement as UCRT, including development and release artifacts |
| Clean-machine executable portability | blocked | `tools/windows-portability/` | A transferable ZIP and standalone PowerShell verifier cover toolchain-free compiler use and prebuilt native programs; local execution is not clean-machine proof | Move only the declared files to a clean Windows environment and retain host/image evidence plus the execution receipt; preserve the separate hosted-source identity requirement |
| TargetSpec maturity flags | blocked | target specification and compiler target registry | Existing support flags stay conservative | Promote only after every mandatory row in this matrix is `proven` on the same target contract generation |

## Bounded production observations (2026-10-23)

At clean `8ba3fab7`, the existing tokenizer and file-search benchmark sources were loaded through
`workspacecompiler.Check`, emitted by `cbackend.EmitStorageTransfers`, and built by
`BuildWindowsMSVCTargetCached` with MSVC 14.51.36231. Both profiles reproduce the benchmark's
final-edit stdout exactly. The [compact record](../benchmarks/completion-followup-20261023.json)
contains all four observations; these are local evidence, not the mandatory same-commit hosted receipt.

- The generated and runtime objects use AMD64 `ADDR32NB` and `REL32`, with `libcmt` and `oldnames`
  default libraries. Optimized objects contain COMDAT sections. No weak externals occur in this sample.
- All four final images directly import `kernel32.dll` only. The `/MT` build path therefore adds no
  direct UCRT/VCRuntime DLL import in these examples. This inspection does not enumerate dynamic
  loads or transitive OS dependencies and does not prove clean-machine portability.
- All four final images contain handlers, chained unwind records and three v2 unwind records.
  These requirements cannot be dismissed as unused. Image inspection alone does not establish
  mixed-stack unwind execution through every such frame.

Keep the table's statuses unchanged. Other production shapes, complete COMDAT/unwind evidence,
hosted mixed-stack evidence and distribution transfer remain distinct work.

## Generated COMDAT retention and removal (2026-10-24)

The [production record](../ops/evidence/windows-generated-comdat-selection-20261024.json) checks
the existing tokenizer and file-search workloads at both MSVC profiles. All four executions retain
their expected output. Of 158 non-associative generated-unit/runtime COMDAT parents, 43 have an
owned map entry and matching final bytes outside relocation fields; 115 have an exact named
discard record and no owned map entry. The observed parent kinds are NODUPLICATES and ANY.

`TestGeneratedCOMDATSelection` uses the shipped bytes example in the existing Windows Go lane.
Both profiles execute. Its optimized image checks 20 retained and 32 discarded parents, and the
checker rejects a missing map row, a missing discard record, contradictory retention/removal and
changed machine code. No workflow or runner was added.

This closes the bounded production-parent observation gap. The child check below subsequently
validates the 376 associated unwind sections. Code relocation values, CRT archive selection and
hosted execution remain separate. The row and target maturity flags stay unchanged.

## Generated unwind children (2026-10-24)

The [child record](../ops/evidence/windows-generated-unwind-selection-20261024.json) reuses the
same four executed images. It checks all 376 direct associative `.pdata`/`.xdata` source sections:
130 are retained and 246 have exact discard records and no owned map entries. Retained `.pdata`
must name its code parent and an `.xdata` sibling. Resolving 318 ADDR32NB fields reproduces all
65 source runtime-function entries and 1,192 source `.xdata` bytes in the final image. Identical-code
folding can share final metadata storage, so these are source counts rather than retained-byte totals.
The checker also rejects nonzero orphan `.pdata` bytes outside the exception directory.

`TestGeneratedUnwindSelection` executes the bytes example under both profiles. The optimized
case validates 88 children and 102 RVA fields. It rejects missing map/discard records, structurally
valid altered final unwind data and a changed source function range. These checks add no runner.

This establishes the bounded metadata selection and relocation result. Live virtual unwinding
through every required v2/handler/chained frame, CRT archive selection, hosted execution and
clean-machine transfer remain open. No target maturity flag changes.

## Bounded mixed-stack execution (2026-10-23)

Mixed-frame Windows maturity means walking live stack metadata and containing exceptions at the C
boundary. It does not mean propagating exceptions through Fairy, which the existing product and
C ABI forbid. Virtual unwinding reads metadata and captured registers; it does not run cleanup.

Four local MSVC 14.51.36231 executions at `/Od` and `/O2` pass in the
[mixed-stack record](../ops/evidence/windows-mixed-stack-20261023.json). Generated Fairy and the C
companion are separate objects. `RtlCaptureContext`/`RtlVirtualUnwind` observes the C inspector,
Fairy callback, C caller and Fairy main in order. The exception code and payload are checked and
caught entirely in C, with abnormal inner and normal outer `finally` observations. A Fairy callback
trap exits with code 134, control tag 6 and no output or C `finally` execution. The existing host
smoke now runs these cases. Both rows remain partial; the hosted lane and remaining target gaps
are not covered by this local run.

## Optimized Fairy symbol lookup (2026-10-24)

The [local record](../ops/evidence/windows-fairy-symbolication-20261024.json) runs
the existing independent-C and Fairy scanner debug-map test. Both indexed and
vector-loop Fairy variants require system DbgHelp to resolve `scanner::scan`,
`scanner::report_work` and `numbers::decimal` at their exact retained function
starts with zero displacement. Expected names come from the emitter's function
table and independently decoded PDB procedure records. Previously, DbgHelp
results were collected only for cost reporting without these assertions.

Changing one byte of each PE's validated RSDS GUID while keeping its code and
PDB causes identity rejection before symbol lookup. The existing six release/
debug executions retain equal scanner results and matching function intervals.
The Fairy images are 211,968 bytes each; the optional test PDB is 4,231,168 bytes.
No PDB is added to default builds or the preview. This proves bounded function
names and identity rejection, not original `.fy` line mapping, universal stack
symbolication or hosted execution. The row and target maturity flags remain unchanged.

## Ownership rule

A future PR for issue #49 must name one or a small adjacent group of rows from this table. It must update the row status and evidence path in the same PR as the executable evidence. Broad cleanup, unrelated refactors, or measurement-only changes do not count as maturity evidence.

Rows owned by another dedicated issue remain separate. In particular, #47 owns public C ABI tagged aggregates. Issue #49 may consume that evidence after it lands but must not duplicate its implementation.

## Same-commit receipt rule

The final promotion receipt must identify the exact repository commit, Fairy language revision, SDK revision, runtime ABI revision, target ID, MSVC toolset identity, Windows runner identity, and development/release profile results. A receipt assembled from successful runs of different commits is invalid.

## Distribution rule

Successful execution on `windows-latest` proves hosted compatibility, not redistributability. Distribution maturity requires artifact transfer to a clean environment containing only the documented Windows baseline and explicitly declared redistributable runtime components.

The [2026-10-24 local portability record](../ops/evidence/windows-portability-local-20261024.json)
at `c8f5964e` records the transfer ZIP, declared file hashes and compiler/native DLL imports.
After ZIP extraction, the verifier passes five toolchain-free compiler operations and both
prebuilt byte/file/console programs with empty child PATH. The development host still has
Visual Studio and the Windows SDK. These results prepare the transfer fixture; they do not
change the clean-machine row or establish dynamic DLL-load closure.

## Promotion rule

`x86_64-windows-msvc` remains below complete maturity while any mandatory row is `partial` or `blocked`. No README, TargetSpec flag, release note, or installer may imply complete target or distribution support before the matrix is fully proven.
