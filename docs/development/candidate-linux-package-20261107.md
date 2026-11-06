# Linux package retained-file generation guard, 2026-11-07

This bounded slice follows the already implemented Linux retained-directory,
archive-cache, HTTPS and online/offline CLI owners. The first 65 queue lines and
the existing package/fetch receipts were inspected; no queue or central index is
updated here. Actual Linux execution remains in the maintainer's combined batch.
No stage, commit, push or hosted dispatch is performed by this worker.

## Implementation

Existing-tree reuse previously checked a file's bytes and its held metadata
before/after reading, but did not bind that held file to the earlier directory
scan fact. A different file generation with identical bytes could therefore be
accepted after the detached scan. This is a source finding, not a reproduced
Linux filesystem race.

`linux_remote_fs::read_observed` now admits a single leaf and a regular, nonzero
inode fact with valid nanoseconds. It checks the existing byte bound, opens the
file through the retained directory, requires a single link, and compares device,
inode, size, modification seconds/nanoseconds and complete mode before allocating
or reading the body. The existing complete-read/extra-byte probe, after-read
metadata check and consuming close remain in use. Ordinary `read` retains its
previous behavior. Existing-tree verification in `linux_remote_offline` now passes
the detached scan fact to this guard; online publication uses that same owner.

Mismatch uses the existing private conflict class 4; malformed expected metadata
and paths fail closed. Bounds retain class 3, missing leaves class 5 and I/O class
1. No new syscall, filesystem mutation, retry, transport selection, dependency,
SDK/native ABI or budget increase is added. No ADR is needed or allocated.
This comparison adds no native observation beyond the existing read path.

The check is an observation, not transactional namespace isolation. Concurrent
hostile edits, complete directory-snapshot confirmation, physical-media durability
and Linux target promotion are not established by this slice.

## Verification

The isolated production-module fixture executes authored Fairy assertions over
the existing independent filesystem model. It checks every metadata field,
including same-byte/different-inode refusal with zero body reads, exact binary
success, invalid facts/leaf paths, absence, case alias, byte bounds, short/growing
reads, hard links, after-read metadata change, close failure and resource cleanup.
Both target C-emission profiles and Windows MSVC low/optimized execution run with
empty PATH. No cached full CLI image was rebuilt; existing cached fetch receipts
and prepared inputs were inspected first.

The first focused test invocation failed before execution because the parallel
Atlas edit temporarily referenced undefined
`verifyCandidateMCPSDKAggregateCleanupCommand`. No Atlas file was changed here.
The subsequent combined command uses the now complete shared test package.

```powershell
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestLinuxRemote(ObservedReadPolicy|ObservedReadNativeInputs|ObservedReadNative|FileSystemPolicy|OfflinePolicy|OnlinePolicy)$'
go vet -mod=readonly ./internal/cli
```

The combined command exits zero in 143.943s: five roots pass, including four
policy fixtures / eight Windows low/optimized generated-program executions;
`TestLinuxRemoteObservedReadNativeInputs` passes, and the actual Linux native
root explicitly skips. The filesystem fixture takes 13.99s, the new observed-read
fixture 8.09s, the native-input guard 1.26s and offline policy 62.15s. Online
policy and all its cached/download/fault cases pass in the same command.
Scoped vet exits zero. `gofmt -l` reports no paths, scoped `git diff --check`
passes, and all six authored files are nonempty UTF-8 without BOM, LF only and
final-newline terminated. Compiler-wide tests, full CLI rebuilds, whole-source
checks/graph-lock refresh and hosted CI are deliberately not run in this slice.

## Deferred native replay

`TestLinuxRemoteObservedReadNative` is authored for real Linux amd64 and explicitly
skips on Windows. Its Windows input guard lowers the actual Fairy entry for both
targets, selects the actual rooted/package-file adapters, and rejects filesystem
model content in the native replay inputs. The native entry observes and reads a
real file, then a test-only POSIX hook creates another inode with identical binary
bytes and renames it over the original. The guarded read must reject; a fresh
ordinary read must still return those unchanged bytes. The hook does not supply
file reads or metadata and is not a product filesystem implementation.

Include this root in the next authorized Linux batch; it reuses one small emitted
program across GCC/Clang low/optimized executions, with empty PATH:

```sh
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestLinuxRemoteObservedReadNative$'
```

Actual Linux retained-directory/cache publication, trusted HTTPS, the existing
full fetch CLI/native online/offline replays and consecutive self-build acceptance
remain required. Windows substitute execution and native input generation do not
prove any of those results. No provider choice or policy relaxation is made.

## Integration scope

- `examples/lexer_candidate/src/linux_remote_fs.fy`
- `examples/lexer_candidate/src/linux_remote_offline.fy`
- `examples/lexer_candidate/tests/linux_remote_observed_read.fy`
- `internal/cli/linux_remote_observed_read_test.go`
- `internal/cli/linux_remote_observed_read_native_test.go`
- This receipt.

Source-of-truth/queue docs, the example graph lock, state/inline/Atlas modules,
public CLI behavior, DB, CI/runners and repository hygiene files are untouched.
The coordinator owns any final whole-source/lock refresh and exact-scope commit.
