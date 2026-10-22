# Validation

- Status: Draft

## Validation Source of Truth

This document defines the stable validation names for this repository.

## Standard Validation Names

- format
- lint
- typecheck
- test
- contract
- migration-check
- smoke
- docs
- check

## Required Final Report

Final responses must list executed validations, passed validations, skipped validations, skip reasons, and remaining risk.

## Runner Policy

Task runner files are optional. Runner `none` means no executable task runner is generated.
If a runner is generated, runner command names must match this document.
Unconfigured runner commands must fail rather than report a fake success.

## Hygiene Validation

Changes to repository hygiene files must be checked for line-ending churn, binary diff
pollution, tracked secret files, ignored build/cache artifacts, and generated-output drift.

## Scope

General validation routes must stay stack-neutral unless a runner file explicitly defines a command.

## Frontend CI routing

`frontend.yml` first runs a lightweight changed-path check. Root Markdown, documentation and
agent-guide Markdown, example/editor/fixture READMEs, example/editor guides, and images under
`assets/brand/`, `docs/`, or `diagrams/` can skip compiler jobs when they are the only changes.
Mermaid files under `diagrams/` also qualify. Other paths, including fixture JSON, grammar,
source, dependency files, and workflows, retain the full compiler checks.

The check reads the complete push diff or PR merge-base diff, including removed paths. Empty
diffs, new branches, tags, missing history, and failed path classification run compiler checks.
Manual Frontend runs also execute the full checks. The scheduled/manual Measurements workflow
keeps its existing triggers.

The Windows core job allows 20 minutes, with a 15-minute Go test timeout per package, because
it includes native compile/run regressions. The separate lexer proof allows 12 minutes so its
native tests and job setup/cleanup fit within the limit. Linux and the separate native/parser jobs
retain their 10-minute limits. The test selection and runner count are unchanged.

The existing `native-c` job uses `ubuntu-24.04` and provisions the
[reference Linux toolchain](docs/abi/linux-toolchain-v0.md) with
`.github/scripts/setup_linux_toolchain.sh`. Only missing compiler, development, linker, and sanitizer
packages are installed; GCC 13 and Clang 18 are exposed through a job-local PATH directory.
Package patch versions remain unpinned. The full native fixture command still validates actual
headers, linking, startup, and sanitizers; version preflight is not a replacement for that execution.

The Linux Go suite now runs exactly once, with `go test -mod=readonly -count=1 -json ./...` piped
into `python3 -B .github/scripts/linux_phase2_evidence.py`, which streams selected test output,
package summaries and failures. The validator emits `fairyc.phase2-linux-evidence/v4` and marks a
run complete only when `TestLinuxFileProgressNative`, `TestSystemHeapOOMNative`,
`TestLinuxPreparedPublication`, `TestBytesEqualExecution`, `TestBytesCompareExecution`,
`TestCheckedI64DecimalExecution`, `TestLinuxTrapReceiptProgressNative`, and
`TestPrivateVectorNestedResourceDrops` pass all 32 GCC/Clang
low and optimized subtests without skips, missing cases or duplicates, with any other suite failure
also rejecting completeness. The schema
version changed because the required evidence set grew from 28 to 32 cases (eight roots, four
compiler/profile cases each). Each trap-receipt case runs six progress/failure scenarios.
The nested-resource root checks parent mutation and active payload cleanup, with one extra
ASan+UBSan execution per compiler inside its low-profile case. These two executions are separate
from the native fixture report's sanitizer count. Hosted v4 execution is pending.
Existing v1/12-case, v2/24-case, and v3/28-case receipts remain historical. The
[v3 Linux Go receipt](https://github.com/0disoft/fairyc/actions/runs/39805612550/job/119918006715)
at `7f7362b87c231c30ea89f3907396957f0d599032` records all 28 cases passing without skips.
The Go suite command and runner count are unchanged, and no native tests are
repeated.
The explicit Bash shell preserves pipeline failure if either the Go suite or receipt checker fails.

Local CI helper checks: `python -B -m unittest discover -s .github/scripts -p 'test_*.py'`.
Workflow syntax: `actionlint .github/workflows/frontend.yml .github/workflows/measurements.yml`.

Local Windows cache-only measurements:
`go run -mod=readonly ./tools/dev/fairyc-windows-host-smoke --cache-measurements-only`.
This mode builds a standalone compiler, then reuses the process-cache, edit-invalidation,
persistent cold/warm, and bounded-growth measurement oracles. It writes four validated JSON
receipts to stdout with the same labels as the full measurement lane and removes its temporary
workspaces and compiler. Compiler construction is outside the recorded durations. It does not
run native or Atlas smokes, and neither workflow selects this mode automatically.

## Repository Shape

cli-tool and monorepo validation must stay focused on repository shape and must not imply generated application source code.
