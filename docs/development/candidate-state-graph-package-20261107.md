# State graph current package identity and interface prerequisites, 2026-11-07

Two disjoint GPT-6.1 Sol high workers implemented current-source body identity
and explicit seven-codec public/source interfaces. The parent reviewed their
changes. This receipt closes these prerequisites, not Complete/cache/native
execution or final acceptance.

The source producer now exposes an immutable Parsed borrow. The identity
adapter derives the actual root descriptor, canonical effects and selected
package coordinate, owning the resulting context/preimage/hash. The interface
profile preserves existing five-codec APIs and adds explicit graph entry points.
Both paths independently use existing Go production APIs as their oracle.

Focused suite reproduction commands (worker results below):

```powershell
go test -mod=readonly -count=1 -timeout 5m -v ./internal/cli -run '^TestCandidateStateGraphSourceIdentity(SealedOwner)?$'
go test -mod=readonly -count=1 -timeout 5m -v ./internal/cli -run '^TestCandidateStateGraphInterface$'
go test -mod=readonly -count=1 ./internal/moduleinterface
```

Source identity passes 46 cases per Windows profile / 92 native processes and
seven constructor/immutability refusals (37.633s package). It covers real source
renames, helpers/branches/borrows, owner rebinding, coordinate drift and malformed
selected rows, caller lifetime and exact/one-short/overdrawn work. Interface
encoding passes 21 cases per profile / 42 native processes, with exact Go public
and source bytes/hashes, bytes/text/map layouts, ordinary refusal, owner isolation
and work boundaries. The worker also reran TestCandidateStateGraphLayouts and
the Go moduleinterface package successfully. TestCandidateOriginalStateLayouts
was explicitly skipped by its existing opt-in gate; it is not a native pass.

A fresh current Go bootstrap checks the staged-input example independently of
the parent's unfinished Complete integration: `check` and `check --locked` pass
717 source files / 8,932 MIR functions. Only that successful lock is copied back.
Records: `.cache/state-package-staged-1d9aa43e/check-results.json`. Scoped vet,
formatting, UTF-8/LF/final-newline and diff checks pass. Native Linux and allocator
failure injection remain deferred. No public CLI/SDK ABI, DB, dependencies,
runner or default-limit changes. No push, CI dispatch, release or deployment.
