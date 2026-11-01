# Windows pinned remote package preparation receipt, 2026-11-01

- Plan/identity slice: `d3cf9eac`; bounded HTTPS/archive/cache slice: `ae24b441`.
- Publication, remaining operation budgets, CLI and final acceptance: containing commit.
- Owners: [ADR 0498](../adr/0498-explicit-pinned-remote-package-sources.md),
  [remote source v1](../package/remote-sources-v1.md), and
  [experimental CLI admission](../cli/command-contract.md#experimental-explicit-remote-package-preparation).
- Host: Windows amd64; Go 1.27.1; native MSVC 14.51.36231, C11 low/optimized.

## Executed and passed

| Slice | Observed Windows result |
| --- | --- |
| Strict input and version selection | Exact pin parsing/binding, duplicate/unknown/noncanonical fields and trailing JSON rejection, invalid UID/version/path/URL/hash and inconsistent root dependency versions reject. No range or registry resolution is claimed. |
| Actual HTTPS transfer | A local TLS server and caller-owned test trust store deliver a real ZIP through the HTTP client. Hash and archive manifest identity verify before immutable cache publication. The cold install requests once; warm reuse and offline restore do not request again. The default client's trust validation rejects the self-signed server. No public package host or system trust store was changed. |
| Failure boundaries | Redirects, non-200 status, oversized response declarations, bad SHA-256, language/UID/version mismatches, unsafe archive paths, traversal/drive/reserved names, case collisions, duplicates, file/directory conflicts, links, nested VCS metadata and oversized entries reject. Remaining compressed/expanded/file budgets are passed before the next read/unpack and reject when insufficient. |
| Verified cache | Cold store uses complete synced temporary bytes and exclusive hard-link publication. Warm and offline reads rehash the archive. Online/offline corrupt-cache and offline-miss cases fail without another HTTP request. Filesystem hard-link support is required. |
| New member publication | The full verified bundle is written under an opened root, then renamed into the new explicit member under the exclusive fetch lock. Identical existing trees reuse; edited, extra or missing files reject without merge or replacement. A cancelled stage cleans up; a preexisting lock fails and remains owned by its original creator. Root/cache symlink rejection passes locally without a skip. |
| Inputs and lock | Source plan, root manifest and preexisting graph lock bytes remain unchanged across fetch. The CLI fetch followed by ordinary check creates the existing graph lock; locked check succeeds. A changed remote dependency is rejected by both explicit snapshot verification and ordinary locked compilation. |
| CLI and native behavior | `experimental fetch-packages --offline` installs and then reuses the cached package. Unknown/duplicate flags and missing plans return the expected invocation/package failures. Output failure returns the internal/output code. The fetched dependency supplies return 42 to actual MSVC low and optimized native workspace runs with empty streams. |
| Related regressions | Existing projectmanifest, lockfile and workspaceloader package tests pass. Focused stable help/invocation/workspace/SDK-lock tests pass. Scoped Go vet, gofmt and Git diff checks pass. A standalone Go-built compiler's help includes the new experimental command. |

The native receipt counts two distinct executions, one per profile. Development reruns do not add
coverage. TLS download/cache/publication integration is tested in the library with an injected
local CA; the CLI fixture exercises the identical prepared cache through the default offline path.
No externally hosted package fetch, release server, login or trust installation was performed.
The standalone binary smoke built and checked help; the native dependency runs use the CLI Run
boundary in the Go tests, not an extra claimed standalone-process matrix.

## Replay

From the repository root:

```powershell
go test -mod=readonly ./internal/packagesource ./internal/projectmanifest ./internal/lockfile ./internal/workspaceloader -count=1
go test -mod=readonly ./internal/cli -run '^(TestRemotePackageCLI|TestStableHelpInventoryOwnsTopLevelCommands|TestHelpAndInvocationErrorsAreSeparated|TestExperimentalWorkspaceCheck|TestStableCheck)' -count=1 -v
go vet -mod=readonly ./internal/packagesource ./internal/projectmanifest ./internal/cli
go build -mod=readonly -o .cache/fairyc-remote-packages.exe ./cmd/fairyc
./.cache/fairyc-remote-packages.exe --help
gofmt -l internal/packagesource internal/projectmanifest/manifest.go internal/cli/package_fetch.go internal/cli/package_fetch_test.go internal/cli/run.go
git diff --check
```

All completed checks above returned 0, with no focused test skipped. Package-source tests were
rerun after the final portable-entry/VCS-metadata rejection change; the accepted native fixture
inputs and the compiler/CLI lowering were unchanged. A standalone-help assertion was corrected
from a PowerShell array comparison to a joined-string comparison; the executable help includes
`experimental fetch-packages [--offline]`.

## Deferred evidence and remaining boundaries

Native Linux remains unexecuted at maintainer request. The ordered queue retains the package
source suite and focused CLI command for later replay. The authored native CLI test selects
GCC/Clang x low/optimized on Linux; this Windows run does not prove those executions or Linux
filesystem/cache publication behavior. No WSL provisioning or hosted runner was started.

Full repository tests, public HTTPS-host validation, fresh-machine transfer, unsupported-filesystem
behavior, deliberate crash/power-loss testing, signatures and publisher identity were not tested.
Exact archive SHA-256 is a reviewed input, not proof that downloaded source is trustworthy. A
hostile process mutating the workspace concurrently is unsupported. A crash can leave a stage or
fetch lock; a later package failure can leave prior completed packages/cache blobs. Cache GC is
manual. Archive filenames containing platform-incompatible spellings reject on every host.

The stable compiler still sees local members and its unchanged graph lock; it does not fetch or
reauthenticate remote origin automatically. Registry selection, range solving/major-compatible
upgrade policy, Git/submodules/LFS, signatures, package publication and Fairy-authored candidate
parity are separate work. No supported registry ecosystem or stable remote CLI is claimed.

This slice adds an experimental Go CLI command, internal source APIs, the pin format and its
owning ADR/package/CLI documentation. Public SDK/runtime/language, manifest/graph-lock schemas,
DB, dependency versions, repository hygiene and CI/runner configuration are unchanged. No push,
deployment, publication, workflow dispatch or version bump was performed.
