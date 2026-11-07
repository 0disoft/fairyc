# Actual Windows state command boundaries, 2026-11-07

The native-console worker adds one disjoint actual-entry test while the
coordinator integrates console ownership and developer native build preparation.
No production source fix is needed for these observations.

```powershell
go test ./internal/cli -run '^TestCandidateStateCommandWindowsBoundary$' -count=1 -v -timeout 10m
```

**47.962s PASS**, low **9.03s** / optimized **25.19s**, six scenarios total.
The unchanged authored `tools/dev/fairyc-state/main.fy` and actual native support
are built in both profiles. Input hashes are checked again at cleanup. Two
scheduling wrappers retain the real local event and observe the real third
successful state lease (open, snapshot, CAS); each calls the original exactly
once and preserves its return/OS cause. Reversing the wrappers restores exact
support bytes. No provider, native graph result, metadata or durable file is
substituted. Isolated children alone own their hidden new console.

Each case starts from version 7 established by seven actual Go provider CAS
operations, preserving unknown binary state. An independent current Go native
command establishes exact expected JSON, snapshot and durable bytes for the
same helper-transitive graph. The authored process then exercises:

- **Closed stdout**: a real OS pipe has no reader. Execute returns exit 4,
  reports `report output failed` and warns publication may already have occurred.
  Commit remains durable and is never retried or reported as cancellation.
- **CAS-entry cancellation**: after the actual CAS lease succeeds, its actual
  local event is signaled and observed latched before returning to Fairy.
  Execute still returns exit 0 with exact Go JSON and no stderr.
- **Competing peer**: the real CAS lease is held while a separate actual Go
  snapshot command runs. That peer returns busy exit 7; the owning Fairy execute
  returns exit 0 with exact JSON. This is peer busy-lock acceptance, not a claim
  that the Fairy command itself encounters a stale-version conflict.

Every successful owner acquires exactly three leases, signals no fourth/retry
observation, and ends at version 8. Two independent Go provider reopens per
scenario preserve the complete expected snapshot and exact durable bytes:
SHA-256 `9ae8c9de131e64312c540276692f9d521ad8f7558af9e0be4f0b510827aef6da`.
Started children are killed/reaped exactly once on failure paths; stdout/stderr
copy goroutines finish before failure diagnostics inspect their buffers. Pipes,
events and provider instances have scoped cleanup.

Full log: `%TEMP%/fairyc-state-command-windows-boundary.log`. Scoped vet,
formatting and diff hygiene pass. Existing current-source lock remains valid:
this slice adds only a Go test and documentation, changing no authored source.
ADR 0543/0544 semantics are exercised without another contract change.

Actual command stale-version conflict/late unknown-publication boundaries,
broader source families, final Go/Fairy/self-build/distribution/clean-machine
and hosted acceptance remain separate. Native Linux remains batched. No
production API/ABI, DB/provider, dependency, repository policy or CI/runner
change, push, dispatch, release or deployment. The full objective remains active.
