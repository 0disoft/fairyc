# Linux directory paths — 2026-11-14

Implemented create, ensure/create-all, empty remove, no-replacement rename and
bounded recursive removal. Parents are opened component by component using
O_DIRECTORY/O_NOFOLLOW. Tree removal refuses a root named `.`, `..` or `/`,
refuses a link as the root, unlinks nested links without traversal, checks opened
directory inode/device identities, and refuses cross-device directories. Limits
are 64 directory levels and 65,536 descendants. Errors preserve the first OS
failure, report the entries already removed and close owned descriptors. Removal
is not transactional or a containment sandbox; same-device bind mounts and
concurrent filesystem changes do not gain stronger guarantees than stated.

The actual production C passed the Windows POSIX tree model in both low and
optimized profiles (0.90 s). It covers create-all, exclusive create, existing
destination preservation, rename, empty remove, recursive content removal,
external link-target preservation, cross-device refusal with partial counts,
root/parent refusal, stream acquisition failure, depth/entry bounds and cleanup.
No model handles remain on any checked path.

Final focused C source/model batch: 3.943 s; full component contract: 0.035 s.
Logs: `.cache/linux-fs-final-20261114.log` and
`.cache/linux-fs-contract-final-20261114.log`. Previously Windows-only test
expectations were narrowed to the still Windows-specific operations, while new
tests assert the exact newly portable closures and unchanged Windows closure.

The separate gcc/clang low/optimized native probe now includes these path/tree
operations. It is skipped on this Windows host, not reported as a Linux pass.
No screen/input tests, SDK/source version bump, release or target promotion.
