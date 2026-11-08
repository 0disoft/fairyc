# Copied closure value CLI cache replay, 2026-11-08

This follows the [command acceptance](fairy-closure-value-cli-20261108.md)
and the first two CLI items in [the handoff](fairy-handoff-20261108.md).
`TestCandidateClosureValuePersistentCache` uses the real Fairy-authored CLI,
ordinary command arguments, a separate provider package and a real disk cache.
The original source returns a copied closure through a factory and ordinary
struct, extracts it, passes it to another function and invokes it after the
factory frame has returned. The original mutable capture is changed before
that return; the invocation must still use its copied snapshot.

## Assertions

- Cold `check` publishes an actual `FYCACHE2` / `FYCKEYS2` project generation.
- Fresh locked check/run processes report whole-project hits, execute the
  current program and preserve every retained generation name/hash and lock
  byte. Snapshot comparisons include corrupt generations, and reject leaked
  unpublished stages.
- A provider body edit changes the native result from 42 to 43. Replacing the
  capture initializer changes it to 44; a caller edit changes it to 45. A
  provider package UID change also invalidates the locked workspace and cache
  identity, while retaining the correct result 45.
- Each edit first fails locked check with status 7 and no cache publication.
  Unlocked run must use a rebuild/mixed route, change the stored generations
  and execute the new result. Subsequent fresh processes must be complete hits.
- All actual `.fyc` generations are truncated to `corrupt`, including any UNIT
  records. Locked check must cold-repair the cache; the resulting complete
  generation and subsequent locked execution must be valid again.
- An invalid main is rejected with status 3 after the stale-lock rejection,
  without changing any retained cache bytes. Restoring the valid source must
  recover a whole-project hit and execute 45.
- Separate non-closure scalar and generic-aggregate projects retain
  `FYCACHE0/FYCKEYS0` and `FYCACHE1/FYCKEYS1`, respectively. Both execute 42
  through development and release native run commands and preserve warm bytes.

There are 27 fresh CLI processes for the closure scenarios and four for each
legacy family, or 35 per producer profile. These are actual process-boundary
checks; in-process Go cache calls do not substitute for them.

## Windows results

| Producer | Result |
| --- | --- |
| low | PASS, 491.82 s including exact-digest host preparation/reuse; 35 CLI processes, scalar 58.10 s and aggregate 58.96 s |
| optimized | PASS, 63.93 s including exact-digest host preparation/reuse; 35 CLI processes, scalar 3.28 s and aggregate 2.93 s |

The first low attempt failed after a correct result 43 because the harness
required the literal word `stored` in the status line. The existing CLI prints
exactly `cache: mixed` for partial UNIT reuse. The final assertion accepts only
the three existing rebuilding status lines (`cold stored`, `mixed`, `units
hit`), still rejects a stale complete hit, still requires changed generation
bytes and still requires the following complete warm hit. No product check,
cache publication requirement or expected runtime value was weakened.

The raw failed attempt is `.cache/closure-cli-cache-low-20261108.log`.
Final logs are `.cache/closure-cli-cache-low-final-20261108.log` and
`.cache/closure-cli-cache-optimized-20261108.log`. Source inspection confirmed
the status spellings in `project_check_cli.fy` and `native_cli.fy` before the
harness expectation was corrected.

Reproduce from the repository root using the existing opt-in gate:

```powershell
$env:GOMAXPROCS='1'
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
$env:FAIRYC_CLOSURE_CLI_ARTIFACTS=Join-Path (Get-Location) '.cache/closure-cli-host-20261108'
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureValuePersistentCache$' -count=1 -timeout=20m -v
```

Select `FAIRYC_CLOSURE_CLI_PROFILE=low` or `optimized` to run one producer.
Host reuse verifies generated header/sources/support/profile and EXE digests;
the companion still consumes fresh C packets from the Fairy CLI.

No compiler or normative contract changes were necessary. `go vet` on
`internal/cli` passed on the final harness. No public API, ABI, DB, dependency,
lockfile, hygiene or CI/runner settings changed. Full-suite, self-host/fixed
point, Linux, distribution and hosted CI checks remain outside this batch.
The older invalid copied-closure Go/Fairy diagnostic/status difference is not
closed by the simple invalid-return rejection used here. No remote action ran.
