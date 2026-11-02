# Windows Fairy candidate inline: full CLI and persistent cache, 2026-11-03

This follows the [source/artifact and native-C receipt](windows-candidate-inline-20261102.md)
for the bounded [candidate inline family](../architecture/candidate-inline-v1.md).
It records whole-command and fresh-process cache evidence for Windows; it does
not expand the admitted body contract, cache wire format or stable ABI.

- Host: Windows amd64, Go 1.27.1, MSVC 14.51.36231.
- Full CLI owner: `internal/cli/candidate_inline_cli_test.go`.
- Persistent-cache owner: `internal/cli/candidate_inline_cli_cache_test.go`.
- Reused packet observer: `internal/cli/candidate_generic_cli_observer_test.go`.
- Reused host owner: `internal/cli/candidate_copied_closure_cli_host_test.go`.
- Full CLI test commit: `c8e37a7b`.
- Persistent-cache test commit: `53a472d3`.

## Execution receipt

| Probe | low | optimized |
| --- | --- | --- |
| Full `check/build/run/test` CLI | Passed, 166.69 s | Passed, 57.91 s |
| Nongeneric cross-package persistent cache | Passed, 353.63 s | Passed, 67.54 s |

Times include current-source host preparation and digest-reuse checks; they are
test-run wall times, not command or build benchmarks. Each full-CLI profile runs
ten separate CLI processes, executes two generated products directly and
performs two packet inspections.

The final test rejects a whole-project hit and an all-unit hit for selected-body,
unused-export and marker-removal edits, while accepting exact `cache: mixed` or
`cache: cold stored`. A test-only overstrict assumption that required cold
storage failed the optimized selected edit in 32.49 s because the actual path
was `cache: mixed`. That is the documented per-unit behavior: the changed class
rejects whole-project replay, but current-source rechecked caller IR may remain
reusable. Only lowering is skipped, and the changed owner supplies the current
inline body during fresh C emission. The corrected optimized replay then passed
all 26 commands in 67.54 s; the final low replay passed in 353.63 s. The assertion
correction changed no product behavior.

The final accepted runs total 72 independent CLI processes and four direct
product executions. Superseded exploratory and overstrict runs are excluded
from these acceptance totals.

## Host and command boundary

Both tests reuse the unmodified authored `file_cli` host: 169 current private C
parts, its original main and command set, and a 16 MiB private stack reservation.
Go builds that host and the original C-only
`tools/candidate/fairyc-candidate-build` companion. Project loading, semantic
admission, cache restoration and C production under test run in Fairy; the Go
frontend is not a workload fallback. Saved-host reuse requires a matching source
digest over the profile, header, C parts and support bytes plus the saved
executable image digest.

The capture proxy saves the packet handed to the original companion, forwards
the exact bytes, arguments and standard streams, and restores the companion
during cleanup. It is test harness only; product images and command behavior are
not modified. This receipt adds no test-only product change.

The root package is nongeneric and imports the nongeneric `answer` dependency.
Each full-CLI profile runs cold `check`, locked warm `check`, development and
release `build`/`run`, JSON test listing, and interpreter, native and
native-release tests. Generated products and `run` return 42 with empty output;
each test invocation reports one passing test. The tests assert once-only
ordered arguments through four `pick` calls, the i32 minimum, division and the
42 result. The product path covers these assertions on the success path; this
receipt does not add a native arithmetic-trap matrix.

Each build packet check currently finds exactly eight
`static inline int32_t fy_fn_` occurrences in the shared header, representing
four reachable inline prototypes and four definitions, and no inline definition
in a body unit. The same check logs one body source unit. The definition count
is asserted; the single body-unit count is observed from the decoded packet.

## Persistent-cache behavior

The cache test contains 26 commands per profile and runs a nongeneric caller with
the nongeneric inline dependency. Both profiles pass all 26 commands. The whole-project v5 key and per-unit reuse are separate:
a class change rejects the whole-project generation, but current-source
rechecked caller IR can still be reused. The test rejects whole-project and
all-unit hits; it does not require zero caller-unit reuse. This follows the
[per-package unit cache](../architecture/candidate-package-unit-cache-v0.md)
boundary. The script covers these paths:

- Cold `check` stores a generation; locked warm `check` must retain identical
  generation names/bytes and lock bytes; locked `run` returns 42.
- Changing the selected `add` body from `x+2` to `x+3` changes the inline class
  and must report exact `cache: mixed` or `cache: cold stored`. The unlocked run
  returns 43 and the next locked run is a complete hit.
- Editing only the unused inline export from 64 to 65 must also report exact
  `cache: mixed` or `cache: cold stored`. This conservative whole-project
  invalidation occurs even though runtime code never calls that export.
- Changing private `spare` from 8 to 9 preserves the inline classes and must
  report exact `cache: mixed`, reusing unchanged units while rebuilding the
  owner. The run and next locked hit return 43.
- Renaming the `add` parameter, adding grouping and spelling 3 as `0x3` preserves
  the typed inline identity and must also report exact `cache: mixed`.
- Removing `@inline` from `add` must report exact `cache: mixed` or
  `cache: cold stored` and reduce the shared header to three definitions, or six
  `static inline int32_t fy_fn_` occurrences. Restoring the marker returns four
  definitions, eight occurrences, and the next locked run is a hit.
- Overwriting every cache generation with corrupt bytes must heal through a cold
  store, followed by a locked hit returning 43 and a valid packet.
- Adding `@inline` to private, unselected `spare` must not be hidden by a valid
  cache. The stale locked check exits 7 with `cache: skipped`; unlocked `check`
  exits 3 with `cache: cold uncached` and a diagnostic at `answer.fy`. Restoring
  the source returns to a cache hit.

Both final runs observed exact `cache: mixed` for selected, unused,
private, equivalent-spelling and marker-removal edits, and `cache: hit` after
marker restoration. The test also permits `cache: cold stored` for a changed
inline class, but both observed paths retained valid caller IR.

Windows `test` retains its non-cache command entry. Complete-cache reuse is
exercised through `check`, `build` and `run` only; no test result is stored or
replayed.

## Reproduction and deferred scope

```powershell
$env:FAIRYC_RUN_PARITY_INTEGRATION = '1'
$env:FAIRYC_CLOSURE_CLI_ARTIFACTS = Join-Path (Get-Location) '.cache/copied-closure-cli'
# Omit FAIRYC_INLINE_CLI_PROFILE to run both, or select low / optimized.
go test -mod=readonly ./internal/cli -run '^TestCandidateInlineCLI$' -count=1 -timeout 15m -v
go test -mod=readonly ./internal/cli -run '^TestCandidateInlinePersistentCache$' -count=1 -timeout 30m -v
go test -mod=readonly ./internal/compiler ./internal/workspacecompiler -run '^(TestCrossPackageInline|TestInlineCache)' -count=1 -timeout 5m
go vet -mod=readonly ./internal/cli
git diff --check
```

The reference Go compiler and workspace inline tests passed in 0.130 s and
3.427 s. Scoped vet and the diff check passed. Native Linux/GNU host replay, full
`go test ./...`, hosted CI, standalone distribution and self-hosting were not
run.

This receipt makes no speedup, C17, public or stable C ABI, production compiler
change or target-promotion claim. It does not newly claim generic-wrapper CLI or
cache acceptance; the earlier generic key parity remains historical evidence.
No public CLI/Go API, SDK ABI, DB, dependency, cache wire format, repository
hygiene or CI/runner change is introduced. No push, publication, deployment or
workflow dispatch is part of this evidence.
