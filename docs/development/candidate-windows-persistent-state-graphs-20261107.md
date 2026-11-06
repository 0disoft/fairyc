# Fairy Windows persistent state graphs, 2026-11-07

Status: explicit authored one-attempt binding implemented and verified on
Windows low/optimized. Ordinary command selection, cancellation and final
acceptance remain separate.

## Product path

`module_state_graph_windows_persistent` uses the real sealed Windows builder,
runner, admitted response and rooted local-state provider. A caller cannot pass
an arbitrary process result or commit receipt. One snapshot is detached, the
actual C graph runs once, and only its admitted final commit overlay reaches
one layout/version CAS. No transaction or CAS retry exists. Empty commits
still compare, while reject, trap and runtime/process/admission failures do not
publish. Source and scratch buffers can be destroyed without losing returned
read/write/trap bytes.

Storage and execution retain separate cumulative work and explicit attempt
counts. Failure owners retain original typed storage information or owned
runner diagnostics; post-run errors also retain the exact admitted response.
The two retention slots are reserved before I/O. Known successful publication
stays acknowledged even if a defensive receipt consistency check disagrees.
Receipt mismatch injection is not claimed for this sealed concrete provider.

The disjoint worker owns only the authored binding and native test. The
coordinator owns provider integration, complete-source checks, documentation,
lock review and commit. Actual product execution uses Fairy and generated C;
Go supplies independent layout/semantic/local-store comparison and reopen.

## Verification

The final worker command passes **20 scenarios across low/optimized** in
**67.934s**, without selected skips:

```powershell
go test ./internal/cli -run '^TestCandidateStateGraphWindowsPersistent$' -count=1 -timeout=12m -v
```

Six graphs cover empty commit/reject, trap, map, transitive helper and present
borrow/view/copy; each also verifies independent Go semantic expectations,
exact real-file state on Go reopen and preservation of an unmentioned binary
key. Additional cases cover no-op/nonempty MAX version, actual readonly-file
rename refusal and real runtime write-limit refusal. Every invocation also
checks 14 pre-I/O options/work refusals and owned runner request-failure causes.
Final log: `%TEMP%/fairyc-graph-windows-persistent-final-pass.log`.

The actual rename failure takes **1 snapshot / 1 run / 1 CAS**, retains the
observed version/OS operation and admitted response, and leaves old bytes
unchanged. The runtime error retains status 4 and its admitted response with
**1 / 1 / 0** calls. Reject/trap and failed/no-op attempts preserve durable file
bytes; publication advances only for nonempty successful commits. Detached
response values remain usable after the source/asset buffer is discarded.

An earlier **71.657s exit-1** matrix is retained separately in
`%TEMP%/fairyc-graph-windows-persistent-final.log`: a borrow fixture returns
runtime status 9 while the harness expects a successful outcome. The final
success case uses the existing self-seeded present-borrow-copy graph instead.
Nonzero runtime status remains a typed failure; no product policy is weakened.
That earlier matrix is not counted as a pass. The native-tested module SHA256
is `b81d113b30371007cefd40404220e3faac733ec26022863054218134f85fe268`.
Integration normalizes its CRLF to LF and removes the extra EOF blank line;
byte comparison verifies this is the only source difference. The final file
SHA256 is `7e361d478f84235414f51c18122b2cfd7de442e27f58ad9af7b7ef47fec13290`.
The unchanged native matrix is reused after that whitespace-only cleanup.

Fresh isolated `check` then `check --locked` pass in **13.578s / 12.438s**,
**748 files / 9,428 MIR functions**, empty stderr. All **1,218 authored files**
match `.cache/state-store-all-be4652cc`; no pending source is excluded. Only the
successful verified lock is copied back. Record:
`.cache/state-store-all-check.json`; bootstrap SHA256
`4e392248a97b8351c8563c904910fcf88c648a9b4d1323be8cda45dc711eb084`.
Scoped `go vet -mod=readonly ./internal/cli`, gofmt and diff checks pass with
empty output. The earlier complete-source check also passes in 14.072s / 13.575s;
the final check above is refreshed specifically for the EOF/line-ending cleanup.
Final source/lock hashes match the successful isolated copy.

## Remaining work

This closes a bounded explicit Windows graph/local-store binding, not the full
native-state queue item. It does not provide context cancellation, ordinary
workspace CLI selection, other storage backends or chain/VM selection. The
existing runner timeout remains bounded; no public ABI hook is invented to
stand in for cancellation. Busy and version conflicts retain the provider's
one-attempt behavior already exercised by the real provider matrix. Forced
stale/conflicting publication between the graph snapshot and CAS and receipt
inconsistency injection are not exercised by this graph matrix.

Full repository regression, complete current-source Go/Fairy and consecutive
self-build comparison, distribution/clean-machine acceptance and the combined
Linux native batch remain open. Same-commit hosted CI requires an authorized
push or dispatch. Public CLI/SDK/runtime ABI, DB/provider choice, dependencies,
runner and repository policy are unchanged. No push, dispatch, release or deploy.
