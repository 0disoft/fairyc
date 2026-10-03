# ADR 0428: Bounded native runtime object reuse

- Status: Implemented (Windows native evidence; Linux native evidence pending)
- Date: 2026-10-03

## Decision

Workspace native compilation reuses only the project-owned runtime/support C
object. Generated Fairy C translation units are always compiled; linking and
executable inspection always run against the current host libraries. Source,
semantic, Flow and MIR validation and lock checking remain fresh. This is not
package MIR reuse, executable reuse, LTO or general incremental compilation.
Stable build/run/native-test and their shared experimental workspace paths use
the adapter. Raw driver probes, sanitizer and cross-compiler evidence remain
uncached.

The key includes a private version, host/target, C consumer mode, profile, exact
compile flags, compiler path/version/image digest, environment, Linux libc
identity, raw runtime source digest and current preprocessed-source digest.
Preprocessing runs on every build: Windows uses MSVC `/EP`; Linux uses `-E`
with system-header markers retained for strict warning semantics. Compiler
stdout and diagnostics are separated under a 16 MiB combined bound and the
existing 60-second command timeout.

The preprocessed snapshot itself is compiled on a miss, so a header update
between preprocessing and object compilation cannot publish a different input
under the old key. Runtime source is passed as the stable relative `runtime.c`
name during preprocessing. Time/file macros are expanded into the snapshot;
changing expansions invalidate reuse. No claim is made about caching arbitrary
external C sources or mutable compiler installations during a build. MSVC
`CL`/`_CL_` injection bypasses this closed path.

## Storage and failure behavior

Objects use the existing digest-checked immutable `packagecache.DiskStore`
record transport, in the OS user-cache directory under
`fairyc/native-runtime-v1/v1`. Windows uses `%LOCALAPPDATA%`; Linux uses
`$XDG_CACHE_HOME` or `~/.cache`. This is separate from the explicitly selected
workspace MIR cache and the repository's bootstrap `.cache`. The store remains
inside an opened user-cache filesystem boundary. `--locked` still checks the
lock exactly; it does not disable this disposable native cache.

The native store allows 64 published generations and 64 MiB combined encoded
record bytes, with a 16 MiB per-record reader/writer bound. Temporary files expire
after 24 hours. It reuses the existing oldest-generation collection policy,
including safe conflict misses and concurrent publication behavior. These bounds
exclude directory metadata, live temporary files and peak compilation space;
concurrent writers can temporarily exceed them until collection converges.
No worker, daemon, lock file, remote cache or CI job is added.

Missing, checksum-corrupt, conflicting or invalid COFF/ELF records cause current
runtime compilation. A reused object rejected during compile/link is quarantined
by exact record identity and gets one fresh runtime compile/link retry. Only
successful links publish objects. Store-open/read/write/collection failures never
convert a successful compilation into a build failure. Preprocessor failure or
overflow selects the original uncached driver. Compiler and linker failures
remain real failures; there is no fake successful artifact or compiler switch.
Removing the private native cache directory discards only regenerable objects.

## Evidence and limits

Tests cover key dimensions, store reopening, checksum corruption, invalid object
payloads, store outage/write failure, successful and persistent link failure with
bounded retry. A native test compiles and executes cold/warm runtime objects,
changes only the generated unit, changes an included header, changes raw source,
profile and compiler identity, corrupts retained records, and substitutes a
checksum-valid but incompatible native object to exercise the real linker retry.
Workspace integration uses Fairy source importing `std::io`, verifies UTF-8
stdout/stderr and reserved normal status on cold/warm development/release builds,
then checks the warm CLI trap channel.

Windows MSVC execution was observed locally. Linux adapter and portable tests are
present with separate GCC and Clang subtests, but Linux native execution cannot
be performed on this Windows host; both executions remain required externally.
Compiler-process counts alone
do not establish faster builds: preprocessing, hashing and cache I/O add work,
particularly on cold builds. No timing or performance improvement is claimed.
SDK, language, runtime ABI and stable CLI output/schema versions are unchanged;
the object-cache format has its own version. No dependency or runner changes.
