# Windows Fairy candidate copied closures: CLI and disk cache, 2026-11-03

This follows the [source/artifact receipt](windows-candidate-copied-closures-20261102.md)
for the bounded [copied closure family](../architecture/candidate-copied-closures-v1.md).
It closes the Windows full-command and fresh-process disk-cache evidence gap;
it does not expand the admitted closure syntax or promote a stable language ABI.

- Host: Windows amd64, Go 1.27.1, MSVC 14.51.36231.
- CLI regression owner: `internal/cli/candidate_copied_closure_cli_test.go`.
- Persistent-cache owner: `internal/cli/candidate_copied_closure_persistent_test.go`.
- Full host owner: `internal/cli/candidate_copied_closure_cli_host_test.go`.
- Test commits: `8f278e33` (full CLI) and `bc841d01` (fresh-process cache).

## Execution receipt

| Probe | low | optimized |
| --- | --- | --- |
| Full `check/build/run/test` CLI | Passed, 260.93 s | Passed, 406.41 s |
| Ordinary and generic cross-package fresh-process cache | Passed, 458.46 s | Passed, 64.51 s |

Both profiles passed: 76 independent CLI processes and four direct executions of
the generated product images. Times include host preparation; they are test-run
wall times, not comparable command/build benchmarks.

The full CLI test runs ten separate CLI processes per compiler profile. It checks
cold `check`, locked warm `check`, development/release `build` and `run`, JSON
test listing, interpreter tests and development/release native tests. The built
closure programs and `run` return 42 with empty program output. Tests execute a
closure whose captured original is subsequently mutated and report one pass.

The disk-cache test runs fourteen separate CLI processes for each ordinary and
generic provider family. It checks cold storage, a locked complete hit with the
same generation names/bytes and lock bytes, and execution returning 42. Changing
the closure body or capture initializer rejects a stale lock with exit 7 and
`cache: skipped`; unlocked recompilation produces 43 or 44 respectively, followed
by another locked complete hit. Corrupting all cache generations forces repair.
A type-invalid closure is rejected with exit 3 rather than admitted from cache;
restoring valid source can reuse its valid generation.

## Host and command boundaries

The host compiles the original Fairy-authored `file_cli` main, including its
`new` command, with 169 private C parts and a 16 MiB private stack reserve.
Direct kernel32 imports used by `new` stay in the C header/native link; no command
is replaced by a stub. `new` itself is retained but not executed in this receipt.
Go builds this host and its C-only build companion; all project loading, semantic
admission, cache restoration and C production under test run in Fairy.

Optional saved host reuse compares a digest of the complete current emitted C,
header, support and profile plus the saved executable digest. Saved host files
are disposable test acceleration, never source or cache-admission authority.
Both compiler-host profiles run development and release product paths. No product
timeout, stack default, cache wire format or parser gate is relaxed.

Windows `test` still uses its existing non-cache command entry. The full command
test proves fresh interpreter/native execution; persistent complete-cache hits
are demonstrated through `check/build/run`. This receipt does not enable Windows
`test --cache-root` or claim that test results are cached.

## Reproduction and deferred scope

```powershell
$env:FAIRYC_RUN_PARITY_INTEGRATION = '1'
$env:FAIRYC_CLOSURE_CLI_ARTIFACTS = Join-Path (Get-Location) '.cache/copied-closure-cli'
# Omit FAIRYC_CLOSURE_CLI_PROFILE to run both, or select low / optimized.
go test -mod=readonly ./internal/cli -run '^TestCandidateCopiedClosure(CLI|PersistentCache)$' -count=1 -timeout 30m -v
go test -mod=readonly ./internal/compiler ./internal/workspacecompiler -run '^TestCopiedClosure' -count=1
go vet -mod=readonly ./internal/cli ./internal/compiler
git diff --check
```

The Go reference closure tests and focused vet passed. The earlier syntax,
identity and restored-C probes retain their separate receipt; they were not rerun
as part of this integration-only change. Native Linux, full `go test ./...`,
hosted CI, self-hosting and clean-machine distribution remain unexecuted. Linux
full CLI and persistent cache replay remain in the combined validation queue.

No production compiler/runtime source, public Go/CLI API, SDK ABI, DB,
dependencies, cache wire version, repository policy, CI or runner configuration
changes. No push, publication, deployment or workflow dispatch.
