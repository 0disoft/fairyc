# Flat Result full CLI and persistent cache — 2026-11-08

> Updated acceptance: [2026-11-14 completion](generic-limit-cli-cache-20261114.md).
> The historical full-helper refusal below exceeded the documented generic cap.

Authority: [ADR 0550](../adr/0550-flat-result-success-generic-transport.md).
Test owner: `internal/cli/candidate_composite_result_cli_cache_test.go`.
Only this receipt and that new test are owned by this task. No production,
source lock, ADR, queue, runner, dependency or ABI change is authorized.

## Execution contract

The parent reported the main-source gate passed: fresh check/locked
14.346/13.388s, 758 sources, 9594 MIR functions, all 1230 Fairy files and lock
exact. These are parent-reported prerequisites, not this task's executed checks.
Host compilation starts only after independent Go admission of the exact fixture.

Use GOMAXPROCS=1, -p=1, -parallel=1, existing host compilation timeout,
256 KiB partitions, 16 MiB host stack and existing per-command timeout.
Native execution is Windows/amd64 only. No Linux, remote action or commit.
Hosts may be reused only after `copiedClosureCLIHost` validates both generated
source digest and image SHA-256 against `host.json`.

## Coverage at the initial stop

The two-package provider exports scalar-generic Result construction, copies,
reassignment, error-parameter binding and propagation. Both success and error
operands of `?` increment a counter; source assertions require one evaluation
each. Explicit extraction/discard handles each Result copy independently.
Ordinary concrete Result wrappers retain all seven success payload types.

The C observer invokes private scalar entries in a freshly captured build
packet. It compares fixed scalar expectations, both copy-selection paths,
floating signed zero and error 77 (78 after edit). It reads only extracted
scalar outputs. It adds no Result-by-value FFI. Ordinary CLI products are built
and executed separately without observer modifications.

| Vector | Low host executed | Optimized host planned; NOT executed |
| --- | --- | --- |
| Cold check, fresh-process locked check/run, byte-stable cache/lock | yes | yes |
| Build/run dev and release; standalone products | yes | yes |
| List + interpreter/native dev/native release test | yes | yes |
| Seven-scalar active success observer + error/count assertions | representative | full |
| Used success-provider invalidation | yes | yes |
| Helper body, success payload type, error value invalidation | omitted | yes |
| Private ordinary body edit, dependent generic key stability | omitted | yes |
| Selected FYCKEYS0 generation corruption and exact repair | yes | yes |
| Must-handle copied Result source refusal; no good-cache mutation | yes | yes |
| Original-copy obligation and E4103 error contract refusal | omitted | yes |

Cache assertions follow actual policy: complete restored generations must hit;
mutations must select a different generation and cannot report a whole-project
hit. Repair may use unit hits, mixed or cold stored outcomes but must restore
the exact good bytes. Only an envelope whose first member is FYCKEYS0 may be
selected for corruption; FYPUNIT0 records remain intact. Restored valid source
refreshes the source lock unlocked before locked-hit assertions.

## Execution ledger

Initial Go compile caught a test-only mismatch comparing int32 native exit code
with int expected exit code. Corrected with an explicit int32 conversion before
running fixture admission. No Fairy host had been launched at that point.

The next admission attempt caught a fixture-only module/path mismatch:
`tests/generics.fy` declared `tests::results`, giving E0002 at bytes 7..21.
Corrected to `tests::generics`; this was before any authored host launch.

Independent Go control command:

```powershell
$env:GOMAXPROCS='1'
go test -mod=readonly ./internal/cli -run '^TestCandidateCompositeResultCLIFixtures$' -count=1 -parallel=1 -p=1 -v
go vet -mod=readonly -p=1 ./internal/cli
```

Controls **PASS**, package **0.596s**, test **0.50s**. Baseline/helper return42,
success/payload return43, error-value/private-body return44. The baseline
interpreter test passes. Both independent-copy refusals report E4101; the
error mismatch reports E4103. Vet **PASS** (empty retained `vet.log`).
Go control executable identity:
`fairyc.compiler-executable-sha256/v1:8ac8290572c1b1fa4f975dac4824101582625296541783a3d9124f3bf5575eee`.

| Input | SHA-256 |
| --- | --- |
| Baseline provider | `94e1660706487328927b18fb6e4b313615b23e0d413dcd5050ce647a5b5e414b` |
| Baseline main | `287053b37c86433ef0fb9e6ecfceaca6e96590289ac7f9ef796344f123d32ea7` |
| Baseline tests | `7f865965141f17bb150d726be4d1a5e2f8c24f7373db6730444f7fab5675baac` |
| Helper provider | `b838041a752ca50d85f7bbe0ea49b71b5723e1b0b32de23b565ebb8c33632141` |
| Success provider | `96ec35a0b29881325a483643a7d94d795bdd2078967074a3e03e81b38573dab1` |
| Payload main | `4712b592f66debb3088998c5aa63e0735a7a43575e83e8990944b310865bbb12` |
| Error-value provider | `14db2bd6466c734917f20e4b82f84307322c704f93a353dae2b85748c9d8c9d8` |
| Private-body provider | `8a2aea26773dbfaa52881dc31d8db5246162029143babd38139f32d721b23efa` |

Full native command, after the parent gate and independent fixture pass:

```powershell
$env:GOMAXPROCS='1'
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
$env:FAIRYC_RESULT_CLI_MAIN_GATE='passed'
$env:FAIRYC_RESULT_CLI_REUSE_ARTIFACTS=Join-Path (Get-Location) '.cache/flat-result-cli-ded379fba8b0'
Remove-Item Env:FAIRYC_CLOSURE_CLI_PROFILE -ErrorAction SilentlyContinue
go test -mod=readonly ./internal/cli -run '^TestCandidateCompositeResultCLICache$' -count=1 -parallel=1 -p=1 -timeout=45m -v
```

Retained native log: `.cache/flat-result-cli-ded379fba8b0/native-attempt1.log`.
The 45-minute test orchestration deadline does not change the existing host or
product command deadlines. Initial low host has 205 C parts. The low outcome is
recorded below. Optimized host compilation was interrupted after the coordinator
redirected scope; no whole CLI acceptance alias is claimed.

Frozen authored input inventory: 1,230 FY files in
`.cache/flat-result-cli-ded379fba8b0/fairy-source-snapshot.json`, SHA-256
`a005e83b31fec93601abcfecb3cd342ce77e0ecb3267dbf43b3109d69b7c2141`.
Main candidate lock SHA-256:
`8af7070e594833433634e2fc86000328853da28cd3058a1b0026e069d6f66261`.

### Low profile: completed

Host preparation **111.1771712s**; full profile **312.1758696s**, including
18 fresh CLI processes and two standalone products. Retained workspace:
`workspace-1670678801` under the evidence directory.

Host input digest:
`46190fc868505491692e8837bdb83f99b47a9708cf9540b7dce7035df80a6b26`.
Host executable SHA-256:
`fb4ef723bbb76be97e95850474e5fc4ae6b2127afabfd126b2cefa0626ba6a67`.

Cold store, byte-stable locked check/release run, dev/release build/standalone
run and CLI run, test list, interpreter/native dev/native release tests all
passed. Baseline release-packet observer passed 26 active payload comparisons
across seven scalars and both copy paths, plus error77 and per-operand counters.

Success-provider edit returned43, selected a new cold-stored project and
changed both package generic keys. Its observer passed 26 comparisons with
second payload selected. Selected-project repair reported `cache: units hit`
and restored every good-cache name/digest exactly. Selected good SHA-256:
`9f60e5aee863b215afd1d36801c325f3bcf22c17095d1ab1af76b9efa9cc96d5`.
Copy-pending refusal returned3 with `cache: cold uncached`, E4101,
`main.fy:29:996`, bytes3336..3344. Good cache remained exact; restored source
hit unlocked and locked. Omitted low vectors in the table were not run.

| Low-host product | SHA-256 |
| --- | --- |
| Baseline dev executable | `c94b748e2609108d3abd7cfd9e3de51b6a58c45558a0869e701ad4e69f522737` |
| Baseline release executable | `7f749712c2dde63b559205d896b1b1492e0c831a5263504d7c54aac519523cd5` |
| Baseline observer executable | `a3224b1573da1187784162d13ff633ed7b997d9dc6adb98fc43064a5823699ce` |
| Mutated-success observer executable | `51d88b9e274a9e6b3241d77db6fe0aece2de94c3fa91950438488cbdeafb7a03` |

### Initial verified stop

At 2026-11-08T00:30:44+09:00 the coordinator redirected work to Fairy structural
type retention only and requested a prompt stop without new runtime admission.
This worker owns the Result CLI/cache test and receipt only; it made no Fairy
structural-retention edits and did not copy files from the closure worktree.
Combined-main source lock refresh and commits remain coordinator-owned.

After verifying its exact Result test command, the owned Go process27084 and
its test/compiler descendants were terminated. Native session7458 closed exit1.
The retained native log contains the complete passing low profile and the
optimized host's 205-part compilation start. Optimized compilation had not
published a host image or `host.json`. There was no native assertion failure
before this deliberate stop. **The whole native test is not a pass.**

No optimized runtime vectors were executed. Helper, payload-type, error-value,
private-body and additional refusal vectors have independent Go fixture
evidence only; their authored CLI/cache matrix remains unverified. No new
runtime admission, executable closure MIR/backend, bound change, source lock
refresh or commit was performed. No alias/whole-CLI acceptance claim is made.

Frozen-input recheck after low completion found **0 changes across all 1,230
FY files**; the source lock digest also remained exact. The two assigned new
files are the entire worker change set. Test source SHA-256:
`4373b57c1f8f715814a0425fd6c73aaba64a53179654222e37cc2d75e4ddbe16`.

Earlier cheap-test failures are retained as explicitly transcribed tool output
in `cheap-failures-transcribed.log`. Native log and completed low host,
workspace/cache and packet artifacts remain in the evidence directory. Vet
session63403 completed exit0; no owned live process/session remains.

## Optimized continuation at current main 94eac0fc

**Observed result: FAIL; whole optimized CLI/cache acceptance is not passed.**
This continuation ran only on Windows/amd64 main
`94eac0fc97e4c7b513313192aa31d2aa466138fe`. The earlier low-host 18-process
completion remains historical evidence and was not rerun. Low/dev products
below are consumers built by the new optimized host, not a second low-host run.

### Independently authenticated admission

At 2026-11-08T01:01:45+09:00, every current FY path and SHA-256 under
`examples/lexer_candidate` was compared with the snapshot named by
`.cache/closure-value-types-source-20261107-all-check.json`. Both inventories
contained **1,231 files with zero differences**; manifest and lock also matched
exactly. That receipt records successful check/locked exits0, empty stderr,
16.0281113/19.4012367s, 758 compiled sources and 9594 MIR functions. Those check
times belong to the prior receipt; this continuation authenticated its exact
inputs rather than rerunning a source checker or refreshing the main lock.
Only after this comparison and the independent Go fixture pass was the existing
`FAIRYC_RESULT_CLI_MAIN_GATE=passed` set for the native child.

| Authenticated input | SHA-256 |
| --- | --- |
| Prior combined check receipt | `bff5ddcb1fa1bb11b94975c41bdec7763cff5042ec0f469b6643ca9131f9f19e` |
| Current and snapshot fairy.lock | `be58b365a33198123325fbe0be9d1285e795e08a9434bcf7e2dcc36f68d83f72` |
| Current and snapshot fairy.toml | `5dd473fe42d35270b362d16d383627167d9981244761a925e42a45f57c8b737f` |
| New per-file FY authentication JSON | `7a8d89be145eda87cc1530ec3ec884596fac0527b64e54f300698f160c5c33e0` |
| Pre-run Go input inventory (2759 files including go.mod/go.sum) | `9dca94d97ee682153ae9b64cbd8c43a0c84da1b8f89f282843d6ca9df071a9f7` |

All new evidence paths below are relative to
`.cache/flat-result-cli-ded379fba8b0/`. Full FY hashes are retained in
`optimized-current94eac0fc-source-authentication.json`; Go hashes in
`optimized-current94eac0fc-go-inputs.json`.

The current Go fixture control passed before native launch: package0.680s,
test0.56s, Go PID30572, compiler/test executable identity
`fairyc.compiler-executable-sha256/v1:6d66b5c36ab34627d6fac20d303d6d39d458beffdc155b996985bb81a8a7f982`.
Provider/main/tests and five edit hashes match the earlier hash table. All six
expected return values and three refusal controls passed. Log:
`optimized-current94eac0fc-fixtures.log`.

### Fresh optimized host and failed full test

Initial process inspection found no live Go/test/compiler process for the
interrupted parent attempt. Its optimized directory had neither a host image nor
`host.json`; it was not accepted as reusable. `copiedClosureCLIHost` freshly
generated the current C digest, then built all205 C partitions without changing
its 256KiB partition limit, timeout or 16MiB stack reserve. Preparation took
**4m23.8716796s**.

| Host identity | SHA-256 |
| --- | --- |
| Generated optimized C input digest | `f78071eb637fe44f7b5c771233d5cd5250bc829bf3cf39c8ad5013e152daa893` |
| Optimized host executable | `788b13e8038bfd41140c47b1f5b2911c7a80291e949826a806172bbe0b00a20b` |

Exact native command and environment:

```powershell
$env:GOMAXPROCS='1'
$env:GOFLAGS='-p=1'
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
$env:FAIRYC_RESULT_CLI_MAIN_GATE='passed'
$env:FAIRYC_RESULT_CLI_REUSE_ARTIFACTS=Join-Path (Get-Location) '.cache/flat-result-cli-ded379fba8b0'
$env:FAIRYC_CLOSURE_CLI_PROFILE='optimized'
go test -mod=readonly ./internal/cli -run '^TestCandidateCompositeResultCLICache$/optimized$' -count=1 -parallel=1 -p=1 -timeout=45m -v
```

Go PID46252 launched at01:02:38+09:00; test PID16444 launched at01:02:43+09:00.
The command completed exit1 at01:07:24+09:00, package280.664s,
optimized subtest280.33s. The test performed **13 fresh CLI commands** before
stopping. The exact Go executable identity was the one reported above.
`optimized-current94eac0fc-processes.jsonl` records observed descendant PIDs,
creation times and command lines; polling is not an exhaustive process trace.

Preserved failure workspace: `workspace-1212925990`.
Preserved full native log: `optimized-current94eac0fc-native-attempt2.log`, SHA-256
`3eac26bd0db06a11e9edc97d2b7c661fa1685300a14dea1bd2e067b2967d165f`.
Its separate stderr log is empty; the test's captured compiler stderr is in the
full log. Earlier interrupted/failed logs remain intact.

| Native vector under the optimized host | Observed outcome |
| --- | --- |
| Baseline cold store, locked check/release run; exact cache/lock bytes | PASS |
| Dev/release builds, two standalone products and CLI runs | PASS, exit42 and empty streams |
| Baseline low/optimized packet observers | PASS, 26 active-payload comparisons each, both copies, error77 and operand counters |
| Test listing, interpreter, native dev and native release tests | PASS |
| Helper mutation stale locked check | PASS, exit7 and exact stale-lock refusal |
| Helper mutation ordinary run | FAIL, exit4 instead of42, cold uncached, internal compiler invariant failed |

Both baseline standalone product hashes equal the earlier low-host product
table. Baseline packet SHA-256:
`0f175d5c0ec45a24ff7ee3d74d6dcb399d074b3aa0dd54b95fe45c8af99e5535`.
New low observer SHA-256:
`2d1c4a9ea375a6880d2c4b981024b1a683d1bd68a1f62cb69648ead7c29ba2cf`;
optimized observer SHA-256:
`a3224b1573da1187784162d13ff633ed7b997d9dc6adb98fc43064a5823699ce`.

### Remaining matrix observations and actual fixture correction

To preserve the failed workspace, supplementary commands used a full separate
copy, `optimized-diagnostics-fb259731`. No fixture was truncated and no compiler
budget was raised. Exact command PIDs, source hashes, exits, streams and timings
for **21 additional CLI commands** are in
`optimized-current94eac0fc-supplemental.jsonl`.

The helper change is exactly `return replaced;` to
`return tail<T>(replaced);` inside `relay<T>`. Uncached ordinary `check` also
returned4 with the invariant failure, PID16904. The unchanged Go control accepts
and executes that exact source. The production cause was not isolated or fixed
in this lane; replacing that valid helper mutation to conceal the failure would
not establish the requested acceptance.

| Supplementary vector | Observed outcome |
| --- | --- |
| Success, payload type, error value and private body mutations | All stale-lock checks exit7; all ordinary runs exit4 with the same invariant message |
| Good-cache preservation after all failed mutations | Exact names and SHA-256 bytes preserved |
| Copy-pending and original-pending at final private-body mutation | Both exit4, not their expected E4101 |
| Original error-contract fixture | Exit3 unsupported f32 signature, not E4103, on both final mutation and baseline |
| Corrected error-contract at final private-body mutation | Exit4, invariant failure |
| Restore baseline unlocked check | PASS, cache hit, lock refreshed only in scratch workspace |
| Corrupt selected baseline FYCKEYS0 project envelope and repair | PASS, exit42, cache: units hit, every good cache name/digest restored exactly |
| Baseline copy-pending and original-pending | PASS, exit3/E4101, good cache exact |
| Corrected baseline error-contract | PASS, exit3/E4103, good cache exact |
| Restored baseline unlocked and locked checks | PASS, cache hit and good cache exact |

The initial error-contract helper used `Result<f32,i32>` as its enclosing return
type. ADR0550 explicitly excludes this shape; the existing transport control
already uses `Result<u32,i32>` to reach propagation checking without invoking
that independent signature refusal. The **only test edit** therefore changes
the enclosing type to `Result<u32,i32>`, discards the extracted f32 and returns
`.ok(1)`. It preserves the f32 operand and the expected E4103. No helper mutation
or acceptance expectation was weakened.

Corrected baseline E4103: main.fy:29:998, bytes3338..3339; actual-error
bytes3306..3309, expected-error bytes3281..3284. Corrected final-mutation source
still fails before this diagnostic can be established. Baseline E4101 spans are
bytes3336..3344 (copied) and3360..3368 (original).

The repaired baseline selected-project SHA-256 is
`4ecd038293d3b9ba35c3c8866f8fe718fb2b8c8c49a1728413deaaf85739b931`.
This is **baseline repair evidence only**. There is no successful mutated
generation to repair. Mutated generic-key stability/invalidation, warm-byte
checks, changed-payload/error78 observers and the final mutated repair/refusal
sequence remain unverified; the original full test did not reach them.

After the one-line fixture fix, the same focused Go fixture command passed again:
package0.585s, test0.49s. Corrected executable identity:
`fairyc.compiler-executable-sha256/v1:c394029e52d9af7922b771fd3060c033f1333a9b8e3276403ef7d94c898e8cec`.
`go vet -mod=readonly -p=1 ./internal/cli` passed with an empty retained log.
The full native test was not rerun after this diagnostic-only correction because
its earlier helper failure and all host inputs were unchanged.

### Final ownership and remaining gate

Final FY comparison at01:11:25+09:00 found **zero changes across all1231 files**;
main lock and manifest hashes remained exact. Comparing the recorded2759 Go
inputs found only the assigned test changed. Its final SHA-256 is
`c9f63217f543e5f82c76a3e248060e43f4e0a4ced1fedd1a8ff68b562b8e93c7`.
`optimized-current94eac0fc-final-inputs.json` retains that comparison;
`optimized-current94eac0fc-evidence-summary.json` retains evidence-log hashes.
Tracked changes are only the assigned test and this receipt. No FY, source lock,
production implementation, API/ABI, DB, other shared test, dependency, runner,
CI, limit, branch, commit or remote state was changed. Linux remains batched.
Final scoped `git diff --check` passed.

The native handle completed exit1 naturally; no duplicate run or forced host
stop was needed. Recorded PID/creation-time recheck at01:11:30+09:00 found
**zero live owned processes**, retained in
`optimized-current94eac0fc-process-cleanup.json`. The remaining acceptance gate
requires the parent/compiler owner to repair the valid helper-call invariant,
then replay the full optimized test using current-source/digest admission.
