# Flat Result forwarding: current-source controls, 2026-11-08

**PASS; ready for parent integration as a separate test change.** This is a
focused uncached source/runtime/generated-C gate, not whole-CLI/cache acceptance.
The 68-specialization full-helper input is outside the supported per-unit
capacity. Its refusal is not evidence of a Result lowering defect.

Work ran only in the shared detached worktree
`fairyc/.cache/closure-packages-20261108`, based at
`94eac0fc97e4c7b513313192aa31d2aa466138fe`, with the graph owner's final current
FY sources. Only these three new files belong to this lane:

- `internal/cli/candidate_flat_result_forward_call_test.go`
- `examples/lexer_candidate/tests/composite_result_forward_call.fy`
- This receipt.

No existing production FY file was edited or proposed for overlap approval.
The main worktree's two Result CLI/cache files remained frozen. The parent owns
integration and commits, including the separately reported Go/cache commit
`b63ac743` and graph/lock commit `1adb76a0`; those are not commits made here.

## Why 60 becomes 68, and where 64 is enforced

The unchanged enforcement is in
[`module_unit_generics.fy:39`](../../examples/lexer_candidate/src/module_unit_generics.fy#L39):
the instance scan counts only records with `argument_count != 0` as generic
specializations, and returns an existing identical declaration/argument tuple
before admission of another record. At
[`module_unit_generics.fy:40`](../../examples/lexer_candidate/src/module_unit_generics.fy#L40)
the exact refusal is:

```text
if(v::len<Instance>(instances)>=instance_limit() || generic>=64 || per>=16){return .err(fail(4503,caller,start,end));}
```

The separate retained-body bound is 256 (`instance_limit`, line28); it does not
replace the 64 generic or 16-per-declaration limits. Existing documentation at
[`examples/lexer_candidate/README.md:382`](../../examples/lexer_candidate/README.md#L382)
and lines464-466 records these bounds and the 64-admitted/65-refused boundary.

The scalar seeds in `internal/cli/candidate_generic_cli_test.go:14-22,39-45`
retain seven extended scalars: i8, i16, isize, u16, u64, f32, f64. The concrete
`pick<i32>` main adds i32, making eight T values. In
`internal/cli/candidate_composite_result_cli_cache_test.go:30-58,74-84`:

| Materialized generic family | Count in full baseline |
| --- | ---: |
| success, relay, bind (E=u32), failure, attempt, forward, select | 7 families x 8 T values = 56 |
| pick<i32> | 1 |
| error_value<u16>, error_once<u16>, counted_error<u16> | 3 |
| tail (declared but unused by baseline) | 0 |
| Total | **60** |

The root `failure<i8>` and error-path `failure<u16>` are already among the eight
failure instances. The `.ok`/`.err` variants and scalar observer wrappers are
not extra generic function specializations. The helper mutation at
`candidate_composite_result_cli_cache_test.go:130` changes only
`return replaced;` to `return tail<T>(replaced);` inside relay. It materializes
tail for those same eight T values: **60 + 8 = 68 > 64**. All individual
declarations remain below16 instances. This is a source call-graph derivation;
the observed Go MIR totals are separately 85 and93 functions, including ordinary
functions, and are not the generic-cap counter.

The fixture preserves all baseline functions and scalar vectors. To isolate
the pure compiler path, the existing answer provider and main are combined into
one root module, removing the import and its `a::` qualifier. This is not a
two-package resolution, artifact/cache or CLI test. The resulting three source
hashes are unchanged from the earlier diagnostic controls:

| Input | Go MIR functions | SHA-256 |
| --- | ---: | --- |
| Minimal relay-to-tail forwarding | 3 | `ae4000abcd5902e36a29457ab9ac86b14d1b9bdb402b48921aca78ee3b214dff` |
| Full baseline, all original vectors retained | 85 | `18de6f23b955c31140ad7dce8b0d6ef9904bd1cfa5e5763083454696c850576e` |
| Full helper, retained as capacity-negative input | 93 | `33baf2da0e53f45e9082ba9ba0902555b25de44ad78faba9e067f7968626f1cb` |

## Exact command and gate

Working directory:
`C:/Users/cherr/Documents/workspace/zerodi-wd1/projects/hobby/opensource/fairyc/.cache/closure-packages-20261108`.

```powershell
$env:GOMAXPROCS='1'
$env:GOFLAGS='-p=1'
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
go test -mod=readonly ./internal/cli -run '^TestCandidateFlatResultForwardCall$' -count=1 -parallel=1 -p=1 -timeout=20m -v
```

The test uses `candidateResultNativeGate` from
`candidate_composite_result_transport_test.go:141-148`: Windows/amd64 and
`FAIRYC_RUN_PARITY_INTEGRATION=1` are required. There is no
`FAIRYC_RESULT_CLI_MAIN_GATE` requirement for this focused test and no closure
profile selector; both producer profiles run explicitly. The temporary baseline
type-overlay implementation was removed entirely before this command.

Each exact input is independently checked by Go before native replay. The
current FY driver calls uncached `core::build_complete_sources`, then
`complete_c::partition`, and emits an actual FYCPARTS1 packet. The existing
companion decoder consumes only that emitted C packet. No Go-prepared AST,
cached FY result or frozen old host substitutes for the FY runtime producer.

The current import snapshot has **402 FY inputs**, including the driver, and
each producer uses94 C parts. Existing 256KiB host partitions, 16MiB stack,
one-minute compile bound and two-minute producer execution bound are unchanged.
The snapshot is copied once for both producer profiles and checked against
current authored source bytes again after all executions.

## Observed current-source result

Completed exit0 at 2026-11-08T01:37:15+09:00. Package **200.117s**, test199.94s;
low producer branch57.11s and optimized branch123.58s, including consumers.

| FY producer | Input | Generated C low | Generated C optimized |
| --- | --- | --- | --- |
| low | Minimal forwarding | PASS, exit42 | PASS, exit42 |
| low | Unchanged full baseline | PASS, exit42 | PASS, exit42 |
| optimized | Minimal forwarding | PASS, exit42 | PASS, exit42 |
| optimized | Unchanged full baseline | PASS, exit42 | PASS, exit42 |

All **eight generated C executions** had empty stdout and stderr. The full
baseline retains the extended scalar entries, while its ordinary main exercises
the existing i32 success and u16 error paths; this lane adds no active seven-type
payload observer or Result-by-value foreign ABI claim.

Both full-helper producer executions **PASS as exact capacity negatives**:
controlled exit4, zero stdout/packet bytes, and exactly:

```text
phase=3 stage=3 code=4503 start=638
```

The wrapper returns the original `core::Problem` as a Result, prints its fields
and returns4. Expected capacity refusal no longer deliberately asserts or
aborts. Any other error, span, diagnostic text, exit or emitted packet fails the
negative assertion. Neither positive assertion was removed or relaxed.

Go launcher PID34616; test PID37176. Producer invocation PIDs in vector order
(minimal, baseline, helper): low38700/9884/25828;
optimized38592/32224/45720. Observed descendant creation times and commands are
retained in `.cache/flat-result-forward-current-processes.jsonl`; polling is
not an exhaustive process trace. Cleanup recheck at01:38:00+09:00 found zero
live owned processes. The launcher/test completed naturally; no forced stop.

## Identities and retained evidence

All following cache paths are inside this detached worktree. Retained source,
producer images and packets: `.cache/flat-result-forward-2904059295/`.

| Identity | SHA-256 |
| --- | --- |
| Full log `.cache/flat-result-forward-current.log` | `2f40af8a94c54ee6fd4abdce2553863179df9d03c25d18256bc609b2fd7368b1` |
| `source-inputs.json`, all402 per-file hashes | `858c5dc8252fd8ec4470039110fb80e5a31d58a5d993d87e7ef7a82a3ede83d4` |
| Current module_types.fy input | `3017463724e852e8e73e909e628930e4f9c25522f989b42b6619dffb0e2afa87` |
| Current closure_value_type_nominal.fy input | `c02afe1640718b116ceeb0f1804f96efbc2ba1f010654efb891ec1154f40134b` |
| Current closure_value_type_source.fy input | `4543122303dea1aec08097f03a0d63f781482da5df9bcbda611adf4eadabb495` |
| Low generated compiler C/profile digest | `26eb193d2dead51e59de97e6ffbdedc83f537ecafbf8e767a8d7e775bea327e7` |
| Optimized generated compiler C/profile digest | `bda33c72c6c416b541f7e9ac088ba0e0c760bfdd705b31daa4f27b2da76e887d` |
| Low producer executable | `7ed8664ed6df8bf9394e74bbfae807f4f57c4c06ec404221f354be92f5a74b98` |
| Optimized producer executable | `7d7e44290605b66f827c02175f23ce1c95f9886f1431a9ddfddb51735256a09f` |
| Minimal packet, identical from both producers | `6483a84564003b47c71f905108cedb83f42ee22c647f338cfd9e128bf686c1f3` |
| Baseline packet, identical from both producers | `ad2894d3dd4caf8bc8c1eaefd8ebfc0c005f419c8cefc64aed66e6fc815744dd` |
| Assigned Go test source | `7771d0421e8f9c46a45fe053955cf48a6c261a8939b623f04cd31f58c82bc232` |
| Assigned FY fixture source | `05b9fd1cd40e7ab9903f7bbd831968382d3bad3321432de72076a68ebf5749fb` |

Go test executable identity:
`fairyc.compiler-executable-sha256/v1:721f844b98fa2770db9d2fdadbfe61591bd4f1d64f275acb6b9f0ba6eb783727`.
`.cache/flat-result-forward-current-summary.json` records all eight consumer
image hashes and packet hashes; `.cache/flat-result-forward-current-cleanup.json`
records the empty owned-process set. The separate native stderr log is empty.

## Earlier failed expectation and limits of this receipt

The earlier expectation that the complete 68-instance helper input should pass
was wrong. It remains a recorded failure: `.cache/flat-result-forward-stage-module.log`,
package73.579s, reported phase3/stage3/code4503 and fixture assert-abort exit134.
The original main full-CLI failure log was not changed or promoted to PASS.
The parent explicitly authorized treating the unchanged full-helper input as
capacity-negative after the64 limit and60-to68 derivation were established.

Preliminary controls using the94eac0fc type-owner overlay were diagnostic only,
not current-source acceptance. Their logs and the original first-stage record
`.cache/flat-result-forward-first-stage.json` remain intact. Earlier fixture
setup failures (array slicing without an explicit view, and a missing root
module declaration when joining the inputs) also remain in the separately named
stage logs. The graph owner's then-incomplete visibility error was not repaired
by this lane. The final successful run uses current source without that overlay.

No limit was raised and no existing positive vector was deleted or truncated.
No production compiler patch was required. Linux, whole-host/CLI mutation,
cache/repair/refusal acceptance and broad repository suites were not rerun: they
are outside this bounded gate. This focused pass does not cure or relabel the
unsupported full-helper CLI vector as a supported positive case.

No main source/lock/queue edit, existing shared-test edit, API/ABI, DB, dependency,
runner/CI, commit, push, deployment or release change was made. The parent can
integrate the three frozen files as its separate test commit.

## Coordinator integration

The three frozen files were copied to main and all bytes compared by SHA-256.
The completed current-source native result above was reused without another
200-second execution. `go vet -mod=readonly ./internal/cli` passed on main with
exit 0 and no diagnostics; Go formatting and `git diff --check` also passed.
The previously completed main `check` and `check --locked` results belong to the
unchanged authored source inputs; this integration adds only a test fixture and
its harness/receipt. The source lock did not need another refresh.

This stage establishes the bounded forwarding pass and the exact capacity
refusal. Fairy executable closure support and full Result CLI/cache acceptance
remain separate. No production compiler, language contract, public API/ABI, DB,
dependency, runner, workflow or repository-hygiene policy change was made here.
Linux remains batched; no push, distribution or hosted CI was run.
