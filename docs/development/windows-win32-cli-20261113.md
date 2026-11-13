# Embedded Win32 starter CLI — 2026-11-13

## Result

The Go Stage 0 `cmd/fairyc` now supports:

```powershell
fairyc new my-tool --uid example.my-tool --template win32
cd my-tool
fairyc check
fairyc test --locked --file tests/domain.fy
python -B package_release.py --compiler C:\tools\fairyc.exe
```

Creation uses only the compiler and writes a new direct child; it does not
invoke Python, inspect MSVC/Windows SDK, download files or launch a GUI. Default
creation and explicit `--template console` retain identical five-file console
contents. UID/template pairs work in either order. Success/output and 0/2/4/7
exit codes, no-overwrite, opened-root writes, sync and ordinary-error rollback
are retained. No lock is generated. Win32 identity limits are 127 ASCII name
bytes / 255 UID bytes to fit existing app resources. Company starts blank;
branding is edited in the generated `app.toml`.

The native app still needs Windows x64/MSVC/Windows SDK to build, and Python
3.11+ for supplied resource/packaging helpers. This extension applies to the Go
product CLI, not the private Fairy-authored candidate CLI. No existing system
installation or PATH entry was replaced in this work.

## Implementation and commits

- `729b29dd`: embedded starter asset and internal scaffold selection, validation
  and source/archive drift tests; ADR 0553.
- `d0c62ce8`: stable CLI option parsing/help, console compatibility and Win32
  frontend tests; command contract v5, product scope and source-owner index.
- Final documentation/verification slice: standalone compiler CLI probe,
  creation/build/package evidence, updated user guidance and refreshed bundle.

`tools/bundle_win32_starter.py` snapshots the existing Python generator's
canonical template/build tooling/IME/icon/license inputs into
`internal/projectscaffold/assets/win32.zip`. It writes sorted entries with fixed
ZIP metadata and normalized LF text, plus hashes of all 26 source inputs and
the archive. Go embeds the ZIP directly; creation never executes the Python
generator. The decoder checks paths, duplicate/nonregular entries, required
files and bounds (128 entries, 1 MiB/file, 4 MiB total), then renders only the
validated identity fields. Original manifest generation owns version/language.

Tests detect changed/missing/new canonical source inputs, changed defaults and
an archive/receipt mismatch. Refresh with:

```powershell
go generate ./internal/projectscaffold
# Equivalent:
python -B tools/bundle_win32_starter.py
```

Generated projects contain independent snapshots. No shared installation,
external template path, automatic updater or public GUI SDK was introduced.
The optional repository Python generator retains its title/company/icon options.

## Executed checks

```powershell
go test ./internal/projectscaffold
go test ./internal/cli -run 'TestStable(New|CommandInventory|ExitCode|ConfigDiscovery)' -count=1
go test ./internal/projectscaffold ./internal/cli -run 'Test(Create|Win32|StableNew|StableHelpInventory|StableExitCode|StableConfigDiscovery)' -count=1
go vet ./internal/projectscaffold ./internal/cli
go build -o .cache/windows-win32-cli-20261113/fairyc.exe ./cmd/fairyc
python -B tools/verify_win32_cli.py --compiler .cache/windows-win32-cli-20261113/fairyc.exe
```

All passed, together with Python AST parsing and scoped `git diff --check`.
No desktop-test function was selected by the Go test filters.

`verify_win32_cli.py` performed the actual process-level checks in an external
Korean/space-containing temporary directory:

1. Copied only `fairyc.exe`, then created the Win32 app with empty PATH. The
   compiler directory still contained only that EXE. No repository lookup,
   generator process, prebuilt lock or hash-product sources were needed.
2. Verified expected creation output, help, contained output, existing-project
   preservation and ten rejected invocations: existing target, unknown template,
   missing UID/value, duplicate option, empty template, traversal name, reserved
   name, invalid UID and excessive name. Invalid cases left no project directory.
3. Edited the generated app identity to `CLI 검증 도구 Ω` and `CliTool.exe` with
   blank company, then ran check, locked check, the pure navigation policy test
   and locked release build through the copied compiler.
4. Ran the generated packaging script, which prepared resources and validated
   the extracted app's **five real CLI cases**, resource-only PE mapping,
   Unicode title, version, blank company, nine icon sizes, ZIP entries, guide,
   license and checksums. The app was never invoked without CLI arguments.
5. Created console apps through both default and explicit console forms,
   compared every generated file byte-for-byte, and checked/tested the default
   console app. Temporary projects/binaries/packages were removed.

The final probe was rerun after updating the bundled documentation. These are
native CLI/resource checks, not evidence of visible layout, IME, Narrator,
keyboard navigation, physical DPI changes or real desktop input. Existing GUI,
clipboard, mouse/keyboard, desktop integration, broad bootstrap/candidate parity
and full release suites were intentionally not run. No hosted CI was invoked.

## Local outputs

- Compiler: `.cache/windows-win32-cli-20261113/fairyc.exe`, **25,196,032 bytes**.
- Compiler SHA-256:
  `8d49705446a2662da1c28002908e4edcfcd71c4b4136eff0201c0783cd988ce1`.
- Embedded compressed template: **33,714 bytes**, including sources/build tools.
- Probe app EXE: **137,728 bytes (134.5 KiB)**.
- Probe app ZIP: **76,597 bytes**.
- Probe ZIP SHA-256:
  `a024a15cc8bfaae166cb9729ac0cec7a6924e93a5c94738f3e83b8c48abbbb19`.
- App EXE SHA-256:
  `cd6ed73520c50eea4494c8bae3a37c86bc130d3ad8405ec663e98e84e8e6d201`.
- Final receipt: `.cache/windows-win32-cli-20261113/receipt.json`.

The test app's ZIP is unsigned. Only the compiler and receipt are retained from
this probe; generating another app is reproducible through the command above.
The compiler size is a current measurement, not a same-revision before/after
size comparison. Source/asset data is committed; local compiler/evidence stays
ignored.

Source-of-truth updates: ADR 0553, `docs/cli/command-contract.md`,
`docs/product/02-spec.md`, and the package/CLI source-owner index. Stable CLI
selection is extended; language/compiler backend/SDK APIs, DB, dependencies,
CI workflows and runners are unchanged. No automatic version bump, signing,
push, release or publication. Concurrent `docs-site/` work was preserved.
