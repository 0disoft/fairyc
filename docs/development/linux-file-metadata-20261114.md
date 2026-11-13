# Linux File metadata and length — 2026-11-14

Added eight separately selected Linux compatibility-ABI bodies: file kind,
path kind, read-only query/update, link count, allocated size, file identity and
set length. All select base File plus the metadata component; existing Windows
selection remains its original File component.

Read-only means absence of UNIX mode write bits, not an effective ACL/access
check. Enabling clears user/group/other write bits; disabling adds owner write
permission and preserves other permission bits. Path kind uses lstat, classing
symlinks as Other. Allocated size is st_blocks * 512 with overflow checks. The
existing FileId carrier stores device and inode low/high words with remaining
words zero. Observations are sequential, not an atomic filesystem snapshot.
ftruncate rejects lengths beyond signed 64-bit off_t and retains file position.

Production C bodies pass explicit Windows-executed POSIX models in both MSVC
profiles. Controls include closed/malformed storage, EINTR, modes, ID bit
preservation, unchanged output on error, allocation overflow and length bounds.
Minimal Linux/Windows dependency checks pass. Focused suites: 2.036 s stdlib,
0.039 s contract; `.cache/linux-metadata-final-20261114.log`.

A real Linux gcc/clang native probe checks actual stat/inode/link/mode/length
and cursor behavior. It is skipped on this Windows host; actual Linux execution
remains unavailable. No GUI/input interaction and no target promotion.
