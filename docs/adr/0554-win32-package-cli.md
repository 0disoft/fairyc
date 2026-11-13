# ADR 0554: Local Win32 packaging in the stable CLI

- Status: Accepted
- Date: 2026-11-13
- Extends: ADR 0552 and ADR 0553
- Command owners: `docs/cli/command-contract.md`, `stable-cli-contract-v0.md`

## Decision

Admit `fairyc package [--locked] [--json]` for Windows amd64 desktop projects.
The exact current root must declare `app.toml` and a valid `fairy.windows.json`
with Windows subsystem. Missing/nonregular/outside-root packaging helpers are
rejected. No parent search, network, installer, custom script/target/output path,
stdin, signing, publication, version bump or GUI automation is added.

Reuse the project's existing Python helpers. Desktop `fairyc build` first runs
its `prepare_resources.py`, so configured title/icon/version changes do not
silently retain stale RC data. Ordinary console and unconfigured Win32 builds
are unchanged. Python 3.11+ is found on PATH as python/python3/py -3; absence
returns an actionable error. MSVC/SDK discovery remains owned by native build.

Package performs a release build with ordinary/locked workspace semantics, then
runs `package_release.py --exe <release-image> --output <root>/dist`. This avoids
recursive compiler invocation. Helpers are project-owned executable build code;
trust the project as for native sources. Default supplied verifiers only run CLI
paths/resource inspection. Editing helpers can change that behavior.

Helper stdout/stderr is bounded to 64 KiB each and execution to three minutes.
The compiler validates the success receipt, rejects signed/published/GUI/input
claims, verifies an inside-dist archive SHA-256/size (up to 33 MiB), and emits
only its own relative-path report. Python is still a build-time dependency;
distributed app EXEs have no added runtime dependency.

Plain success is `packaged: dist/<archive>.zip (unsigned, release, local only)`.
`--json` selects `fairyc.package-report/v1` with schema, relative archive, sha256,
exeBytes, zipBytes, signed=false and published=false. No helper output or absolute
path is forwarded. Statuses reuse 0/2/3/4/6/7: success, invocation/unsupported,
source, internal/output, tooling/artifact and workspace/manifest/lock failure.
Missing package helper is 7; invalid resource settings/tooling/receipt is 6.

Preflight/build failures retain the prior EXE/ZIP where replacement has not
started. A successful build followed by package failure can retain the new EXE
with the old ZIP. ZIP/sidecar replacement is not a two-file crash transaction.
Output transport failure after publication returns 4 and does not undo outputs.
This Go Stage 0 extension does not claim private Fairy candidate CLI parity.
