# Authored private state-command options, 2026-11-07

This slice adds a pure Fairy argument parser for the **existing developer**
`tools/dev/fairyc-state` explicit native-graph invocation. The authority is
`tools/dev/fairyc-state/main.go`, `native.go`, and ADR 0500's explicit native
graph developer mode. Those authorities are unchanged.

This is a prerequisite for a real authored developer entry. It does not select
an entry, acquire arguments from the OS, load source, open a store, build or run
a graph, produce a command report, or implement cancellation. It neither replaces
the existing Go command nor completes the broader authored command queue.
No public main CLI surface is added.

## Exact API

Module: `examples/lexer_candidate/src/module_state_command_options.fy`.

```fairy
pub struct Options {
    u32 action;       // numeric action: 1 init, 2 execute, 3 snapshot
    bool release;     // false low, true optimized
    usize store_index;  // always 1, into the original argv
    usize source_index; // always 2, into the original argv
}
pub enum Parsed { Help; Native { Options options; }; }
pub enum InvocationError { Invalid; UnsupportedGoMIR; TooLarge; Nul; }
pub fn Result<Parsed,InvocationError> parse(ref v::Vec<bytes::Buffer> argv);
pub fn i32 exit_code(InvocationError error); // 2 for every invocation error
pub fn usize byte_limit();                  // 1048576 aggregate bytes
```

`v` means `std::vec`; `bytes` means `std::bytes`. The vector excludes the
executable. `Options` and both enums contain only copied scalars, with no borrowed
path or retained buffer ownership. The caller retains the original argv for host
dispatch and must keep it if it later wants to dereference the returned indexes.
The parser allocates nothing and accepts an immutable reference. It never trims,
normalizes, rewrites or copies argument bytes, or performs filesystem/process I/O.
Returned scalar values remain usable after the caller destroys its arguments.

Exactly these forms are admitted:

```text
--help
<init|execute|snapshot> <nonempty-store> <source.fy> --engine native-graph --profile <low|optimized>
```

The seven argument positions and case are fixed. The source predicate mirrors
Windows `filepath.Ext(path) == ".fy"`: scan backward to the first dot or either
directory separator. `.fy` itself is valid; `.FY`, a trailing separator, a dot
in a parent component alone, a trailing space/dot, and `.fy:stream` fail.
There is no path cleanup, existence check, whitespace policy or UTF-8 validation.
For example, a nonempty whitespace store or a store spelled `--help` is admitted
as a literal store path. Only **standalone** `--help` returns `Parsed.Help`.

The original three-argument Go-MIR form returns `UnsupportedGoMIR`; it never
silently selects graph admission or an interpreter fallback. This private parser
does not claim the full existing Go developer command's accepted language.

Admission order and bounded work:

1. Reject counts other than 1, 3, or 7 as `Invalid`, without traversing argv.
2. Check all argument lengths against the 1 MiB aggregate budget before scanning
   bytes. Use subtraction before addition to avoid overflow. Reject as `TooLarge`.
3. Scan admitted bytes for NUL and return `Nul` before any future host admission.
4. Recognize standalone help; classify any three-argument vector as unsupported;
   otherwise validate the exact native syntax and return `Invalid` on mismatch.

This uses constant auxiliary space and bounded linear work (at most 1 MiB of
NUL scanning, plus bounded comparisons and a source extension scan). NUL and the
private byte budget are explicit admission restrictions beyond the Go syntax
predicate. Too-large input is never truncated, even if it contains help or NUL.
All parser failures map to invocation exit 2. Help maps to success 0 at the future
entry. Existing source rejection 3, I/O/backend 4, transaction rejection 6,
conflict/layout/busy 7 and language trap 8 stay with the execution owner; the
parser does not manufacture runtime outcomes.

## Verification receipt

Final Windows amd64 command, from the repository root:

```powershell
go test -mod=readonly ./internal/cli -run '^TestCandidateStateCommandOptions$' -count=1 -v -timeout 5m
go vet -mod=readonly ./internal/cli
gofmt -l internal/cli/candidate_state_command_options_test.go
git diff --check -- examples/lexer_candidate/src/module_state_command_options.fy internal/cli/candidate_state_command_options_test.go docs/development/candidate-state-command-options-20261107.md
```

The final test passed in **10.279s** (test body 10.17s): **116 independent cases
per profile, 232 native parser processes** across actual Windows MSVC low and
optimized builds, with no selected skips. The final verbose output was filtered
through PowerShell `Select-String` for source hashes, observation counts, failures
and pass/timing lines. Vet and gofmt passed without output. The explicit-path
diff check passed; since these three new files were untracked, their whitespace
and final newlines were also inspected directly before delivery.

The independently authored accept/reject matrix covers all action/profile pairs,
empty arguments in every position, case-sensitive source extensions, slash and
backslash component boundaries, raw non-UTF-8 bytes, whitespace retention,
extra/reordered/duplicate flag spellings, every help position, arity, explicit
Go-MIR refusal, NUL in every slot and before `.fy`, 64 KiB arguments, exactly
1 MiB, one-byte-over budget, and error-precedence controls. A separate Go syntax
reference uses the actual Windows `filepath.Ext` behavior and the existing
seven-position gate. NUL/budget refusals are checked as deliberate private
admission restrictions rather than misrepresented as full Go CLI equivalence.
The developer runner is not invoked by these parser tests.

Each native probe reads a bounded test-only binary stdin envelope, constructs
`Vec<bytes::Buffer>`, invokes the authored parser, checks **every input byte and
length on both success and failure**, destroys argv, then consumes and copies
the scalar result. NUL and large inputs are never OS process arguments. The
envelope/main is test-only and has a separate 2 MiB input/1024-element guard.

Earlier attempts are not counted as passing validation: initial test-probe
scalar-reference syntax and unconsumed-Result errors were corrected in this
owned test file. Concurrent cancellation test files temporarily blocked package
compilation on `WAIT_TIMEOUT` types. A temporary explicit-file attempt first
included a Linux-only test (`syscall.Mkfifo`), then correctly selected Go's host
file lists and reached the probe error. No cancellation/host file was edited.
The final canonical package command and vet above pass without excluding files.

The reused `candidateAtlasProbeSize` / `candidateParityBuildPartsWithLimit`
helpers only emit the Windows profile; they do not offer both-target emission.
Linux emission and execution were not run. State execution/persistence,
cancellation and end-to-end authored command entry remain separate work.

Coordinator commit-scope `check` / `check --locked` pass in **13.434s / 12.604s**,
**749 files / 9,433 MIR functions**, empty stderr. All **1,219 authored files**
match `.cache/state-options-check-3525fe2d`. The pending cancellation module is
excluded and its capture/selector/pin edits are restored from HEAD in this
isolated copy; no uncommitted cancellation code is credited to this commit.
The successful lock is copied back. Record: `.cache/state-options-check.json`.

## Source identities

SHA-256 over exact on-disk bytes for this receipt:

| Source | SHA-256 |
| --- | --- |
| `examples/lexer_candidate/src/module_state_command_options.fy` | `9023bf61111e79721107a71a75c2e321aceb58771013fb0fa161bedd3a185666` |
| `internal/cli/candidate_state_command_options_test.go` | `73c84b5cdb6c31c8b56767eeead1617eceb3391f9892d5ce10f5a3c35a3bfba7` |
| `tools/dev/fairyc-state/main.go` | `4f7d237250a899e90ee3570041bce2eb8715c1237c60d5007e05f367f5efca84` |
| `tools/dev/fairyc-state/native.go` | `16b92964eb6645561b061ae78b3ea5377a3221c3c58e74cf5a30c008f2b78497` |
| `docs/adr/0500-local-contract-state-execution-and-persistence.md` | `ae4673ed89b0c145443ceff7f79977b182293f82b035f8631ea86a37029a4e20` |

The worker adds only the module, test and this receipt. The coordinator reviews,
updates the source index/work queue and verified lock, and commits this slice.
No ADR, SDK, database, runner, dependency, public CLI or remote change is made.
