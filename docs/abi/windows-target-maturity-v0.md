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
| Same-commit hosted LLP64 baseline | partial | `.github/workflows/frontend.yml`, `cmd/fairyc-windows-host-smoke/` | Windows CI executes the native smoke on the PR commit | Publish one compact receipt that records compiler SHA, target ID, development/release profiles, and every mandatory sub-receipt consumed by maturity promotion |
| Scalar Fairy to C and C to Fairy calls | proven | `docs/abi/c-abi-v0.md`, native ABI fixtures | Existing independently compiled crossings cover the admitted scalar ABI | Keep in mandatory smoke and prevent fallback to same-translation-unit evidence |
| Aggregate Fairy to C and C to Fairy calls | partial | `docs/abi/c-abi-v0.md`, `cmd/fairyc-windows-host-smoke/` | Bounded aggregate crossings exist | Broaden same-commit crossings to every aggregate shape claimed by the promoted target profile; public tagged aggregates remain tracked by #47 |
| COFF relocation families used by emitted/runtime objects | partial | `internal/windowsrelocationprobe/`, Windows host smoke, relocation ADRs | Multiple AMD64 relocation formulas and linked-image observations exist | Enumerate every relocation emitted by current Fairy/runtime objects and require an object plus linked-image assertion for each mandatory family |
| Weak externals and fallback chains | partial | Windows relocation/link probes and associated ADRs | Bounded weak-external alias and search-mode behavior has dedicated probes | Add weak chains and cycles used by production objects and require selected plus discarded-target evidence |
| Default-library resolution | partial | Windows linker/native smoke | Current MSVC link paths resolve required runtime libraries | Record exact default-library inputs and prove expected selection/failure behavior without relying on developer-machine accident |
| Archives and COMDAT selection | partial | COMDAT/archive probes and ADRs under `docs/adr/` | Associative COMDAT and discarded-child behavior have direct evidence | Cover every archive/COMDAT form emitted by the promoted production path and retain exact selected/discarded evidence |
| Basic AMD64 unwind tables | proven | unwind probes, PE inspection, Windows host smoke | Generated/runtime frames have bounded `.pdata`/`.xdata` evidence at both profiles | Keep mandatory in same-commit hosted receipt |
| Unwind v2, handlers, and chained unwind | partial | Windows unwind probes and ADRs | Selected handler/relocation forms are already inspected | Complete v2/handler/chain cases that can be emitted by the admitted target and execute unwind through those exact frames |
| Fairy frame to foreign frame unwind crossing | partial | Windows native smoke | Individual Fairy and foreign frame behavior exists | Add direct mixed-stack execution proving the required unwind direction across independently compiled boundaries |
| Foreign frame to Fairy frame unwind crossing | partial | Windows native smoke | Individual Fairy and foreign frame behavior exists | Add the reverse mixed-stack execution and exact failure/trap mapping |
| CodeView object records | partial | ADR 0238 and CodeView probes | Function-targeted `SECREL`/`SECTION` and RSDS/PDB association are checked | Expand only the records required by the promoted debug contract; do not claim full CodeView support |
| PDB association | partial | ADR 0238, Windows host smoke | Ephemeral MSF 7 PDB identity is associated with final PE RSDS data | Add hosted same-commit artifact retention/retrieval evidence before any distribution/debug-package claim |
| DbgHelp symbolication | partial | ADR 0239, `cmd/fairyc-windows-host-smoke/symbolication_windows.go` | One exact function and source-line canary resolves through system DbgHelp | Cover representative optimized Fairy frames and failure behavior; one canary never promotes the whole debugger surface |
| UCRT dependency closure | partial | runtime/link smoke and PE import inspection | Current generated programs execute on hosted Windows runners | Produce an explicit deployment manifest and clean-machine execution proving no undeclared developer-tool installation dependency |
| VCRuntime dependency closure | partial | runtime/link smoke and PE import inspection | Current MSVC-linked artifacts execute in CI | Same clean-machine manifest/execution requirement as UCRT, including development and release artifacts |
| Clean-machine executable portability | blocked | future distribution fixture | Hosted build runners are not a clean-machine distribution proof | Build artifact on one hosted job, move only declared distributable files to a clean Windows environment, execute the mandatory smoke without source tree or compiler toolchain |
| TargetSpec maturity flags | blocked | target specification and compiler target registry | Existing support flags stay conservative | Promote only after every mandatory row in this matrix is `proven` on the same target contract generation |

## Ownership rule

A future PR for issue #49 must name one or a small adjacent group of rows from this table. It must update the row status and evidence path in the same PR as the executable evidence. Broad cleanup, unrelated refactors, or measurement-only changes do not count as maturity evidence.

Rows owned by another dedicated issue remain separate. In particular, #47 owns public C ABI tagged aggregates. Issue #49 may consume that evidence after it lands but must not duplicate its implementation.

## Same-commit receipt rule

The final promotion receipt must identify the exact repository commit, Fairy language revision, SDK revision, runtime ABI revision, target ID, MSVC toolset identity, Windows runner identity, and development/release profile results. A receipt assembled from successful runs of different commits is invalid.

## Distribution rule

Successful execution on `windows-latest` proves hosted compatibility, not redistributability. Distribution maturity requires artifact transfer to a clean environment containing only the documented Windows baseline and explicitly declared redistributable runtime components.

## Promotion rule

`x86_64-windows-msvc` remains below complete maturity while any mandatory row is `partial` or `blocked`. No README, TargetSpec flag, release note, or installer may imply complete target or distribution support before the matrix is fully proven.
