# Fairy Windows local-state lock authority, 2026-11-07

Status: actual nonblocking rooted Windows authority/lease prerequisite
implemented; no state record publication or graph CAS claimed.

A disjoint high-reasoning worker implements the two sealed resource owners,
the fixed-name native open/identity/one-byte-lock wrappers and independent
Windows tests. The coordinator reviews those paths, connects the authored
runtime selector, regenerates the current source pack/pin and verifies its
complete Go/Fairy runtime output before committing.

The worker's final scoped Windows test results pass both profiles for
exclusive-create/reopen byte preservation, Go-owner/Fairy-contender and
Fairy-owner/Go/second-process contention, exactly one locked byte, denied
namespace replacement while leased, normal scope/early-return release,
unchanged handle counts, forced-process-death release, retained identity
replacement/missing/different-root refusal and missing/nonregular conditions.
The existing six rooted filesystem fixture bodies also pass both profiles,
including retained-directory replacement, staged binary publication, collision/
failed-publication preservation, cleanup, bounded inventory and stale entries.
Four StateLock/StateLease construction/field-access refusals pass. The worker
also reports the three existing Go Windows host selector tests passing.

The coordinator explicitly replays the two file/directory symlink refusal
cases under low to resolve privilege/skip uncertainty; both execute and pass,
without skip, in **2.213s**. Log: `.cache/state-local-lock-symlink-final.log`.
This focused replay does not recount the unchanged worker matrix as a new run.

The runtime pack now contains **23 sources / 164,536 bytes**, SHA256
`b515e9b924731eb9a0213790b930830dd712a87f63b2c33e48030b624e8b63fe`.
`TestCandidateHTTPSRuntimePackaging` passes low/optimized in **7.685s**,
comparing complete emitted C for nine selector combinations/profile, including
all three new state-lock host symbols and unchanged legacy/HTTPS combinations.
Log: `.cache/state-local-runtime-packaging-final.log`.

Fresh complete-source check then locked check pass in **12.937s / 12.229s**,
at **745 files / 9,347 MIR functions**, empty stderr. All **1,215 authored
.fy files** match snapshot `.cache/state-local-check-962f3c67`; its already
verified lock is restored after the preceding codec-only commit. Record:
`.cache/state-local-check-latest.json`. The unchanged source-check compiler
SHA256 is `4e392248a97b8351c8563c904910fcf88c648a9b4d1323be8cda45dc711eb084`.
Scoped Go vet, gofmt and diff hygiene pass. No broad native regression is
repeated after documentation-only changes.

Selected commands:

```powershell
go test -mod=readonly -count=1 -timeout 5m -v ./internal/cli -run '^TestCandidateStateLocalLock(Windows|RootedCompatibility|Sealed)$'
go test -mod=readonly -count=1 -timeout 4m -v ./internal/cli -run '^TestCandidateStateLocalLockWindows$/low/symlink-precondition'
python -B tools/package-candidate-runtime-assets.py --update-pin --output .cache/state-local-runtime-assets.pack
go test -mod=readonly -count=1 -timeout 5m -v ./internal/cli -run '^TestCandidateHTTPSRuntimePackaging$'
go vet -mod=readonly ./internal/cli ./internal/candidatehost
```

Linux native execution, actual Fairy record I/O/atomic publication, persistent
layout/version CAS and graph execution, ordinary CLI and final acceptance
remain open. Three compiler-private host helpers/allowlist entries and their
runtime pin change; public CLI/SDK ABI, DB/provider/dependencies, runner
configuration and repository policy do not. No push, dispatch, release or deploy.
