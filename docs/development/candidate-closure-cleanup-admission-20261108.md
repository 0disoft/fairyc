# ADR0551 private cleanup type admission, 2026-11-08

Worker slice on `main`, baseline `1ff68c241b2d`. The parent owns concurrent
unit call/value frontend work and integration. Only these six files belong to
this slice; no shared lock, index, commit, dependency, CI or remote action was
changed by this worker:

- `examples/lexer_candidate/src/module_unit_cleanup_mir_model.fy`
- `examples/lexer_candidate/src/module_unit_cleanup_mir_types.fy`
- `examples/lexer_candidate/src/module_unit_cleanup_mir_verify.fy`
- NEW `examples/lexer_candidate/tests/closure_cleanup_mir.fy`
- NEW `internal/cli/candidate_closure_cleanup_mir_test.go`
- This NEW receipt.

## Deliberate family selection

The single new compiler-private constructor is
`assemble_closures(types, frames, glues, fields, operands, drops, work, project, aggregates)`.
It returns the existing cleanup `Program`, with an owned private `closures` flag;
`closures(ref Program)` exposes only the selected mode. Existing project and
aggregate bool policies remain independent, including all four combinations.
No existing signature changes. This is an internal module API addition, not a
language/CLI/ABI contract or a new execution operation.

`assemble`, `assemble_project`, `assemble_aggregates`,
`assemble_project_aggregates`, `assemble_artifact` and
`assemble_aggregate_artifact` always select `closures=false`. There is no new
artifact constructor. Existing decoders still reconstruct through the old
constructors, so their type23 refusal is preserved. The ordinary lowerer and
the existing graph fixture's `executable_code(23,0)==1001` remain unchanged.

Only the new family admits canonical graph kind23 with **all five payload
fields zero**. Existing duplicate detection and glue validation apply without
exceptions: glue is exactly kind0 with zero fields/frame/effects/abilities and
`nonempty=false`. The unchanged requirement/drop checks leave this leaf unowned
and reject any attempt to drop it. A regular function result or value operand
can describe this leaf without creating cleanup obligations.

Array/Option/Result/Vec/borrow/pointer/tuple and generic-resource descriptors
cannot directly reference kind23. Both Result alternatives and both tuple
positions are covered. Resource glue fields and cleanup receiver types also
reject it. Existing exact pointer/borrow/tuple restrictions remain in effect.
Nominal identities alone do not expose ordinary struct fields; this detached
owner does not claim to prove hidden nominal containment or provider layouts.
Those source/layout checks remain the parent's separate authority.

These are source-independent cleanup facts. They prove neither current source,
a closure target/capture set, executable body, Complete, cache restoration nor
publication. Existing program/project/body formats and cleanup codec modules
are untouched. In particular, the existing cleanup encoder has no representation
for the new mode: callers must not use its old bytes as closure-family transport;
the old decoder deliberately refuses such a leaf. A future transport owner must
add an exact private format gate before enabling closure round trips.

## Focused fixture coverage

Five fixture functions cover:

1. Positive scalar controls for all six old constructors; explicit closure leaf,
   trivial glue, mode-bit compatibility, aggregate/scalar-tuple compatibility,
   and unchanged fresh verifier work of 40962 for a single leaf.
2. Every nonzero leaf payload, duplicate leaves, all observable direct wrappers,
   Result error and tuple-tail positions, out-of-range links, a maximal scalar
   payload and unsupported future kind24.
3. Binding/expression operands without drops; forbidden drops for each operand
   kind, closure cleanup receiver, resource-field misuse with a scalar positive
   control, and every nontrivial glue payload field.
4. Independently written FYCLEAN0/FYCLEAN1 bytes under both project flags:
   positive scalar decoding, kind23/kind24 rejection, and rejection of an extra
   record appended to an otherwise valid old payload.
5. Exact/over unit and project producer-work bounds, exact/over independent
   verifier work, glue/type count mismatch, empty graph, 4097 types, 16385
   operands and over unit/project drop counts. Synthetic count probes are never
   published as valid programs. No production count, budget or charge changed.

## Executed validation

Windows AMD64, `GOMAXPROCS=1`:

```powershell
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureCleanupMIRSource$' -count=1 -timeout=3m -v
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureValueGraphOracle$' -count=1 -timeout=2m -v
gofmt -l internal/cli/candidate_closure_cleanup_mir_test.go
```

- Final source check **PASS**: test 2.51s, package 2.694s. This parses the actual
  fixture and authored import closure, then runs the real CLI `check` in an
  isolated temporary project. It does not build native Fairy code. Initial
  fixture checks exposed a reserved identifier and conditional integer type;
  both were repaired before this pass.
- Existing graph oracle **PASS**: 20 structural cases, 3 distinct closure
  admissions and 8 malformed controls, package 0.126s. This executes the Go
  oracle; it does not execute the Fairy `executable_code` assertion.
- Go formatting, scoped tracked `git diff --check`, LF/final-newline checks for
  all five code/test files, and unchanged existing graph fixture, normal cleanup
  lowerer and cleanup encoder/decoder: **PASS**.

Parent integration then executed the five fixture groups on Windows AMD64:

```powershell
$env:GOMAXPROCS='1'
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureCleanupMIR(Source|Native)$' -count=1 -timeout=8m -v
```

- Final SDK-integrated source **PASS**, 2.34s; native **PASS**, 11.84s;
  package **PASS**, 14.311s.
- Both `run` and `run --release --locked` returned exit 0 and empty stdout/stderr.
- Evidence: `.cache/closure-cleanup-mir-integration-20261108.log`.

**Not run:** whole compiler native/self-build, Linux, closure artifact/cache round
trips, CI or release checks. The new mode remains private type admission, not
closure body execution or publication.

No normative/source-of-truth document, DB, runner or repository-hygiene config
changed. The parent owns commits and the shared source lock.

## Final worker source SHA-256

Paths are relative to the repository; this receipt is excluded.

| File | SHA-256 |
| --- | --- |
| `examples/lexer_candidate/src/module_unit_cleanup_mir_model.fy` | `54fd90bba2b1aa7cef2ef4ea7b83f0d7ff7f6f3ac6de18125e5d55629ee1aa35` |
| `examples/lexer_candidate/src/module_unit_cleanup_mir_types.fy` | `b30f22751d589ac609fa257174ab4ca6b1a134986cadfaf18eacbf808b07149b` |
| `examples/lexer_candidate/src/module_unit_cleanup_mir_verify.fy` | `cf01384560949bc8c64c864737c37b12bd89a848cd2996e238fb05fab09e4c10` |
| `examples/lexer_candidate/tests/closure_cleanup_mir.fy` | `7039a37b56424d2e1eb4bc5ca5d01d832654dc310f16c9cc5bedfee0fc3b58cc` |
| `internal/cli/candidate_closure_cleanup_mir_test.go` | `992bde400360d71b0e0afcfed1e4e54fd392f77f76050d7a0bff9edf32ff73ad` |
