# CLI Configuration

- Status: Draft
- Repository Type: cli-tool

## Repository Type Contract

This repository type owns command behavior, arguments, flags, config loading, exit codes, terminal output, JSON output, runtime compatibility, and shell integration contracts.

## Source of Truth

- Product decision: UNDECIDED
- Technical owner: UNASSIGNED
- Related ADRs: `docs/adr/0159-bounded-workspace-check-session.md` and `docs/adr/0166-experimental-persistent-workspace-cache-command.md`

## Required Decisions

- Command list and flag ownership: UNDECIDED
- Exit-code taxonomy: UNDECIDED
- Machine-readable output contract: UNDECIDED
- Config precedence and default behavior: UNDECIDED
- Runtime compatibility floor: UNDECIDED

## Implemented provisional cache inputs

`experimental check-workspace-session` accepts no config file or environment-variable defaults.
Its target, request ceiling, cache entry ceiling, retained-byte ceiling, artifact-byte ceiling, and
optional exact-lock mode are explicit command arguments. The process reads only the versioned
NDJSON request stream from stdin. These inputs are unstable and do not settle precedence or default
behavior for the future release-stable CLI.

`experimental check-workspace-cache` likewise accepts no config file or environment-variable
defaults. It requires a known target, a canonical relative directory below ignored `build/`, and
positive explicit entry, retained-byte, record-byte, artifact-byte, and abandoned-temp-age limits.
It derives only the exact compiler-build identity from the running executable bytes. This proof
surface does not select a future user cache location, shared-cache ownership model, or default quota.

## Review Blockers

- A command changes without updating help, examples, output, and exit-code expectations.
- JSON output exposes generated or existing file contents.
- Runtime compatibility changes without smoke validation.
