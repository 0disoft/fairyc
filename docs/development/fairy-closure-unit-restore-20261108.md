# ADR0551 source-bound closure unit admission, 2026-11-08

Authority: ADR0551. This bounded continuation consumes the separately owned
FYPROG02 Program codec. Component artifacts and Program codec files stay frozen.

## Explicit unit boundary

`module_unit_artifact_admission::restore_closures` has the same caller-owned
source/catalog/signature/plan/range/work inputs as the old restore APIs. It:

1. Calls `module_program_artifact_decode::decode_closures(raw, work)`, which
   requires FYPROG02/FYCLEAN2/FYBODY01 and completes the fresh full structural
   Program proof before returning a detached owner.
2. Requires unit scope (`project == false`), closure mode and an aggregate flag
   matching current metadata. It does not infer either scope bit from types.
3. Runs `compile::range_identified` against the current original-source range.
   That path checks bodies/Flow/borrows, lowers and attaches current source-plan
   identities. `ids::extract_closures` then validates the explicit canonical
   kind23 family and the fresh owner's identity snapshot.
4. Encodes that current owner through the fixed FYPROG02 encoder and compares
   the complete canonical recipe bytes with the supplied artifact. The shared
   artifact meter charges the comparison as well as decoding/encoding.
5. Rechecks state and SDK source matching and returns the **fresh current-source
   identified unit**, including its current borrow observations.

This is deliberate fresh checking/lowering and exact recipe comparison. It is
not a lowering-skip or warm-cache performance claim. A detached Program never
becomes source authority. The artifact does not contain an independent source
text envelope; this boundary binds its structural recipe to current checked
source semantics, not to byte identity of irrelevant source comments.

The existing project coverage/call recipe matcher independently rejects
invocation kind4. Extending that family would need additional source writes,
so this small slice does not use or weaken it. The fresh-owner recipe approach
keeps the continuation within the authorized source-recipe/bound-unit scope.
Existing `restore`/`restore_aggregates` continue to select FYPROG00/01 and reject
the closure family. Project/cache/disk/Complete/publication paths are unchanged.

## Declaration identity and source comparison

`module_project_artifact_bodies` preserves expression kind36's `target` as a
helper **declaration ID**. Other existing target operands continue through the
canonical type mapping. A helper handle must not become zero or another type
when its numeric value exceeds or collides with the type arena.

`module_unit_artifact_identities::extract_closures` is an explicit family gate
around the existing current-owner extraction; it does not manufacture new
generic identities for closure leaves or helpers. Kind23 keeps an empty
specialization row. No on-disk identity format is extended.

## Acceptance fixture

`tests/closure_unit_restore.fy` and
`internal/cli/candidate_closure_unit_restore_test.go` author:

- Thirteen current original-source controls: the eleven existing accepted
  closure controls, a helper-declaration/type-arena separation case with twelve
  preceding scalar functions, and a return-only closure copy.
- Original source -> scoped metadata/source plan -> checked/lowered/identified
  unit -> FYPROG02 encode -> explicit source-bound restore -> exact reencode.
  The returned owner must have current borrow observations, canonical kind23
  identity rows, unit scope and a valid full structural proof.
- Direct current-source/body recipe comparison independently checks the kind36
  declaration fix without entering whole-project call coverage/publication.
- Three alternative, detached-valid artifacts: changed lifted-helper arithmetic,
  changed invocation argument and changed callable effects metadata. Each must
  fail against the other current source. Separately supplied stale metadata/plan
  with replaced original source must also fail.
- Scalar refusal, old/aggregate API refusal of FYPROG02, invalid unit ranges,
  truncation/trailing data and exhausted artifact work with no source work.

These are authored runtime assertions. Source validation checks their syntax,
imports, types and compiler IR; it does not execute them.

## Validation and ownership

With `GOMAXPROCS=1`:

```powershell
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureUnitRestoreSource$' -count=1 -v -timeout=240s
```

Final Source PASS after the identity-gate simplification: **22.50s**, package
**22.849s**. Initial Source also passed: 18.99s, package 19.284s. Scoped tracked
diff checks and explicit LF/final-newline/trailing-whitespace checks pass.

Parent `TestCandidateClosureUnitRestoreNative`: PASS, 182.90s. The existing
two-CL partitioned compiler-host builder compiled 81 parts for each low and
optimized profile, keeping the one-minute command bound and 16MiB stack.
Thirteen source-bound controls and the applicable source/helper/metadata/stale
refusals executed with exit 0 and empty stdout/stderr. This includes the three
detached-valid mismatches, so structural validity cannot replace source matching.
Log: `.cache/closure-unit-artifact-restore-parts-20261108.log`.
Final Source recheck passed in 15.81s.

The earlier CLI fixture passed under `run`; its monolithic optimized build
exceeded the unchanged 60-second limit. That failure remains in
`.cache/closure-artifact-project-native-20261108.log`. Partitioned acceptance
does not mark monolithic optimized CLI acceptance as passed. Linux, whole
candidate/self-host, persistent-cache and hosted CI remain separate work.

Only the assigned admission, identity and body-comparison files, the new
Fairy/Go fixtures and this receipt are written. No component/Program codec,
project/cache/disk/Complete/publication source, lockfile, normative document,
public API/ABI, database, dependency, warning/limit/timeout policy, runner,
workflow or repository-hygiene setting is changed. No staging, commit, push,
deployment or other remote mutation is performed by this worker.
