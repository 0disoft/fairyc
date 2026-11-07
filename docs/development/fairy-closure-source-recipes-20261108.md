# Copied closure source call and recipe matching, 2026-11-08

ADR 0551's source comparison lane now accepts actual kind-4 value calls. It
does not manufacture an ordinary provider, specialization or static target.
The invocation's module, declaration and instance remain exactly zero. The
existing kind-1/kind-2 admission and partial-unit behavior are unchanged.

`module_project_artifact_calls` authenticates the checked call record, named
immutable local/parameter, mapped kind-23 callable type, mapped i32 result and
single checked i32 argument. Each binary lookup comparison charges the shared
work history. Dense expression indexing is justified by the preceding full
body matcher: equal counts and exact source_index equality at every ordinal.

This uses the existing full body and fresh Program proof. The body comparison
preserves kind-36 helper declarations as declarations rather than type IDs,
checks original-source construction targets and ordered capture snapshots,
and compares every expression, lexical binding, statement and retained byte.
Frame mapping already handles prepared private helpers as ordinary kind-3
frames with empty specialization identities. Type mapping already compares
the canonical kind-23 leaf descriptor. No types/frames/coverage code change
is required. The fresh proof independently authenticates real pure helper
bodies/signatures, initialized callable operands, storage and exact captures.
The recipe layer retains trivial kind-0 closure glue and no closure cleanup
obligation; existing resource drop/recipe and metadata coverage checks follow.

`closure_source_recipes.fy` creates Programs from original written sources,
encodes and decodes fresh FYPROG02 owners, then exercises calls, full recipes,
partial-unit recipes, checked-range coverage and whole-source coverage. The
controls include captures 0 through 4, returned factories, local/struct copies,
different helpers passed to one parameter, a signature-only consumer and
mutable-source capture snapshots. An ordinary scalar roundtrip preserves the
old gate and remains refused by the closure encoder.

Same-width source alternatives test helper arithmetic, capture order, captured
initializer and invocation argument changes. Byte mutations independently
retain detached validity while changing an argument span, the named callable
binding or its syntax token. The argument mutation must first pass full body
matching before the call recipe refuses it. Nonzero static targets and
uninitialized callable locals must fail fresh detached reconstruction. Calls,
recipes and both coverage modes author exact-work, one-over and already-invalid
history assertions without resetting the caller's ledger.

## Validation and handoff

Windows `GOMAXPROCS=1 go test -mod=readonly ./internal/cli -run
'^TestCandidateClosureSourceRecipesSource$' -count=1 -v` passed: test 14.22s,
package 14.508s, 16 accepted original closure sources plus the scalar control.
The first attempt found a fixture-only double-reference error; it was fixed
before this pass. Source checking validates the matcher and fixture but does
not execute Fairy's runtime assertions. The parent schedules Native after the
product-source freeze:

```powershell
$env:GOMAXPROCS='1'
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureSourceRecipesNative$' -count=1 -v -timeout=10m
```

Native uses the existing partitioned compiler host, two-CL bound, one-minute
per-command limit and 16MiB stack, running low and optimized images. It is
reserved for sequential parent execution, including checking detached-valid
mutation assumptions. Project envelope routing, reordered-module rebase and
Complete/cache publication are separately owned by the parent. Linux, CI,
release, distribution and remote state remain outside this slice.

The first parent Native run failed after 47.34s: low image exit 134 and empty
stdout/stderr, before optimized execution. Log:
`.cache/closure-source-recipes-native-20261108.log`. This was an executed
failure, not a successful runtime receipt. The fixture now prints its helper
and assertion checkpoint to stderr only on failure. Native profile subtests
allow a coordinated `-run '^TestCandidateClosureSourceRecipesNative/low$'`
diagnostic without compiling an optimized image; the normal command retains
both profiles and stops after the first failure. The diagnostic fixture's final
Source check passed in 21.81s (package 22.133s). Diagnosis and the final profile
results follow below.

The parent identified a real mixed-ledger defect during diagnosis:
`bodies::body` passed the artifact's 1GiB caller history directly into
`copied_closure_reach::build`, whose independently bounded traversal rejects
history at 1MiB. With a prepared closure source, exact-work calls starting
near 1GiB therefore failed before doing their small reachability traversal.
The authorized write scope was extended to `module_project_artifact_bodies`.
Reach now starts at local measured=0 and folds its cost once into the artifact
ledger on success and failure. Its original 1MiB traversal bound is preserved,
and the exact/one-over assertions are unchanged. The first uninstrumented
failure alone does not establish its checkpoint; Native still must execute
the corrected fixture before claiming a successful diagnosis.
After the ledger fix, the focused Source check passed in 15.74s (package
16.013s). Product sources were frozen before the next Native snapshot.

The corrected full Native run passed in 214.72s (package 214.955s): low
41.94s and optimized 153.41s, session 37431 terminal exit 0. Each image executed
all 16 original closure controls, the ordinary scalar gate, three detached-valid
forgeries, two detached-invalid forgeries, four source mismatches and exact,
one-over and already-invalid work histories for all five matcher/coverage
modes. Both images returned 0 with empty stdout/stderr. No fixture assertion
or expected 1001/1002 status was weakened. Failure-only diagnostics remain in
the fixture for future regressions. The parent separately reported whole
candidate check/locked success on 775 sources and 9765 MIR functions; this
worker did not rerun that broader check.

Calls, the body reachability work ledger and recipe comments change in product source, plus this new receipt,
the new Fairy fixture and Go harness. No normative source-of-truth document,
public SDK/API/ABI, dependency, lock, DB, workflow or runner changes. No staging,
commit or push is performed by this worker.
