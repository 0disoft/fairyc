# Fairy mixed closure UNIT cache acceptance (2026-11-08)

## Scope and owners

`tests/closure_mixed_cache.fy` enters the existing
`project_compiler_core::build_complete_mixed` and
`build_complete_aggregate_mixed` factories. Both signatures remain unchanged.
The current-source routing implementation belongs to
`module_project_identified_compile.fy`, outside this test-only change.
Whole-project cache acceptance belongs to a separate workstream.

The fixture loads real manifest workspaces and retains a real cache directory
across fresh producer processes. It never substitutes a standalone UNIT build,
synthetic cache record, detached project, or raw C emitter for the mixed pipeline.
After Complete succeeds it selects the actual root entry with
`project_compiler_entry::select`, consumes that Complete with
`module_project_complete_c::partition_entry(complete, selected, 2)`, verifies the
bundle retains Complete, and serializes `c::project_emission(bundle)`.

## Focused scenarios

- Three packages: root caller, forwarding maker, and helper provider. The provider
  returns canonical `fn i32(i32)` values with zero through four copied i32 captures.
  Values pass through an ordinary non-repr(C) struct, struct copies, extraction to
  a local, and invocation after the original factory returns. Probe calls exercise
  all five capture counts; the selected four-capture result is 42.
- Cold is exactly hits/misses/stored `0/3/3`; fresh-process warm is `3/0/0`.
  Changing only the four-capture provider body produces 43 with `2/1/1`.
  Restoring its old generation and changing only the caller argument also produces
  43 with `2/1/1`. Each changed source is then replayed warm.
- Changing the root package UID changes real canonical package/unit identity.
  It requires `0/3/3` and three additional generations. Restoring the manifest
  reuses the old generations with `3/0/0`.
- A body-only type error in a later dependency is checked with both an existing
  retained set and an empty cache. A different, valid root body creates a cold
  prefix first. The fixture requires phase 3, a nonzero failing module index,
  at least one successful miss, and zero stored records. The disk snapshot must
  remain identical, including an entirely empty directory for the cold failure.
- A second three-package workspace has no function signature containing a
  closure. Only a typed body-local safe-fn binding introduces the leaf. Its cold
  directory must contain one `FYCACHE2` and two legacy scalar `FYCACHE0` envelopes;
  the next process must hit all three. This catches global-signature-only routing.
- One cold/warm pair exercises the unchanged aggregate mixed factory. Its root
  wraps the closure result in a real `Cell<i32>` specialization; the same
  closure/struct/provider project remains in use. It does not introduce another
  leaf-case matrix.

Every successful mixed result is compared to a fresh Complete built from the same
current input: all three public interface owners, ordinary/source hashes and
encoded descriptors. Ordinary/local C packets also match that Complete exactly.
Aggregate C packets match an independently cold-lowered Complete with the same
package UNIT partitioning, as explained below. Checked project IR
must retain closure mode, the selected aggregate mode, and verified source-bound
lifted helper module/declaration identities. The capture-count mask is checked.

Cache snapshots compare exact filenames and SHA-256 values, authenticate the
content hash embedded in each filename, preserve old generations, and assert the
expected number of additions. Warm replay and failed builds must leave the
retained set unchanged. Success output is exclusively the exact C packet; authored
failures print a checkpoint, and compiler failures include phase/stage/code and
source coordinates before trapping.

## Restore evidence and limits

Warm runs require actual cache hit counters plus nonzero source/artifact admission
work. UNIT closure restore retains the decoded owner and skips replacement
lowering in its owning implementation. Complete consumes the intermediate UNIT
plan and exposes only the linked project; it does not expose those decoded UNIT
owners or their individual `ir::work == 0` values. This fixture therefore makes no
claim that linked-project work measures skipped UNIT lowering, and does not add
a second standalone UNIT restore merely to inspect that field. The existing UNIT
restore acceptance owns the direct decoded-owner proof.

## Validation and parent scheduling

The Go harness source-checks the complete imported Fairy fixture and both real
original workspace shapes without building or running native images. The final
source check passed on 2026-11-08: test body 23.91 seconds, Go package 24.279
seconds. A fixture-only module/path mismatch found during validation was fixed
by using `src/main.fy` for `module main`. No product-source defect was identified
by this check. Native assertions remain unexecuted by this worker. The parent
schedules native execution after shared sources are frozen:

```powershell
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureMixedCacheSource$' -count=1 -timeout 5m -v
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureMixedCacheNative$' -count=1 -timeout 15m -v
```

Native work is Windows AMD64 only. Low and optimized producer profiles each run
their own fresh workspaces/cache directories. Each emitted C packet executes
under the corresponding low or optimized consumer profile and must return the
actual 42 or 43, with empty stdout/stderr. Matching profiles avoid a redundant
producer/consumer cross product. A failed low subtest stops before optimized work.
No Linux runs, CI changes, product-source edits, stage/commit/push operations, or
shared lockfile updates are part of this worker's scope.

Source validation and native validation results must be reported separately;
authoring these assertions is not evidence that they have executed.

## Parent fixture correction

The first Native low run reached the final aggregate scenario and failed at
checkpoint 35 with hits/misses/stored `0/3/0` (282.780s package). All preceding
ordinary, invalid-source, identity-change and local-only assertions had executed;
the overall test and low profile still failed. The original log is retained as
`.cache/closure-mixed-cache-native-20261108.log`.

That scenario selected the aggregate factory but used only an ordinary struct.
UNIT lowering derives its aggregate bit from actual kind20/21/22 types, so the
fixture's aggregate key disagreed with its nonaggregate units. The corrected
input constructs `Cell<i32>` around the result. No cache feature checks, byte
comparisons, hit/miss/store counts, publication rules or profile limits were
weakened.

An attempted stable Go `check` on that new input correctly refused private
generic-aggregate specialization. Source validation now uses the existing
explicit `compiler.CheckGenericPackages` oracle for this one private input;
ordinary/local inputs retain stable `check`. The complete fixture and all three
input shapes passed Source (18.23s body, 18.566s package). Logs:
`.cache/closure-mixed-cache-source-final-20261108.log` (failed stable attempt)
and `.cache/closure-mixed-cache-source-fixed-20261108.log` (final Source).

## Matching the aggregate C baseline to UNIT partitioning

The next low run passed actual aggregate cache publication but found a header
difference at byte 3386 (both headers 4228 bytes). Diagnostic output showed only
private field-number ordering: the ordinary struct used `fy_f_2` rather than
`fy_f_1`, with the generic cell using the opposite number. The mixed factory
splits package UNITs; the default source factory uses its own capacity grouping.
Private cleanup field indices can consequently differ. Exact C byte identity
across those different grouping plans is not a cache invariant.

The default independently built Complete remains in the fixture for all source,
verified-IR, helper/capture and public-interface checks. For aggregate **C byte
comparison**, a second Complete is cold-lowered using the same package UNIT
partitioning. This calls the real aggregate mixed factory with its I/O work
already exhausted, requiring exactly hits/misses/stored `0/3/0` and nonzero
source/link work. It cannot restore a saved UNIT or publish a replacement.
The Go retained-set hash assertions also cover these additional baseline calls.
Ordinary/local C comparison still uses the default independent factory.

All header/literal/unit byte comparisons and actual C execution remain required;
the correction controls the grouping variable instead of accepting unequal
packets. No product field naming, format, admission or cache behavior changed.
The aggregate scenario now runs first, keeping the full ordinary/local schedule
and both profiles while diagnosing any new combination failure sooner.

Failed logs are retained: `.cache/closure-mixed-cache-native-final-20261108.log`
(checkpoint 23), `.cache/closure-mixed-cache-diagnostic-fixed-20261108.log`
(bounded header excerpt). Diagnostic-helper Source attempts also caught a
missing explicit usize annotation and path-dependent resource ownership; the
final fixture uses explicit indices and an early-return reference helper.

## Final parent Windows acceptance

The complete corrected schedule passed without skipping a case or profile:

```powershell
$env:GOMAXPROCS='1'
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureMixedCache(Source|Native)$' -count=1 -timeout 15m -v
```

```text
--- PASS: TestCandidateClosureMixedCacheSource (17.02s)
--- PASS: TestCandidateClosureMixedCacheNative (361.87s)
    --- PASS: TestCandidateClosureMixedCacheNative/low (203.05s)
    --- PASS: TestCandidateClosureMixedCacheNative/optimized (141.25s)
ok github.com/0disoft/fairyc/internal/cli 379.000s
```

There were 15 fresh producer processes per profile, including two invalid-source
rejections: 30 producer replays and 26 successful Complete C packets. Each packet
compiled and executed under its matching consumer profile with the expected
42/43 and empty streams. Exact key/family/hash retention, `0/3/3` cold,
`3/0/0` warm, `2/1/1` partial reuse, UID changes, failed-prefix nonpublication,
two scalar plus one closure local-only records, and three aggregate/closure
records all passed. Source and Native remain distinct results. Log:
`.cache/closure-mixed-cache-acceptance-final-20261108.log`.

The only product change owned by this completed mixed slice is explicit
closure key/load/save routing in `module_project_identified_compile.fy`.
The Complete C adapters were committed with project cache `b1fb3eee`.
No additional runtime policy, public CLI/SDK ABI, normative source document,
dependency, DB, workflow or runner change was made. This does not complete
full CLI, legacy-family regression, self-host, distribution or Linux acceptance.
