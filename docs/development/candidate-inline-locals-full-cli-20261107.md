# Inline immutable locals: full authored CLI acceptance (2026-11-07)

Status: **passed** on Windows AMD64 after the parent's explicit **API READY**
notification. Both low and optimized full authored CLI host profiles passed.

Scope: [ADR 0509](../adr/0509-inline-immutable-local-bindings.md), following the
[local-binding implementation receipt](candidate-inline-locals-20261107.md).
The accepted fifteen-file implementation slice is unchanged by this follow-up.

## Acceptance path

`TestCandidateInlineLocalsFullCLI` reuses `runCandidateInlineTrapCLI` and the
existing `inlineLocalsTraps` fixture. It clears the optional inline-host profile
selection so both low and optimized host subtests run. Existing Windows AMD64
and `FAIRYC_RUN_PARITY_INTEGRATION=1` gates remain in
`copiedClosureCLIProfiles`; the wrapper does not bypass or duplicate those gates.

The authored main and independent CLI processes remain intact. The Go companion
consumes the actual unchanged Fairy-generated C packet, without supplying a Go
frontend or cached AST. Two unused immutable local initializers execute in source
order: division by zero first, or integer overflow after successful division.
Both ordinary and generic bool-marker callers retain the same acceptance:

- Cold development and warm locked/release runs have exact expected cache status,
  empty stdout and exact Go-matching provider trap diagnostics at columns 55/72.
- Warm traps must not change cache generation names or bytes.
- The release packet must contain exactly one reachable inline prototype and
  definition in the shared header, and no inline definition in body units.
- Interpreter, native development and native release test reports must match
  the independent Go reference exactly. A successful sentinel after the two
  failing tests checks `locals(40,2)==42` and `add(40)==42`.

Each host profile expects fourteen independent authored CLI processes: four
cold/warm pairs and six test reports. Both profiles together expect twenty-eight
processes, plus the runner's independent Go reference checks.

## Parent-owned prerequisite

The parent implemented
`cdriver.BuildWindowsMSVCCompilerHostPartsWithStackReserve` with a private 8MiB
compiler-host header bound. Inspection confirms
`candidate_copied_closure_cli_host_test.go` calls that API. The ordinary project builder retains its 2MiB header contract.
This follow-up does not edit either the driver or the shared helper.

The prior full host header measured 2,099,985 bytes, exceeding 2MiB by 2,833 bytes;
205 body parts were within the existing part-count bound. This is a private-host
harness prerequisite, not a permanent product blocker. Heavy acceptance starts
only after the parent's explicit API-ready notification.

## Focused command and results

After explicit API readiness, executed from the repository root:

```powershell
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
$env:FAIRYC_CLOSURE_CLI_ARTIFACTS=Join-Path (Get-Location).Path '.cache/inline-locals-full-cli-20261107'
go test -mod=readonly ./internal/cli -run '^TestCandidateInlineLocalsFullCLI$' -count=1 -v -timeout 15m
```

Result: **PASS**, package duration **610.715s**, test duration **610.59s**.

- Low host: **287.86s**, fourteen independent authored CLI processes, four
  cold/warm trap pairs and six exact Go-matching test reports.
- Optimized host: **298.49s**, the same fourteen processes and assertions.
- Combined: **twenty-eight** authored CLI processes, **eight** cold/warm trap
  pairs and **twelve** exact Go-matching reports, plus independent Go reference
  execution before the host subtests.

All expected trap commands returned `ExitProgramTrap` (5). Exact source diagnostics,
provider columns, cache statuses/generation stability, packet/header placement,
interpreter/native development/native release reports and post-trap successful
sentinels passed through the existing shared runner. The previous header-admission
blocker is resolved for this full compiler-host path without changing this
follow-up's test assertions or the ordinary builder's header limit.

The shared helper retains each compiler image and a `host.json` receipt binding
its exact generated C source digest and executable digest under the artifact
directory's `low` and `optimized` subdirectories. A later run reuses only a
matching source/image pair; the companion remains a C-only consumer.

## Retained exact-source artifacts

Artifacts remain under `.cache/inline-locals-full-cli-20261107/{low,optimized}`.
Each directory contains `fairyc.exe`, `host.json` and the C-only
`fairyc-candidate-build.exe`. After execution, both image SHA-256 digests were
independently recomputed and matched `host.json`. The source digest binds the
profile name, generated header, support and every generated C unit. Future helper
runs regenerate the inputs and reuse an image only when both source and image
digests match; no unchanged verified C needs recompilation.

| Profile | Source SHA-256 | Image SHA-256 | Image bytes |
| --- | --- | --- | ---: |
| low | `ca670ad53310e6795f160e1141f804db7f3f7ce5a48962bdda274c874c0f9112` | `66afce8b93e37f5bdab629ff6453fdb6b0dd4ab53d7a26a20472306efc01ed3f` | 16,116,736 |
| optimized | `62b502f4e9f8ebac51a96f4c36bf3326b90a2ddb5eea50be8f0624b1ba3e0fe6` | `8db580d0fd23886e1a641939391f0edffc8801160cd1aa43beee16875678dece` | 6,073,856 |

## Exact follow-up paths and remaining gates

- `internal/cli/candidate_inline_locals_full_cli_test.go`
- `docs/development/candidate-inline-locals-full-cli-20261107.md`

Only these two new files belong to this follow-up. Parent owns staging and commit.
No lockfile, shared implementation/index/queue, prior accepted file, driver,
shared harness, API/DB, dependency, runner or repository-hygiene change is made.
Native Linux, the full test suite, hosted dispatch, push, release and deployment
remain outside this follow-up.

Final hygiene: both new files are nonempty UTF-8/LF with final newlines; the
new Go test is gofmt-clean. No staging, commit or lock refresh was performed.
