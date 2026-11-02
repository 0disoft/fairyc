# Windows Fairy candidate generics: full CLI and disk cache, 2026-11-03

This follows the [source and native receipt](windows-candidate-generics-20261102.md)
for the bounded [candidate generic transport v1](../architecture/candidate-generic-transport-v1.md).
It adds Windows full-CLI and cross-package persistent-cache evidence for the seven
extension scalars. They remain direct transport types; no literal, arithmetic or
complete interpreter-runtime admission is added.

- Host: Windows amd64, Go 1.27.1, MSVC 14.51.36231.
- Full CLI owner: `internal/cli/candidate_generic_cli_test.go`.
- Test-only observer: `internal/cli/candidate_generic_cli_observer_test.go`.
- Persistent-cache owner: `internal/cli/candidate_generic_cli_cache_test.go`.
- Reused host owner: `internal/cli/candidate_copied_closure_cli_host_test.go`.
- Test commits: `9fa4a003` (CLI) and `376c292d` (persistent cache).

## Execution receipt

| Probe | low | optimized |
| --- | --- | --- |
| Full `check/build/run/test` CLI | Passed, 117.01 s | Passed, 34.40 s |
| Generic cross-package persistent cache | Passed, 196.41 s | Passed, 40.87 s |

The times include current-source host preparation and saved-host digest-reuse
checks; they are not performance benchmarks. The completed run covers 48
independent CLI processes, four direct product executions, fourteen test-only C
observer images and 98 exact scalar byte comparisons across seven types.

Both compiler profiles use the unchanged full authored `file_cli` host, with 169
private C parts and a 16 MiB private stack. The original Fairy-authored `file_cli`
main and commands remain present; Go builds the host and its C-only build companion.
Saved host reuse is optional test acceleration through
`FAIRYC_CLOSURE_CLI_ARTIFACTS=.cache/copied-closure-cli`: a saved executable is
reused only when the current profile, C parts, header and support digest and the
image digest match. It is not cache-admission authority.

The CLI test starts ten separate processes per profile. It checks cold `check`,
locked warm `check`, development and release `build`/`run`, JSON test listing,
interpreter tests and development/release native tests. Generated products and
`run` return 42 with empty output; each successful test invocation reports one
passing test. Normal interpreter tests exercise the original scalar constructors,
so they do not prove new seven-type literals, operations or full runtime execution.

The cache test starts fourteen fresh processes per profile. Its provider package
`answer` contains both the public `pick<T>` and private `sink<T>` used by the
importing caller. It checks cold storage and a locked hit with unchanged cache
generation names, bytes and lock bytes; `run` returns 42. Changing the provider
result from first to second rejects the old lock with exit 7 and `cache: skipped`;
an unlocked run returns 43, and the next locked run is a hit returning 43. Both
runs switch the actual seven-scalar observer value. Editing the caller from 42/43
to 44/45 again rejects the lock, then an unlocked run and locked hit return 45.
Corrupting every `.fyc` generation produces cold-storage repair and a warm hit
returning 45. An invalid provider `return true` is rejected by the stale lock,
then cold `check` exits 3 without admitting a cached result; restoring the valid
source returns to a cache hit. The test checks five seven-scalar observer images
per cache profile.

## Observer and host boundaries

Fixtures retain private typed observer functions in C reachability through calls
inside a literal-false branch; recursive source-only seed functions never execute.
The test-only capture proxy saves the actual fresh `--packet-input` bytes and
forwards unchanged arguments and streams to the original C-only companion. It
restores that companion on cleanup.

A separate test-only observer consumes the captured packet, finds each entry by
its exact unique C signature, appends one entry call and invokes all seven scalar
helpers. It compares complete object bytes, including signed zero, `UINT64_MAX`
and signed negatives. The observer uses exact signatures rather than hard-coded
function IDs or public source exports. It is compiled with the low C consumer
profile only and is separate from unmodified product builds, which use development
and release profiles.

The CLI test contributes two observer images per compiler profile. The cache test
contributes five per cache profile. These are test-only native builds, not product
builds or cached test results. Both profiles restored the original C-only companion
after completion; each restored binary was 5,291,520 bytes.

## Reproduction and deferred scope

```powershell
$env:FAIRYC_RUN_PARITY_INTEGRATION = '1'
$env:FAIRYC_CLOSURE_CLI_ARTIFACTS = Join-Path (Get-Location) '.cache/copied-closure-cli'
# Omit FAIRYC_GENERIC_CLI_PROFILE to run both, or select low / optimized.
go test -mod=readonly ./internal/cli -run '^TestCandidateGenericTransportCLI$' -count=1 -timeout 15m -v
go test -mod=readonly ./internal/cli -run '^TestCandidateGenericTransportPersistentCache$' -count=1 -timeout 15m -v
go test -mod=readonly ./internal/compiler -run '^(TestGenericSignature|TestBroaderGenerics)' -count=1
go test -mod=readonly ./internal/workspacecompiler -run '^TestBroaderGenericsScalarCache$' -count=1
go vet -mod=readonly ./internal/cli
git diff --check
```

The compiler reference selection, workspace scalar cache, focused CLI vet and diff
check passed. Windows `test` retains its non-cache command and parser entry; the
complete-cache checks use `check`, `build` and `run`, and no test result is cached.

Native Linux, full `go test ./...`, hosted CI, self-hosting and clean-machine
distribution were not run. This new fixture uses C11 and does not add candidate
C17 validation; the earlier Go repr(C) fixture's C17 result remains separate. The
seven-type interpreter-operation, static-interface and copied-closure slices keep
their earlier boundaries.

No production compiler/runtime, grammar, public CLI/Go API, SDK ABI, DB,
dependency, cache wire format, repository policy, CI or runner changes. No push,
publication, deployment or workflow dispatch.
