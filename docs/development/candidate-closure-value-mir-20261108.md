# Fairy copied closure MIR boundary - 2026-11-08

Authority: ADR0551. Worker scope: MIR model, verifier, a focused fixture and
Go harness. Parent owns source preparation, type checking, HIR/lowering and
format gates; the C worker owns emission. No shared lock edits or commits.

## Private API

- Type13 is `fn i32(i32)`; operation40 constructs it and terminator8 invokes it.
- `ClosureTarget { usize target; usize captures; }` is stored in
  `Program.closures`. `closure_count` and zero-based `closure_at` expose rows.
- `assemble_closures` has exactly the `assemble_resources` arguments followed
  by `Vec<ClosureTarget>` after drops. Existing assembler signatures stay intact.
  Old assembly refuses closure types and records. New assembly permits empty
  metadata for signature-only consumers without operation40. Every actual
  create requires matching metadata before assembly succeeds; every metadata
  row requires a matching create. External/orphan helper rows are not admitted.
  `has_closures` detects the new record/type family.
- Create40 uses `operator=target`, `type=owner_type=13`, ordered i32 capture
  edges0..4. Zero captures use `argument_start=0`. Unused fields are zero/false.
- Call8 uses `value=callable13`, `left=argument1`, `type=1`, `destination=result1`
  and one same-function normal `target`. Other fields are zero/false.

Metadata is checked against a real private, safe, zero-effect local body with
an i32 result and captures+1 i32 parameters. All blocks, including dead blocks,
must use immutable i32 locals and admitted checked arithmetic. No calls, effects,
foreign facts, ownership, other types, repeated writes or cyclic CFG are allowed.
Checked arithmetic traps remain permitted with effects0. Source-owned literal
identity and source16-local/64-expression admission remain parent responsibilities.

Capture edges and call destinations participate in ordinary shape accounting
and definite initialization. A callable/argument must be initialized before
the call destination becomes initialized. Direct ordinary closure struct fields
are allowed; nested carriers, wrappers, resource/enum/ABI/pointer/borrow transport,
carrier mutation and functional update remain closed, matching current Go MIR.
Transport uses the existing flow-scale67108864 work ceiling and charges linear
nominal lookups. Existing arena limits are unchanged.

## Focused validation

`internal/cli/candidate_closure_value_mir_test.go` provides independently selected
syntax, source-check and Windows native tests. Each uses an isolated temporary
workspace and the actual authored import closure; no shared `fairy.lock` is used.

The hand-built `tests/mir_closure_values.fy` has36 distinct graph cases:
captures0..4, checked division/trap and direct struct field controls, plus27
rejections covering arity5, missing/uninitialized/wrong-typed callable/captures,
forged targets and canonical fields, unsafe/effectful/foreign/public helpers,
wrong signature/mutable parameters, cross-function successors, dead calls and
overwrites, cycles, missing/duplicate/mismatched metadata, old assembly gates,
and nested carrier fields. An unused `fn i32(i32)` identity has no constructor
or metadata and passes only the explicit new family. Initialization failures
additionally require1002; old assembler and missing-create-metadata controls
require4. There are8 admitted and28 rejected distinct graph cases.

Executed source checks before native coordination:

```powershell
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureValueMIR(Syntax|SourceCheck)$' -count=1 -p=1 -parallel=1 -v
```

Syntax and isolated Go source check passed (92 authored source files,
1293 MIR functions at that source-check snapshot). Earlier source-check
attempts exposed a parent source-preparation E1006 and then a fixture resource
join; both were corrected. A harness expectation was corrected to allow normal
successful check output. Those attempts stopped before native compilation.

After parent coordination, executed:

```powershell
$env:GOMAXPROCS='1'
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureValueMIR(Syntax)?$' -count=1 -p=1 -parallel=1 -v
```

PASS: fixture syntax0.01s; source check92 authored sources/1294 MIR functions;
native verifier assertions in both default Windows and release/locked profiles,
20.28s for the combined source/native test, package20.420s. The36 distinct cases
execute in each profile. The native fixture exercises actual Fairy verifier
code; source-to-closure execution and C emission are other workers' checks.
Read-only gofmt and scoped Git whitespace checks passed. All owned files remain
below40KB. No process remains running from these commands.

Linux, full self-build/baseline compiler, package/cache restoration, full release
checks and remote actions were not run by this worker. No shared lock was edited.

Integration reminders: parent callable copy now uses `owner_type=13` and
zero-capture creates use argument_start0. Metadata collected
from HIR must not retain a create removed by MIR reachability; old format output
must reject the new family until a deliberate format version is implemented.

Coordinator integration: the MIR-only main commit keeps C emission closed for
this family until the source/HIR/lowering/C stage lands. Its temporary render
gate returns code 4 before writing output; old no-closure C remains unchanged.
