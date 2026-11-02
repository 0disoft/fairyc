# Windows Fairy candidate static operations: full CLI and persistent cache, 2026-11-03

This follows the [source/artifact and native-C receipt](windows-candidate-static-operations-20261102.md)
for the bounded [candidate static operation family](../architecture/candidate-static-operations-v1.md).
It records whole-command and fresh-process cache evidence for Windows; it does
not expand the admitted interface syntax, cache wire format or stable ABI.

- Host: Windows amd64, Go 1.27.1, MSVC 14.51.36231.
- Full CLI owner: `internal/cli/candidate_static_operation_cli_test.go`.
- Persistent-cache owner: `internal/cli/candidate_static_operation_cli_cache_test.go`.
- Reused packet observer: `internal/cli/candidate_generic_cli_observer_test.go`.
- Reused host owner: `internal/cli/candidate_copied_closure_cli_host_test.go`.
- Test commits: `2579f375` (full CLI), `a1a6765f` (persistent cache).

## Execution receipt

| Probe | low | optimized |
| --- | --- | --- |
| Full `check/build/run/test` CLI | Passed, 137.22 s | Passed, 50.40 s |
| Static-operation cross-package persistent cache | Passed, 314.00 s | Passed, 86.41 s |

Times include current-source host preparation and digest-reuse checks; they are
test-run wall times, not command or build benchmarks. Both CLI profiles pass ten
separate processes each. Each cache profile passes exactly 23 separate processes
and four test-only observer images. The complete receipt is 66 CLI processes,
four direct product executions, twelve observer images and 84 scalar byte
comparisons across seven extended types.

An initial optimized-cache attempt failed because the test expected
`cache: mixed stored`. The existing CLI contract returns exactly `cache: mixed`;
only that test expectation was corrected before the successful 86.41 s rerun.
Product behavior was unchanged.

## Host and command boundary

Both profiles reuse the unmodified authored `file_cli` host: 169 private C parts,
its original main and commands, and a 16 MiB private stack. Go builds that host
and its C-only build companion. Project loading, semantic admission, cache
restoration and C production under test run in Fairy; the Go frontend is not a
workload fallback. Saved-host reuse remains optional test acceleration and is
accepted only when the current profile, C parts, header, support digest and image
digest match.

Each CLI profile runs cold `check`, locked warm `check`, development and release
`build`/`run`, JSON test listing, interpreter tests, and development/release
native tests. Generated products and `run` return 42 with empty program output;
each successful test invocation reports one passing test. Normal interpreter tests
exercise i32, i64, u8, u32, usize and bool operations plus zero/four parameters.
The test-only observer covers the seven extended provider types.

The tests reuse `installGenericCLIPacketCapture` and `observeGenericCLIPacket`.
The proxy saves the actual packet handed to the original C-only companion, forwards
the original arguments and streams, and restores the companion during cleanup.
The separate observer modifies the private C entry to call seven extended scalar
wrappers with real boundary values and compares complete object bytes, including
negative zero and `UINT64_MAX`. It is compiled with the low C consumer profile
only. Product development and release builds remain unmodified. This is not a new
C17 check.

Literal-false recursive source seeds retain the seven wrapper specializations in
the reachable packet but never execute them as Fairy source. All thirteen provider
types are admitted and participate in the same graph. The ordinary program/tests
execute the original six scalar families and zero/four-argument operations; the
observer specializes the other seven. No source literal, arithmetic or full
seven-scalar interpreter-runtime admission is added.

## Persistent-cache behavior

The cross-package workspace imports the public operation from `answer`. The cold
store and locked warm check retain identical generation names, bytes and lock
bytes; locked `run` returns 42. After that, both profiles verify these
paths:

- Changing selected i32 and all seven extended `pick` providers to return their
  second argument rejects the old lock with exit 7 and `cache: skipped`. An
  unlocked run returns 43; the next locked run is a complete hit, and C
  observation verifies 43 from all seven extended providers.
- Editing the unused public `seed<i64>` provider from 64 to 65 also rejects the
  old lock. The observed unlocked status is `cold` or `mixed`, rather than a
  complete hit, and does not report all units hit. The run and warm hit return
  43, proving dependent identity changes even when that provider is not selected
  by runtime code.
- Editing unrelated private `spare()` from 8 to 9 rejects the old lock, then the
  unlocked run reports exactly `cache: mixed` and returns 43. The owner unit is
  rebuilt while unchanged units are reused; the next locked run is a hit.
- Corruption of every `.fyc` generation causes cold-storage repair, followed by
  a warm locked hit returning 43.
- Removing the i32 provider tuple is rejected as E4502 at the original `main.fy`
  root. A wrong i32/bool signature and a duplicate i32 tuple are rejected as
  E4501. Each control first rejects the stale lock with exit 7, then unlocked
  `check` exits 3 with `cache: cold uncached`. Restoring valid source returns to a
  cache hit.

Windows `test` retains its non-cache command entry. Only `check`, `build` and `run`
exercise the complete cache, and no test result is stored or replayed.

## Reproduction and deferred scope

```powershell
$env:FAIRYC_RUN_PARITY_INTEGRATION = '1'
$env:FAIRYC_CLOSURE_CLI_ARTIFACTS = Join-Path (Get-Location) '.cache/copied-closure-cli'
# Omit FAIRYC_STATIC_CLI_PROFILE to run both, or select low / optimized.
go test -mod=readonly ./internal/cli -run '^TestCandidateStaticOperationCLI$' -count=1 -timeout 15m -v
go test -mod=readonly ./internal/cli -run '^TestCandidateStaticOperationPersistentCache$' -count=1 -timeout 30m -v
go test -mod=readonly ./internal/compiler ./internal/workspacecompiler -run '^TestStaticInterface' -count=1
go vet -mod=readonly ./internal/cli
git diff --check
```

The Go compiler/workspace static-interface tests passed in 2.167 s and 0.619 s;
scoped vet and the diff check passed. Native Linux remains queued. Full
`go test ./...`, hosted CI, self-hosting and clean-machine distribution were not
run. The source/artifact receipt remains the owner of its earlier native and
identity results; this follow-up repeats them only as reused host/cache behavior.

No production compiler/runtime, grammar, public CLI/Go API, SDK ABI, DB,
dependency, cache wire format, repository policy, CI or runner change is made. No
push, publication, deployment or workflow dispatch is part of this evidence.
