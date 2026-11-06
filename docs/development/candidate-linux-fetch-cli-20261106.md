# Linux candidate package-fetch CLI connection, 2026-11-06

This continues the private Linux implementation in
[the filesystem/HTTPS/online receipt](candidate-linux-package-fetch-20261106.md).
Linux execution remains batched by user request. Windows model observations do
not certify Linux syscalls, TLS trust or deployment prerequisites. No push,
hosted CI dispatch, release or deployment is authorized by this receipt.

## Shared output and Linux command leaf

`project_fetch_output.fy` owns the unchanged success line, bounded error reasons,
cleanup suffix, help/invocation output and exit codes: success 0, invocation 2,
output failure 4 and fetch failure 7. It takes scalar counts/stages rather than
depending on either platform's filesystem/HTTPS result type. `project_fetch_cli`
retains its entry signatures and Windows owners. Linux `linux_fetch_cli` uses
the Linux offline/online owners and `linux_cli_data`'s Linux-specific help.
Numeric native causes remain in the private online API; CLI presentation keeps
the existing bounded reason policy rather than printing OS/library error text.

Syntax/help resolves before any filesystem, monotonic-clock or request access.
Only explicit `experimental fetch-packages` calls the online owner; `--offline`
calls only the offline owner. Cache corruption never falls back to transport.
The command's existing help/unknown-command precedence and optional-flag rules
are unchanged. No stable compiler operation gains networking.

Windows command-leaf validation:

```powershell
go test -mod=readonly -count=1 -timeout 8m ./internal/cli -run '^TestLinuxFetchCommand$'
go test -mod=readonly -count=1 -timeout 5m ./internal/cli -run '^TestCandidateFetchCommand$'
go vet -mod=readonly ./internal/cli
# From examples/lexer_candidate:
go run -mod=readonly ../../cmd/fairyc check
go run -mod=readonly ../../cmd/fairyc check --locked
```

The final Linux command-leaf command passes in 24.831s and the Windows command
regression in 20.676s, both low/optimized without selected skips. Scoped vet,
Go formatting and diff checks pass. Reference full-source check and locked check
pass 697 files / 8,635 MIR functions, with the example graph lock refreshed.

The leaf runs actual Fairy command parsing, output, offline/online policy,
hashing/archive/cache/publication owners and native Windows stdout/stderr with
independent Linux filesystem/HTTPS ABI substitutes. Both target C emission
profiles are checked; Windows low/optimized images execute with empty PATH.
Expected final trees and request counts include independent Go reference results.
Syntax cases compare Go response ordering with an explicit Linux-help substitution.
The model also requires zero handles/scans/clock observations/requests for syntax
and help, rather than merely checking an unchanged final tree.

Cases cover exact 4,095/4,096-byte argument capacity and 4,097-byte rejection,
unknown/newline commands, help precedence, duplicate/unknown/trailing options,
missing plan, offline miss without network, fresh offline install, warm reuse,
online cold install, invalid plan/ZIP before publication, corrupt cache without
network, retained verified cache on a package conflict, native/cleanup/deadline/
cache collision/lock controls, and broken stdout/stderr. Completed publication
survives a final stdout failure. The shared fetch-process helper now explicitly
sets empty PATH, including the Windows command regression.

Initial Go compilation caught an incorrect fixture-path helper name. The next
run exposed the help-reader's trailing-newline parsing, fixed with checked source
boundaries. A formatting command was initially run from the example directory;
the file is formatted from the repository before final validation. These harness
corrections do not relax product limits or warning flags.
An undersized non-ZIP fixture correctly reached the existing minimum-size bound;
the final command oracle keeps separate 21/22/23-byte cases for bound versus
malformed ZIP classification, all before cache publication.

## Full Linux entry and authored native command replay

`linux_file_cli` now selects experimental mode 11 and calls `linux_fetch_cli`.
Its help delegates to the same Linux-only data as the command leaf. Only the
outer dispatcher gains the network effect needed for this explicit command;
stable check/build/run/test paths keep their existing owners. The existing
Python preparation tool selects this actual authored entry without a new source
rewrite or an online/offline implementation substitution.

```powershell
go test -mod=readonly -count=1 -timeout 12m -v ./internal/cli -run '^TestLinuxFileCLI$'
python -B tools/prepare-candidate-linux-cli.py --output .cache/linux-fetch-cli-20261106-dispatch
# From examples/lexer_candidate and the prepared directory, separately:
go run -mod=readonly ../../cmd/fairyc check
go run -mod=readonly ../../cmd/fairyc check --locked
go vet -mod=readonly ./internal/cli
```

The full-entry command exits zero in 30.868s. Both target profiles emit the full
Linux entry without Windows host symbols and select closed Linux runtime support;
the Linux result has 23 C partitions and a 1,817,749-byte header. Portable dispatch
executes low/optimized on Windows (5.35s combined), including experimental mode 11.
The outer test then explicitly skips actual Linux compilation/execution on this
Windows machine. Report this as passed emission/selection/portable observations
and a native skip, not as a complete native CLI pass.

The prepared-source command succeeds. Original reference check and locked check
pass 697 source files / 8,635 MIR functions; the prepared entry adds one Linux
directory-I/O helper and passes 697 / 8,636. Scoped vet, formatting and diff checks
pass. These checks do not execute libcurl, child processes or TLS on Linux.

The existing actual Linux full-CLI test now reuses each built profile for the new
fetch command replay, with no extra full compiler/companion build. It checks exact
Linux help/invocation output, offline fresh install and reuse, online warm cache,
Go-reference complete final namespace bytes, restored-workspace check/locked,
and one absent-cache request rejecting an untrusted loopback TLS certificate
without reaching HTTP, retrying or changing the namespace. Programs run with
empty PATH. No system CA installation is used. This code is authored, not run.
The native source requires the optional libcurl development headers plus the
system runtime/CA prerequisites recorded in ADR 0501; baseline provisioning has
not been changed or certified here.

The GNU companion's authored runtime selector/asset pack now includes the
package-directory/file/clock/HTTPS assets. The independent Windows low/optimized
oracle matches 118 plans and 76 complete runtime C outputs against live Go
selection; see `candidate-linux-runtime-sync-20261106.md`. Actual Linux
compilation, trusted TLS and consecutive full Linux self-build remain unverified.

Public CLI spellings/output/exit policies, SDK ABI, dependencies, DB, CI/runners
and repository hygiene are unchanged. Internal implementation routing and queue
documents record this progress. Actual trusted Linux download, toolchain/runtime
provisioning, broad Go/Fairy comparisons, self-build and distribution remain open.
