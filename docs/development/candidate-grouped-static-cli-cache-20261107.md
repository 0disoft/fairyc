# Grouped static full CLI / persistent-cache proof — 2026-11-07

Scope: isolated checkout
`C:/Users/cherr/Documents/workspace/zerodi-wd1/projects/hobby/opensource/fairyc/.cache/grouped-static-20261107`.
Baseline HEAD: `c5b6c9b430037de0b57cf1d0bc53a3fd520aa7f1` plus the coordinator's
frozen grouped source. This worker owns only this receipt and
`internal/cli/candidate_grouped_static_cli_cache_test.go`. No production, lock,
queue, ADR, existing test, main-root file, or commit is owned here.

## Prerequisite and scoped checks

Coordinator explicitly reported the grouped pipeline gate passed: attempt3,
low/optimized producers, all four consumers exit42, 227.778s; the independent
24-case admission and whole-source check/locked gates also passed. These are
coordinator-reported prerequisites, not executions owned by this receipt.

All local commands use `GOMAXPROCS=1` and existing host/product bounds.

```powershell
go test -mod=readonly ./internal/cli -run '^TestCandidateGroupedStaticCLIFixtures$' -count=1 -parallel=1 -p=1 -v
go vet -mod=readonly -p=1 ./internal/cli
```

Go controls: **PASS**, package 0.326s. Vet: **PASS**, no output.
Final `gofmt -l` is empty; both new files have no whitespace diagnostics from
`git diff --no-index --check` (exit1 means new-file differences from `/dev/null`).
Post-run image SHA-256 checks match both recorded host receipts exactly.
The fixtures cover baseline and four cumulative equal-width edits, independent
Go interpreter values, selected specialization names, and three unused-i64
provider refusals. No large Fairy host was launched before the prerequisite
notification and these cheap controls.

Native invocation, after the coordinator notification:

```powershell
$env:GOMAXPROCS='1'
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
$env:FAIRYC_GROUPED_STATIC_CLI_GATE='passed'
$env:FAIRYC_CLOSURE_CLI_PROFILE=''
go test -mod=readonly ./internal/cli -run '^TestCandidateGroupedStaticCLICache$' -count=1 -parallel=1 -p=1 -timeout=30m -v
```

The 30-minute Go test watchdog covers two sequential full-host builds and all
CLI subprocesses; the helper's host command timeout, stack reserve, 256KiB
partition limit, ordinary artifact bounds and strict warnings are unchanged.

## Fixture and assertions

The library declares `Arithmetic` with public `add` and `multiply` operations,
complete i32/i64 provider bundles, an exported generic `algorithm<T>` selecting
only add, and an ordinary exported function calling the i32 provider directly.
Neither main nor tests requests multiply or any i64 specialization. The test
module includes ordinary and generic calls plus a once-only argument evaluation
check. Main returns 42, or 44 after the used-provider edit.

| Vector | Main result | Expected stable package generic keys out of two |
| --- | --- | --- |
| Baseline | 42 | Initial ledger |
| Used add/i32 body: offset 0 to 2 | 44 | 0 |
| Unused multiply/i32 body: offset 0 to 1 | 44 | 0 |
| Unused multiply/i64 body: offset 0 to 1 | 44 | 0 |
| Unrelated private body: return 1 to 2 | 44 | 1 |

Expected values are authored vectors. The test reads actual canonical generic
key members from the stored project ledger and compares their hashes; it does
not recreate production key construction. The Go fixture control verifies only
add/i32 and algorithm/i32 are specialized.

Whole-project records and unit records share both the `fyproject0-*.fyc` name
family and `FYCACHE0` envelope. Project selection checks the actual first member
marker `FYCKEYS0` at byte 16; it never selects a `FYPUNIT0` unit record. Ledger
lengths/counts follow the existing big-endian u64 wire format. The fixture has
exactly two package rows.

The native matrix actually established the following, with per-profile scope
and interruption boundaries recorded below:

- Full authored low and optimized compiler hosts, through existing
  `copiedClosureCLIHost` / `copiedClosureCommand` helpers.
- Cold check, fresh-process locked check/run hits, byte-stable cache generations
  and lockfile, dev/release build/run and standalone executable execution.
- Interpreter, native dev and native release test dispatch, with one passed test.
- Group dependency closure after each equal-width edit; unrelated private edit
  keeps the dependent generic key and uses the existing mixed-cache policy.
- Corruption of only the newly selected whole-project generation, accepting
  unit-hit/mixed/cold-stored recovery and requiring exact good bytes restored.
- Partial unused-i64 bundle, duplicate unused tuple and wrong unused signature
  refusal without changing the good cache. Restoring valid source first runs
  unlocked check to refresh the source lock while requiring the preserved hit,
  then checks locked byte stability again.

No exact historical generation count or blanket two-package miss count is
asserted for the authored cache. Unit semantic admission may yield mixed reuse
after a group edit. Byte comparisons occur only in a warm/refusal/repair window
for this bounded fixture and the existing retention policy.

## Source identities

SHA-256 values of exact fixture bytes:

| Input | SHA-256 |
| --- | --- |
| Initial owned test file | `a1d6399226c36b1305608d37557b11a4cba9712db278aa995eaf71cd8473c88e` |
| Revised owned test file after cost steering | `0887c6e19a0e4fb71060d26dfe2316a6ab89cc12e8acfdd6c8af59be3a20e0f5` |
| Root manifest | `ffe0ede347076b52256c79d5e60c3451ec5ca8254aa26cd7f2cf901605519479` |
| Provider manifest | `47afc2e193d070ee7a8d21b5e373e232157976f8de6617fc2761ed241d3db4a3` |
| Provider baseline | `9ad916ade2e242811cc310353e743c81bd24dc73cc1f5e7fb8f4ac763efa18db` |
| Main | `63f0c56002775c39a912a0be7e60d09f4523c8d639b45b83d76b3afdcb9e2be1` |
| Test module | `7cb5152cf61d8eb6204fb617cecf7a81f3e3e8960c74b14446fa0a629dcbc368` |
| Used-provider edit | `d5a6627ffcdb835bc50bccbc45bff75ddcdfd73a0666804b507b43d0307d22be` |
| Unused sibling edit | `e776e43d79daced1ca0f5738a2af29b9e04acb88dd73ba11e0a84d7b03d5324b` |
| Unused i64 tuple edit | `c320efb7fd1a55e409586cbbdb4397d1c52fe1e4afbd3f8d3dfa7033c0356747` |
| Private body edit | `73339e6e644f42375d3cc59e1cc548ad3a178ba51d0fa99f4dcf1f2cd2ab14a2` |
| Partial unused i64 | `9c7018ce9e1d8542ebda6fcb79cbd19fb13d6014e174d28b1bc17c6ef74084d7` |
| Duplicate unused i64 | `492dcb9f52b2155446a4e23d05868a287bca9fb9a03b299893337d221da1fd09` |
| Wrong unused i64 signature | `edeb27fa1f53cae6968a4e4af8cf9043e0e69f23801a07394a8fbb3c7954a7bd` |

Go control executable:
`fairyc.compiler-executable-sha256/v1:c9ef975cd968cf7e16439794eaef9a5de5e5d4ca3bcefaaeeac4f24866bfd53b`.

## Native attempts

Attempt1 started after the gate, at 2026-11-07 23:51 KST. Unique retained directory:
`.cache/grouped-static-cli-3025737042` under this isolated checkout. Host receipts
record SHA-256 of generated C/support/profile inputs and exact executable images.
Native outcome: **PASS**, completed on 2026-11-08 KST. The original low run was
interrupted for explicit cost steering; its completed assertions and resumed
tail are distinguished below. Preparation is not counted as execution.

### Cost steering and interrupted attempt1

At the user's speed steering, attempt1 was explicitly stopped by terminating
only its verified Go process tree (Go PID 8624, test PID 23696). Test process
start was 23:51:57 KST; termination was 23:57:57 KST, approximately 360 seconds.
The low image was written at 23:53:55.944, so it was ready approximately 118
seconds after test-process start; this timestamp is not a separate compilation
stopwatch measurement. No product assertion had failed at interruption.

Before termination, low completed baseline check/build/run/list/test in all
three test modes, both standalone products, and all four mutation/ledger
assertions. It was interrupted during the private-edit warm tail. Its repair
and refusal assertions had not executed; optimized had not started. The retained
output below is the exact observed attempt1 output, not a completed-suite pass.

The revised default low matrix keeps baseline + representative used-provider
edit + selected project repair + one partial-unused tuple refusal. Optimized
retains every mutation/warm/refusal vector. For this run only, explicit resume
arguments select the already-verified private-edit low workspace and selected
FYCKEYS0 generation, so baseline and mutations are not repeated. The host helper
still regenerates/hash-checks its C input and verifies the retained image hash.
No Fairy source, C image, host timeout, bounds or product behavior was changed.
The revised Go control passed again in 0.335s, executable
`fairyc.compiler-executable-sha256/v1:4fe0219ee35b2792eb3530d7ee1fa314ec62f4dddd9c815b7122c579e72829fc`.
Revised vet also passed (exit0, no output).

Resumed low tail passed in **98.6030204s**, including **29.2552818s** to
regenerate/hash-check the unchanged host inputs and rebuild its ordinary C-only
companion. Six fresh low CLI processes covered locked reuse, unit-hit project
repair, partial-unused-i64 refusal, unlocked source-lock refresh and a final
locked hit. Cache bytes matched the good snapshot after repair and refusal.
Combined with the interrupted run, approximately 459s of low-profile test time
was spent; this is not a single uninterrupted subtest duration.

Low host generated-C input digest:
`a625d6aab50b82b9e3d7ddb66414d232252da991ddb896179ef1fca664e107d0`.
Low host image SHA-256:
`42ab29abb7adb73e7cf32b04b3568ec433c4aa6428dc05684422d63b96515b1b`.
The resumed helper independently confirmed both exact digests before reuse.

```text
=== RUN   TestCandidateGroupedStaticCLICache
    candidate_grouped_static_cli_cache_test.go:414: unique retained host/workspace artifacts=C:\Users\cherr\Documents\workspace\zerodi-wd1\projects\hobby\opensource\fairyc\.cache\grouped-static-20261107\.cache\grouped-static-cli-3025737042
=== RUN   TestCandidateGroupedStaticCLICache/low
    candidate_grouped_static_cli_cache_test.go:417: full authored CLI low: 205 C parts
    candidate_grouped_static_cli_cache_test.go:422: host profile=low identity={"Source":"a625d6aab50b82b9e3d7ddb66414d232252da991ddb896179ef1fca664e107d0","Image":"42ab29abb7adb73e7cf32b04b3568ec433c4aa6428dc05684422d63b96515b1b"}
    candidate_grouped_static_cli_cache_test.go:423: retained workspace=C:\Users\cherr\Documents\workspace\zerodi-wd1\projects\hobby\opensource\fairyc\.cache\grouped-static-20261107\.cache\grouped-static-cli-3025737042\workspace-2279767119
    candidate_grouped_static_cli_cache_test.go:423: fairy.toml sha256=ffe0ede347076b52256c79d5e60c3451ec5ca8254aa26cd7f2cf901605519479
    candidate_grouped_static_cli_cache_test.go:423: libs/answer/fairy.toml sha256=47afc2e193d070ee7a8d21b5e373e232157976f8de6617fc2761ed241d3db4a3
    candidate_grouped_static_cli_cache_test.go:423: libs/answer/src/answer.fy sha256=9ad916ade2e242811cc310353e743c81bd24dc73cc1f5e7fb8f4ac763efa18db
    candidate_grouped_static_cli_cache_test.go:423: src/main.fy sha256=63f0c56002775c39a912a0be7e60d09f4523c8d639b45b83d76b3afdcb9e2be1
    candidate_grouped_static_cli_cache_test.go:423: tests/grouped.fy sha256=7cb5152cf61d8eb6204fb617cecf7a81f3e3e8960c74b14446fa0a629dcbc368
    candidate_grouped_static_cli_cache_test.go:423: CLI [check --cache-root cache --cache-status] => 0
    candidate_grouped_static_cli_cache_test.go:423: selected project=fyproject0-b28c73824363b7065d4ef150f97f2e17987c8d922e0308cf29b9ef5485c61c7a-add6e1ea537d7c8d3d20831e1090b0c72c49f58903efc62c8f7a8608f90cab0b.fyc sha256=add6e1ea537d7c8d3d20831e1090b0c72c49f58903efc62c8f7a8608f90cab0b ledger-sha256=68bf8b831697b1a7bf0fe72e099ce9c3976f79ba8abf5110ff066c93cfcb822e
    candidate_grouped_static_cli_cache_test.go:423: CLI [check --locked --cache-root cache --cache-status] => 0
    candidate_grouped_static_cli_cache_test.go:423: CLI [run --release --locked --cache-root cache --cache-status] => 42
    candidate_grouped_static_cli_cache_test.go:423: CLI [build --locked --cache-root cache --cache-status] => 0
    candidate_grouped_static_cli_cache_test.go:423: product flags=[] sha256=22ce6372a9d664540cadfa488cd3b8a081e9f6fc8ec17a3618384cabdf0142a5 exit42 empty streams
    candidate_grouped_static_cli_cache_test.go:423: CLI [run --locked --cache-root cache --cache-status] => 42
    candidate_grouped_static_cli_cache_test.go:423: CLI [build --locked --release --cache-root cache --cache-status] => 0
    candidate_grouped_static_cli_cache_test.go:423: product flags=[--release] sha256=5c419dca9f41c88766e47b4901d32a50a2874cd6b7b7c11f5fe239fda13e3138 exit42 empty streams
    candidate_grouped_static_cli_cache_test.go:423: CLI [run --locked --release --cache-root cache --cache-status] => 42
    candidate_grouped_static_cli_cache_test.go:423: CLI [test --list --json --locked] => 0
    candidate_grouped_static_cli_cache_test.go:423: CLI [test --json --locked] => 0
    candidate_grouped_static_cli_cache_test.go:423: CLI [test --json --locked --native] => 0
    candidate_grouped_static_cli_cache_test.go:423: CLI [test --json --locked --native --release] => 0
    candidate_grouped_static_cli_cache_test.go:423: edit=used-provider provider-sha256=d5a6627ffcdb835bc50bccbc45bff75ddcdfd73a0666804b507b43d0307d22be
    candidate_grouped_static_cli_cache_test.go:423: CLI [check --locked --cache-root cache --cache-status] => 7
    candidate_grouped_static_cli_cache_test.go:423: CLI [run --cache-root cache --cache-status] => 44
    candidate_grouped_static_cli_cache_test.go:423: selected project=fyproject0-a0648ec9ae3ef01c3175bd6a29f9a73d197aeda0c336fb4eea1800357fa33d81-e9aea61e51d5ac460fb543edb887b450950b96d1d68fd38124a1b65b65e7f75f.fyc sha256=e9aea61e51d5ac460fb543edb887b450950b96d1d68fd38124a1b65b65e7f75f ledger-sha256=afaa211c060dc61d92dac4df081e626063a2e47241b3f788ada8eff1075a5078
    candidate_grouped_static_cli_cache_test.go:423: edit=used-provider cache=cache: mixed stable-package-generic-keys=0/2
    candidate_grouped_static_cli_cache_test.go:423: CLI [check --locked --cache-root cache --cache-status] => 0
    candidate_grouped_static_cli_cache_test.go:423: CLI [run --release --locked --cache-root cache --cache-status] => 44
    candidate_grouped_static_cli_cache_test.go:423: edit=unused-sibling-provider provider-sha256=e776e43d79daced1ca0f5738a2af29b9e04acb88dd73ba11e0a84d7b03d5324b
    candidate_grouped_static_cli_cache_test.go:423: CLI [check --locked --cache-root cache --cache-status] => 7
    candidate_grouped_static_cli_cache_test.go:423: CLI [run --cache-root cache --cache-status] => 44
    candidate_grouped_static_cli_cache_test.go:423: selected project=fyproject0-09784b66c9631946085e216770588d9dcc20c942b60d4ac8ba5cafc06f7ecb3e-0bed6574b7ac30b2bf12b23caab3038885337c564bbb39b5b0cbc1d9e3b4f803.fyc sha256=0bed6574b7ac30b2bf12b23caab3038885337c564bbb39b5b0cbc1d9e3b4f803 ledger-sha256=9c85c98bafc626d980535f5b2407d46c67d90373c9e98cdd3cb66e0df7d10fc0
    candidate_grouped_static_cli_cache_test.go:423: edit=unused-sibling-provider cache=cache: mixed stable-package-generic-keys=0/2
    candidate_grouped_static_cli_cache_test.go:423: CLI [check --locked --cache-root cache --cache-status] => 0
    candidate_grouped_static_cli_cache_test.go:423: CLI [run --release --locked --cache-root cache --cache-status] => 44
    candidate_grouped_static_cli_cache_test.go:423: edit=unused-i64-provider-tuple provider-sha256=c320efb7fd1a55e409586cbbdb4397d1c52fe1e4afbd3f8d3dfa7033c0356747
    candidate_grouped_static_cli_cache_test.go:423: CLI [check --locked --cache-root cache --cache-status] => 7
    candidate_grouped_static_cli_cache_test.go:423: CLI [run --cache-root cache --cache-status] => 44
    candidate_grouped_static_cli_cache_test.go:423: selected project=fyproject0-eddaf7d73542c141a500e0d7dcf2c6476b43f6d8c730840a961891cd3dad187f-fa996d1b861681b7f2d00ae5f215e96c4b8bf5abb171b4a712c4e31c5ec76825.fyc sha256=fa996d1b861681b7f2d00ae5f215e96c4b8bf5abb171b4a712c4e31c5ec76825 ledger-sha256=2a7c03c2993ce833baafce9c7b9bfee5f224aa2771cfc1bef8fe4d3f506a8306
    candidate_grouped_static_cli_cache_test.go:423: edit=unused-i64-provider-tuple cache=cache: mixed stable-package-generic-keys=0/2
    candidate_grouped_static_cli_cache_test.go:423: CLI [check --locked --cache-root cache --cache-status] => 0
    candidate_grouped_static_cli_cache_test.go:423: CLI [run --release --locked --cache-root cache --cache-status] => 44
    candidate_grouped_static_cli_cache_test.go:423: edit=private-body provider-sha256=73339e6e644f42375d3cc59e1cc548ad3a178ba51d0fa99f4dcf1f2cd2ab14a2
    candidate_grouped_static_cli_cache_test.go:423: CLI [check --locked --cache-root cache --cache-status] => 7
    candidate_grouped_static_cli_cache_test.go:423: CLI [run --cache-root cache --cache-status] => 44
    candidate_grouped_static_cli_cache_test.go:423: selected project=fyproject0-3063c05f2706fe1fd93c7a57f6979a2102ea2827783b2033c2738bbfc6b6c8fe-6ad0b55e8be5d203d4927c11b424e7191333eb545cab54f667a0325d9637e561.fyc sha256=6ad0b55e8be5d203d4927c11b424e7191333eb545cab54f667a0325d9637e561 ledger-sha256=35e236dd1244b32977c7e4c16eba9f5d12fc714095ff7ed63af05705c9306e8c
    candidate_grouped_static_cli_cache_test.go:423: edit=private-body cache=cache: mixed stable-package-generic-keys=1/2
```

### Attempt2 completion and exact per-profile coverage

The second command used the same gate/profile settings and test command above,
plus these explicit retained-artifact resume settings:

```powershell
$env:FAIRYC_GROUPED_STATIC_CLI_REUSE_ARTIFACTS=(Join-Path (Get-Location) '.cache/grouped-static-cli-3025737042')
$env:FAIRYC_GROUPED_STATIC_CLI_RESUME_LOW=(Join-Path $env:FAIRYC_GROUPED_STATIC_CLI_REUSE_ARTIFACTS 'workspace-2279767119')
$env:FAIRYC_GROUPED_STATIC_CLI_RESUME_PROJECT='fyproject0-3063c05f2706fe1fd93c7a57f6979a2102ea2827783b2033c2738bbfc6b6c8fe-6ad0b55e8be5d203d4927c11b424e7191333eb545cab54f667a0325d9637e561.fyc'
```

Second command: **PASS**, exit0, package **368.565s**. Reported subtests:
low **98.81s**, optimized **269.44s**. Internal profile timers were
**98.6030204s** and **269.2278411s**, respectively. Optimized host preparation
took **236.5397612s**, leaving approximately **32.688s** for the full acceptance
matrix. No assertion failed in either CLI attempt; the first process exit1 was
the deliberate cost-steering termination, not a product failure or timeout.

| Evidence | Low | Optimized |
| --- | --- | --- |
| Cold check; fresh-process locked byte-stable check/run | Passed in attempt1 | Passed in attempt2 |
| Dev/release build, standalone exit42, run exit42 | Both passed in attempt1 | Both passed in attempt2 |
| Test list; interpreter; native dev; native release | All passed in attempt1 | All passed in attempt2 |
| Used-provider edit, both generic keys changed, result44 | Passed in attempt1 | Passed in attempt2 |
| Unused sibling/i32 edit, both keys changed, result44 | Passed in attempt1 before steering took effect | Passed in attempt2 |
| Unused i64 provider edit, both keys changed, result44 | Passed in attempt1 before steering took effect | Passed in attempt2 |
| Private edit, one key stable, result44 | Key comparison passed in attempt1; ordinary warm tail was interrupted; resumed locked check passed | Comparison and complete warm cycle passed |
| Selected project repair, exact good bytes restored | Resumed tail, units hit | Passed, units hit |
| Partial unused i64 refusal, good cache unchanged | Resumed tail | Passed |
| Duplicate unused tuple / wrong unused signature refusal | Not run on low | Both passed, good cache unchanged |
| Restore valid source and require byte-stable locked reuse | Resumed locked check | Complete locked check/run cycles |

Optimized completed **45 fresh authored CLI processes + 2 standalone products**.
The low resumed tail completed six processes; the interrupted low log owns the
earlier baseline and mutation evidence. The default low test now stops after
the used-provider vector and one partial-bundle refusal. No claim is made that
the revised reduced low default was rerun from cold; existing valid evidence
was reused at the user's explicit speed steering.

Retained paths under `.cache/grouped-static-cli-3025737042`:

- `low/fairyc.exe`, `low/host.json`; workspace `workspace-2279767119`.
- `optimized/fairyc.exe`, `optimized/host.json`; workspace `workspace-3685712951`.

Optimized host generated-C/support/profile input digest:
`e8fd125d83200d0f195c5562128203b4f1c44382ce68447ee93275b2c3ee8a04`.
Optimized host executable SHA-256:
`81a1c5fc24a756dbf4066dc02cdd1a7d3882b2c26a9a3782a6c69890c79d4871`.
These `Source` values are the existing host helper's exact NUL-separated
profile/header/support/C-parts digest, not a hash of only the original Fairy
text. Both hosts used 205 C parts.

Both hosts' baseline generated products executed with empty output and matched
these hashes in this run (no general reproducibility claim):

- Dev executable: `22ce6372a9d664540cadfa488cd3b8a081e9f6fc8ec17a3618384cabdf0142a5`.
- Release executable: `5c419dca9f41c88766e47b4901d32a50a2874cd6b7b7c11f5fe239fda13e3138`.

Selected repaired whole-project digests:

- Low: `6ad0b55e8be5d203d4927c11b424e7191333eb545cab54f667a0325d9637e561`.
- Optimized: `1b0633aaf1f0278522f981580951636c0567f78e26347dc5f1b3810dd025ad0b`.

Native test executable:
`fairyc.compiler-executable-sha256/v1:4fe0219ee35b2792eb3530d7ee1fa314ec62f4dddd9c815b7122c579e72829fc`.
Selected exact final-run output:

```text
    candidate_grouped_static_cli_cache_test.go:483: host profile=low resolve-elapsed=29.2552818s identity={"Source":"a625d6aab50b82b9e3d7ddb66414d232252da991ddb896179ef1fca664e107d0","Image":"42ab29abb7adb73e7cf32b04b3568ec433c4aa6428dc05684422d63b96515b1b"}
    candidate_grouped_static_cli_cache_test.go:485: selected-generation repair=cache: units hit selected=fyproject0-3063c05f2706fe1fd93c7a57f6979a2102ea2827783b2033c2738bbfc6b6c8fe-6ad0b55e8be5d203d4927c11b424e7191333eb545cab54f667a0325d9637e561.fyc good-sha256=6ad0b55e8be5d203d4927c11b424e7191333eb545cab54f667a0325d9637e561
    candidate_grouped_static_cli_cache_test.go:485: refusal=partial-unused-i64 provider-sha256=9c7018ce9e1d8542ebda6fcb79cbd19fb13d6014e174d28b1bc17c6ef74084d7 diagnostics="cache: cold uncached\nlibs/answer/answer.fy:1:1: E4501: Generic source preparation rejects a malformed, colliding, shadowing, or reserved declaration or binding. [bytes 0..591]\n"
    candidate_grouped_static_cli_cache_test.go:485: low resumed tail complete: 6 processes, repair + partial unused tuple refusal; baseline/mutations owned by interrupted attempt1
    candidate_grouped_static_cli_cache_test.go:489: profile=low actual elapsed=1m38.6030204s
    candidate_grouped_static_cli_cache_test.go:483: host profile=optimized resolve-elapsed=3m56.5397612s identity={"Source":"e8fd125d83200d0f195c5562128203b4f1c44382ce68447ee93275b2c3ee8a04","Image":"81a1c5fc24a756dbf4066dc02cdd1a7d3882b2c26a9a3782a6c69890c79d4871"}
    candidate_grouped_static_cli_cache_test.go:487: edit=used-provider cache=cache: mixed stable-package-generic-keys=0/2
    candidate_grouped_static_cli_cache_test.go:487: edit=unused-sibling-provider cache=cache: mixed stable-package-generic-keys=0/2
    candidate_grouped_static_cli_cache_test.go:487: edit=unused-i64-provider-tuple cache=cache: mixed stable-package-generic-keys=0/2
    candidate_grouped_static_cli_cache_test.go:487: edit=private-body cache=cache: mixed stable-package-generic-keys=1/2
    candidate_grouped_static_cli_cache_test.go:487: selected-generation repair=cache: units hit selected=fyproject0-add5984a5dc5d97bd0b1f1e4e9e1c0162090eb3a3db6da6e57203710abef9401-1b0633aaf1f0278522f981580951636c0567f78e26347dc5f1b3810dd025ad0b.fyc good-sha256=1b0633aaf1f0278522f981580951636c0567f78e26347dc5f1b3810dd025ad0b
    candidate_grouped_static_cli_cache_test.go:487: refusal=partial-unused-i64 provider-sha256=9c7018ce9e1d8542ebda6fcb79cbd19fb13d6014e174d28b1bc17c6ef74084d7 diagnostics="cache: cold uncached\nlibs/answer/answer.fy:1:1: E4501: Generic source preparation rejects a malformed, colliding, shadowing, or reserved declaration or binding. [bytes 0..591]\n"
    candidate_grouped_static_cli_cache_test.go:487: refusal=duplicate-unused-i64 provider-sha256=492dcb9f52b2155446a4e23d05868a287bca9fb9a03b299893337d221da1fd09 diagnostics="cache: cold uncached\nlibs/answer/answer.fy:1:1: E4501: Generic source preparation rejects a malformed, colliding, shadowing, or reserved declaration or binding. [bytes 0..688]\n"
    candidate_grouped_static_cli_cache_test.go:487: refusal=wrong-unused-i64-signature provider-sha256=edeb27fa1f53cae6968a4e4af8cf9043e0e69f23801a07394a8fbb3c7954a7bd diagnostics="cache: cold uncached\nlibs/answer/answer.fy:1:1: E4501: Generic source preparation rejects a malformed, colliding, shadowing, or reserved declaration or binding. [bytes 0..616]\n"
    candidate_grouped_static_cli_cache_test.go:487: completed 45 fresh authored CLI processes + 2 standalone products; full-vectors=true retained=C:\Users\cherr\Documents\workspace\zerodi-wd1\projects\hobby\opensource\fairyc\.cache\grouped-static-20261107\.cache\grouped-static-cli-3025737042\workspace-3685712951
    candidate_grouped_static_cli_cache_test.go:489: profile=optimized actual elapsed=4m29.2278411s
    candidate_grouped_static_cli_cache_test.go:496: test executable=fairyc.compiler-executable-sha256/v1:4fe0219ee35b2792eb3530d7ee1fa314ec62f4dddd9c815b7122c579e72829fc
--- PASS: TestCandidateGroupedStaticCLICache (368.30s)
    --- PASS: TestCandidateGroupedStaticCLICache/low (98.81s)
    --- PASS: TestCandidateGroupedStaticCLICache/optimized (269.44s)
PASS
ok  	github.com/0disoft/fairyc/internal/cli	368.565s
```

The actual authored refusal diagnostics report E4501 at
`libs/answer/answer.fy:1:1`, bytes `0..591` (partial), `0..688` (duplicate), and
`0..616` (wrong signature). This test proves refusal and cache preservation;
it does not claim more precise source-location parity from these CLI messages.

## Boundaries

No Linux, hosted CI, remote action, dependency installation, production fix,
bound increase, source-of-truth change or commit is performed by this worker.
Full host/native outcome is recorded above. Both assigned files remain
uncommitted for coordinator review. Sessions 34733 (interrupted) and 43163
(exit0) are closed; no live process/session or blocker remains.

After integration, the coordinator verified every changed worktree file was
copied to main (the admission receipt has an added main-source check), moved its
entire ignored `.cache` to `.cache/grouped-static-evidence-20261107/` in main,
then removed the detached worktree. Historical artifact paths above now resolve
under that evidence root; no failed log or host image was discarded.
