# SDK/user aggregate cleanup command acceptance, 2026-11-07

This bounded follow-up to `candidate-atlas-sdk-aggregate-20261107.md` uses the
same exact source: SDK Vec calls inside `Check<bool>` cleanup, and a generic
function returning the concrete user struct `Cell<i32>`. Production admission
and command wiring already exist; no compiler or product behavior is rewritten.

## Acceptance added

The small Windows fixture compiles the actual `project_atlas_cli`,
`project_mcp_cli`, and `project_edit_cli` import closures once per low/optimized
profile. Its test main only chooses the actual command entry point. It does not
provide maps, Complete objects, aggregate-mode choices or Go-produced source
facts to Fairy. MCP/editor commands run with PATH empty and use their own
executable SHA-256 for current compiler identity. The focused SDK-query replay
below independently prepares the exact published source and checks the same
native image identity; it does not supply maps or Go facts to Fairy.

The MCP verifier reuses the existing real stdio session acceptance: initialize,
notification and the single read-only tool, an independent Go-matching impact
query for original SDK `at`, stale source error, same-session recovery after
independent lock refresh, workspace immutability, no paths/source leakage,
protocol-only stdout and clean EOF/stderr.

The editor verifier reuses exact Go preview/apply report comparisons, invalid
bool-field overlay and SDK declaration-write refusals, all four prepublication
source/overlay checks, exact published body bytes, unchanged graph lock and no
staging/lock debris. The old request first fails against the stale lock, then
fails against its old source SHA-256 after independent refresh.

The replacement private leaf creates `Check<bool>` and calls `cell<bool>(true)`.
The concrete `Cell<bool>` and `cell<bool>` implementations replace their i32
counterparts; neither bool implementation exists in the original source. The
cleanup continues to call SDK `empty<i32>`, `push<i32>` and `at<i32>`. The Go
oracle verifies exact implementation inventories, bool field layout, cleanup
call spans and cross-package specialization endpoints, changed root source
identity, and unchanged pinned SDK declarations, symbols, source/interface
identity and concrete SDK implementations.

An optional test-only current-source hook on the shared editor fixture then
queries original SDK `at` and original user `cell` through the actual impact
command on the published, independently relocked workspace. Exact Go subjects,
nodes, contexts, traversal, omissions and executable identity must match.
This adds no extra preparation to existing fixture callers.

Both new verifiers are registered as `mcp-sdk-aggregate-cleanup` and
`edit-sdk-aggregate-cleanup` in the unmodified-main acceptance harness. The
bounded command fixture is not execution of that full authored-main harness.
Native map-by-map acceptance for the original source remains the prior receipt;
this follow-up compares edited Go map facts and actual native impact results.
The bounded fixture separates `mcp`, `edit` and `current-sdk` groups so each can
resume independently. `edit` keeps the user post-edit query; `current-sdk`
prepares the identical edited bytes and independently checks the SDK subject.
The full-main editor verifier retains both post-edit queries.

## Observed Go inventory

| Package | Declarations | Ordinary functions | Implementations | Relations |
| --- | ---: | ---: | ---: | ---: |
| example.session.app | 5 | 2 | 3 | 8 |
| example.session.answer | 1 | 1 | 0 | 0 |
| fairyc.std | 278 | 156 | 14 | 516 |

These counts hold before and after the edit. The edited graph has 128 resolved
calls and exactly three cross-package user-cleanup calls with concrete source
and target specialization identities. Original private-storage E0101 and
nominal-argument E4502 source refusals remain covered by the selected fast tests.

## Commands and results

```powershell
go test -mod=readonly -count=1 -failfast -timeout 1m -v ./internal/cli -run '^TestAtlasSDKAggregateCleanup(EditOracle|Oracle|Refusals)$'
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
go test -mod=readonly -count=1 -failfast -timeout 18m -v ./internal/cli -run '^TestCandidateAtlasSDKAggregateCleanupCommands$'
# Select the still-unfinished profile/group without changing the default helper.
go test -mod=readonly -count=1 -failfast -timeout 8m -v ./internal/cli -run '^TestCandidateAtlasSDKAggregateCleanupCommands$/^optimized$'
go test -mod=readonly -count=1 -failfast -timeout 22m -v ./internal/cli -run '^TestCandidateAtlasSDKAggregateCleanupCommands$/^low$/^edit$'
go vet -mod=readonly ./internal/cli
```

Fast Go source/edit/refusal acceptance: PASS, 3.107s, no skips. The new edit
oracle itself took 1.50s. Scoped CLI vet passes. The same native selection without
the private integration opt-in passes in 0.169s with its single heavy test
explicitly skipped. Go formatting and all five owned files' UTF-8/LF, final
newline and trailing-whitespace checks pass; the scoped tracked diff is clean.

The initial combined native command FAILED at its overall 18-minute Go timeout
(1080.229s). Its low MCP group returned successfully before low/edit started.
Low preview, invalid-overlay refusal, SDK write refusal, apply (4m6.8479294s),
exact published bytes, unchanged lock/no debris and stale-lock refusal had
passed. The timeout stack is waiting at the old-source-hash refusal after
independent refresh, `candidate_edit_sdk_command_test.go:158`; that refusal and
the subsequent edited-source impact queries were not completed by this command.
This failure remains a failure. Only its exact orphan image child, PID 46348,
was stopped after the parent had exited; requery confirmed zero remaining owned
children. Original log: `.cache/atlas-sdk-aggregate-commands-20261107.log`.

The separate optimized-only selection PASSED in 149.285s (test 149.06s,
profile 126.97s, MCP 12.20s, edit 32.14s), no selected skips. Actual apply took
9.1025816s. All editor refusals, exact publication and both post-edit current
queries passed. Its log is
`.cache/atlas-sdk-aggregate-commands-optimized-20261107.log`.

At the coordinator's request, only the unfinished low/edit group is resumed
under the explicit 22-minute integration-test bound. Low/MCP and optimized are
not rerun. This resume FAILED in 874.371s (test 874.25s, profile 858.30s,
edit 842.50s). Apply completed in 3m30.8706887s. Preview, invalid-overlay and SDK
write refusals, exact publication/lock/debris checks, stale-lock refusal and the
previously unfinished old-source-hash refusal passed. The user `cell` current
impact query returned and matched the independent Go result. The final SDK
`at` current impact query hit the unchanged three-minute native CLI bound before
returning a result. Its result comparison and the final two-query workspace
immutability/count assertions therefore did not run. This is a bounded timing
failure, not an observed semantic-result mismatch or a successful low/edit gate.
Both completed low test roots have zero remaining owned native children. At
that checkpoint no command limit had been raised. The failed resume log is:
`.cache/atlas-sdk-aggregate-commands-low-edit-resume-20261107.log`.

The coordinator then isolated only the unfinished current SDK query and gave
this newly added integration query an explicit six-minute bound. The existing
shared three-minute helper default, six-minute apply, eight-minute MCP session
and all product timeouts are unchanged. No acceptance assertion is removed.
All completed MCP/editor/user-query observations above are retained; those slow
workflows are not repeated again.

```powershell
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
$env:FAIRYC_ATLAS_RESUME_IMAGE_SHA256='b06156fa650e64f77b06167526dc66c4cad254d65084e7940c7436f4c14a1403'
go test -mod=readonly -count=1 -failfast -timeout 9m -v ./internal/cli -run '^TestCandidateAtlasSDKAggregateCleanupCommands$/^low$/^current-sdk$'
```

This focused native selection PASSES in **192.445s** (query group 168.35s).
Its freshly generated low image has the exact original SHA-256 above and the
source has exact edited SHA-256
`6bc8280da93a19b09c1ea12501c05efe3d018ead5854e28bb4d9a3098de9e130`.
The independent Go 128-call SDK query agrees in subjects, nodes, contexts,
traversal and omissions, with exact current compiler identity. Its final
one-query count and complete workspace immutability assertions pass. Log:
`.cache/atlas-sdk-aggregate-current-sdk-low-20261107.log`.

These observations together establish the bounded command behaviors in both
profiles. Neither of the two failed overall commands is relabeled a pass, and
no single uninterrupted combined low/root run is claimed. Future native replay
should select the three low groups separately (MCP, edit, current-sdk) and the
optimized profile, or allow an adequate opt-in overall bound for their sum.
The optional resume-image environment value is a strict equality assertion on
the freshly built image, not an alternative compiler or production selection.

Profile/group resume needs no further helper change or new profile option:
existing Go subtest filtering
selects a profile before its native C build and a command group before its
verifier. The outer Go test bound is separate from the unchanged existing
three-minute ordinary CLI, six-minute apply and eight-minute MCP session bounds;
only the added current SDK query has its separately recorded six-minute bound.

## Preserved identities

The expired original low snapshot is preserved under
`.cache/atlas-sdk-aggregate-commands-20261107/low-snapshot`, with 415 Fairy source
files, its manifest and its 9,990,144-byte image. The per-file input/image
manifest is `evidence.json` in that parent directory, SHA-256
`052ca4e3bfb98d94b0d274b7c200260834ab8a4db039126ce014575eea3f7f9a`.

| Input/image | SHA-256 |
| --- | --- |
| Original exact client source | `f3730143845162f9eb1910333c2220e4f6c5a14162da16f0552a8a5f2723fab9` |
| Exact edited client source | `6bc8280da93a19b09c1ea12501c05efe3d018ead5854e28bb4d9a3098de9e130` |
| Narrow test main (533 bytes) | `a38764121075e4f885013dd1f58817382981d95d97195bf5cf8714ddc38206f3` |
| Original low image | `b06156fa650e64f77b06167526dc66c4cad254d65084e7940c7436f4c14a1403` |
| Completed optimized image | `2d65345b2c625c46fca5e8e45a11f27cc2c13038c653b19cac61196a8c0ed70c` |

The optimized image identity is recorded by the successful current-query
verifier. Its temporary image has already been cleaned normally; the original
low image and copied source inputs remain preserved. These are separate local
snapshots, not same-commit whole-compiler promotion evidence.

The low/edit resume source/manifest/image inventory matches every one of the
original 417 file hashes, including the same low executable hash above. Its
copied snapshot is `low-edit-resume-snapshot` in the same cache directory;
`low-edit-resume-evidence.json` records that inventory at checkout `21b9c835`
(SHA-256 `c8f6957eb19652b47759355b0f3ad53260125581a3e95759006fb1ba9cc8ec1d`).
These preserved cache files remain ignored and are not proposed Git changes.
`results.json` alongside them records the three actual command outcomes and
their completed log SHA-256 values without changing the input manifests.

The new native test uses the existing private integration opt-in, keeping heavy
execution out of default Go tests. The final default selection passes in 1.385s:
the three Go oracle/refusal roots pass and the heavy native root explicitly
skips. Scoped vet passes. Native assertions, warning policy, completeness and
source-authority checks remain intact; the additional query's test-only bound
is recorded above. The five authored files retain UTF-8/LF/final-newline hygiene.

## Remaining gates and boundaries

Original low/MCP completion is preserved within the first failed overall run.
The failed low/edit resume preserves its completed editor/user-query checks;
the focused SDK replay independently completes the final query and workspace
assertions. Optimized MCP/edit acceptance is complete. This remains composed
group evidence over identical inputs, not a successful uninterrupted low root.

Full authored-main replay for these two new groups, edited native map-by-map
inspection, native build/run of the edited client, native Linux execution, the
broader source matrix, full repository tests, consecutive self-build, final
distribution/clean-machine checks and same-commit hosted gates remain separate.
Linux stays batched. These local timings are acceptance observations, not
isolated performance benchmarks.

No owning grammar, public CLI/protocol/schema, SDK/runtime ABI, DB, dependency,
runner, product timeout, shared index/source-of-truth/queue or ADR change. State,
inline and package-fetch modules are outside this slice. No staging, commit,
push, dispatch, release or deployment; the coordinator owns review and commits.
