# ADR0551 source-backed unit cleanup lowering, 2026-11-08

Bounded parallel worker slice on `main`, baseline `b5af44e4`, final source-check
HEAD `7b513405` after the parent's shared SDK commit. The parent owns native CPU,
bodyIR/C integration, shared source lock,
commits and the later Linux batch. This worker changed only:

- `examples/lexer_candidate/src/module_unit_cleanup_mir_lower.fy`
- NEW `examples/lexer_candidate/tests/closure_cleanup_lower.fy`
- NEW `internal/cli/candidate_closure_cleanup_lower_test.go`
- This NEW receipt.

## Change and preserved boundaries

The existing type-copy loop records whether the owned values graph contains
kind23. Only that case selects the existing compiler-private
`assemble_closures(graph,frames,glues,fields,operands,drops,work,false,aggregate_types)`.
The independent aggregate flag still follows kind20/21/22. No added graph pass,
work charge, API parameter or cleanup record is needed.

Graphs without kind23 still select the original `assemble` or
`assemble_aggregates` call with the same inputs and producer work. All three
public lower signatures are unchanged. Raw `lower` still checks move diagnostics,
constructs cleanup policy and rejects its first diagnostic before lowering.
`lower_checked` and `lower_checked_catalog` still validate source freshness and
reuse the policy owned by the unforgeable checked-bodies plan. The private
`lower_policy` entry remains private.

The existing cleanup MIR verifier owns canonical leaf payload, duplicate,
trivial-glue, no-drop, wrapper and resource-field restrictions. Selecting its
closure family does not bypass any of them. Cleanup model/types/verify were not
edited. Neither were artifact formats or old artifact constructors/decoders.

This follows the previous receipts
`candidate-closure-cleanup-admission-20261108.md` (private family signature and
independent modes) and `fairy-closure-unit-values-20261108.md` (source-owned
canonical kind23 values). Neither receipt's earlier native results are reused
as evidence for this change.

## Original-source fixture and harness

Go supplies only original source text, including the actual optional SDK Vec
source for its wrapper control. No inferred Go graph, helper, move fact or MIR
record is supplied to Fairy. The authored producer independently scans/parses,
builds signatures, binds, prepares the unit call closure, builds values and
moves, evaluates cleanup policy, and calls raw cleanup lower.

For admitted inputs the fixture is written to compare raw, sourcechecked and
catalog-sourcechecked cleanup programs: all graph descriptors, glues, frames,
fields, operands, drops, mode flags and producer work. It checks exact source
frame identity, correspondence to move cleanup requests, canonical kind23 with
all five payload fields zero, exactly trivial closure glue and no closure drop.
Sourcechecked freshness controls change one source byte without changing length,
swap catalogs and remove a source; these must return stale-source code12.

The Go harness contains 25 original-source controls: 10 admitted and 15 refused.
They cover all four closure/aggregate mode combinations; scalar and resource
legacy controls; signature-only closure invocation; returned factories, copies
and ordinary struct fields; different helper literals sharing the same type;
ordinary and generic resource cleanup beside closures; and authorized cleanup.

Refusal controls cover missing cleanup effects (7102), absent drop ability
(1108), Option, either Result position, array, borrow, pointer, Vec, direct and
ordinary-struct-contained resource fields, repr(C), foreign transport, unsafe C
callback storage and mutable closure bindings. Policy failures must be returned
unchanged by raw lower and by checked-bodies construction, including their spans.
Budget/allocation/stale-record failures cannot satisfy a language-refusal
assertion. A missing SDK import cannot satisfy the Go Vec control: real CLI
`check` must specifically return source-diagnostic exit3 with E4502.

The native harness uses the existing private partitioned compiler builder on
Windows AMD64. It invokes every control, then the freshness group, under low and
optimized profiles, requiring one ordered receipt per original source. The same
complete input is retained in both profiles, with the existing 60-second bound
per C compile/link command. Linux remains batched. This producer exercises
source-to-cleanup lowering, not the input closure body's runtime execution.

## Aggregate prerequisite correction after parent native failure

The parent log `.cache/closure-cleanup-lower-integration-20261108.log` reports
native failure on `old-aggregate`, phase4/code9. The preceding seven admitted
controls reached cleanup MIR, and the two expected policy refusals returned
7102/1108. That partial run is not a 25-case native pass.

The fixture incorrectly used ordinary `signatures::build` for all inputs.
As in `project_compiler_core::build_complete_aggregate_sources` and the existing
generic aggregate fixtures, the unchanged Box<T> inputs require:

```text
project_source_plan::build_aggregate_sources
  -> generic_aggregate_types::build
  -> module_signatures::build_aggregates
  -> unit binding -> values -> moves -> policy -> cleanup lower
```

`prepared_signatures` now chooses that prerequisite by inspecting the original
root declarations for generic struct/resource definitions. It receives no Go
expectation flag, fabricated graph or synthetic concrete layout. Each of the
three aggregate controls owns one module with no imports; the fixture explicitly
enforces that plumbing and requires exactly one source-prepared aggregate
instance. Its private package coordinate is a SHA-256 digest of the actual
original module bytes. Plan/template/source freshness is checked by the existing
production preparation owners. Source preparation has its separate ledger;
the raw/checked cleanup work comparison retains the original move-policy ledger.

The other 22 inputs retain ordinary signature construction. In particular, the
Vec refusal's root contains no user generic aggregate declaration and its actual
SDK provider does not select the user aggregate route. The two closure/aggregate
signatures are supported by the dependency walker's existing safe-function
type-child traversal; no copied-closure literal/kind36 occurs in these three
aggregate inputs. Other literal controls keep their already exercised route.
All 25 original strings and expected admissions are unchanged (10 accept,
15 refuse); neither a case nor an acceptance was removed to make the check pass.

The Go workspace helper now copies the newly imported planning/hash modules and
their actual import closure. Its first correction check exposed that missing
copy as a package-graph failure; it was fixed before the final passing source
test. No production change beyond the existing lowerer is needed or claimed.
Cleanup model/types/verify, checked policy/moves and all parent-owned files remain
outside this worker's write set.

## Executed validation

Windows AMD64, `GOMAXPROCS=1`:

```powershell
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureCleanupLowerSource$' -count=1 -timeout=3m -v
gofmt -l internal/cli/candidate_closure_cleanup_lower_test.go
git diff --check -- examples/lexer_candidate/src/module_unit_cleanup_mir_lower.fy
```

- Final source Go test after aggregate preparation: **PASS**, test 12.23s,
  package 12.429s. All 25 original-source
  Go controls match their declared admissions. The real CLI checked the authored
  Fairy fixture and its import closure in an isolated temporary workspace.
- Go formatting and scoped tracked diff whitespace: **PASS**.
- Cleanup model/types/verify: no tracked diff from HEAD.
- LF, final-newline and trailing-whitespace checks for this slice: **PASS**.

Parent native integration **PASS**, 176.49s (package 176.811s): 68 C parts in
each producer profile, all 25 ordered original-source receipts per profile and
the source-freshness assertions. Evidence:
`.cache/closure-cleanup-lower-native-20261108.log`.

The initial corrected monolithic CLI run passed low, but optimized compilation
exceeded the unchanged 60-second MSVC command limit. That failed acceptance is
retained in `.cache/closure-cleanup-lower-monolithic-20261108.log`; it is not
reported as a CLI optimized pass. The private partitioned builder verifies the
complete producer with the same limit and no source/control removal.

**Not executed in this slice:** input closure bodies, full native candidate or
self-build, artifact/cache round trips, Linux, remote CI, release or deployment.

Parent native reproduction:

```powershell
$env:GOMAXPROCS='1'
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureCleanupLowerNative$' -count=1 -timeout=8m -v
```

No public language/API/ABI contract, normative source-of-truth document, DB,
dependency, manifest, shared lock, runner, CI or hygiene configuration changed.
Other workers' changes were left intact. The parent owns commits; no push or
remote action was performed.

## Final code SHA-256

| Path | SHA-256 |
| --- | --- |
| `examples/lexer_candidate/src/module_unit_cleanup_mir_lower.fy` | `74988ae37d3449e377f7c7ea785d729a345e547fe703cd81b2fe8097c3ab848a` |
| `examples/lexer_candidate/tests/closure_cleanup_lower.fy` | `33bcc69402afd57f32748552960c5fdc7d08cccb558fdbf2e8c38453f19ed4b9` |
| `internal/cli/candidate_closure_cleanup_lower_test.go` | `f6c60c60213d1f7026a0712de394ba60939a4fa9067e824b2cddde32a1a3814f` |
