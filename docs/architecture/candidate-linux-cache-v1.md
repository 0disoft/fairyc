# Private Linux candidate workspace cache v1

The Linux dispatcher accepts `check/build/run --cache-root <existing directory>`
and optional `--cache-status`, together with each command's `--locked` and
build/run `--release` flags. The shared option parser rejects missing roots,
duplicates, unknown options and status without a root before workspace access.
Caching remains opt-in; it does not create the caller's cache directory. Cache
options for Linux `test` remain separate work. For `run`, cache flags after `--`
belong to the program and do not select compiler cache behavior.

## Platform selection and admission

The preparation tool selects these authored boundaries in its isolated source
copy, in addition to the existing Linux CLI selections:

- `project_cache_disk` uses `linux_cache_file` transport and `linux_rooted_file`
  handle types.
- `project_cache_environment` queries the actual image path through
  `linux_host_paths`; `project_compiler_image_hash` streams that file through the
  rooted Linux workspace reader, retaining the existing 96 MiB image bound.
- `project_cache_profile` selects `x86_64-linux-gnu` and the target owner's LP64,
  System V AMD64, ELF64 and DWARF CFI ABI identity. Windows cache keys remain
  separate even when source text and the bundled SDK match.

The compiler image SHA, bundled SDK identity, source/lock/dependency admission,
per-unit keys, envelope/digest checks, fresh IR validation, work histories and
cold compilation fallback remain shared. No cached native object is reused.
Storage/initialization failures are disposable misses; invalid source and lock
failures retain their diagnostics and exit codes. `--cache-status` writes the
existing hit/mixed/units-hit/cold-stored/cold-uncached/skipped status to stderr.

`linux_build_cli` selects `project_emit.evaluate_cached_run` when an explicit
cache root is present, and the ordinary evaluation path otherwise. Both paths
finish with the same entry selection, fresh C packet emission and native
companion invocation. Development and release share verified IR; their native
compile/link profiles still differ. A hit never reuses a native image or bypasses
the sibling companion. Frontend/lock failures are reported after any requested
cache status; status-output failure returns 4 before native staging.

Run retains the current compilation source snapshot on cold and warm paths for
trap locations. The native output remains a private stage image; published
`build/<name>` contents, program argument/stream handling and exit codes retain
their existing behavior. Cache status is emitted before collected program stderr.

## Filesystem policy

The cache root must already exist. Descriptor-relative traversal rejects symlink
prefixes and parent traversal. Reads require regular nofollow leaves. Linux
metadata retains device, inode, size, signed seconds, nanoseconds and mode;
ordering never converts Linux time to Windows ticks. Conditional removal passes
the exact observed metadata to the existing Linux retention adapter.

The shared disk policy delegates metadata ordering, temporary aging and
publication-collision classification to its selected storage module. Windows
retains named staging, tick-based aging and its existing collision codes. Linux
uses anonymous `O_TMPFILE` stages, sync-before-exclusive-link publication and
`EEXIST` collision classification. The generated shared stage-name argument is
validated but never creates a Linux directory entry. Unsupported `O_TMPFILE`
filesystems fall back to uncached compilation. Close releases unpublished
anonymous stages; no named fallback or recursive deletion is introduced.

Linux inventory retains only recognized generation names. Foreign files,
symlinks, directories and Windows-style `.stage` names are preserved. They still
consume the directory scanner's item/work limits. The 64-entry/1 GiB command
policy, 256 MiB record limit and existing bounded scan/I/O work remain in force
for recognized generations. Corrupt digest-named collisions are repaired only
after metadata-conditional removal; changed objects are preserved. Publication
does not establish directory-sync durability or a global concurrent-writer bound.

## Verification boundary

`TestLinuxCacheFilePolicy` emits both target profiles and executes the wrapper
with controlled native substitutes in low/optimized host profiles. It covers
anonymous ownership, binary partial writes/reads, exclusive publication and
errno preservation, filtered inventory, signed timestamps, exact removal
metadata, close-after-removal failure and work/path rejection.
`TestCandidateCacheDiskWindows` runs the five existing disk-cache fixtures with
the actual private Windows host support in low/optimized profiles, including
corruption recovery, immutable collisions, retention and live-stage preservation.

`TestLinuxFileCLI` checks whole-entry both-target emission, absence of Windows
foreign symbols, closed Linux support selection and the selected target/ABI
profile. Its Linux-only integration includes cold/warm locked check/build/run,
development-to-release IR reuse, warm trap source locations, binary program
streams, cache-looking arguments after `--`, corruption recovery, source/lock
invalidation, missing-root fallback and symlink-root rejection. Cached build
also verifies that companion failure preserves the previous published image.
Those Linux-native cases require Linux amd64 and a GNU toolchain;
Windows substitute execution does not establish Linux execution proof.
`TestLinuxBuildCacheArguments` executes the existing shared cache-option fixtures
in low/optimized host profiles, covering compiler/program flag separation,
duplicates, missing roots/status dependencies and NUL error precedence.

No public language/SDK API, database, dependency, repository hygiene rule,
workflow job, runner or target maturity flag changes in this slice. Linux native
execution, test cache wiring, target-specific public C emission,
self-build and distribution evidence remain open. Go stays the default compiler.
