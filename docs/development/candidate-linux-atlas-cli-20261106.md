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
the sole network operation. This first slice routes impact only; subsequent
sections record MCP and symbol-query selection separately. Edits remain open.

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
MCP/stdio and symbol selection have separate slices below. Calls/pack selection
and edit filesystem publication remain open, along with actual Linux TLS/runtime
and final gates.

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

## MCP command

Linux experimental mode 2 now selects the shared `project_mcp_cli` owner with
Linux help through `arguments_with_help`; the original `arguments()` still
supplies default help. Root, explicit capability, all budgets, framing, current
locked-source admission and response/error policies remain owned by the shared
implementation. `project_mcp_stream` uses the existing target console ABI; Linux
support selects the direct POSIX read/write implementation. No separate stream
adapter, network listener, child query provider or write tool is introduced.

`TestLinuxAtlasMCPCommand` uses the same independent Go query/context oracle and
live-session verifier as the default command, with exact Linux help aliases.
Initialization, valid/unavailable/valid query recovery, message/notification
limits, empty-PATH execution, response before stdin EOF, changed-source refusal
and recovery, unchanged workspace bytes and broken stdout are covered. The
default verifier now accepts an explicit expected help string internally; its
existing entry retains the unchanged expectation. Actual full-entry Linux replay
reuses each already-built CLI image via `checkLinuxAtlasMCP` without another full
compiler or companion build.

```powershell
go test -mod=readonly -count=1 -timeout 10m -v ./internal/cli -run '^(TestLinuxAtlasMCPCommand|TestCandidateMCPCommand|TestCandidateMCPStream)$'
go test -mod=readonly -count=1 -timeout 5m -v ./internal/cli -run '^TestLinuxFileCLI$'
go test -mod=readonly -count=1 -timeout 4m ./internal/cli -run '^TestLinuxFetchCommand$'
python -B tools/prepare-candidate-linux-cli.py --output .cache/linux-atlas-mcp-20261106
```

The final command/stream group passes in 309.388s, with no selected skips:
default MCP 153.38s (low 53.67s / optimized 80.84s), Linux-help MCP 151.86s
(53.87s / 82.61s), stream 3.94s. Stream controls include 4,096-byte chunk and
65,536/65,537-byte frame boundaries, CRLF/final EOF, progress before failure and
live multi-frame responses. Fetch regression passes both profiles in 28.927s,
including the final independent MCP help-presence expectation.

Full-entry both-target emission and closed Linux support selection pass in
38.232s: 24 C partitions, 1,976,208-byte header. Portable dispatch passes both
Windows profiles in 7.06s, including exact MCP selection and near-name refusals.
The outer test explicitly skips native Linux compilation/execution. Original
source check/locked passes 698 files / 8,640 MIR functions; freshly prepared
Linux check/locked passes 698 / 8,641. Scoped vet, formatting and canonical-text/
diff checks pass. The example lock changes only current root identity/order.

Native Linux stdio, default-SIGPIPE/closed-pipe behavior and session freshness
remain unverified. The console writes directly to the descriptor; a Windows
broken-pipe observation cannot establish Linux signal/error behavior. Retain
that explicit signal-policy check in the combined native replay. No Linux
MCP/fetch/syscall/TLS/self-build pass is claimed here. Symbols/calls/pack/edit,
all broader source-family comparisons and every final acceptance gate remain
open at this slice; symbol selection continues below. Public API/SDK ABI, DB,
dependency, timeout, CI/runner and repository
policies are unchanged; no push, hosted dispatch, release or deployment.

## Symbol catalog command

Linux experimental mode 3 selects `project_catalog_cli::arguments_with_help`;
the original `arguments()` retains default help. Shared option parsing, fresh
source/Complete production, requested v1/v2/v3 catalog family, selection/output
budgets and lock policy are unchanged. Linux help now lists the admitted
`atlas-symbols list` flags and optional filters/cursor/locked mode. Unknown and
near-name commands retain fetch's existing refusal policy.

`TestLinuxAtlasSymbolsCommand` executes the Linux-help entry with real Windows
workspace I/O in low/optimized. The original Go source/catalog oracle supplies
all expected identities, cursor boundaries, selection/output byte counts and
canonical complete JSON independently of candidate output. It checks all three
catalog versions, name filtering, pagination, exact/one-short/one-over byte
ceilings, invalid selectors/arguments, stdout failure, locked byte preservation,
stale-source refusal and actual unlocked lock refresh. The probe also retains
the original default entry in the same image via a test-only argv prefix, and
separately requires exact default help for all three aliases at both direct and
`list` positions. This prefix is absent from product dispatch.

The full-entry Linux replay reuses its already-built per-profile image for the
same Linux-help and query oracle. The portable entry checks exact mode 3 and
near-name refusals. Both-target full entry emission and closed Linux support
selection pass in 39.378s: 25 C partitions and a 1,988,729-byte header. Portable
Windows dispatch passes both profiles in 6.76s; the outer 39.08s test explicitly
skips native Linux execution.

```powershell
go test -mod=readonly -count=1 -timeout 12m -v ./internal/cli -run '^TestLinuxAtlasSymbolsCommand$'
go test -mod=readonly -count=1 -timeout 5m -v ./internal/cli -run '^TestLinuxFileCLI$'
go test -mod=readonly -count=1 -timeout 4m ./internal/cli -run '^TestLinuxFetchCommand$'
python -B tools/prepare-candidate-linux-cli.py --output .cache/linux-atlas-symbols-20261106
```

Fetch regression passes both profiles in 36.628s, including independent presence
checks for all connected help lines. Original check/locked passes 698 files /
8,641 MIR functions; freshly prepared Linux check/locked passes 698 / 8,642.
The final catalog command passes in 490.118s (test body 489.97s): low 383.74s,
optimized 92.06s. Each profile executes 44 independent-oracle command cases,
six Linux-help cases and six unchanged default-help cases: 112 generated command
executions across both profiles, all with empty PATH. There are no selected
command-leaf skips. Scoped vet, Go formatting, UTF-8/LF/final-newline and diff
checks pass. No compiler/program deadline or warning policy is relaxed.

No public CLI/SDK ABI, DB, dependency, timeout, CI/runner or repository policy
changes. Native Linux filesystem/lock/publication, pipe/signal behavior, all
remaining command ports and the complete final acceptance queue stay open.

## Call catalog command

Linux experimental mode 4 selects `project_calls_cli::arguments_with_help`.
The existing default `arguments()` signature/help remains intact. Both entries
use the same complete source-backed call-query owner, 0/1/2 catalog selection,
outgoing/incoming/both directions, original package/symbol IDs, construction and
output budgets, error classifications and locked/unlocked policy. Linux help
lists only that existing admitted syntax; no query or source producer is copied.

`TestLinuxAtlasCallsCommand` reuses the independent Go workspace-call oracle for
all version/direction combinations, recursion, exact/one-short/one-over output
bytes and package/call ceilings, missing/invalid selectors and arguments,
read-only locked execution, stdout failure, stale lock refusal and an actual
unlocked source/lock refresh. Fresh Go expectations restore the old lock before
the unlocked candidate invocation, so its publication is not supplied by setup.
The same probe image checks both default and Linux help aliases at direct and
`list` positions. The default selector is test-only, absent from product dispatch.

`checkLinuxAtlasCalls` attaches the same Linux-help/query verifier to the existing
per-profile native full-CLI image. It adds no full compiler/companion build.
Portable full-entry selection checks exact mode 4 and near-name refusals;
both-target emission must retain closed Linux support without Windows symbols.
Actual Linux filesystem/lock/pipe behavior and all final acceptance gates stay
batched and unverified.

```powershell
go test -mod=readonly -count=1 -timeout 12m -v ./internal/cli -run '^TestLinuxAtlasCallsCommand$'
go test -mod=readonly -count=1 -timeout 5m -v ./internal/cli -run '^TestLinuxFileCLI$'
go test -mod=readonly -count=1 -timeout 4m ./internal/cli -run '^TestLinuxFetchCommand$'
python -B tools/prepare-candidate-linux-cli.py --output .cache/linux-atlas-calls-20261106
```

Final calls command execution passes in 273.41s: low 186.94s and optimized
73.31s. Each profile runs 26 independent Go-oracle query/error/lock cases, six
Linux-help aliases/actions and six unchanged default-help aliases/actions:
76 generated command executions with empty PATH, without selected command skips.

The additional `TestLinuxCLIHelp` compares every connected experimental command
line with the independent Go usage contract. It corrects the calls version flag
to optional, symbols kind to required/repeatable and impact depth to 0..64,
retaining the exact existing command policy. Its final isolated run passes in
0.100s. The initial combined test command exited 1 after calls passed because a
redundant help-only native probe incorrectly used a partition-required helper.
That duplicate native probe was removed; the existing calls test already checks
exact generated help bytes. No product source changed after the passing calls
execution, and that result is retained rather than rerunning its 76 commands.

Current regression command:

```powershell
go test -mod=readonly -count=1 -timeout 12m -v ./internal/cli -run '^(TestLinuxAtlasCallsCommand|TestLinuxCLIHelp)$'
go test -mod=readonly -count=1 -timeout 1m -v ./internal/cli -run '^TestLinuxCLIHelp$'
go test -mod=readonly -count=1 -timeout 5m -v ./internal/cli -run '^(TestLinuxFileCLI|TestLinuxFetchCommand)$'
python -B tools/prepare-candidate-linux-cli.py --output .cache/linux-atlas-calls-final-20261106
```

The fetch regression passes both profiles in 26.49s. Current full-entry emission
and closed Linux support selection pass with 25 C partitions and a 1,996,624-byte
header; portable dispatch passes in 5.46s. The full-entry test takes 32.18s and
explicitly skips actual Linux execution; its combined regression command exits
0 in 58.984s. Original check/locked passes 698 files / 8,642 MIR functions;
freshly prepared Linux check/locked passes 698 / 8,643. Scoped vet, Go formatting,
UTF-8/LF/final-newline and diff checks pass.

Pack/edit selection, native Linux filesystem/lock/publication, pipe/signal
behavior, broader source-family parity and every final acceptance gate remain
open. Public CLI policies, SDK ABI, DB, dependency, timeout, CI/runner and
repository rules remain unchanged; no push, hosted dispatch, release or deployment.

## Context pack command

Linux experimental mode 5 selects `project_pack_cli::arguments_with_help`.
The original `arguments()` continues to select default help. Overview/show/around
share the existing current-source producer, v1/v2/v3 schema selection, exact
package/symbol identity, direction/relation filters, traversal/selection/output
budgets and lock policy. Linux help carries the three exact Go-owned command
lines, checked by `TestLinuxCLIHelp` independently of the Fairy output data.

`TestLinuxAtlasPackCommand` reuses the existing independent Go partition/pack
oracle and ordinary command fixture. Both default and Linux-help entries are
checked in the same per-profile Windows image through a test-only prefix, with
all three help aliases at direct and overview/show/around positions. The prefix
is absent from production dispatch. The actual query checks retain v1/v2/v3,
all directions, exact/one-short/one-over output ceilings, invalid selectors,
selection/depth/relation policy, source-lock preservation/refusal and unlocked
refresh. `checkLinuxAtlasPack` attaches the same verifier to the existing native
Linux full-CLI image without adding a compiler or companion build.

```powershell
go test -mod=readonly -count=1 -timeout 15m -v ./internal/cli -run '^(TestLinuxAtlasPackCommand|TestLinuxCLIHelp)$'
go test -mod=readonly -count=1 -timeout 5m -v ./internal/cli -run '^(TestLinuxFileCLI|TestLinuxFetchCommand)$'
python -B tools/prepare-candidate-linux-cli.py --output .cache/linux-atlas-pack-20261106
```

Current original check/locked passes 698 files / 8,643 MIR functions; freshly
prepared Linux check/locked passes 698 / 8,644. Both-target full-entry emission
and closed Linux support selection pass with 25 C partitions and a 2,032,407-byte
header. Portable dispatch passes in 5.67s; the outer full-entry test takes 32.22s
and explicitly skips native Linux execution. Its combined fetch/full-entry
regression command exits 0 in 62.466s; fetch passes both profiles in 29.92s.

The final pack/help command exits 0 in 419.539s. Pack takes 419.42s: low
336.66s and optimized 69.86s. Each profile executes 48 independent Go-oracle
query/error/lock cases, twelve Linux-help cases and twelve default-help cases:
144 generated command executions with empty PATH, without selected command
skips. The independent command-help contract comparison also passes. Scoped
vet, Go formatting, UTF-8/LF/final-newline and diff checks pass. No product
compilation or runtime deadline is relaxed.

Edit selection, native Linux filesystem/lock/publication and stdio/signal
behavior, broader source-family parity and every final acceptance gate remain
open. Public CLI policy, SDK ABI, DB, dependencies, product deadlines, CI/runners
and repository policy remain unchanged; no push, dispatch, release or deployment.
