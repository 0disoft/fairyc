# Copied closure locals: Windows full CLI and disk cache, 2026-11-07

This integration-only slice follows ADR 0546 and
[the source/artifact receipt](copied-closure-locals-20261107.md).
The checkout started at `0bb199e7` on `main`. The dedicated owner is
`internal/cli/candidate_copied_closure_locals_cli_cache_test.go`.
No production Fairy source, source contract, queue, lockfile, dependency,
public API/ABI, DB, runner or CI configuration is changed.

## What the tests exercise

`TestCandidateCopiedClosureLocalsCLI` retains the actual authored `file_cli`
main through the existing `copiedClosureCLIHost` helper. The helper uses the
explicit private compiler-host builder (ADR 0510), including its separate 8 MiB
header allowance and 16 MiB private stack reserve. Ordinary product bounds,
warning flags, native command deadlines and packet validation are unchanged.
The Go companion consumes Fairy-produced C; it does not replace the Fairy
frontend, command dispatcher or disk-cache restoration.

Each low/optimized host runs ten fresh CLI processes: cold and locked warm
`check`, development/release `build` and `run`, JSON test listing, interpreter
tests, and development/release native tests. The two built images are also
executed directly. The main source reuses the new immutable-locals program;
the test source independently exercises typed/inferred/transitive locals,
snapshot timing, repeated calls, outer-name hygiene, creation without executing
an unused trapping initializer, and the admitted 16-local/64-node boundaries.
The main program includes generic instantiations and arithmetic coverage.

`TestCandidateCopiedClosureLocalsPersistentCache` runs ordinary and generic
cross-package providers in separate temporary workspaces. The generic consumer
uses both i32 and i64 specializations. Every invocation is a new OS process.
Each provider covers:

- Cold storage, then locked complete hits executing the expected return value.
- Equal-width initializer and result edits changing 42 to 38 and 43.
- Unused initializer value/order edits retaining 42 but changing stored
  generation identity; stale locks fail with exit 7 and `cache: skipped`.
- Unlocked recompilation followed by release execution from a locked complete
  hit. Locked checks/runs preserve generation names, generation bytes and lock.
- Corrupting all stored generations, then repairing from valid locked source.
- Mutable and unused-bool locals, 17 locals and 65 shared expression nodes
  refusing at package-input admission instead of reusing or altering valid cache
  generations. Restored valid source runs from a locked complete hit again.

`TestCandidateCopiedClosureLocalsCLIFixtures` uses the Go frontend and interpreter
to independently admit the generated accepted sources, reject the four invalid
families, and run the authored test fixture before native compiler work.

Windows `test` keeps its existing command entry without `--cache-root`.
Persistent complete-cache behavior is exercised through `check/build/run`;
these tests do not claim cached test results.

## Commands and observations

Host: Windows amd64, Go 1.27.1. Native verification is bounded by
`GOMAXPROCS=1`; subtests are sequential and use no `t.Parallel`.
The existing compiler-host worker selection is at most two CL processes and
also respects GOMAXPROCS. Logs and optional digest-checked host reuse are kept
under `.cache/copied-closure-locals-cli-cache-20261107/`, separate from the
coordinator's Atlas artifacts.

```powershell
$env:GOMAXPROCS='1'
go test -mod=readonly -count=1 -timeout 2m -v ./internal/cli -run '^TestCandidateCopiedClosureLocalsCLIFixtures$'
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
$env:FAIRYC_CLOSURE_CLI_ARTIFACTS=Join-Path (Get-Location) '.cache/copied-closure-locals-cli-cache-20261107/hosts'
# Leave FAIRYC_CLOSURE_CLI_PROFILE unset for low and optimized.
go test -mod=readonly -count=1 -timeout 30m -v ./internal/cli -run '^TestCandidateCopiedClosureLocals(CLI|PersistentCache)$'
```

The fixture check passes: test 0.29 s, package 0.452 s; shell wall time
33.5196542 s including the initial Go test build.
Log: `.cache/copied-closure-locals-cli-cache-20261107/fixtures-attempt1.log`.

Native attempt 1 started at `2026-11-07T22:07:34.6367463+09:00`.
Full CLI passes: low 201.84 s, optimized 263.77 s, total 465.61 s.
That is 20 independent CLI processes and four direct built-image executions.
The attempt then fails in the low/ordinary persistent-cache harness after a
successful corruption-repair `check`: the reused strict snapshot helper rejects
an older corrupt generation that is not selected by that command. It is not a
product repair failure. `project_cache_disk.fy` explicitly skips corrupt reads
and repairs a colliding selected generation; it does not promise to repair all
historical files. The fixture had deliberately corrupted every generation.

The old owned test process and its child were stopped after this concrete
failure, before repeating the same defect through all four combinations. The
coordinator's Atlas process was untouched. The failing aggregate attempt remains
failed: package 887.362 s, shell 892.287713 s, exit 1, ending
`2026-11-07T22:22:26.8577722+09:00`. Its successful full CLI subtests remain
separate reusable observations. Log:
`.cache/copied-closure-locals-cli-cache-20261107/native-attempt1.log`.

The dedicated snapshot helper now requires at least one normally framed
generation and hashes every project-generation file, including old corrupt
ones, for exact name/byte stability. Initial cold storage still uses the strict
existing validator. Post-repair locked complete hits and actual return values
remain mandatory. No record is removed or silently excluded to make the test
pass, and product cache admission/repair is unchanged.

After the harness fix, `TestCandidateCopiedClosureLocalsCLIFixtures` passes
again: test 0.25 s, package 0.349 s, shell 33.663501 s including Go compilation
after concurrent compiler edits. Log: `fixtures-attempt2.log` in the log directory.
Attempt 2 low/ordinary passes the previously failing repair assertion, then
observes a diagnostic parity difference: a mutable local refuses in Fairy
package-input admission with exit 7, `cache: skipped` and
`fairyc: package workspace failure: libs/answer/answer.fy: invalid package input`.
The independent Go `check` returns source diagnostics (exit 3) for this fixture.
The assertion now records the observed Fairy package-input refusal, empty stdout and
unchanged valid cache generation names/bytes; it does not accept either outcome
indiscriminately. Changing the expectation establishes rejection/no publication;
it does not fix or establish Go/Fairy exit-code or diagnostic parity. No owning
contract authorizing this difference was established in this slice; the parent
is inspecting that contract. This remains a CLI diagnostic parity gap unless
the owning contract establishes intentional command-specific behavior. Product
diagnostic and cache policy are unchanged.
That attempt remains failed: test 259.90 s, package 260.093 s, shell
264.3049856 s, ending `2026-11-07T22:29:03.1325625+09:00`.
Log: `native-attempt2-low-ordinary.log`.

Attempt 3 low/ordinary passes all 38 independent CLI processes: family
345.25 s, parent test 373.85 s, package 374.018 s, shell 404.711153 s,
ending `2026-11-07T22:37:23.7558586+09:00`. The host is reused only after exact
generated-C/image digest checks. Log: `native-attempt3-low-ordinary.log`.
It passes repair then locked check/release execution (42), all four observed
package-input refusals, unchanged cache bytes and restored locked execution.

The remaining low/generic and optimized ordinary/generic matrix runs started
at `2026-11-07T22:37:52+09:00`. Two independent Go processes each retain
`GOMAXPROCS=1`; together they permit at most two CL processes. Profile host paths
and log files are disjoint. Tests within each process remain sequential.
The coordinator authorized this bounded overlap on the 12-logical-CPU/32-GiB
host. This changes no product command limit or compilation policy.

Attempt 5 optimized passes both families, 76 independent CLI processes:
ordinary 30.34 s, generic 38.32 s, parent test 99.90 s, package 100.248 s,
shell 104.957829 s, ending `2026-11-07T22:39:37.5156669+09:00`.
Log: `native-attempt5-optimized.log`. The exact digest check reuses the original
optimized CLI image; no new optimized compiler host is built for this replay.
Attempt 4 low/generic also passes all 38 independent CLI processes: family
364.34 s, parent test 394.85 s, package 395.032 s, shell 399.7419783 s,
ending `2026-11-07T22:44:32.2616856+09:00`.
Log: `native-attempt4-low-generic.log`.

The completed scoped matrix is 172 successful independent CLI invocations
plus four direct built-image executions: 20 full CLI invocations and 152
persistent-cache invocations across low/optimized and ordinary/generic.
Failed/aborted attempts are retained separately and are not added to that count.

| Passed observation | Go package time | Shell wall time | Log under the evidence directory |
| --- | --- | --- | --- |
| Full CLI low + optimized | Subtest parent 465.61 s; aggregate attempt later fails | Included in failed attempt 1 | `native-attempt1.log` |
| Persistent low ordinary | 374.018 s | 404.711153 s | `native-attempt3-low-ordinary.log` |
| Persistent low generic | 395.032 s | 399.7419783 s | `native-attempt4-low-generic.log` |
| Persistent optimized ordinary + generic | 100.248 s | 104.957829 s | `native-attempt5-optimized.log` |

These timings include different host preparation/reuse and bounded overlap;
they are execution receipts, not comparable performance benchmarks.

```powershell
$env:FAIRYC_CLOSURE_CLI_PROFILE='low'
go test -mod=readonly -count=1 -timeout 15m -v ./internal/cli -run '^TestCandidateCopiedClosureLocalsPersistentCache$/^low$/^ordinary$'
# native-attempt3-low-ordinary.log; attempt2 retained separately as a failure.
go test -mod=readonly -count=1 -timeout 15m -v ./internal/cli -run '^TestCandidateCopiedClosureLocalsPersistentCache$/^low$/^generic$'
# native-attempt4-low-generic.log
$env:FAIRYC_CLOSURE_CLI_PROFILE='optimized'
go test -mod=readonly -count=1 -timeout 20m -v ./internal/cli -run '^TestCandidateCopiedClosureLocalsPersistentCache$'
# native-attempt5-optimized.log
```

No full CLI repeat is needed because those assertions and fixtures did not change.

The focused `go vet -mod=readonly ./internal/cli` passes in 6.7904697 s;
`.cache/copied-closure-locals-cli-cache-20261107/vet.log` retains its result.
The dedicated Go file is gofmt-clean and the scoped diff check reports no
whitespace errors (0.5318347 s, `hygiene.log` in the same directory).
After both harness corrections, the final scoped vet also passes in 7.739207 s
(`vet-final.log`).

## Authenticated host identities and concurrent work

Other workers changed `module_types.fy`, `generic_signature_diagnostics.fy`
and Go compiler/C backend files for flat Option transport during this run.
The initial CLI tests completed on their previously copied source. The
persistent-cache test subsequently copied fresh source and correctly rebuilt
its host when the generated-C digest changed. This receipt establishes the
observed commands on authenticated generated-C hosts; it does not establish a
clean single-HEAD build or acceptance of the complete final compiler checkout.
No unrelated edits are included in this worker's change.
At final handoff the parent has moved `main` to `250e6f97`; this worker has
made no commit. Only its two dedicated files are added by this slice.

The helper's `host.json` `Source` field hashes profile name, emitted header,
runtime support and all emitted units; `Image` is SHA-256 of the executable.
The initial CLI receipts were copied before persistent-host replacement to
`cli-low-host.json` and `cli-optimized-host.json` under the log directory.
Persistent receipts are retained as `persistent-low-host.json` and
`persistent-optimized-host.json`; optimized reuses the same source/image pair
as its first CLI replay.
Both first CLI images used 205 C units. The hashes below retain their exact
identities even when the reusable image path is overwritten later.

| Observation | Generated-C/profile SHA-256 | Executable SHA-256 |
| --- | --- | --- |
| CLI low | `378715f3ba766614cd715f2e29b832c9e1f74485abbd4d6f9133326c1d05c434` | `3a21543cbbb1a59c5359bd63363d1b72cff080e735ae4dcc22be3e3ac37b1e70` |
| CLI + persistent optimized | `7743161be3093332a56258f6206d5404a00c8b100138d6aebc076e884bde648c` | `c775f0a5928f8b0e7e312541d9bc762ecfe59f5beb1032600b019b28dea4cf9a` |
| Persistent low (failed + corrected replay) | `429e846711cc30a5f914b28da12681084b7f3c22e8b453f28e7fea9771f61da9` | `5b3210e6f9ba605ef6b519bc8165c9493ebbce01b6e56499b37e0a58772125c9` |

## Scope and remaining gates

This is Windows integration evidence only. Full selfbuild/fixed-point checks,
native Linux (batched by the coordinator), full Go suite, distribution testing,
hosted CI and allocator-failure injection are not run by this slice. The
separate source/artifact pipeline owns native initializer-trap provenance;
the new full CLI fixture checks that unused closure creation does not trap.
No push, publication, deployment or workflow dispatch. This worker makes no
commit; the coordinator owns review and the eventual small local commit.
The Go exit-3 versus Fairy exit-7 invalid-closure diagnostic gap remains visible
above. Queue edits are outside this worker's write scope; the coordinator owns
any queue entry and subsequent production fix or contract decision.
