# Candidate per-package unit cache v0

This is a private compiler slice that adds per-package unit reuse on top of the
existing disk store. It changes no public Go CLI behavior and no stable SDK ABI,
and no default cache location or enablement exists, so the ordinary check and
product commands keep their current behavior.

## Private boundary

[`project_cache_unit_store.fy`](../../examples/lexer_candidate/src/project_cache_unit_store.fy)
shares the bounded store and retention policy of the whole-project generations
but keeps its own envelope, so unit admission stays isolated from the
whole-project routes in the same directory. `encode` writes `FYCACHE0` plus a
`FYPUNIT0` identity member plus a `FYPROG00` IR-program member, and `restore`
re-reads those members in that order.
[`project_cache_unit_key.fy`](../../examples/lexer_candidate/src/project_cache_unit_key.fy)
writes the `FYPUNIT0` preimage.

## Route and key

`collect` binds the 8 environment compatibility fields, the profile target, the
environment standard library, the selected package `instance_id`, the numeric
range `[first, stop)`, the package rows (owner, namespace, path) and the resolved
import rows (module index, declaration, target, alias, binding). It omits the
other packages' source bytes on purpose: current semantic admission decides
whether consumed contracts still match. Numeric source slots and resolved imports
stay in the route, so reordered topology is a miss, and a range whose origins
cross two owners is rejected with code 1001.

## Admission

[`module_unit_artifact_admission.fy`](../../examples/lexer_candidate/src/module_unit_artifact_admission.fy)
never skips current checking. For a saved unit it decodes its IR, runs
`compile::check_range` over the current sources, rebuilds the current
original-source identities with `identified::build`, snapshots them through
`ids::extract`, then compares exact IR body, Flow, call and type metadata
coverage through `coverage::checked_range` before state layouts and SDK units
build. Parsing and source checking therefore always run; only lowering is
avoided. A saved unit that cannot attach to the current canonical type and
planner numbering is rejected, and the caller treats any rejection as a
disposable miss.

## Lookup and observed status

[`project_compiler_cached.fy`](../../examples/lexer_candidate/src/project_compiler_cached.fy)
is two-level: it tries the whole-project generation first through
`artifacts::from_disk`, and only on a miss enters the per-unit mixed path in
`project_compiler_core.build_complete_mixed`. The observed `path` is 0
unattempted, 1 cold, 2 complete whole-project hit, 3 mixed and 4 all units
reused. The optional CLI status reports `cache: mixed` for 3 and `cache: units hit`
for 4, distinct from `cache: hit` for the whole-project hit at 2.

## Invalidation

An ordinary cross-unit call checks its saved callee signature against current
metadata, so editing a dependency body leaves the caller unit reusable while the
changed owner unit is rebuilt; a changed public interface rejects the caller.
A generic whose selected body changes invalidates its consumer, because the
specialization body is part of the compared coverage. Admission failure never means the on-disk generation is
corrupt, so `load` keeps it instead of deleting it. Reordered topology and
incompatible type or planner numbering are conservative misses.

## Publication and fallback

[`project_compiler_core.build_complete_mixed`](../../examples/lexer_candidate/src/project_compiler_core.fy)
builds each unit group through `append_cached`, links the whole project with
`identified::build`, builds all package interfaces, then calls `publish_cached`
to store the cold units recorded in the `Work` bitmap. No unit is published
before every source and the complete linked owner succeed, so a later source
error cannot publish a successful prefix. Whole-project lookup exhaustion falls
back directly to `build_complete_reporting`; a rejected unit is cold-compiled
through the ordinary `range_identified` path. Cache errors remain disposable
misses, and cold compilation owns source diagnostics.

## Bounds

`project_cache_unit_store.Work` tracks at most 512 units in the cold bitmap and
carries independent key, scan, I/O, source and artifact histories. Inside the
envelope the `FYPUNIT0` identity member is capped at 16 MiB (16777216), the
IR-program member at the 256 MiB transport limit (268435456) and aggregate work
at 1073741824 charged units, all from `module_program_artifact_wire`. Source
planning keeps its 16 MiB history and project linking its independent 16 MiB history. These
are resource ceilings, not a measured result.

## Evidence

Admission evidence for this slice is recorded at
[candidate-package-unit-admission-20261015.json](../ops/evidence/candidate-package-unit-admission-20261015.json),
with `privateUnitAdmissionChanged`, `publicApiChanged=false` and
`fullProjectCallsStillRequireOwnedTargets=true`. Three compiler-API tests pass
Go-native and both candidate C profiles in the
[mixed compiler receipt](../ops/evidence/candidate-mixed-package-cache-20261015.json).
The [CLI receipt](../ops/evidence/candidate-mixed-package-cache-cli-20261015.json)
records 108 passing process observations across both candidate C profiles through
the 410-source authored check/build/run entry points. The current 511-source full
private dispatcher still stops in project linking with code 1002. No product
performance result or latest full-dispatcher fixed point is claimed here.

## Explicit aggregate unit family, 2026-11-05

`project_cache_unit_key.collect_aggregates` derives the same authenticated
environment, selected package and topology preimage using `FYPUNIT1` instead of
`FYPUNIT0`. Its opaque identity retains aggregate mode. The different preimage
produces a different route; neither a caller boolean supplied to restoration nor
the presence of aggregate-shaped bytes changes an ordinary key into this family.
Aggregate units use `FYCACHE1` / `FYPUNIT1` / `FYPROG01` in that order. Ordinary
unit bytes and `collect` remain unchanged. Encoding rejects a key/program mode
mismatch and linked-project programs; reading requires the exact key, envelope,
program and current-signature modes before current admission.

`module_unit_artifact_admission.restore_aggregates` uses the named aggregate
program decoder, then the existing current checked-range, original identity,
full body/Flow/call/cleanup/metadata coverage, state and SDK owners. Both unit
admission routes reject linked-project payloads. A reconstructed unit has no
producer lowering history; it still runs fresh current checking and may be
discarded as a miss when canonical numbering or consumed contracts differ.

`project_compiler_core.build_complete_aggregate_mixed` prepares current aggregate
source/type/signature facts, selects the aggregate key family for package-owned
ranges, and relinks reused and cold units through the same project verifier.
Publication uses `publish_cached_aggregates` only after complete linked interfaces
succeed. Cache failures fall back to the existing current-source cold unit
compiler, and a failed Complete publishes no successful unit prefix. Ordinary
mixed compilation keeps its existing entry point and key family.

The shared raw disk store transports both `FYCACHE0` and `FYCACHE1`, checking their
existing content-addressed generation identity and retained-directory policy.
It confers no semantic authority. Unit and whole-project readers still own their
exact version and first-member contracts, so an opaque raw record cannot cross
their admission boundary. Storage naming, retention, publication and cleanup
semantics are unchanged.

Windows unit-cache evidence is recorded in the aggregate implementation plan.
This explicit compiler-core entry does not wire aggregate whole-project disk
restoration or actual CLI commands. Linux execution, full CLI/self-build,
distribution and same-commit hosted acceptance remain separate required work.
