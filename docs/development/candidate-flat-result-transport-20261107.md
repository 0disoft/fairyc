# Fairy flat Result transport replay, 2026-11-07

Owner: accepted ADR 0550. Work is limited to the isolated
`fairyc/.cache/flat-result-20261107` checkout at baseline
`bf719aa872375ac22706c20cb232db42efb28700`.

This worker owns only these new files:

- `examples/lexer_candidate/tests/composite_result_transport.fy`
- `internal/cli/candidate_composite_result_transport_test.go`
- `internal/cli/candidate_composite_result_transport_pipeline_test.go`
- This receipt.

Production type/admission/backend code, existing tests, ADRs, locks, queue and
commits belong to the coordinator/other owners. No production edit is made here.

## Controls and source ownership

The seven success leaves are i8, i16, isize, u16, u64, f32 and f64. Go controls
check 105 original source cases: flat and resolved-error signatures, closed
wrong-error/nested/reference/pointer/array shapes, independent must-handle
obligations on both copies, and propagation error byte origins with UTF-8/CRLF.
The error mismatch uses an enclosing `Result<u32,i32>` so it tests propagation
without first invoking the independent `Result<S,i32>` signature refusal.
Seven composite generic argument refusals and the exact pipeline source are
also checked. Source ASTs are compared before and after Go preparation.

Fairy graph tests cover descriptor, interned and written canonicalization of
each S7 payload and the resolved E=u32 path. Capacity graphs already contain
u32 and a valid array backbone: at 4094 nodes the new success scalar plus wrapper
reach 4096, while at 4095 nodes the failed wrapper must roll back its new scalar.
All fields of every original record are compared after failure. Wrong-error and
shallow-boundary failures, unknown error lookup and a failed ordered batch also
prove complete rollback, including a previously successful prefix.

Native source diagnostics compare actual Fairy signature reports and body
diagnostics to Go's original spans/features. The source pattern and expectations
are created independently from Fairy's output.

## Restored artifact/native replay

The native source handles original, copied and replaced Result obligations
individually. It constructs and extracts `.ok` and `.err`, uses ordinary written
and generic signatures including resolved E=u32, and exercises both outcomes of
`?`. A separate C counter requires each operand to execute once. It observes only
the active scalar payload, including signed minima, unsigned maxima and floating
positive/negative zero through the existing private scalar bridges. There is no
Result-by-value foreign ABI and no extended scalar arithmetic/literal dependency.

The existing `static_operation_pipeline.fy` helper supplies actual current-source
Complete compilation, nested body/program/project artifact encoding/decoding,
source-bound restoration and generated C partitioning. It locates bridges by
written probe name. Neither Go-expanded AST nor alternate provider/body code is
fed into Fairy. Both producers use one frozen import closure, with low and
optimized C consumers for each. Each of four products must return 42 with empty
output, preserve thirteen scalar observations, and execute 26 `?` operands.
Failed pipeline copies/images and separately named attempt logs are retained.

## Checks and gate

All executions set `GOMAXPROCS=1`. Native checks additionally set
`FAIRYC_RUN_PARITY_INTEGRATION=1` and require Windows AMD64.

The coordinator reported the Go/source gate passed: 28 focused top-level Go
tests and source/restored-MIR C11/C17 low/optimized native products. This is an
external coordinator result, not a new test run by this worker.

Own Go control attempt 1 failed (0.11s): signature and propagation cases and the
exact pipeline input passed, but the initial must-handle expectations used a
declaration origin instead of the initializer evaluation origin, and omitted
the later original evaluation needed to create a new pending obligation. The
controls now check the initializer use and later re-evaluation explicitly, in
agreement with the owning Go tests. Log:
`.cache/flat-result-go-controls-attempt1.log`.

Attempt 2 failed (0.02s) at the added graph fixture syntax preflight because the
logical source path omitted `tests/`, producing the expected module/path E0002.
Only the supplied source path was corrected. Log:
`.cache/flat-result-go-controls-attempt2.log`.

Go control attempt 3 passed (0.174s package wall time), including the graph
fixture syntax preflight, all 105 source cases, seven composite argument
refusals and the exact pipeline source. Log:
`.cache/flat-result-go-controls-attempt3.log`.

Native graph/source attempt 1: the graph test passed in 15.92s, including its
ordinary/release-locked executions and all original-record rollback checks.
The combined unrolled 105-case source diagnostic driver then failed before
actual comparison: the normal run command returned exit 4, `internal compiler
invariant failed` (10.56s). This is a failed bootstrap-driver attempt, not an
assertion failure or a successful source diagnostic claim. Log:
`.cache/flat-result-graph-source-attempt1.log`.

The source driver now keeps all 84 signature cases in the existing Option
binary-reader fixture, and isolates the 21 body/evaluation cases in a separate
helper module. Main calls two small drivers instead of owning 105 resource-heavy
blocks. No existing fixture is edited; the private copy receives its actual case
count. This still uses two native builds, not one build per scalar.

Native source attempt 2 **passed** in 89.30s (89.487s package wall time), including
the cheap source check, both ordinary/release-locked executions, all 84 original
signature spans/features and 21 body/evaluation byte origins. No product type or
flow defect reproduced with the bounded reader/helper driver. Log:
`.cache/flat-result-source-diagnostics-attempt2.log`.

Restored-artifact pipeline attempt 1 **passed** in 184.17s (184.328s package wall
time). Both low and optimized actual Fairy producers used the same frozen source
closure and 103 C parts. All four low/optimized producer/consumer products
compiled, linked and executed with exit 42 and empty output. Each independently
checked thirteen payload observations and 26 once-only operand evaluations;
52 payload observations and 104 evaluations total. Source-bound body/program/
project restoration into Complete and subsequent actual C partitioning succeeded
in each producer. Log: `.cache/flat-result-native-pipeline-attempt1.log`.

Executed focused selections (each `-mod=readonly -count=1`):

```powershell
go test ./internal/cli -run '^TestCandidateCompositeResultGoControls$' -timeout=3m
go test ./internal/cli -run '^TestCandidateCompositeResult(Graph|SourceDiagnostics)$' -timeout=8m -v
go test ./internal/cli -run '^TestCandidateCompositeResultSourceDiagnostics$' -timeout=8m -v
go test ./internal/cli -run '^TestCandidateCompositeResultTransportPipeline$' -timeout=12m -v
go vet ./internal/cli
gofmt -l internal/cli/candidate_composite_result_transport_test.go internal/cli/candidate_composite_result_transport_pipeline_test.go
```

Scoped vet, read-only gofmt checks and whitespace checks of all four new files
passed. Valid graph/Go control results were reused after the diagnostic-driver
change; unchanged selections were not rerun. The combined graph/source command's
failed status remains recorded separately from its successful graph subtest.

## Final freeze and remaining gates

The parent completed whole isolated-source checking independently. This worker
read `.cache/flat-result-source-20261107-all-check.json` without rerunning it:

- `check`: exit 0, 14.958s, lock updated, empty stderr.
- `check --locked`: exit 0, 13.154s, lock verified, empty stderr.
- Both: two instances, 756 loaded source files, 9564 MIR functions.
- Snapshot: `.cache/flat-result-source-all-8dd376fc`, all 1228 authored FY files,
  no excluded uncommitted sources, `source_match=true`, `lock_copied=true`.
- Go bootstrap SHA-256:
  `00be89383b0049a349b0bf8559ea2c81c86e8af85feddf349069001ce53982fa`.
- Parent log: `.cache/flat-result-source-attempt1.log`.

At final handoff, `GOMAXPROCS=1 go vet ./internal/cli` passed again and read-only
gofmt checks of both owned Go files produced no output. No extra native run was
launched. All three owned code-file SHA-256 values below remain unchanged.

Finalized 2026-11-08 KST. HEAD remains
`bf719aa872375ac22706c20cb232db42efb28700`. Exactly the four new files listed at
the top are worker changes. No existing test or production/ADR/lock/queue file
was edited, and no commit or remote action was made. No owned process/session
remains running and there is no remaining worker blocker.

These SHA-256 values fingerprint authored inputs at receipt finalization. The
last two are coordinator-owned files read only for the receipt. Successful
private source/image copies were cleaned; all failed attempt logs remain.

| File relative to the isolated checkout | SHA-256 |
| --- | --- |
| `examples/lexer_candidate/tests/composite_result_transport.fy` | `e8e5a200dc77f0aa5b7eb959c3acd277316921732095f36ab0d9f8b8206824f9` |
| `internal/cli/candidate_composite_result_transport_test.go` | `7483817ec8bc91c72960fb598c052be78abffc0dcd2dc2798b07544b536b853e` |
| `internal/cli/candidate_composite_result_transport_pipeline_test.go` | `97c72f05617aaed09f746ce37afc130f2af8b3664056710e07929ff0cc1cf8f1` |
| `examples/lexer_candidate/src/module_types.fy` | `59a0c2d3b3678140bb0398af1ef807f794fec1064ce1d7b0ede873bcff8d5fc6` |
| `examples/lexer_candidate/src/generic_signature_diagnostics.fy` | `4a8981013410296e5c22e6c7850ff8eb7425305cca6f97122af7eec99c03c39a` |

No production source/work/runtime/stack bound, public API/ABI, artifact/cache
wire version, DB, dependency or runner contract is changed by this worker.
The Result-only lock is tied to the isolated bf719aa8 base. It must not be copied
into main. The parent will copy the owned files and regenerate the combined
Group+Result lock in main; this worker has made no lock edit or copy.
Linux, full CLI/persistent cache, combined Group+Result source checking,
self-build/fixed-point, distribution, hosted CI and remote actions remain
separate coordinator gates. This owned Result test/receipt slice is ready.

## Combined main integration

The coordinator copies only the two production sources and the three test files
into main, preserving the already-integrated grouped interface implementation.
The baseline-bf719aa8 lock is not used. A fresh actual-main snapshot passes
`check` / `check --locked` in **14.346s / 13.388s**, with **758** source files,
**9,594** MIR functions and all **1,230** authored Fairy files plus the new lock
matching. Log: `.cache/flat-result-main-integration-check.log`.
