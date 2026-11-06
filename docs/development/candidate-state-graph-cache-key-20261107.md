# State-graph cache identity prerequisite, 2026-11-07

ADR 0522 supplies canonical graph-key bytes and a separate digest family from
sealed Complete source/coordinate/body/interface facts. The GPT-6.1 Sol high
worker owned only the new key source and test; the parent owns integration,
source checks, lock, documentation and commit.

```powershell
go test -mod=readonly ./internal/cli -run '^TestCandidateStateGraphCacheKey$' -count=1 -timeout=15m -v
```

The final run PASSes in **105.823s**, **41 cases per Windows profile / 82 native
processes**, with no selected skips. A separate Go framing/SHA-256 implementation
uses actual Go source checking, state-body identity and public/source interfaces
to construct expected bytes. It does not hash Fairy's reported preimage to
create its expectation.

Cold/source-restored keys and same-coordinate owner rebinding agree. Exact
source comments, body operations, schema/interface changes, coordinate, standard
library and all eight environment members invalidate identity. Each environment
field has empty and invalid-UTF-8 refusal controls; invalid standard identities,
ordinary Complete, oversized compatibility input and already-overdrawn graph/
ordinary work are rejected. Exact/one-short work is checked for compute and
the real sealed-environment adapter. That adapter is actually executed using
the running image and bundled SDK, then compared to its explicit snapshot.
Source, coordinate, compatibility and Complete destruction cannot alter the
returned identity.

Initial attempts 1 and 2 fail while parsing oracle fixture module names;
attempt 3 fails on a probe `usize` type mismatch. These are preparation failures,
not product passes. Attempt 4 passes 37 cases/profile; the final fifth run adds
four adapter/work controls and supersedes it. Logs are ignored local
`.cache/graph-cache-key-attempt1.log` through `graph-cache-key-attempt5.log`.

Fresh complete-source `check` and `check --locked` pass **720 source files /
9,004 MIR functions**, empty stderr. All **1,190 authored files** match the
isolated checked copy; only its successful lock is copied back. Snapshot locator:
`.cache/state-graph-key-check-latest.json`. Scoped vet, formatting, diff and
UTF-8/LF/final-newline checks pass.

This is key derivation, not disk restoration, filesystem persistence or ordinary
command selection. Full Go suite, allocator failure injection, native graph
execution, self-build/distribution and hosted acceptance remain unexecuted.
Linux stays batched. Public CLI/SDK/runtime ABI, DB, dependencies, default
limits, repository and runners are unchanged. No push or dispatch.
