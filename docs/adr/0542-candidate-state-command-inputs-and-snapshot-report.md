# ADR 0542: Authored state-command inputs and snapshot reporting

- Status: Accepted for the explicit compiler-private developer command
- Date: 2026-11-07

The existing Go developer command in `tools/dev/fairyc-state` owns the explicit
seven-argument native-graph syntax, fresh single-source identity and v2 report.
Three independent authored helpers prepare its real entry without changing the
main CLI, choosing a provider/chain or admitting the default Go-MIR invocation.

`module_state_command_arguments` reads actual process argv, excluding the image,
and delegates syntax to the existing `module_state_command_options` parser.
It checks arity before allocating/traversing arguments, bounds aggregate bytes
to the parser's 1MiB policy and retries only a too-small argument buffer with
the exact reported size. It owns the exact copied argv together with the parsed
scalar selection, keeping source/store indexes bound to their backing owner.
Host status/index/needed and allocation failures remain distinguishable from
invocation failures; exit codes are respectively 4 and 2. It performs no source,
asset or state-file I/O and does not dispatch a command.

`module_state_command_source::prepare` accepts only actual source bytes. The
existing bounded source producer admits its layout and contract UID. The UID,
version `experimental-state-graph-source/1` and empty features derive precisely
the existing Go developer command coordinate. A private fixed owner 0 binds that
coordinate; no caller coordinate, serialized body, cache or arbitrary graph can
replace fresh source admission. Building the sealed Complete reuses the same
cumulative work ledger. The bounded source is parsed twice rather than adding a
new constructor that weakens Complete ownership. The existing 64KiB authored
source ceiling and admitted language subset remain explicit.

`module_state_command_snapshot_report::snapshot` accepts the copied command
options, sealed Prepared and actual admitted local State. Only init/snapshot
are admitted, and expected layout identity must agree. Output follows the
existing Go `fairyc.local-state-run/v2` JSON, including profile, layout/body/
transport program identities, full-width versions, binary keys and base64/null
values. It returns one complete owned newline-terminated buffer. Output has a
32MiB ceiling with a cumulative 1GiB work ledger; errors expose no partial JSON.
There is no console write or storage publication in the formatter.

These helpers do not yet load a source/asset path, compose the real command
entry, select a console cancellation handler or serialize executed outcomes.
Their acceptance does not complete the native-state implementation/final queue.
Linux execution remains batched, and no public CLI/SDK/native ABI, database,
dependency, CI/runner or repository policy is changed.

Evidence: `docs/development/candidate-state-command-inputs-20261107.md`.
