# Windows Fairy candidate inline receipt, 2026-11-02

- Sources: `324042ea` (function identity), `61ac698f` (admission, detached IR and
  cache classes), `39365ee3` (shared-header emission and packet readers).
- Host: Windows amd64, Go 1.27.1, MSVC 14.51.36231, C11 low and optimized.
- Scope: [candidate inline v1](../architecture/candidate-inline-v1.md), extending
  the earlier [encoder/no-inline baseline](windows-candidate-parity-20261102.md).

## Executed and passed

| Probe | Result |
| --- | --- |
| Function identity | Six valid Go-checked bodies match all SHA words in actual Fairy execution at both profiles. Coverage includes rename/group/literal-spelling stability, body changes, prefix and all admitted binary operators, zero/four parameters and i32 minimum. Twelve invalid declaration/body shapes reject. |
| Actual signature authority | Both previously rejected bounded inline examples now pass the actual authored signature factory; the ordinary function control remains accepted, at both profiles. |
| Complete source-derived keys | Five source graphs (`Plain`, `Aliases`, `Inline`, `InlineEdit`, `InlineFormat`) cover fifteen package instances per profile. All ordinary/generic canonical bytes agree with independently produced Go templates; the ledger also verifies digest text and executing-image SHA binding. Both profiles pass; each ledger is 39,765 bytes. |
| Artifact/source re-entry | Normal whole-project restoration succeeds. Removing a valid marker and attaching an equivalent valid marker to an unannotated function both pass detached structural decoding but fail current-source Complete admission. Four marked body components also round-trip and reject changed digest bytes at both profiles. |
| Partitioned C execution | Each native Fairy probe emits a cross-package program with four inline functions. All four definitions appear in the shared header, with none in body units. Candidate low/optimized x generated-program low/optimized yields four passing executions with exact exit 23, empty output, once-only ordered argument evaluation, arithmetic and i32 minimum assertions. |
| Packet readers | Go's 390-case oracle and actual Fairy decoding agree at both profiles, including an inline-only entry, duplicate headers, header/body duplicate IDs, wrong placement and out-of-range inline frames. Existing deadline and rejected-input controls pass. |
| Whole candidate check | The isolated source workspace passes `check` and `check --locked`: two instances, 578 source files and 7,263 MIR functions. Scoped vet and diff checks pass. |

Commands executed from the repository root:

```powershell
go test -mod=readonly ./internal/cli -run '^(TestCandidateCacheKeyV5Parity|TestCandidateInlineAdmissionBoundary|TestCandidateInlineIdentity|TestCandidateInlinePacketParity)$' -count=1 -timeout 5m -v
$env:FAIRYC_RUN_PARITY_INTEGRATION = '1'
go test -mod=readonly ./internal/cli -run '^(TestCandidateCompleteCacheKeysV5|TestCandidateInlinePipeline)$' -count=1 -timeout 20m -v
# After adding the explicit missing/extra-marker source-admission controls:
go test -mod=readonly ./internal/cli -run '^TestCandidateInlinePipeline$' -count=1 -timeout 12m -v
Remove-Item Env:FAIRYC_RUN_PARITY_INTEGRATION
go test -mod=readonly ./internal/candidatebuild -run '^(TestPacketDecodeOracle|TestFrontendDeadlineAdmissionAndPacket|TestFreshPacketInputRejectsBeforePublication)$' -count=1 -v
go vet -mod=readonly ./internal/cli ./internal/candidatebuild
git diff --check
```

The complete-key probe passes in 165.11 seconds. The final pipeline probe,
including current-source restoration controls, passes in 171.07 seconds. These
are whole probe durations, not compiler throughput or inlining speed claims.
Both heavy probes require explicit opt-in and use the existing private C build
API, partitioned sources, a 16 MiB stack reservation, 60 seconds per compiler
invocation and two minutes per private execution. Product deadlines, ordinary
link admission and default stack policy were not changed.

The full source check used the existing ignored workspace
`.cache/parity-check-3d916675`, refreshed from the current authored sources:

```powershell
Push-Location .cache/parity-check-3d916675
./fairyc-bootstrap.exe check
./fairyc-bootstrap.exe check --locked
Pop-Location
```

## Deferred and limitations

Linux execution remains in the [combined queue](../development/ordered-work-20261031.md).
Identity, signature and packet probes have unexecuted Linux paths. The complete
key and pipeline private build harnesses currently require Windows; their native
Linux companion paths still need implementation. No WSL provisioning occurred.

Full `go test ./...`, C17 consumption, whole-compiler sanitizer instrumentation,
clean-machine installation, a rebuilt standalone whole-candidate CLI acceptance
run and hosted CI were not performed. This receipt covers arithmetic assertions
and normal execution; it does not add a new native arithmetic-trap matrix.
Broader generics, interfaces, closures and wider inline body families retain
their separate candidate-parity work.

Contracts/index/queue are updated; grammar, public API/SDK ABI, DB, dependencies,
repository hygiene files, CI and runner settings are unchanged. The private IR
cache revision deliberately changes to `/1`. No push, release or deployment.
