# Private Linux directory enumeration v1

`internal/candidatehost/linux_directory.c` owns the implemented native cursor from commit
`3fb7c48f`. The private `linux_directory.fy` module implements the bounded, ownership-safe scan.
This enumeration contract excludes deletion, retention, CLI, and public SDK surfaces.

## Native cursor (implemented)

The private ABI is:

```c
uint32_t fairyc_private_linux_directory_scan_open_v1(uint32_t root, size_t *output);
uint32_t fairyc_private_linux_directory_scan_next_v1(size_t handle, uint8_t *name,
        size_t capacity, uint32_t *length, uint32_t *state, void *metadata);
uint32_t fairyc_private_linux_directory_scan_close_v1(size_t handle);
```

`open` takes a retained `rooted::Directory` handle. It calls
`openat(root_fd, ".", O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC)` and hands that fresh
descriptor to `fdopendir`. It does not `dup` the root descriptor, so each cursor has an independent
directory offset. The published cursor is the owned `DIR *` pointer encoded as `size_t`; zero means
empty. Success returns zero with a nonzero cursor. Failure leaves output zero. If `fdopendir`
fails, `open` closes the freshly opened descriptor before returning the error.

After requiring non-NULL `length`, `state`, and metadata pointers, each `next` call resets `length`
and `state` to zero and clears the 40-byte metadata output. It accepts only a nonzero cursor and a
non-NULL 256-byte `name` buffer. It performs exactly one `readdir` attempt per call. Dots,
vanished names, retained records, and the EOF probe are therefore separate calls. A successful EOF
probe returns state 0. A `readdir` error returns its errno.

The entry must extend beyond `d_name`'s offset. Name scanning is capped at 256 bytes, allowing at
most 255 bytes plus a NUL. Empty names, embedded `/`, or a missing NUL in a full record are
rejected. `d_type` and `d_ino` are ignored. A non-dot name is checked with
`fstatat(dirfd(stream), name, &info, AT_SYMLINK_NOFOLLOW)`, retrying `EINTR`. `.`, `..`, and an
`ENOENT` from that check return state 2 with the name populated and zero metadata. Other stat errors
return errno.

State 1 copies the name and actual metadata observed by `fstatat`. The 40-byte
`rooted::Metadata` layout is, in order: `device` (`st_dev`), `identity` (`st_ino`), `size`
(`st_size`), `modified_seconds` (`st_mtim.tv_sec`), `modified_nanos` (`st_mtim.tv_nsec`), and
`mode` (`st_mode`). The native layer rejects negative sizes and negative or out-of-range nanos.
The Fairy wrapper accepts only the seven Linux inode types (`S_IFIFO`, `S_IFCHR`, `S_IFDIR`,
`S_IFBLK`, `S_IFREG`, `S_IFLNK`, `S_IFSOCK`) and nanos 0..999,999,999; other values are ABI
problem 6. The scan is not a snapshot: a name that disappears
before its stat can be skipped as state 2, while metadata for a replaced entry describes the
object stat sees at that moment.

`close` consumes the cursor and calls `closedir` exactly once; a zero handle is a no-op. Close is
never retried, and a failure returns its errno after ownership has been consumed. The Fairy caller
clears its raw handle before the one explicit close attempt.

## Implemented Fairy scan

The module imports `linux_rooted_file as rooted`. Its public operation is
`scan(ref rooted::Directory root, usize maximum, ref mut usize work) -> Result<Entries, Problem>`,
where `1 <= maximum <= 4096`. `Entries` owns two aligned vectors: `names` as
`Vec<bytes::Buffer>` and `facts` as `Vec<rooted::Metadata>`. Returned names and metadata have
detached ownership and retain the operating system's unsorted `readdir` order.

The caller owns one cumulative `usize work` counter shared across scans. Each scan charges 1 unit
before opening and 297 before every `next`, including dot and vanished records and the final EOF
probe. A retained entry additionally charges its name length plus 40 bytes before allocation or
retention. The 16 MiB (`16_777_216`) cap limits this logical, byte-weighted work count; it is not a
memory peak, allocation limit, or wall-clock bound. A `fstatat` `EINTR` retry adds no work and is
not time-bounded. Checks happen before the charged operation, so crossing the cap fails without
returning a prefix.

The wrapper validates the native `length`/`state` agreement and every retained name as valid UTF-8
of 1 through 255 bytes without NUL or `/`. All non-dot names are validated even when state 2 says
the entry vanished. Exact `.` and `..` names are skipped only in state 2. Any malformed result,
work limit, item limit, allocation failure, or native error fails the entire result.

`Problem.kind` is 1 invalid input, 2 allocation, 3 open, 4 next, 5 close, 6 native ABI/malformed
state, 7 work or item limit, and 8 UTF-8 or invalid name. `os_code` retains the native errno where
applicable. If a scan already failed, `close_code` preserves a secondary close error; if scanning
succeeded but close failed, the result is problem kind 5. Raw ABI values are confined to private
cursors and rooted resources, with cleanup calling native close exactly once.

## Verification boundary

`TestPrivateLinuxDirectorySelection` and `TestLinuxDirectoryAdapterBoundaries` pass on Windows amd64
with MSVC low and optimized profiles using POSIX substitutes. `TestLinuxDirectoryPolicy` passes
both-target emission and Windows low/optimized execution, covering maximum 4096 and failure at
4097, exact 16,777,216-unit work and the one-unit-short boundary, skipped-only work churn, name and
metadata ownership after a second scan and root close, invalid UTF-8 on retained and vanished
names, and preservation of a primary error when close also fails.

`TestLinuxDirectoryAdapterNative` covers GCC/Clang low and optimized profiles, 604 retained
entries after 601 files plus a symlink, directory, and FIFO, root rename, and independent cursors.
It skips on Windows, so no native Linux pass is claimed. Both `check` and `check --locked`
pass for 545 sources and 6971 MIR; whole `go vet ./...` and related directory/rooted tests pass.

References: [readdir(3)](https://man7.org/linux/man-pages/man3/readdir.3.html),
[fdopendir(3)](https://man7.org/linux/man-pages/man3/fdopendir.3.html), and
[stat(2)](https://man7.org/linux/man-pages/man2/stat.2.html).
