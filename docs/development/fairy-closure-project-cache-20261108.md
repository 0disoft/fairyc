# ADR0551 whole-project persistent cache, 2026-11-08

This slice extends the existing project lookup, complete key ledger, transport
and artifact/disk operations. It reuses the current-source Complete restorer and
the shared content-addressed disk store. No second cache implementation is added.

## Families and API

| Selected family | Route domain suffix | Envelope | Ledger | Project member |
| --- | --- | --- | --- | --- |
| Scalar | `00 00` | `FYCACHE0` | `FYCKEYS0` | `FYPRJ000` |
| Aggregate | `00 01` | `FYCACHE1` | `FYCKEYS1` | `FYPRJ001` |
| Closure | `00 02`, then aggregate u64 | `FYCACHE2` | `FYCKEYS2` | `FYPRJ002` |

The domain prefix remains `fairyc.candidate-project-lookup`; route filenames
retain `fylookup0-sha256-`. The closure route and ledger each encode the explicit
aggregate bit as canonical big-endian u64, exactly 0 or 1. Existing scalar and
aggregate encoder paths emit their previous bytes without adding the bit.

`project_cache_lookup.collect_closures(loaded,env,aggregates,work)` derives the
route from the resolved current graph and actual compiler environment.
`project_cache_keys.collect_closures(loaded,complete,env,aggregates,work)` requires
the checked Complete's closure and aggregate modes to match, then derives all
package keys atomically from authenticated source provenance and interfaces.
Legacy collectors reject a closure Complete. No supplied key hash or raw marker
can replace these facts.

The artifact module exposes these explicit entries (all work parameters are
mutable references):

```text
from_disk_closures(store,loaded,env,bool aggregates,DiskWork work)
to_disk_closures(store,loaded,complete,env,bool aggregates,DiskWork work)
encode_closures(loaded,complete,env,bool aggregates,key_work,artifact_work)
restore_closures(raw,loaded,env,bool aggregates,projection_work,source_work,
                 artifact_work,link_work,key_work)
```

The wire module exposes `preflight_closures(raw,aggregates,work)`,
`encode_keys_closures(keys,work)`, `matches_closures(raw,keys,work)` and
`encode_closures(keys,program,work)`. Existing wire encoders/matchers refuse
closure keys. Preflight requires complete framing, exact markers, a canonical
matching aggregate bit and no trailing data. In particular, `FYPUNIT2` and
`FYPROG02` are not project-ledger/project-member markers even though both UNIT
and project transport use `FYCACHE2`.

## Admission and disk behavior

Restore uses `decode_closures` followed by
`core::restore_complete_closure_sources(saved,current_sources,current_owners,
current_imports,current_coordinates,aggregates,source_work,artifact_work,
link_work)`. The resulting verified Complete produces a fresh key ledger which
must match the saved ledger exactly. Current-source recipe/target/body checks
and interfaces retain authority. Restore never cold-lowers replacement IR.

The existing disk traversal, bounds, content hashing, collision repair and
publication logic are shared. Every failed candidate preserves accumulated work;
resource failures stop traversal. Closure semantic mismatches remain disposable
misses without deleting a potentially compatible retained generation. Legacy
families retain their existing rejection policy. Raw corruption is detected by
the store's content hash; cold publication repairs that content-addressed file.
There is no speed claim or measured performance result.

Parent integration owns `project_compiler_cached.build`: current-source
`command::closures` selects these named disk entries independently of the
aggregate bit. Parent also owns mixed UNIT compilation/publication and the
Complete C adapters. The fixture imports and calls that real cached wrapper;
it is not limited to a detached restore alternative. No parent-owned source is
edited by this slice.

## Authored acceptance

`examples/lexer_candidate/tests/closure_project_cache.fy` and
`internal/cli/candidate_closure_project_cache_test.go` cover:

- Real manifest/workspace loading, actual running compiler identity, two source
  modules and closure helpers, returned factory/local/struct copies.
- Separate scalar, aggregate and closure routes, opposite aggregate-bit refusal,
  legacy API refusal, exact project versus UNIT markers, canonical feature bits,
  ledger tampering, changed source, truncated/trailing bytes and exhausted work.
- Independent cold compilation compared byte for byte with every admitted
  project artifact. Warm restoration requires `ir::work == 0` and nonzero
  current-source/artifact/link/key histories.
- Real cached-wrapper cold publication (`observed.path == 1`, `stored == true`,
  miss phase/code 2/2, no store error) and warm Complete hit (`path == 2`, no
  lookup/store error or repeated publication, artifact IR work origin).
- Fresh process replays, corruption followed by exact repair, changed closure
  result 42 to 43, invalid-source nonpublication, source reversion, artifact and
  I/O exhaustion without publication, and combined closure/aggregate mode.
- Low and optimized producer profiles, each replay in a new process, with every
  successful emitted Complete C packet built/run under both consumer profiles.
  The generated producer uses strict existing warning policy, 60-second native
  command bounds and a 16MiB stack reserve.

Native is authored for parent execution only after all parallel source changes
are frozen. The authored schedule has 16 producer replays per profile (one
invalid-source replay without output), and 30 generated consumer executions per
profile. These counts are planned coverage, not executed passes.

## Validation and remaining boundary

Worker Source-only command:

```powershell
$env:GOMAXPROCS='1'
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureProjectCacheSource$' -count=1 -v -timeout=60s
```

The first Source attempt failed with `missing-visible-module` (3.66s); the
temporary workspace import roots omitted `module_project_complete_c`. After
adding that copy dependency, the next attempt found an E1006 error in the
fixture (12.44s): its `loaded` parameter shadowed the helper of the same name.
Renaming that parameter fixed the error; no product gate or assertion was
relaxed. Final Source passed in 23.04s (package 23.317s), with the real wrapper,
its observation assertions and the complete authored fixture in scope.
Scoped `gofmt -l` and `git diff --check` also passed. `gofmt` applies only to the
new Go test. No Native execution occurred in this worker.

Parent Native command, intentionally not run by this worker:

```powershell
$env:GOMAXPROCS='1'
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureProjectCacheNative$' -count=1 -v -timeout=20m
```

Source checking does not execute fixture assertions or establish native cache
hits, corruption recovery, generated-C results, or performance. The parent run
below supplies the named Native evidence. Independent legacy scalar/aggregate
regression, Linux, full CLI/self-host/distribution and hosted CI remain separate.

No normative source-of-truth, public CLI/SDK/ABI, DB, dependency, lockfile,
repository hygiene, warning/time/stack limit, workflow or runner configuration
was changed by this slice. No stage, commit, remote write, Linux or CI run was
performed.

## Parent Windows acceptance

After the worker source freeze, the exact Native command above passed:

```text
--- PASS: TestCandidateClosureProjectCacheNative (518.19s)
    --- PASS: TestCandidateClosureProjectCacheNative/low (295.01s)
    --- PASS: TestCandidateClosureProjectCacheNative/optimized (187.53s)
ok github.com/0disoft/fairyc/internal/cli 518.368s
```

Both producer profiles completed all 16 fresh-process replays, including one
invalid-source rejection per profile. The 30 successful project packets ran
under both consumer profiles: 60 generated-C executions with the expected 42
or 43 and empty stdout/stderr. This includes real cached-wrapper cold/warm
selection, corruption repair, source generations and aggregate/closure mode.
Warm project restore retained zero replacement IR work. The sealed Complete C
publisher now chooses its explicit closure adapter; legacy raw C APIs retain
their old admission gates. Log: `.cache/closure-project-cache-native-20261108.log`.

This is bounded cache and Complete-publisher acceptance, not stable CLI,
self-host, distribution, native Linux or hosted-CI acceptance. No cache speed
improvement is claimed. The parent will commit the exact completed paths; no
push, dispatch, release or deployment is authorized by this receipt.
