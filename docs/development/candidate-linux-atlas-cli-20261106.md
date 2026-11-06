# Linux candidate Atlas CLI connection, 2026-11-06

Native Linux execution remains batched at the user's request. Each command is
connected independently, retaining the existing current-source/lock and query
policies. No push, hosted dispatch, release or deployment is authorized here.

## Impact command

`linux_experimental_cli` routes exactly `atlas-impact` to the shared
`project_atlas_cli` owner. Its new `arguments_with_help` entry accepts the Linux
help text; the existing `arguments()` still supplies the same Windows/default
help and preserves its signature. Linux help lists only connected commands.
Fetch retains unknown-command, help and overlong-argument precedence and remains
the sole network operation. This does not route MCP, edits or symbol queries.

The prepared Linux workspace already selects its retained workspace readers,
compiler-image reader and cache/host-path environment. Atlas impact consumes
those existing owners, without introducing a new filesystem facade, subprocess
oracle or unauthenticated map source. Every request still enters the shared
fresh locked-source verifier, Complete producer, current map admission and
bounded join/traversal/render pipeline. Query results retain explicit partial
coverage and never claim safe edits or behavior equivalence.

`TestLinuxAtlasImpactCommand` executes the shared command with Linux help on
Windows MSVC low/optimized and the real Windows workspace I/O. It reuses the
independent Go graph/context/identity oracle, exact source/lock read-only checks,
depth/symbol/output/package/call cutoffs, missing/stale lock, syntax and output
failure controls. Linux help aliases and broken-help-output have separate exact
expectations. Generated query processes use empty PATH.

`TestLinuxFileCLI` checks exact nested command selection, emits the full Linux
entry for both targets and rejects Windows foreign symbols. Its existing
per-profile native Linux image now runs `verifyLinuxAtlasImpact`, reusing the
same Go oracle and command controls without another compiler/companion build.
This actual Linux replay is authored, not executed on the Windows host.

Validation commands and final results are recorded in the completion section.
MCP/stdio, symbols/calls/pack selection and edit filesystem publication remain
separate implementation slices, as do actual Linux TLS/runtime and final gates.

No public command policy, SDK ABI, DB, dependency, timeout, CI/runner or
repository-hygiene change. This adds private Linux dispatch and a help-injection
entry; the original shared command behavior remains covered by its regression.

## Completion checks

The original source passes `check` and `check --locked` with 698 files / 8,639
MIR functions. The freshly prepared Linux CLI passes both commands with
698 / 8,640. Scoped `go vet -mod=readonly ./internal/cli`, Go formatting,
UTF-8/LF/final-newline and diff checks pass.

The existing Windows `TestCandidateAtlasCommand` passes low/optimized in
565.38s (469.25s / 79.47s). It retains the original ordinary query/error controls
and the 17-declaration/two-ID family oracle. This is the unchanged default-help
wrapper's command regression, not actual Linux execution.

The selected command is:

```powershell
go test -mod=readonly -count=1 -timeout 12m -v ./internal/cli -run '^(TestLinuxAtlasImpactCommand|TestCandidateAtlasCommand|TestLinuxFileCLI|TestLinuxFetchCommand)$'
python -B tools/prepare-candidate-linux-cli.py --output .cache/linux-atlas-impact-20261106
# In examples/lexer_candidate and the isolated Linux workspace, separately:
go run -mod=readonly ../../cmd/fairyc check
go run -mod=readonly ../../cmd/fairyc check --locked
```

That combined command times out at the Go harness's 12-minute overall limit
(720.160s), while compiling the Linux-help command's optimized Windows image.
The completed default-wrapper regression above is retained. The incomplete
Linux command and the two not-yet-entered roots are not counted as passes.
No compiler invocation, program timeout or product resource policy is changed.
No compiler child remains after the timed-out harness exits. The remaining
roots are rerun separately, without repeating the completed family regression:

```powershell
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^(TestLinuxAtlasImpactCommand|TestLinuxFileCLI|TestLinuxFetchCommand)$'
```

The final separate command exits zero in 352.584s. Linux-help impact passes both
Windows profiles in 257.38s (low 134.80s, optimized 102.18s); fetch regression
passes both in 47.97s. There are no selected command-leaf skips. The full Linux
entry emits for both targets with closed Linux support and no Windows foreign
symbols: 24 C partitions, 1,965,103-byte header. Its portable dispatch passes
both Windows profiles in 7.81s; the outer test takes 46.76s and explicitly skips
actual Linux compilation/execution. These observations do not certify native
Linux impact, fetch, syscalls, TLS or self-build.

The original default-wrapper pass is retained from the first command because
its source and verification inputs are unchanged. No failed/incomplete test is
counted as a native pass. Native Linux replay, all other Atlas command ports,
the broader comparison/distribution gates and the full implementation objective
remain open.
