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

At this slice, command-leaf behavior is implemented; full Linux entry dispatch,
prepared-source/full-entry checks and native command replay remain the next step.
The GNU companion's authored runtime selector/asset pack also needs the recently
added package-directory/file/clock/HTTPS assets before consecutive full Linux
self-builds can be certified. Go native support already selects those assets;
the separate authored selector must not be assumed equivalent without its oracle.

Public CLI spellings/output/exit policies, SDK ABI, dependencies, DB, CI/runners
and repository hygiene are unchanged. Internal implementation routing and queue
documents record this progress. Actual trusted Linux download, toolchain/runtime
provisioning, broad Go/Fairy comparisons, self-build and distribution remain open.
