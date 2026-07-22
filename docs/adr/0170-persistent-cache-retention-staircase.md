# ADR 0170: Preserve the persistent-cache retention staircase as blocker evidence

- Status: Accepted measurement contract
- Date: 2026-07-22
- Implementation: `internal/buildmeasure/persistent_cache_growth.go`, `cmd/fairyc-windows-host-smoke/main.go`
- Storage prerequisite: `docs/adr/0163-bounded-immutable-disk-cache-store.md`
- Artifact prerequisite: `docs/adr/0162-stable-package-mir-artifact.md`

## Context

Fresh-cache cold/warm pairs prove cross-process reuse, but they say nothing about disk growth while a
developer edits one long-lived workspace. A two-package warm hit can coexist with unbounded old
generations, overbroad invalidation, or repeated conflict cleanup. Selecting default limits from the
fresh-pair observations would therefore be measurement theater.

The current stable package artifact still duplicates the complete workspace type context in every
package payload. Its cache key is package-oriented. A private root edit can leave the dependency key
unchanged while changing the dependency artifact bytes through that duplicated context. The
immutable store correctly retains both digest-named generations; the next read treats the two valid
generations as a producer conflict, deletes them, and falls back. This is safe but wastes writes,
retention, and a later lookup.

## Decision

Add `fairyc.persistent-cache-growth-measurement/v0` to the configured Windows host smoke. One
standalone compiler executes nine separate processes against one six-entry cache. Source mutation
happens before each process and elapsed time is deliberately absent. The sequence is:

1. baseline cold;
2. first private root edit;
3. second private root edit, which quarantines the dependency-key conflict;
4. private dependency edit;
5. public dependency edit, which reaches the entry ceiling;
6. public-state no-op verified hit;
7. private root edit while the public dependency remains;
8. no-op conflict recovery;
9. healed no-op verified hit.

The report preserves per-process package hits and misses, store outcomes, MIR function count,
retained entries and bytes, signed retained deltas, raw record observations, evictions, and verified
hits. It includes exact compiler and environment identity but no path, source, duration, speedup, or
default-policy claim. Retry policy is fixed to `none;preserve-first-observation`.

Validation requires the current entry staircase `2, 4, 4, 6, 6, 6, 6, 5, 5`. The second private
root edit and conflict-recovery no-op each quarantine two raw record generations. The public
dependency and later private root edit each evict two generations. The public no-op and final healed
no-op must be complete two-package verified hits with no lowering.

## Evidence

On the local Windows amd64 smoke with MSVC 14.51.36231, the accepted sequence reached 24,844 retained
bytes under the six-entry ceiling and finished with five entries and 20,158 bytes. Four generations
were evicted and four conflicting generations were quarantined. The final process returned two
verified hits, zero misses, zero lowering, zero rejection, and zero eviction. These exact byte values
are one local observation, not a regression budget or a reference-machine baseline.

Unit tests reject missing or reordered samples, altered entry or byte deltas, a larger entry policy,
missing eviction or conflict evidence, store failures, no-op lowering, retries, and any default or
speedup decision. The configured smoke executes the real standalone CLI, lock refresh, opened-root
disk store, separate processes, and bounded collection.

## Consequences and limits

- The cache proves bounded recovery under this control sequence; it does not justify an ordinary
  command default, 6-entry product limit, or representative repository-growth claim.
- Package-oriented keys and workspace-wide duplicated type contexts remain a measured efficiency
  blocker. The next artifact task is a package-scoped stable context or another exact key-to-bytes
  contract that preserves private-edit reuse without valid-generation conflicts.
- Replacing the immutable conflict rule with arbitrary last-writer-wins would hide the mismatch and
  make concurrent producer behavior nondeterministic. This ADR does not permit that shortcut.
- A broader workspace distribution, declared reference environment, stored baseline distribution,
  accepted noise rule, and policy decision remain required before enabling persistent caching for
  ordinary commands.
