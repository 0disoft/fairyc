# Windows Fairy generic inline caller: full CLI and persistent cache, 2026-11-03

This receipt records Windows command and cache evidence for generic callers of the bounded
[cross-package inline family](../architecture/candidate-inline-v1.md). The exports remain
public, safe, nongeneric i32 arithmetic; see the
[ADR](../adr/0493-cross-package-inline-bodies-and-cache-v5.md) and
[nongeneric receipt](windows-candidate-inline-cli-cache-20261103.md).

The later [MSVC inline-trap receipt](windows-candidate-inline-traps-20261103.md) is separate
regression evidence for parameterized inline checked arithmetic.

- Host: Windows amd64, Go 1.27.1, MSVC 14.51.36231.
- Full CLI: `internal/cli/candidate_generic_inline_cli_test.go`.
- Persistent cache: `internal/cli/candidate_generic_inline_cli_cache_test.go`.
- Shared runners: `candidate_inline_cli_test.go` and `candidate_inline_cli_cache_test.go`.
- Reused harness: the existing packet observer, copied host and C-only companion.
- Commits: CLI/Go-reference `92db2f29`; persistent cache `d5cf2d49`.

## Caller graph

The root caller is generic; the dependency is not. `fn i32 evaluate<T>(T marker)`
discards `marker`, executes `answer::mix` through four ordered `pick` calls, checks i32
minimum and division, and returns `answer::add(40)`. `main` calls `evaluate<i32>(1)`
and `evaluate<bool>(true)`, asserts equality, and returns 42; the test executes both.
The picks increment once in order, yielding 1234 and a count of four. The exports
remain bounded pure i32 arithmetic; no generic inline export is admitted.

## Execution receipt

| Probe | low | optimized |
| --- | --- | --- |
| Generic-caller `check/build/run/test` CLI | PASS, 142.98 s | PASS, 48.81 s |
| Generic-caller persistent cache, 26 commands | PASS, 281.72 s | PASS, 64.02 s |

The separate in-process Go reference ran once in 3.32 s and made two calls:
unedited source returned 42; after changing `add` from `x+2` to `x+3`, it returned
43, with empty output and diagnostics.

Each full-CLI profile runs ten independent CLI processes and executes two generated
products directly; each cache profile runs 26 CLI processes. The final acceptance
total is 72 independent Fairy CLI processes, four direct product executions and
two in-process Go reference calls. These are test-run times, not build benchmarks.

## Host and command boundary

The tests reuse the unmodified authored `file_cli` host: 169 current private C parts, its
original main and commands, and a 16 MiB private stack reservation. Reuse requires exact
source and executable-image digest matches over the profile, header, C parts and support
bytes. The original C-only `tools/candidate/fairyc-candidate-build` companion remains the
direct product path. Loading, admission, restoration and C production run in Fairy.

The shared runner extraction accepts a workspace callback; existing nongeneric entrypoints
and their command sequence are unchanged. No test-only product or observer image is added;
the capture proxy forwards the packet, arguments and streams unchanged and restores the
companion during cleanup.

Each CLI profile runs cold and locked warm `check`, development and release `build`/`run`,
JSON test listing, and interpreter, native and native-release tests. Products return 42
with empty output. Windows `test` remains a non-cache entry.

Each development and release build packet in both CLI profiles requires four reachable
inline definitions in the shared header and none in body units. It observes eight
`static inline int32_t fy_fn_` occurrences, four prototypes plus four definitions. A
generic caller moves no body.

## Persistent-cache behavior

Both profiles pass the unchanged 26-command matrix. Cold `check` stores a generation,
locked warm `check` reports `cache: hit`, and `run` returns 42. Changing selected `add`
from `x+2` to `x+3` returns 43, an unused export change invalidates its class, and marker
removal reduces the shared header from four definitions to three; restoration returns four.

Selected-body, unused-export and marker-removal edits may store `cache: mixed` or
`cache: cold stored`, and reject whole-project or all-unit replay. Private `spare`
edits and the `add` rename/grouping/hex spelling change require exact `cache: mixed`.
Corruption heals through a cold store. A private unselected `@inline` annotation is
rejected from source before a valid cache can hide it. Both profiles logged
`cache: mixed` for selected, unused, private, format and removal edits, then
`cache: hit` after marker restoration.

Never infer zero unit reuse from a whole-project v5 miss. Fresh source checking may
reuse current caller IR while C emission consumes the changed owner's body. This is
the [candidate per-package unit cache](../architecture/candidate-package-unit-cache-v0.md)
boundary, distinct from whole-project and all-unit hits.

## Reproduction and deferred scope

```powershell
$env:FAIRYC_RUN_PARITY_INTEGRATION = '1'
$env:FAIRYC_CLOSURE_CLI_ARTIFACTS = Join-Path (Get-Location) '.cache/copied-closure-cli'
# Omit FAIRYC_INLINE_CLI_PROFILE to run both, or select low / optimized.
go test -mod=readonly ./internal/cli -run '^TestCandidateGenericInlineCLI$' -count=1 -timeout 15m -v
go test -mod=readonly ./internal/cli -run '^TestCandidateGenericInlinePersistentCache$' -count=1 -timeout 30m -v
go test -mod=readonly ./internal/cli -run '^TestCandidateGenericInlineCLIReference$' -count=1 -timeout 5m -v
go vet -mod=readonly ./internal/cli
git diff --check
```

Scoped `go vet` passes with the final reference source, and `git diff --check`
passes, as do 92 local Markdown links. Native Linux, full `go test ./...`,
a C17 or new arithmetic-trap matrix, hosted CI, distribution and self-hosting remain
deferred. No production compiler, public API, ABI, DB, dependency, cache wire format,
repository hygiene, CI or runner change is included. No push or deployment is part of
this evidence.
