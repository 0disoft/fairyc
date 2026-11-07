# Flat Result full CLI and persistent cache — 2026-11-08

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

## Coverage

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

### Final verified stop

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
