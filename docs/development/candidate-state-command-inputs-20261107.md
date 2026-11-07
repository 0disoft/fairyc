# Authored state-command inputs and snapshot reports, 2026-11-07

This slice adds three private helpers for the existing explicit Windows native
developer command, not an installed/selected command entry. The coordinator owns
actual process argv admission; two parallel workers separately own fresh source
identity and snapshot JSON. The coordinator alone owns this receipt, ADR 0542,
the source index/queue, the checked lock and final integration commit.

## Actual arguments

`module_state_command_arguments::read` returns a sealed Invocation owning the
exact process argv and the options parser's scalar selection. `arguments` borrows
the owned vector; `parsed` copies the selection. Arity is checked before reading
argument bytes. At most seven admitted arguments share the 1MiB aggregate policy.
Each too-small scratch is grown only after validating its reported required
length against remaining bytes, then the argument is copied in full. Unknown
host statuses and inconsistent reported lengths are never treated as truncation.
No argument normalization, path rewrite, file I/O or command dispatch occurs.

Typed invocation failures map to exit 2, host/allocation failures to 4. Original
host status, executable-inclusive index and exact needed bytes remain available.
The existing parser owns NUL/syntax/Go-MIR refusal and its documented precedence.

```powershell
go test -mod=readonly ./internal/cli -run '^TestCandidateStateCommandArguments$' -count=1 -v -timeout 5m
```

Windows low/optimized pass **6.768s**, **24 actual OS invocations/profile, 48
total**. Checks cover help, invalid arity, explicit Go-MIR refusal, all six
action/profile combinations, every empty slot, exact 127/128/129-byte scratch
boundaries, 9,000 UTF-8 bytes, astral Unicode, spaces, tabs/newlines, quotes,
backslashes, literal help as a store and `.fy` as source. Returned argv is checked
byte-for-byte, with exact scalar indexes, process exit and empty stderr/trailing
output. No source/store needs to exist. Host argument failure, allocation failure
and impossible-through-Windows command lines above the OS limit are not injected;
the existing pure parser covers its aggregate byte/NUL boundaries separately.

Early test scaffolding was corrected for named enum-pattern syntax, the actual
stdout SDK names, the admitted cast subset and the partitioned probe size before
this passing run. No language/host policy was expanded for that fixture. Log:
`.cache/state-command-arguments.log`.

## Fresh source

`module_state_command_source::prepare(actual, work)` returns the existing sealed
Complete. Its actual admitted layout UID, the existing developer version identity
and empty features determine the coordinate. A fixed private owner 0 binds the
coordinate table; the caller supplies no cached artifact, coordinate or graph.
Both bounded parsing passes and coordinate derivation share the cumulative caller
ledger. Source and all returned identity owners remain detached from caller bytes.

```powershell
go test ./internal/cli -run '^TestCandidateStateCommandSource$' -count=1 -v -timeout 8m
```

Worker observation: **25.855s PASS**, low **4.70s** / optimized **10.31s**,
**14 cases/profile, 28 total**. Independently compiled Go source/native handoff
expectations match source, package coordinate, canonical body bytes/hash and both
public interface identities/encoding. Cases include cells, maps, helpers,
borrow/copy, reject/trap, changed UID, exact 65,536-byte source and one byte above,
empty/malformed/missing-UID/unknown-field/invalid-UTF-8 sources. Valid cases also
exercise exact/one-short/exceeded cumulative work, unchanged input and output
ownership after destroying input. An initial fixture type error was corrected
before the full passing run. Log:
`%TEMP%/fairyc-state-command-source.log`.

## Snapshot JSON

`module_state_command_snapshot_report::snapshot(options, prepared, state, work)`
accepts only init/snapshot and a State whose layout matches the sealed Prepared.
Its 32MiB output limit and 1GiB cumulative work limit preserve original error
codes (1001 invalid, 1002 work/bound, 1003 allocation). Entry visits, emitted
bytes and base64 work are charged; errors return no partial JSON. Actual
transport/body/layout identities supply metadata. Full-width numeric versions,
32-byte binary keys, null/empty/base64 values and the final newline match Go.

```powershell
go test -mod=readonly ./internal/cli -run '^TestCandidateStateCommandSnapshotReport$' -count=1 -v -timeout=6m
```

Worker observation: **34.532s PASS**,
**8 combinations/profile, 16 total**, with actual Prepared plus real rooted local
provider snapshots. Complete output matches independent Go `json.Marshal` bytes
and decodes back to the actual Go snapshot. The matrix crosses init/snapshot,
version 0/MAX and empty/unknown-binary/null/empty-value records in low/optimized.
Every case also checks exact/one-short/exceeded work, invalid actions, layout
mismatch, durable file bytes and owned JSON after source/Prepared/State teardown.
Maximum output/entry-size stress and allocation-failure injection are not claimed.
The original full test output was observed in the worker chat but not saved.
`.cache/state-command-snapshot-report-receipt.json` explicitly preserves a worker
report, not a reconstructed original log. Its file hashes were collected after
the successful run, rather than claiming a test-time input attestation. The
coordinator reuses this valid worker result without another native test run.

## Integration checks

The coordinator's fresh complete isolated `check` / `check --locked` pass in
**15.279s / 14.273s**, **753 source files / 9,467 MIR functions**, empty stderr.
All **1,223 authored files**, including every new module, match the tested
`.cache/state-store-all-8a540a0c` snapshot; no pending source is excluded. Only
the successful checked lock is copied back. Record:
`.cache/state-store-all-check.json`, log `.cache/state-command-inputs-check.log`.
Scoped `go vet -mod=readonly ./internal/cli`, gofmt and diff hygiene pass.

## Remaining scope

The actual authored command entry still needs source/asset file loading, builder/
store/run orchestration, executed-outcome/error reporting and cancellation/console
selection. Full Go/Fairy comparison, Linux execution, self-build/distribution
and hosted final acceptance are not inferred from the helper tests. The broader
implementation objective remains active; no push, CI dispatch or release.
