# CLI Configuration

- Status: Draft
- Repository Type: cli-tool

## Repository Type Contract

This repository type owns command behavior, arguments, flags, config loading, exit codes, terminal output, JSON output, runtime compatibility, and shell integration contracts.

## Source of Truth

- Product decision: UNDECIDED
- Technical owner: UNASSIGNED
- Related ADR: `docs/adr/0159-bounded-workspace-check-session.md` for the unstable bounded session only

## Required Decisions

- Command list and flag ownership: UNDECIDED
- Exit-code taxonomy: UNDECIDED
- Machine-readable output contract: UNDECIDED
- Config precedence and default behavior: UNDECIDED
- Runtime compatibility floor: UNDECIDED

## Implemented provisional session inputs

`experimental check-workspace-session` accepts no config file or environment-variable defaults.
Its target, request ceiling, cache entry ceiling, retained-byte ceiling, artifact-byte ceiling, and
optional exact-lock mode are explicit command arguments. The process reads only the versioned
NDJSON request stream from stdin. These inputs are unstable and do not settle precedence or default
behavior for the future release-stable CLI.

## Review Blockers

- A command changes without updating help, examples, output, and exit-code expectations.
- JSON output exposes generated or existing file contents.
- Runtime compatibility changes without smoke validation.
