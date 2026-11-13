# ADR 0553: Embedded Win32 starter in the stable CLI

- Status: Accepted
- Date: 2026-11-13
- Extends: ADR 0204 (starter selection), ADR 0552 (native Windows build inputs)
- Command owner: `docs/cli/command-contract.md`

## Decision

Add `fairyc new <package-name> --uid <package-uid> --template win32`.
The existing invocation and explicit `--template console` select the unchanged
five-file console starter. UID/template option pairs may appear in either order;
unknown/repeated/incomplete options and unknown template names are rejected
before filesystem access. There is no arbitrary source path, network fetch,
prompt, overwrite/merge mode or implicit project creation in the current folder.

Win32 generation uses the same contained, exclusive direct-child creation,
sync, rollback, diagnostics, output and 0/2/4/7 exit codes as the default starter.
No lock is fabricated. The new app starts at the existing default version and
language. Package name is also its initial title/EXE/ZIP prefix; UID supplies its
assembly identity. The company field is blank. Win32 names are limited to 127
ASCII bytes, UIDs to 255, to fit the existing identity/resource constraints.
Edit `app.toml` and regenerate resources for custom title/company/icon/EXE names.

Bundle the hash-free source template, IME guard, default icon/license and shared
Python build tooling into the Go compiler. Generation works from the compiler
alone, with no repository checkout, Python, MSVC or SDK lookup. Generating a
Windows app is host-independent; its native build requires Windows x64,
MSVC/Windows SDK, and its packaging scripts require Python 3.11+. No native app,
GUI or package command runs as a side effect of `new`.

Canonical editable inputs stay in `templates/win32_desktop`, `tools/win32_app`
and the existing example assets. `tools/bundle_win32_starter.py` makes a sorted,
bounded, deterministic embedded ZIP and source checksum receipt. Go tests reject
stale inputs/defaults and a mismatched archive. Text hashes normalize CRLF to LF
so checkout line endings do not create cross-host drift. Generated apps own a
standalone snapshot; this is not a template upgrader or a public SDK interface.

The Python repository generator remains available for its existing configurable
title/company/icon workflow. Native GUI behavior and physical input validation
remain separate from CLI creation/build/package evidence.
