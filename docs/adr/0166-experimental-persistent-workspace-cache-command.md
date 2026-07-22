# ADR 0166: Expose persistent MIR reuse only through an explicit experimental check command

- Status: Accepted experimental surface
- Date: 2026-07-22
- Implementation: `internal/cli/workspace_cache.go`
- Output schema: `spec/cli/experimental-workspace-cache-check-v0.schema.json`
- Build identity prerequisite: `docs/adr/0165-executable-digest-compiler-build-identity.md`

## Context

Stable package MIR, bounded immutable disk storage, verified Session integration, process tests, and
an exact executable identity existed, but no real `fairyc` command could exercise them. Enabling a
hidden default cache would be premature: no reference workload has selected a location, byte budget,
retention age, or performance threshold, and a cache directory is a new write surface.

## Decision

Add the unstable one-shot command:

```text
fairyc experimental check-workspace-cache
  --target <target-id>
  --cache-dir <build/path>
  --cache-max-entries <count>
  --cache-max-bytes <bytes>
  --cache-max-record-bytes <bytes>
  --cache-max-artifact-bytes <bytes>
  --cache-temp-max-age-seconds <seconds>
  [--locked]
```

Every value is mandatory except `--locked`; there are no config, environment, or compiled defaults.
The directory is canonical, forward-slash relative, and below the workspace's ignored `build/`
directory. Entry and byte ceilings reuse the existing 65,536-entry and 1 GiB safety ceilings;
temporary age is a canonical positive integer no greater than one year. Artifact bytes may not
exceed record bytes, and record bytes may not exceed retained bytes.

The command opens the workspace directory once and creates and opens the cache through that
filesystem boundary. Every cache operation remains relative to the resulting opened root. A
symlink, Windows junction, parent traversal, or concurrent pathname replacement cannot redirect
cache I/O outside the opened workspace tree. The displayed `cacheDirectory` remains the validated
relative spelling and never exposes the host path.

Each invocation hashes its own executable bytes for the compiler-build cache-key member, performs
ordinary workspace loading, lock refresh or exact locked verification, and full semantic analysis.
Only complete verified stable-package hits skip MIR lowering. Corrupt, incompatible, partial, or
mixed-context state falls back to lowering. Oversized writes are `storeSkipped`; disk write failures
after successful compilation are `storeFailed` and do not make derived cache state authoritative.

Success writes one exact `fairyc.experimental.workspace-cache-check/v0` JSON object with target,
executable identity, relative cache directory, workspace/MIR counts, current cache outcome, retained
entry/byte counts, raw record counters, and verified-hit count. It emits no source contents, absolute
paths, artifacts, or toolchain paths. Diagnostics remain on stderr with existing exit codes.

## Evidence

CLI tests require cold two-package fallback and a warm second invocation with two hits and no
lowering. They corrupt one immutable generation and require one surviving hit, one miss, ordinary
lowering, rejection accounting, and refreshed artifacts. Parser tests cover missing, duplicate,
unknown, noncanonical, overflowing, relationally inconsistent, escaping, and Windows-ambiguous
inputs. Schema tests require the Go report fields and JSON Schema fields to match exactly.

The configured Windows host smoke separately builds `fairyc.exe`, invokes the new command twice
against the checked workspace and one cache directory, and requires the same executable identity,
cold fallback, warm all-hit, zero warm lowering, two retained records, and zero store failures.

## Consequences and limits

- Users and CI can now test real cross-process persistent reuse without WSL or a daemon.
- Ordinary `check`, build, run, and session commands remain unchanged and cache-free or process-local.
- The command is a proof surface, not the final cache UX. No default directory, budget, retention,
  cleanup command, L1 layer, speedup claim, or stable command spelling is selected.
- The cache path is workspace-local disposable state. A future user-level shared cache needs a
  separate ownership, permission, multi-repository quota, inspection, and cleanup decision.
