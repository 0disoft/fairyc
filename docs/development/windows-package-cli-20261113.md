# Win32 build/package CLI — 2026-11-13

ADR 0554 adds `fairyc package [--locked] [--json]` to the Go Stage 0 stable CLI.
Configured Windows desktop `build` prepares resource files from app.toml first;
package always builds release, runs the existing project-owned Python packager
and independently checks the retained bounded inside-dist archive hash/size.
The compiler emits only its relative-path plain report or fairyc.package-report/v1.
Python 3.11+/MSVC/SDK remain explicit build dependencies, with no automatic install.
No app runtime framework, signing, upload, release or version bump was added.

The headless `tools/verify_win32_package_cli.py` passed outside the repository
using a copied compiler and empty PATH for creation. It checks actual changed
Unicode title resources after build without a manual prepare step; JSON/plain
package reports, SHA-256/sidecar/ZIP, missing Python and invalid app settings
preserving previous output, stale locked package preserving lock/ZIP, ordinary
lock refresh and missing helper rejection. Each package runs the supplied 26 CLI
cases (including nine reads and twelve saves) on the extracted app. No window or
physical input is used. Probe EXE: 171,520 bytes; probe ZIP: 93,423 bytes.
Local receipt: `.cache/windows-jobs-20261113/package-cli.json`.

Scoped scaffold, starter, stable build/package/help/exit/config and receipt tests
passed; Go vet passed for internal/cli and internal/projectscaffold. A missing
Python diagnostic identifies Python 3.11+ and PATH. Reports reject outside-dist,
wrong hash/size and signed/published/GUI/input claims. Helper streams are bounded
to 64 KiB and time to three minutes. Project helpers are trusted executable build
code, not a sandbox. A build can succeed before packaging fails; ZIP/sidecar are
not a two-file crash transaction. Output failure cannot roll back local artifacts.

Contract owners changed: ADR 0554, CLI command/stability/output contracts, product
scope and source-owner index. Root/template/tool guides and embedded source ZIP
were refreshed. Public CLI changed; language/public SDK, DB, dependency, CI/runner
and versions did not. Broad bootstrap, GUI/IME/Narrator/DPI tests and hosted CI
were skipped as outside this headless task. Local commit only; no push/publication.
