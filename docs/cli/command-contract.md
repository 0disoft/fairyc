# Command Contract

- Status: Draft
- Repository type: cli-tool
- Runtime implementation: Go Stage 0

This document owns planned `fairyc` command semantics. It does not claim that a command is implemented. Implemented commands must be marked with tests and a language revision before release.

## Implemented experimental frontend surface

The current Go prototype implements two deliberately unstable developer commands:

| Command | Purpose | Output |
| --- | --- | --- |
| `fairyc experimental parse <file.fy>` | Parse one source-root-relative file and verify lossless syntax plus error-free AST lowering | `fairyc.experimental.parse/v0` JSON on stdout, defined by `spec/cli/experimental-parse-v0.schema.json` |
| `fairyc experimental check-syntax <file.fy>` | Check lexing, parsing, recovery, module identity, lossless reconstruction, and error-free AST lowering | Success line on stdout or human diagnostics on stderr |

These commands treat the current directory as their temporary source root, accept exactly one
relative regular file, reject root escape including escaping symbolic links, and read no more than
the current implementation byte budget. Their spelling and numeric exit codes are prototype-only.
The JSON report includes relative path, byte length, module identity, token kinds, CST node kinds,
half-open byte spans, child indexes, and structured diagnostics. It excludes source text, token
text, cooked literal values, and absolute local paths. When bounded loading detects more bytes than
the implementation limit, `truncated` is true and `byteLength` is a lower bound rather than a
claimed full-file size.

`fairyc check` remains unimplemented. It must not be aliased to `check-syntax`, because the planned
command below also requires name resolution, typing, resource, borrow, region, and effect analysis.

## Stage 0 surface

| Command | Purpose | Code generation |
| --- | --- | --- |
| `fairyc check [target]` | Parse, resolve, type-check, and run resource, borrow, region, and effect analysis | No |
| `fairyc build [target]` | Compile and link the selected target with the configured backend | Yes |
| `fairyc run [target] -- [args...]` | Build and run a program while preserving the program exit status | Yes |
| `fairyc test [target]` | Build and run semantic test targets | Yes |
| `fairyc emit <ast|hir|mir|c> [target]` | Emit a requested compiler representation for inspection | Requested form only |
| `fairyc explain <diagnostic-code>` | Explain a stable diagnostic code | No |
| `fairyc targets` | Report known target profiles, status, evidence, and unavailable requirements | No |

Stable positional syntax, package selection, workspace selection, target triples, feature flags,
and configuration precedence remain UNDECIDED. The implemented `experimental` spellings do not
settle those stable decisions.

## General requirements

- Commands are non-interactive by default unless an explicit future subcommand states otherwise.
- Input, output, and cache paths are resolved under documented roots and reject unsafe traversal.
- Human diagnostics go to stderr; requested artifacts and machine-readable results go to stdout or an explicit output path.
- `--json` output uses a versioned schema and contains no ANSI escapes.
- Machine-readable build and target receipts identify the applicable compiler, language, MIR, target-schema, ABI-profile, runtime, standard-library, diagnostic, package, and Atlas compatibility contracts rather than collapsing them into one compiler version.
- Deterministic commands preserve ordering across identical inputs.
- `check` must not invoke a backend or linker.
- `emit` must not silently choose a different semantic profile from `build`.
- `run` separates compiler arguments from program arguments with `--`.
- Cache inspection and cleanup are distinct from project output cleanup.
- `targets` reports `emittable`, `conformant`, and `supported` independently and cannot promote a profile while a required fixture is skipped or absent.

## Planned profile flags

The stable spelling is UNDECIDED, but the semantic profiles are fixed:

- Stage 0 development: C reference backend, low optimization, bounded reuse, and requested debug information;
- Stage 0 release: C reference backend, ordinary optimization, stripped artifact, and no default LTO;
- later development and production: backend choice is enabled only by a separate measured contract;
- release LTO: explicit ThinLTO request;
- profiling: explicit instrumentation generation and profile use;
- unsafe optimization: source-local unsafe or unchecked operations, never a project-wide silent semantic switch.

## Exit categories

Numeric codes remain UNDECIDED. The CLI must distinguish success, invalid invocation or configuration, source diagnostics, backend or linker failure, toolchain or target unavailability, and executed-program exit status.

## Review blockers

- Help, documentation, JSON schema, examples, tests, and exit behavior disagree.
- A normal development command enables expensive global optimization.
- A build option changes safe-language semantics.
- Output embeds absolute local paths without an explicit diagnostic or debug contract.
- A machine-readable result includes source contents, secrets, or unbounded compiler logs by default.
