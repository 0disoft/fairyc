# Output and Exit Codes

- Status: Draft
- Repository Type: cli-tool

## Repository Type Contract

This repository type owns command behavior, arguments, flags, config loading, exit codes, terminal output, JSON output, runtime compatibility, and shell integration contracts.

## Source of Truth

- Product decision: `docs/cli/command-contract.md`
- Technical owner: UNASSIGNED
- Related ADRs: `docs/adr/0027-experimental-c-emission-cli.md`,
  `docs/adr/0028-experimental-linux-scalar-build-cli.md`, and
  `docs/adr/0180-stable-workspace-c-emission-v1.md`, and
  `docs/adr/0181-stable-workspace-semantic-test-v1.md`, and
  `docs/adr/0204-stable-starter-project-creation-v1.md`, and
  `docs/adr/0206-starter-project-guide-v2.md`, and
  `docs/adr/0207-stable-windows-workspace-run-v2.md`, and
  `docs/adr/0208-starter-project-guide-v3.md`

## Required Decisions

- Broader command list and remaining flag ownership: UNDECIDED
- Exit-code taxonomy outside the stabilized command subsets: UNDECIDED
- Machine-readable output contract: UNDECIDED
- Config precedence and default behavior: UNDECIDED
- Runtime compatibility floor: UNDECIDED

## Prototype frontend v0

`fairyc new <package-name> --uid <package-uid>` writes one success line only after all five starter
files, including the compiler-owned guide, are created and synced. It stabilizes code `0` for success, `2` for argument or identity rejection,
`4` for internal/output failure, and `7` for parent, existing-target, permission, filesystem, or
rollback failure. An existing target produces no stdout and is never opened or changed. Errors name
only the relative requested child, never the absolute parent or raw host error. If stdout fails
after creation, code `4` cannot roll back the already completed project.

`fairyc check [--locked]` stabilizes a bounded subset of the numeric taxonomy: `0` success, `2`
invalid invocation, `3` source diagnostics, `4` internal or output failure, and `7` package,
manifest, or lock failure. Success writes exactly one `check ok:` line documented by ADR 0175;
failures emit no partial success. Workspace syntax errors retain their ordinary diagnostic codes and
positions on stderr instead of being collapsed into package failure text. No stable JSON mode exists.

`fairyc build [--release] [--locked]` stabilizes codes `0`, `2`, `3`, `4`, `6`, and `7` for
success, invalid invocation or unsupported admitted subset, source diagnostics, internal/output
failure, toolchain/link/artifact failure, and package/manifest/lock failure. Success writes exactly
`built: <path> (x86_64-windows-msvc, <development|release>, lock
<updated|unchanged|verified>)`. No success line is emitted before the inspected executable replaces
the prior default artifact. An output-stream failure after replacement returns `4` but does not
delete the successfully built artifact. No stable JSON mode exists.

`fairyc run [--release] [--locked] [-- <program-args...>]` v4 emits no compiler success line. After
successful compilation and image inspection it attaches caller stdin without pre-reading it and
forwards arguments after `--` to the program's process arguments. After launch it forwards bounded
program stdout and stderr to their matching streams, then preserves an ordinary
`main` return as the complete 32-bit process status. A recognized language trap is identified by a
matching byte on the separate private control pipe, preserves all program stderr, appends exactly `trap: <category>\n`, and returns
`5`. Therefore codes `2`, `3`, `4`, `5`, `6`, and `7` mean compiler failures only before launch; the
same ordinary post-launch values are program statuses. A forwarding failure after execution is code
`4` and does not undo earlier bytes or program effects. An input transport failure after launch is an
execution failure with the same no-rollback limit. The shared captured-output ceiling is 64 KiB;
stdin has no separate byte ceiling and is consumed only on program demand. Cross-stream order is
unspecified, program arguments forward verbatim after `--` with embedded NUL rejected before
compilation, and no temporary executable is retained.

`fairyc explain <diagnostic-code>` stabilizes codes `0`, `2`, and `4` for successful exact lookup,
invalid invocation or unknown code, and output failure. Success writes exactly three ANSI-free lines
with code/meaning, category, and the provisional Stage 0 catalog identity. Failure emits no partial
stdout. The command has no filesystem, process, network, target, or toolchain side effect.

The implemented `fairyc experimental` commands use provisional process codes only for prototype
automation. They are not the stable exit-code taxonomy:

| Code | Prototype meaning |
| ---: | --- |
| `0` | The requested prototype operation completed successfully. |
| `2` | Invocation, diagnostic-code lookup, or bounded input loading failed, the experimental semantic subset cannot represent the requested syntax, or an interpreter capability limit was reached. |
| `3` | One or more source diagnostics were produced. |
| `4` | An internal invariant or output write failed. |
| `5` | The experimental MIR interpreter reached a canonical language trap. |
| `6` | The experimental native build could not resolve or run its toolchain, link the scalar runtime, or safely materialize its artifact. |
| `7` | The package workspace, lockfile, package graph, or package-to-build mapping failed before native compilation. |

`experimental atlas-text source` and `documentation` write exactly one compact canonical
`fairyc.attributed-text-json/v0` value without a trailing newline on success. An admitted fragment
that exceeds the explicit fragment or aggregate text limit is still code `0`: the document contains
one checksummed `fragment-byte-limit` omission and no text record. An undocumented declaration under
the documentation action is also code `0` with one `kind-unavailable` omission. Missing, duplicate,
unknown, noncanonical, or unsupported options; an unknown package or exact SymbolId; and a complete
envelope that exceeds `--max-output-bytes` use code `2` with empty stdout. Source diagnostics use
`3`; internal identity/invariant and output-write failures use `4`; manifest, workspace, graph, and
lock failures use `7`. Ordinary mode may update `fairy.lock` before a later selector or budget
failure; `--locked` does not write. An external writer can accept a prefix before returning code `4`,
so stream rollback is impossible even though encoder-side size failure is atomic.

`experimental atlas-evidence show` writes exactly one compact canonical
`fairyc.atlas-evidence-json/v0` value without a trailing newline. Successful attributed-text
`kind-unavailable`, safety, or size omissions keep code `0` and remain independently checksummed
inside their child documents. Missing, duplicate, unknown, noncanonical, or unsupported options; an
unknown package or exact SymbolId; insufficient query selection; and a facts, attributed-text child,
or complete bundle exceeding its explicit bound use code `2` with empty stdout. Source diagnostics
use `3`; internal identity/invariant and output-write failures use `4`; manifest, workspace, graph,
and lock failures use `7`. Ordinary mode may update `fairy.lock` before a later failure; `--locked`
does not write. Encoder-side failures are atomic, but an external writer can still retain a prefix
before returning code `4`.

The stable `fairyc explain` output is exactly three deterministic, ANSI-free lines on success: the
canonical code and meaning, its category, and `fairyc.diagnostic-catalog/v0 (provisional Stage 0)`.
It reads no source or configuration and has no filesystem, process, network, toolchain, target, or
write side effect. Missing, extra, malformed, and unknown codes use code `2` with empty stdout.
Output-write failure uses code `4`. The successful output reserves a catalog meaning and does not
claim implementation, fixture coverage, or release status.

`fairyc targets` writes one deterministic, ANSI-free human block per validated TargetSpec in
canonical ID order. Each block includes schema, emission subset, three independent maturity states,
and sorted missing-evidence markers. The command accepts no arguments and has no source,
configuration, filesystem, process, network, toolchain, target-probe, WSL, or write side effect.
The block grammar, blank-line separation, final newline, and canonical ordering are stable while
TargetSpec rows and evidence evolve independently. Success uses code `0`; unexpected arguments use
code `2`; invalid compiler-owned target data and output-write failure use code `4`, without partial
stdout.

`fairyc emit c ...` is no longer a stable command. It writes no stdout and returns code `2` with a
migration direction to `fairyc experimental emit-workspace-c --target <target-id> [--locked]`.
The experimental command writes exactly one raw same-version deterministic ISO C11 application
unit to stdout. It adds no banner, receipt, JSON envelope, ANSI escape, absolute path, or FY source
text. Compilation and emission failures write no unit; an output writer may accept a prefix before
returning a code-`4` transport error, and that stream cannot be rolled back. Success uses code `0`;
invalid form/flags, missing or unknown target, missing/invalid root entry, and unsupported backend
capability use code `2`; source diagnostics use code `3`; internal and output failures use code `4`;
manifest, workspace, graph, and lock failures use code `7`. The command writes no output artifact,
but ordinary lock maintenance may precede emission or an output failure. `--locked` verifies the
lock without writing.

`fairyc test [--json] [--locations] [--native [--release] [--stdin | --stdin-file <portable-workspace-file>] [--timeout-seconds <1..60>] [--capture-output] [-- <program-args>...]] [--workspace | --package <package-uid>] [--dir <portable-test-directory>]... [--file <portable-test-path>]... [--exact <test-identity>]... [--locked]` buffers one deterministic report after compilation and capability
admission. Default and singular-package identities are `<module>::<function>`; workspace identities
are `<package-uid>::<module>::<function>`. Completed tests use `test <identity> ... ok` or `test
<identity> ... FAILED (<trap-category>)`, followed by `test result:
<ok|FAILED>. <passed> passed; <failed> failed; lock <updated|unchanged|verified>`. Zero tests emit
only the successful final line. Without `--json`, the command writes no ANSI escape, absolute path,
source text, artifact, JSON, or native program stream. Default mode uses the semantic interpreter. `--native`
uses one inspected temporary Windows image per test but retains the same report; native program
output is rejected instead of mixed into it. Arguments after the first `--` are forwarded only in
native mode and do not appear in the report. Explicit native-only `--stdin` requires exactly one
selected test after complete compiler admission and attaches caller input directly after PE
inspection; default input remains EOF, and stdin bytes do not appear in the report. Success uses code `0`; invocation, interpreter/backend
capability, or native-program-output rejection uses `2`; source and `E9001` test-declaration
diagnostics use `3`; internal and output failures use `4`; one or more language traps use `5`;
native toolchain/build/inspection/execution failure uses `6`; workspace, test-source, graph, and lock
failures use `7`. Exact selectors preserve canonical report order; missing, duplicate, or unknown
selectors use code `2` with no report. Missing, duplicate, unknown, unreachable, or
feature-instance-ambiguous package selection also uses code `2` with no report. A trap does not stop later selected tests.
Exact file selectors use the portable workspace-relative path domain emitted by `--locations`,
preserve canonical report order, and likewise reject missing, duplicate, or unknown paths with code
`2`. A known zero-test file succeeds with the ordinary zero-test report. `--file` and `--exact` are
mutually exclusive, while the complete selected test overlay is still compiler-admitted. Exact
directory selectors use only ancestor directories derived from that admitted file catalog, recurse
by a `/` segment boundary, preserve canonical order, and reject missing, duplicate, alternate-
separator, case-mismatched, dot-segment, and trailing-slash spellings with code `2`. A directory
containing only admitted zero-test files succeeds; an empty filesystem directory is not cataloged.
`--dir` is mutually exclusive with `--file` and `--exact`.
Compilation/capability failure emits no report;
an external writer may still accept a report prefix before returning code `4`. Ordinary lock
maintenance can precede later failure, while `--locked` is the no-write form.

Native `--json --capture-output` selects `fairyc.test-report/v4` and accepts bounded program output.
Every outcome requires `stdoutBase64` and `stderrBase64`; canonical padded base64 preserves arbitrary
binary bytes and empty streams are empty strings. Trap control is carried outside stderr, so every
program stderr byte is encoded unchanged. `source` remains conditional on `--locations`.
Without the flag, output is still rejected as code `2`. Capture-limit or execution-infrastructure
failure emits no JSON report, and no report represents cross-stream ordering.

Execution-mode `--json` emits one compact newline-terminated object after every selected outcome is
classified. Default and `--package` use `fairyc.test-report/v1`; `--workspace` uses
`fairyc.test-report/v2`. It contains the exact `engine`, nullable `target` and
`profile`, canonical `tests` outcomes with nullable trap categories, matching `count`, `passed`, and
`failed` totals, aggregate `status`, and `lock`. Interpreter target/profile are null; native target
is `x86_64-windows-msvc` with `development` or `release`. All-passing still exits 0; one or more
language traps writes the complete JSON and exits 5. Failures that previously produced no human
report likewise produce no JSON report, and writer failure remains code 4 after a possible prefix.
With `--locations`, JSON uses `fairyc.test-report/v3` and every outcome adds the exact-current
portable source anchor. Without it, root/package v1 and workspace v2 bytes remain unchanged.

`fairyc test --list [--json] [--locations] [--workspace | --package <package-uid>] [--dir <portable-test-directory>]... [--file <portable-test-path>]... [--locked]` buffers one `test <identity>` line per admitted test in
canonical order, followed by `test list: <count> discovered; lock
<updated|unchanged|verified>`. Zero tests emit only the final line. Listing performs the complete
test-overlay compiler admission but executes no test, backend, toolchain, or subprocess and writes
no artifact. Success uses code `0`; incompatible or duplicate flags and compiler capability
rejection use code `2`; source and `E9001` diagnostics use code `3`; internal and writer failures
use code `4`; workspace, test-source, graph, and lock failures use code `7`. Codes `5` and `6` are
unreachable because listing has no language-test or native-build outcome. Compilation failure emits
no list report; ordinary lock maintenance may already have completed, while `--locked` writes
nothing. `--json` replaces only the success bytes with one compact newline-terminated object:
default and `--package` use `fairyc.test-list/v1`, while `--workspace` uses
`fairyc.test-list/v2`. Both contain `schemaVersion`, canonical `tests`, matching `count`, and
`lock` (`updated`, `unchanged`, or `verified`). Without `--locations` it adds no ANSI escapes,
paths, source text, metadata, or host inventory and retains exact text/JSON bytes. With
`--locations`, text appends the portable path and 1-based position, while JSON uses
`fairyc.test-list/v3` and adds the equally ordered exact-current `locations` array. Neither form
emits source text or an absolute host path.

`experimental parse` writes one `fairyc.experimental.parse/v0` JSON document, defined by
`spec/cli/experimental-parse-v0.schema.json`, to stdout even when source diagnostics exist.
`experimental check-syntax` writes human diagnostics to stderr and does not emit a machine result.
`experimental check-semantics` writes one success line with typed-HIR module and function counts for
a multi-file input, semantic diagnostics on stderr, or an explicit unsupported-subset message on stderr.
`experimental check` writes exactly
`check ok: <files> file(s) (verified MIR: <functions> function(s), <foreign> foreign declaration(s))`
after the complete explicit closure passes MIR verification. It emits no partial success output on
source diagnostics or subset failure and requires no executable entry, backend, toolchain, or write.
`experimental check-workspace-session` reads at most the caller's bounded number of versioned NDJSON
requests and writes exactly one `fairyc.experimental.workspace-check-session/v0` response line for
each completed request. Response status/code pairs are `ok`/`0`, `source-diagnostics`/`3`, and
`package-failure`/`7`. Human source and package diagnostics remain on stderr. Malformed framing uses
code `2` with no response for the rejected request; internal, stdin-read, and stdout-write failures
use code `4` and terminate. EOF or the request bound returns the last completed request code, so a
later successful repair can make the session process return `0`. Empty input uses code `2`.
`experimental check-workspace-cache` writes exactly one
`fairyc.experimental.workspace-cache-check/v0` object on success, defined by
`spec/cli/experimental-workspace-cache-check-v0.schema.json`. It reports exact executable identity,
relative cache directory, target/lock/workspace/MIR counts, verified reuse, retained state, raw record
counters, and skipped or failed stores. Source or package diagnostics produce no partial JSON and use
the existing codes `3` or `7`; invalid flags and limits use `2`; invariant, executable-identity, or
output failures use `4`.
`experimental atlas-pack overview` writes exactly one canonical
`fairyc.context-pack-json/v1` value with no trailing newline. It requires an exact manifest-v1
package UID plus explicit record, query-selection-byte, and final-output-byte limits. Invalid or
unknown selectors, malformed limits, and insufficient budgets use code `2`; semantic diagnostics
use `3`; manifest, workspace, source-loading, graph, and lock failures use `7`; executable identity,
compiler invariant, and output failures use `4`. Ordinary lock maintenance may precede a later
failure; `--locked` is no-write. No partial pack is intentionally emitted, though an external writer
may accept a prefix before returning an output failure.
`experimental atlas-pack show` and `experimental atlas-pack around` use the same output and code
families. Malformed visible/full SymbolIds, unsupported directions or relations, duplicate policy,
unknown exact symbols, and insufficient record/relation/depth/selection/output budgets use code `2`.
`around` accepts canonical depth zero through 64; `show` rejects a depth option. Both emit no partial
pack before output begins and preserve the same external-writer prefix caveat.
`experimental atlas-symbols list` writes exactly one canonical
`fairyc.symbol-catalog-json/v1` value with no trailing newline. Invalid or duplicate kinds,
malformed or duplicate exact names, malformed full-ID cursors, cursors outside the complete
module/name/kind-filtered set, unknown packages or modules, and insufficient symbol,
selection-byte, or final-output budgets are invocation failure `2`. Zero exact-name matches are
successful complete output. Workspace/lock failures are `7`, source diagnostics
are `3`, and compiler invariants or output failures are `4`. No partial catalog is intentionally
written on pre-output failure; an external writer may retain a prefix when its own write fails.
`experimental atlas-symbols list --catalog-version 2` instead writes one canonical
`fairyc.symbol-catalog-json/v2` page with original generic declaration identities and v2 currentness
context. Explicit `--catalog-version 1` is byte-identical to the default. Invalid or duplicate
versions use code `2`; source diagnostic, workspace/lock, budget, and output failures retain the
existing command policy. No partial output is intentionally emitted on failure.

`experimental atlas-calls list` writes exactly one canonical
`fairyc.workspace-call-catalog-json/v0` value with no trailing newline. It requires an exact package
UID, canonical visible/full SymbolId, outgoing/incoming/both direction, and explicit positive
package, call, and final-output limits. Invalid or duplicate policy, unknown packages or symbols,
construction-budget exhaustion, and final-output overflow use code `2`; source diagnostics use `3`;
manifest, workspace, graph, and lock failures use `7`; internal join contradictions and output
failures use `4`. A valid zero-call selection is successful complete output. No partial catalog is
intentionally written before output begins; an external writer may retain a prefix when its own
write fails.
`experimental target-spec` writes one deterministic `fairyc.target-spec/v0` JSON document, defined
by `spec/cli/experimental-target-spec-v0.schema.json`, to stdout. Missing, extra, or unknown target
IDs produce code `2`, no JSON, and a bounded invocation diagnostic on stderr.
`experimental emit-c` writes exactly one deterministic C11 translation unit to stdout on
success. It adds no banner, JSON envelope, ANSI escape, absolute path, or source text. Source
diagnostics and capability failures produce no partial C output.
`experimental build` writes a successful executable only to the explicit ignored `build/` path and
then writes `built: <relative-path> (x86_64-linux-gnu, <compiler>, <profile>)` to stdout. Toolchain,
link, and artifact failures write bounded human diagnostics to stderr with provisional code `6`.
An existing output is not replaced and no partial C translation unit is written to stdout.
`experimental build-target` writes a successful PE only to the explicit ignored `.exe` path below
`build/`, then writes `built: <relative-path> (x86_64-windows-msvc, msvc <version>, <profile>)` to
stdout. Invalid or unavailable targets use code `2`; source diagnostics use code `3`; an admitted
source outside the provisional target subset uses code `2`; toolchain, compile, link, and artifact
failures use code `6`. The exact admitted SystemHeap, Buffer, Arena, owned-text/CStr, and UTF-8 standard-library calls are linked automatically;
every other foreign or unproven standard-library symbol fails with code `2` before toolchain
resolution. An existing output is never replaced.
`experimental run-target` writes
`native result: <i32> (x86_64-windows-msvc, msvc <version>, <profile>)` on a zero-output ordinary
execution outside the reserved product-trap status range. Reserved statuses write
`trap: <category>` to stderr and return code `5`; this experimental command cannot distinguish a
normal result in that range.
Source and target-subset failures use codes `3` and `2`; toolchain, compile, link, PE/unwind
inspection, unexpected program output, and execution failures use code `6`. It writes no artifact
below the source root.
`experimental build-workspace` writes a successful PE only to the explicit ignored `.exe` path,
then writes `built workspace: <relative-path> (x86_64-windows-msvc, msvc <version>, <profile>, lock
<state>)`. `experimental run-workspace` retains no PE and writes `native workspace result: <i32>
(x86_64-windows-msvc, msvc <version>, <profile>, lock <state>)`. Lock state is `updated`,
`unchanged`, or `verified`. Package/lock failures use code `7`; source diagnostics use code `3`;
entry or backend capability failures use code `2`; toolchain, compile, link, PE/COFF/unwind,
unexpected-output, and execution failures use code `6`; canonical traps use code `5`. Neither
command emits partial success output. Workspace run consumes a compiler-private trap token, so all
normal `i32` results remain distinct from code-5 traps.
`experimental exec-target` instead writes the program's captured stdout and stderr bytes to their
matching streams without a compiler receipt, then returns the program's full 32-bit Windows process
status. A nonzero status in this command is a program result, not automatically a compiler failure.
The streams share one 64 KiB capture budget and have no cross-stream ordering promise. Toolchain,
compile, link, PE/COFF/unwind inspection, timeout, capture-limit, and CLI output-write failures keep
their ordinary prototype diagnostic codes and do not masquerade as program statuses.
`experimental build-host` writes a successful PE only to the explicit ignored `.exe` path below
`build/`, then writes `built: <relative-path> (windows/amd64 host-smoke, msvc <version>, <profile>)`
to stdout. It uses code `6` for toolchain, compile, or artifact failures and never replaces an
existing output.
`experimental run-host` writes `native result: <i32> (windows/amd64 host-smoke, msvc <version>,
<profile>)` after a target-neutral temporary MSVC build and execution. A canonical runtime trap uses
the same `trap: <category>` stderr form and code `5`; toolchain, compile, temporary-artifact, timeout,
unexpected-output, or execution failures use code `6`. It writes no persistent executable.
`experimental run` writes `result: <i32>` to stdout on return, or `trap: <category>` to stderr with
provisional code `5` on a language trap. Its bounded `step-limit` is a capability result with code
`2`, not a language trap. None of these commands
uses ANSI escapes. `fairyc explain` and `fairyc targets` likewise use no ANSI escapes. Human positions are one-based line and byte-column values backed
by the diagnostic's zero-based half-open byte span.

The parse JSON schema intentionally excludes source text, token text, cooked literal values, and
absolute paths. The target JSON schema excludes local compiler, SDK, and sysroot paths. The
workspace-session schema excludes source text, source paths, artifacts, and toolchain paths while
reporting bounded cache and compiler decisions. These reports can therefore describe structure,
target semantics, or sequential check outcomes without becoming a second source copy or host
inventory. Stable test JSON behavior is defined by `spec/cli/stable-test-list-v1.schema.json` for
listing and `spec/cli/stable-test-report-v1.schema.json` for semantic or native execution.

## Review Blockers

- A command changes without updating help, examples, output, and exit-code expectations.
- JSON output exposes generated or existing file contents.
- Runtime compatibility changes without smoke validation.
