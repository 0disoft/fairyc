# Private candidate Linux rooted files v1

`examples/lexer_candidate/src/linux_rooted_file.fy` owns handle resources, path admission,
failure mapping, staged publication, and consuming close. The closed native adapter in
`internal/candidatehost/linux_rooted_file.c` implements the private C ABI. Neither file owns a
cache format, artifact decoder, compiler policy, or public SDK surface. Public SDK and CLI
contracts remain unchanged.

## Implemented read slice

The completed first-slice commit `bf2a4a52` implements directory, file, metadata, read, and
close operations. Its six private entry points are `fairyc_private_linux_directory_open_v1`,
`fairyc_private_linux_root_directory_open_v1`, `fairyc_private_linux_root_file_open_v1`,
`fairyc_private_linux_root_file_info_v1`, `fairyc_private_linux_root_file_read_v1`, and
`fairyc_private_linux_root_file_close_v1`.

`open` accepts an existing root path of 1 through 4095 UTF-8 bytes and appends one NUL byte.
`linux_command::path` validates the text, then the adapter opens it relative to `AT_FDCWD` with
`O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC` and checks the descriptor with `fstat`. The
path may be absolute or relative to the process working directory. Admission rejects a trailing
`/` except for the single-character `/`, because a trailing slash would otherwise let
`O_NOFOLLOW` traverse a final symlink to a directory.

`directory` and `open_read` accept one flat leaf relative to a retained Directory. The Fairy
wrapper requires valid UTF-8 of 1 through 255 bytes, rejects `.` and `..`, and rejects NUL and
`/`. The native adapter independently enforces the NUL terminator, rejects embedded NUL and a
slash in a leaf, and bounds the encoded leaf at 256 bytes including its terminator. A child
directory must resolve to a directory; a file open must resolve to a regular file. Both use
`openat` against the retained Directory descriptor with `O_NOFOLLOW | O_CLOEXEC`. File opens
also use `O_NONBLOCK | O_NOCTTY` so a substituted FIFO cannot block before the type check.

The root open applies `O_NOFOLLOW` to its final component. Intermediate symlinks in the initial
root path may resolve normally. Later child opens use the retained directory object, so replacing
or renaming the root pathname cannot redirect them, and a final symlink is rejected. This is not
a sandbox or snapshot: hard links can alias the same object elsewhere, another process can edit
it in place, and a later read can observe different bytes or metadata.

The 32-bit handle is encoded as `descriptor + 1`; zero means no owned descriptor, while OS
descriptor zero remains valid. Valid handles range from 1 through 2147483648, corresponding to
descriptors 0 through `INT_MAX`. Child operations reject zero and out-of-range values; this
private ABI keeps no descriptor registry. Explicit close clears its local consumed handle before
one native close attempt and reports errors. Automatic cleanup calls native close once without
clearing a stored handle first and discards the result. Close is never retried because Linux may
already have released a descriptor when `close` reports `EINTR`.

Each `file_read` request is limited to 65536 bytes. A zero-length request succeeds with zero
copied bytes; a larger request, invalid handle, null buffer with nonzero size, or output count
larger than the supplied buffer fails. Short reads and zero at end of file are ordinary results;
the cap is per call rather than an aggregate payload limit.

`file_info` calls `fstat` on the retained File descriptor. It rejects non-regular files, negative
sizes, sizes that do not fit `size_t`, and out-of-range nanoseconds. The private 40-byte metadata
layout is, in order: `device` (`st_dev`), `identity` (`st_ino`), `size` (`st_size`),
`modified_seconds` (`st_mtim.tv_sec`), `modified_nanos` (`st_mtim.tv_nsec`), and `mode`
(`st_mode`). Failures are reported as errno values, with wrapper stages 1 input/bound,
2 allocation, 3 directory, 4 file, 5 metadata, 6 read, 7 close, and 8 native ABI. `missing`
recognizes `ENOENT` only for directory or file stage.

Earlier first-slice evidence remains: the original `TestLinuxRootedFileAdapterBoundaries` passed
with Windows MSVC POSIX substitutes in low and optimized profiles, and the original
`TestLinuxRootedFilePolicy` passed both-target Fairy emission plus Windows low and optimized
execution. Focused vet passed then. Native Linux GCC/Clang low/optimized execution was unavailable
because the host is Windows and WSL is unavailable; `wsl --list` failed with
`REGDB_E_CLASSNOTREG`.

## Implemented anonymous write and publication slice

The current sources add `create`, `write_all`, `sync`, and `publish_new`, backed by
`fairyc_private_linux_root_file_create_v1`, `..._write_v1`, `..._sync_v1`, and
`..._publish_new_v1`. `create(root)` opens `.` relative to the retained Directory with
`O_TMPFILE | O_RDWR | O_CLOEXEC` and post-validates a regular file. It creates no stage
filename. `O_TMPFILE` support is required; an unsupported filesystem fails at stage 9 and has
no named-file fallback.

`write_all` accepts only a staged File and loops until the whole input is written, using chunks
of at most 64 KiB for native `write`. A short write advances the loop. A native error returns
stage 10 with `completed` set to the successful prefix; a zero write or an over-count is a
malformed native result at stage 8 and preserves that prefix. The native entry point also caps
each call at 64 KiB.

`publish_new` validates the leaf, automatically calls `sync` before publication, and then
invokes the native publisher; the caller does not need an explicit sync first. A sync failure is
stage 11. The C publisher requires a regular file with `st_nlink == 0` before the exclusive
link, then tries `linkat(file_fd, "", directory_fd, leaf, AT_EMPTY_PATH)`. Direct-link
`EPERM`, `EINVAL`, and `ENOENT` alone trigger the `/proc/self/fd/<file_fd>` plus
`AT_SYMLINK_FOLLOW` fallback. Hosts where the direct link lacks permission therefore require
usable procfs. `EEXIST` and all other errors return directly, so collisions never use the
fallback; `collision` recognizes stage 12 with `EEXIST`.

Publication is exclusive and does not replace an existing name. The wrapper exposes no named
delete or replacement operation. A successful native publish marks `staged = false`; further
`write_all` or `publish_new` calls fail at the wrapper boundary. That freeze is an API rule,
not filesystem immutability: other processes can still modify the published name or inode.
Closing an unpublished anonymous inode reclaims it because it has no directory entry; a published
file remains named after close.

The sequence does not sync the parent directory and does not establish physical power-loss
retention. A crash or device failure can lose or change a directory entry or data after success.
Directory enumeration is implemented in the separate [scan adapter](candidate-linux-directory-v1.md).
CLI connection, retention decisions and retention erase remain open.

Authoritative verification is complete. Candidate check and `check --locked` pass with 544
sources and 6954 MIR functions; all `go vet ./...`, related C adapter, process, capture, argv,
and host-path regressions pass. Rooted wrapper emission passes both targets and Windows low and
optimized execution. The historical first-slice checkpoint remains 544 sources and 6949 MIR
functions. `TestLinuxRootedFileAdapterNative` skips on Windows; WSL is unavailable and
`wsl --list` reports `REGDB_E_CLASSNOTREG`. Native Linux execution is deferred, and no native
Linux pass is claimed.

The authored native probe covers real symlink and FIFO rejection, retained-directory use after
the root is renamed, anonymous publication with an `EEXIST` collision, and unpublished close.
Wrapper fixtures cover a 131079-byte chunked write, an error after the 7-byte prefix, sync before
link, collision cleanup, and rejection of further writes after publication.

Primary Linux references: [openat(2)](https://man7.org/linux/man-pages/man2/openat.2.html),
[open(2)](https://man7.org/linux/man-pages/man2/open.2.html),
[fstat(2)](https://man7.org/linux/man-pages/man2/fstat.2.html),
[read(2)](https://man7.org/linux/man-pages/man2/read.2.html),
[close(2)](https://man7.org/linux/man-pages/man2/close.2.html),
[write(2)](https://man7.org/linux/man-pages/man2/write.2.html),
[fsync(2)](https://man7.org/linux/man-pages/man2/fsync.2.html),
[linkat(2)](https://man7.org/linux/man-pages/man2/linkat.2.html), and
[proc(5)](https://man7.org/linux/man-pages/man5/proc.5.html).
