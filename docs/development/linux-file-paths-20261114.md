# Linux portable file path operations — 2026-11-14

Added dependency-closed Linux implementations for std::fs file rename, hard
link and byte copy using the existing compatibility ABI names. Windows source
selection is unchanged. Source-only File users do not select the new component.

Executed: `TestLinuxFilePathsModel` in both MSVC profiles and the Linux/Windows
component-selection controls. Production C bodies execute against an explicit
POSIX syscall model: exclusive-create preservation, no-replace rename, hard
links, EINTR reads, short/zero writes, partial-error counts, descriptor cleanup,
invalid paths and directory-source refusal. Focused suites pass (0.925 s C
stdlib, 0.041 s contract). `.cache/linux-path-final-20261114.log`.

`TestLinuxFilePathsNative` is added for real gcc/clang low/optimized execution
and real inode, destination-preservation and rename behavior. It is skipped on
this Windows host. WSL discovery timed out after twelve seconds; no Linux
compiler/container is available on PATH. Native Linux behavior is unverified
here; no target maturity or release claim was made. No desktop/input interaction.

Metadata and directory/enumeration methods remain separate implementation units.
