# ADR 0169: Bind persistent cache I/O to an opened workspace root

- Status: Accepted storage hardening
- Date: 2026-07-22
- Implementation: `internal/cli/workspace_cache.go`, `internal/packagecache/disk.go`, `internal/durablefile/`
- Supersedes: Windows publication primitive in `docs/adr/0168-platform-durable-cache-publication.md`

## Context

The experimental command accepted only canonical `build/...` paths, and the disk store rejected a
non-directory key leaf. Those checks did not make the later absolute pathname operations safe. A
workspace could make `build` a symlink or Windows junction to another directory, or another process
could replace a checked ancestor before a later create, read, delete, scan, or rename. A detached
`EvalSymlinks` plus prefix check would have the same time-of-check/time-of-use gap.

ADR 0168 used absolute verbatim paths with `MoveFileExW` to request Windows write-through rename.
That API did not bind lookup to the workspace directory handle, so retaining it would preserve a
stronger durability request at the cost of an unclosed path-escape boundary. Cache bytes are
disposable; writes outside the selected workspace are not an acceptable cache failure mode.

## Decision

Open the workspace directory once with Go 1.26 `os.OpenRoot`. The CLI passes that boundary plus the
validated relative cache path through `workspacecompiler` and `packageartifact`. `DiskStore` creates
and opens its version root under that boundary, retains the resulting `os.Root`, and performs every
create, open, read, remove, scan, and rename with root-relative operations.

The direct `NewDiskStore(path, limits)` API still treats its explicit path as the caller-selected
boundary, then opens the version root before use. The new `NewDiskStoreInRoot` API is the required
entry point when containment within a larger owner such as a workspace matters. Stores and
persistent Sessions expose `Close` so the directory handle has an explicit lifetime.

Publication remains a same-directory rename of a fully written, file-synced, and closed unique
temporary. On Unix-like hosts, the opened containing directory is synced after rename. Windows uses
the handle-relative `os.Root.Rename`, reopens the final file with write access, and flushes it.
Windows has no documented portable directory-handle flush contract here, so the guarantee remains
a verified hit or safe miss after interruption rather than retention after every physical power
loss. This supersedes the absolute-path `MoveFileExW` implementation and its stronger write-through
request.

The opened-root temporary creation, immutable convergence/repair, same-directory publication, and
platform sync mechanics now live in `internal/durablefile`. The package cache remains responsible
for cache keys, generations, bounds, collection, and verification. This extraction changes no cache
layout or failure policy and lets other derived artifacts reuse the same containment primitive.

## Evidence

- Disk-store tests reject parent traversal under an opened root.
- A directory symlink to an external temporary directory is rejected where the host permits the
  test to create one, and the external directory remains untouched.
- All ordinary, concurrent-process, corruption, eviction, and deterministic publication-crash tests
  run through root-relative operations.
- CLI cold/warm and corruption-recovery tests run with the opened workspace boundary.
- The Windows host smoke builds a standalone compiler and validates repeated persistent-cache
  invocations without WSL.

## Consequences and limits

- Workspace-local cache I/O is no longer authorized by a lexical string check. The opened workspace
  handle is the authority and remains stable if the directory is renamed.
- Symlinks that resolve outside the root fail. Internal aliases cannot produce executable cache
  truth because generation bytes, digest, embedded key, interface, artifact, and MIR are still
  independently verified.
- A caller that uses the direct path constructor explicitly selects that path as its root; it does
  not gain containment within an unpassed parent directory.
- Windows post-rename file flush is weaker than a universal directory-metadata durability guarantee.
  The cache is derived state, so loss is a miss and never a compilation-authority failure.
- Linux bind mounts and special filesystem semantics retain the documented `os.Root` limitations.
  No claim is made beyond the Go 1.26 opened-root contract and the configured local tests.

## References

- Go `os.Root`: <https://pkg.go.dev/os#Root>
