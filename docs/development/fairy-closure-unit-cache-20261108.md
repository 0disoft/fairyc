# ADR0551 closure UNIT persistent cache, 2026-11-08

This private UNIT slice selects an explicit closure family. It changes no public
CLI, stable SDK ABI, cache location, or default enablement. It does not introduce
project-envelope admission or a new Complete publication path.

## Exact families

| API/key family | Envelope | Identity | Program |
| --- | --- | --- | --- |
| `collect` / existing store APIs | `FYCACHE0` | `FYPUNIT0` | `FYPROG00` |
| `collect_aggregates` / existing store APIs | `FYCACHE1` | `FYPUNIT1` | `FYPROG01` |
| `collect_closures` / named closure store APIs | `FYCACHE2` | `FYPUNIT2` | `FYPROG02` |

`project_cache_unit_key.collect_closures` takes the existing source input,
original source plan, environment, package range, an explicit aggregate bit and
work history. `FYPUNIT2` starts with that aggregate bit as a canonical big-endian
u64 (0 or 1), then the unchanged range/environment/package/topology fields. Both
closure feature combinations are separate from each other and the old families.
The route retains `fylookup0-sha256-` plus the digest of the canonical preimage;
the new preimage, rather than a different filename parser, separates the domains.
The old scalar/aggregate preimages and successful encoder bytes are unchanged.

`project_cache_unit_store` exposes `encode_closures`, `restore_closures`,
`load_closures` and `save_closures`. Old APIs reject closure identities and
closure programs. Closure encoding rejects old keys, linked-project programs,
missing closure mode, and aggregate/key disagreement. Restore requires the exact
envelope, identity and program markers, complete framing with no trailing bytes,
byte equality with the supplied key and current metadata aggregate agreement.
Changing an outer marker cannot promote a record to another family.

## Current source is authority

Warm UNIT restoration calls
`module_unit_artifact_admission.restore_closures`. The integrated entry decodes
FYPROG02, checks UNIT/closure/aggregate modes, checks current bodies and Flow,
reattaches current source identities and compares every retained body, call,
cleanup recipe and metadata through full checked-range coverage. Current state
and SDK facts and fresh borrow observations precede the returned owner. The
same closure call-recipe matcher is independently exercised on malformed and
source-mismatched artifacts.

This repeats current checking but skips replacement cleanup/body lowering.
The retained Program is the freshly decoded and fully reverified saved graph;
`closure_unit_restore.fy` asserts its artifact work origin (`ir::work == 0`).
No measured warm speedup is claimed. Source bytes remain absent from the route:
a changed recipe is a miss on the same route, and compatible old generations
remain available if the source is reverted.

The shared store continues to own content-addressed filenames, compiler-image
and environment compatibility, publication, collision repair and retention.
Malformed bytes, key/version mismatch, source mismatch and incompatible old
generations are disposable misses. No semantic mismatch deletes a generation.
Existing UNIT resource failures 10/11/1002/1003 stop candidate traversal;
disk inventory/read failures still collapse to 1001, and a failed save returns
false. All scan/I/O/source/artifact work histories are retained. This patch does
not relax a bound or change the existing disk-error policy.

## Acceptance fixtures

`examples/lexer_candidate/tests/closure_unit_cache.fy` authors:

- Three key domains, distinct closure aggregate-bit routes, old/new API refusal,
  exact envelope/member markers, identity-bit tampering, truncation/trailing bytes
  and exhausted key/artifact histories.
- Real workspace loading, actual running compiler-image identity, disk open,
  cold encode/save and fresh-process warm source-bound load. Every admitted owner
  is compared byte for byte with a separate cold compilation.
- Corrupted content-addressed bytes becoming a cold miss and exact repaired
  publication; a changed lifted closure body returning a different answer;
  invalid current source publishing no record or output; reversion reusing the
  retained compatible generation after a newer incompatible one.
- Exhausted artifact work retaining 1002; exhausted I/O remaining a miss;
  neither exhausted attempt publishes a record.
- Returned factory values, local copies, struct copies and invocation after the
  creator frame exits. Scalar closure C packets are built/run under both native
  consumer profiles. The combined closure/generic-aggregate mode gets disk
  roundtrip coverage; its UNIT C emission remains separately gated.

`internal/cli/candidate_closure_unit_cache_test.go` copies the entire authored
transitive import closure through `copyCandidateParityModules`, including the
Windows filesystem/profile/compiler-image dependencies. The native producer uses
`candidateParityBuildParts`, which derives both SDK and private Windows host
support from its actual foreign-symbol inventory. No PATH-provided compiler
identity or substitute file fixture supplies cache authority.

Source-only command (no C compiler or fixture assertion execution):

```powershell
$env:GOMAXPROCS='1'
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureUnitCacheSource$' -count=1 -v
```

Parent-only native command, after all parallel source writes are frozen and the
raw transport integration below is present:

```powershell
$env:GOMAXPROCS='1'
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureUnitCacheNative$' -count=1 -v -timeout=20m
```

Native runs both compiler-producer profiles sequentially. Each replay launches a
fresh process and reopens the same real cache directory; both scalar consumer
profiles must execute the emitted packet with 42, or 43 after the body change.
This is authored acceptance, not an executed native result from this subtask.

## Parent integration

The first parent Native attempt executed both producer profiles but failed at
ordinary cold replay with exit 134 and empty output (292.69s total). This remains
a failed run: `.cache/closure-unit-cache-native-20261108.log`. The fixture's
entry selection was tightened to the actual source declaration named `main`,
instead of assuming only one kind3 frame. Failed assertions now emit their
checkpoint, and a failed low subtest stops before building the optimized image.
The revised fixture Source passed in 33.12s (package 33.516s).

Integration also replaces the former cold-rebuild unit restorer with the full
current-source recipe lane described above. The existing unit restoration
fixture now asserts the retained artifact Program work origin. Its 13 accepted
originals and malformed/source/helper/metadata controls remain present. Source
checking passed in 12.96s; the cache Source check before diagnostics passed in
20.12s (combined package 33.344s).

Corrected Native cache replay passed in 462.11s (package 462.258s): low 189.08s,
optimized 234.78s. Each producer profile executed ten ordinary replays and two
combined closure/aggregate replays in fresh processes. Nine ordinary outputs
per producer produced C packets, each executed under both consumer profiles:
24 fresh producer processes and 36 generated consumer executions in total.
All expected 42/43 outcomes, old/new key refusals, corruption repair, changed
source, invalid-source nonpublication, source reversion, artifact exhaustion
and I/O exhaustion assertions passed. Log:
`.cache/closure-unit-cache-final-20261108.log`.

Independent `TestCandidateClosureUnitRestoreNative` also passed in 186.37s
(package 186.511s): 81 compiler fixture parts per low/optimized profile, all
13 original controls, source/helper/metadata/stale-input refusals and the
retained artifact Program work-origin assertion. Each image returned zero with
empty stdout/stderr. Log:
`.cache/closure-unit-recipe-restore-native-20261108.log`.
No source-only result counts as a native cache hit.

## Integration boundaries

The worker owns the UNIT key/store, two new fixtures and this note. Parent
integration also updates the source-bound unit restorer, its existing fixture
and raw disk-envelope recognition.
`project_cache_unit_lookup.fy` and `project_cache_unit_artifact.fy` do not exist;
their responsibilities live in the existing store module.

Integration allows `FYCACHE2` in `project_cache_disk.envelope`. This is raw
transport recognition only; exact UNIT key/program and current-source gates
remain necessary. Existing content hashes, naming and retention are unchanged.

Compiler-core routing/publication and whole-project caching remain parent work:
select closure keys and the named load/save APIs explicitly, preserve separate
aggregate mode, and publish cold units only after successful complete linking
and interfaces. This subtask's fixture saves one fully checked independent UNIT;
it does not establish all-or-nothing project publication. Parent must also retain
old scalar/aggregate regression coverage while integrating. Linux, full CLI,
self-build/distribution, hosted CI and performance measurements are not run here.

No normative document, public CLI/SDK/ABI, DB, dependency, repository hygiene,
timeout/warning policy, workflow or runner configuration is changed. No push,
release, hosted dispatch, deployment or Linux execution was performed.
