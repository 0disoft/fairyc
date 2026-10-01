# Nine original CLI workloads: c34718bd revalidation

2026-10-01. All **754/754** baseline cases passed in **18** new Windows MSVC
native builds (development and release for each workload). There were no final
failed or skipped baseline cases and no compiler/SDK regression was observed
within this corpus. The 692 original file/reparse observations matched before
and after the run, with zero changes.

The pinned compiler source is `c34718bd60ef08596629113e554321357cf2c3fd`, with
SDK `0.0.0-stage0.73`, language `/20`, runtime ABI `/0`, and embedded
`vcs.modified=false`. Compiler executable SHA256:
`7d58dd9c8860f01723f55f2f289a609d0941fa3cd6b06699954ca46336eeb917`.
The checked-in [evidence receipt](evidence/practical-revalidation-c34718bd-20261001.json)
records source and executable digests, audit results, local artifact hashes, and
configured command receipt paths.

| Original workload | Baseline cases passed | New native builds |
| --- | ---: | ---: |
| 01 text statistics | 40 | 2 |
| 02 config validation | 68 | 2 |
| 03 CSV aggregation | 128 | 2 |
| 04 file content search | 62 | 2 |
| 05 directory sizes | 32 | 2 |
| 06 duplicate files | 32 | 2 |
| 07 BMP grayscale | 118 | 2 |
| 08 arithmetic calculator | 156 | 2 |
| 09 task manager | 118 | 2 |

## Scope and provenance

Each workload ran once to completion, in order, from a fresh copy of its
original manifest, Fairy sources, verifier and independent expected values/oracle.
Original binaries, build histories and runtime residue were not reused. Sources
and manifests remained byte-identical. The new compiler refreshed SDK locks;
verifier adaptations changed compiler paths/digests, execution recording and
UTF-8 stream decoding, retaining existing assertions. The required `red.bmp`
fixture was copied with its original SHA256 before the first BMP run.

The local run is preserved under
`fairyc/.cache/revalidation-c34718bd-round3`, including the Korean reports,
full summary, before/after original inventories, compiler metadata, adapted
verifiers, EXEs and raw stdout/stderr/exit records. Every original case name/order,
case record versus actual execution, manifest/source digest and final EXE digest
was audited. Directory reparse/hardlink fixtures were exercised without skips.
The checked-in receipt includes checksums for these ignored local artifacts;
it is a bounded result record, not a replacement for the original source contracts.

## Verifier failures and recovery

Project 04 initially failed a Korean-output assertion because its process reader
used Windows codepage 949 to decode UTF-8 bytes. An independent byte capture of
the already-built EXE matched the fixed expected UTF-8 bytes exactly. Explicit
strict UTF-8 stdout/stderr decoding corrected the copied verifiers for 04–06;
no compiler, SDK, product source or expected-output change was needed.

The development EXE from this run was reused only after its checksum matched
the byte-diagnosis receipt; the release EXE was then built for the first time.
There were 754 final baseline executions plus four initial project-04 executions
and one independent byte diagnostic: **759** native program executions total.
One attempted resume failed in the wrapper before any new native execution;
its failed receipt is retained separately. Compiler `check` ran 10 times;
the native build total remained 18. Initial wrapper preparation also failed due
to Windows backslashes in a JavaScript path literal, before copying projects;
using `path.join` fixed that setup issue. These are verifier/runner failures,
not compiler product regressions or successful baseline cases.

Configured execution: `fairyc_regression_build`, `fairyc_regression_prepare`,
`fairyc_regression_project_1` through `_9`,
`fairyc_regression_encoding_diagnosis`, and `fairyc_regression_finalize`.
The scoped strict command-contract check passed. It retained warnings about
unrelated parent manifest drift and older shared-write declarations; those files
were not repaired for this task.

## Limits

This is compatibility evidence for the original workloads. Their original i32
CSV limits and task-manager two-rename save behavior remain in the historical
sources. Passing them does not prove migration to the new i64, shared reborrow,
mutable subview, repeated-array or contextual-literal features. Their owning
ADR 0413–0417 tests provide separate feature evidence; the updated task-manager
[prepared-save example](../architecture/task-manager-prepared-save.md) provides
separate storage-fault and forced-termination evidence.

No physical power cut, full-volume/quota exhaustion, new injected OS faults,
concurrent save/rollback stress, other OS/CPU, clean-PC installation, editor GUI,
fuzzing or performance experiment ran here. The generation/journal recovery
layer and physical power-loss durability remain outside this result. See
[the capability and storage boundaries](../architecture/practical-workload-improvements.md).
No push, release, installation or system-setting change was performed.
