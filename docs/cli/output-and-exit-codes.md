# Output and Exit Codes

- Status: Draft
- Repository Type: cli-tool

## Repository Type Contract

This repository type owns command behavior, arguments, flags, config loading, exit codes, terminal output, JSON output, runtime compatibility, and shell integration contracts.

## Source of Truth

- Product decision: UNDECIDED
- Technical owner: UNASSIGNED
- Related ADR: UNDECIDED

## Required Decisions

- Command list and flag ownership: UNDECIDED
- Exit-code taxonomy: UNDECIDED
- Machine-readable output contract: UNDECIDED
- Config precedence and default behavior: UNDECIDED
- Runtime compatibility floor: UNDECIDED

## Experimental frontend v0

The implemented `fairyc experimental` commands use provisional process codes only for prototype
automation. They are not the stable exit-code taxonomy:

| Code | Experimental meaning |
| ---: | --- |
| `0` | The requested syntax operation completed without source diagnostics. |
| `2` | Invocation or bounded input loading failed, the experimental semantic subset cannot represent the requested syntax, or an interpreter capability limit was reached. |
| `3` | One or more source diagnostics were produced. |
| `4` | An internal invariant or output write failed. |
| `5` | The experimental MIR interpreter reached a canonical language trap. |

`experimental parse` writes one `fairyc.experimental.parse/v0` JSON document, defined by
`spec/cli/experimental-parse-v0.schema.json`, to stdout even when source diagnostics exist.
`experimental check-syntax` writes human diagnostics to stderr and does not emit a machine result.
`experimental check-semantics` writes one success line with typed-HIR module and function counts for
a multi-file input, semantic diagnostics on stderr, or an explicit unsupported-subset message on stderr.
`experimental emit-c` writes exactly one deterministic ISO C11 translation unit to stdout on
success. It adds no banner, JSON envelope, ANSI escape, absolute path, or source text. Source
diagnostics and capability failures produce no partial C output.
`experimental run` writes `result: <i32>` to stdout on return, or `trap: <category>` to stderr with
provisional code `5` on a language trap. Its bounded `step-limit` is a capability result with code
`2`, not a language trap. None of these commands
uses ANSI escapes. Human positions are one-based line and byte-column values backed
by the diagnostic's zero-based half-open byte span.

The JSON schema intentionally excludes source text, token text, cooked literal values, and absolute
paths. A report can therefore identify structure and spans without becoming a second copy of the
source file. Stable `--json` behavior remains UNDECIDED.

## Review Blockers

- A command changes without updating help, examples, output, and exit-code expectations.
- JSON output exposes generated or existing file contents.
- Runtime compatibility changes without smoke validation.
