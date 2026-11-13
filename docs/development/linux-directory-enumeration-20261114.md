# Linux directory enumeration — 2026-11-14

Implemented the existing ReadDir resource ABI with fdopendir/readdir and
consuming closedir/drop. Directory prefixes are opened with openat and
O_DIRECTORY/O_NOFOLLOW; names/path bytes are UTF-8 validated. Entries are
classified with fstatat(AT_SYMLINK_NOFOLLOW). Symlinks are reported as Other
with the reparse flag, and are not traversed. A short destination buffer or
transient metadata error retains the same pending entry for retry. EOF is
stable. Relative dot/parent components retain ordinary filesystem meaning;
this is not a workspace containment API.

Actual production C runs against a Windows-hosted POSIX model in both profiles
(1.32 s), covering prefix-link refusal, invalid encoding/interior NUL, buffer
retry without writes/advancement, metadata retry, symlink classification, EOF,
failed stream acquisition, consuming close and idempotent drop. Linux/Windows
minimal dependency tests pass (0.034 s). Log:
`.cache/linux-directory-model-20261114.log`.

The separate real Linux gcc/clang test is added but skipped here. WSL remains
unavailable; this is source/model evidence, not actual Linux filesystem proof.
No desktop/input interaction or target promotion.
