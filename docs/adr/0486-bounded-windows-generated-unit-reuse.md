# ADR 0486: Bounded Windows generated-unit reuse

- Status: Accepted for prototype
- Date: 2026-10-23
- Extends: [ADR 0428](0428-native-runtime-object-reuse.md)

## Decision

Reuse the generated C translation unit's object on the Windows workspace native path. Source,
lock, semantic and MIR checks, C emission and header preprocessing remain fresh. Link the current
unit/runtime objects against the current host libraries and inspect the resulting PE/unwind on
every build. The cache stores no executable, source tree, compiler process or background service.
The raw driver probes and Linux path retain their existing behavior.

The [current cost observations](../benchmarks/completion-followup-20261023.json) put loading/checking
and C emission near 100 ms combined, while a warm runtime-only release build still takes over
three seconds. Generated-unit object reuse targets that repeated C compilation, without retaining
another frontend graph.

## Identity and bounds

The unit key has a separate versioned domain and the same compiler path/version/image, host,
target, consumer mode, flags, profile, effective environment, raw source and preprocessed snapshot
dimensions as the runtime key. MSVC preprocesses the stable relative name `unit.c` with `/EP`.
The exact returned snapshot is compiled on a miss, so header changes invalidate the object and
cannot change compiler input after hashing. `CL` or `_CL_` injection bypasses both caches.
Mutable toolchain installation during a build remains outside this cache's claim.

The existing opened-root, digest-checked `DiskStore` is reused in the OS user-cache directory at
`fairyc/native-unit-v1`. It keeps at most **two generations and 4 MiB of encoded records in total**,
with a 2 MiB per-record bound and the existing 24-hour temporary-file expiry. This is shared across
projects and profiles; switching among more than two unit inputs can evict a useful object. It is
not a per-project cache multiplied by the number of projects. Oversized objects simply do not
persist. Runtime retention stays at its existing independent 64-generation/64-MiB limit.

As with the existing store, limits exclude directory metadata, live temporary files and compilation
peak space. Concurrent publication converges through the existing collection policy.

## Failure behavior

Cache unavailability and snapshot failure use the original generated-unit compile path. Invalid
or checksum-corrupt records miss. A structurally valid object rejected by the real linker is
removed by exact record identity, then compiled once from the current snapshot. A persistent
compile/link failure remains an error and publishes no replacement object. Only successful links
publish cache records. The nested runtime/unit recovery paths allow at most one fresh retry per
object per attempt, hence at most four link attempts if both cached layers are rejected.

Link-only builds use `link.exe` beside the resolved `cl.exe`; prebuilt object validation, library
environment, output inspection and temporary cleanup still use the common Windows builder. No
dummy source file, extra linker implementation or new library dependency is introduced.

## Validation

The Windows native test checks cold/warm use in both profiles, generated-source and included-header
changes, runtime-only changes, checksum corruption, a checksum-valid object without `main`, a
persistent missing-symbol link failure, cache outage and eight edits beyond the retention limit.
Key tests cover every identity dimension and separation from the runtime key. Existing runtime
cache and stable CLI probes retain their independent output/trap expectations.

CLI flags, stdout, exit codes, language/SDK/runtime revisions and target maturity are unchanged.
Linux generated-unit reuse is not enabled by this decision. Timing and retained-byte observations
must be recorded before asserting a performance improvement.
