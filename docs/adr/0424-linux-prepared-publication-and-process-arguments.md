# ADR 0424: Linux prepared-file publication and process arguments

- Status: Accepted
- Date: 2026-10-03

## Context

Linux contributors can build the Go compiler, but the task manager's public
`read_exact`, `remove_file`, `replace_file` and process argument calls previously
selected Windows-only companions. PowerShell evidence scripts also required a
maintainer's local artifact tree. A fork must provide its own source-based tests.

## Decision

SDK `0.0.0-stage0.79` selects target-owned implementations through neutral
`fy_std_file_read_exact_v0`, `fy_std_file_remove_v0` and `fy_std_file_replace_v0`
symbols. The public Fairy signatures, language/20 and runtime/0 do not change.
Windows adapters delegate to the existing implementations. Historical Windows
symbols remain admitted for existing internal probes.

On Linux GNU amd64, `read_exact` repeatedly calls the existing descriptor read,
returns status 7 and the initialized prefix on premature EOF, preserves the
unread suffix, and checks the handle even for an empty view. OS errors retain
`errno`. `remove_file` unlinks one name; directories fail. `replace_file` validates
bounded nonempty CStr paths and a regular prepared source with `lstat`, then
calls `rename` once. It rejects source symlinks and directories. A missing live
name is allowed. It never deletes the destination first, copies across mounts,
flushes file contents, promotes orphans or chooses a recovery generation.

Callers must use distinct names for different objects, prepare a sibling on the
same mounted filesystem, check write/sync/close, exclude competing writers and
trust the directory. The `lstat` check is not a race-proof identity gate. Linux
can replace an open destination: existing handles still refer to the old inode.
This differs from Windows sharing denial. Same-inode names can yield an OS no-op;
alias publication is outside the caller contract. Cross-mount errors are returned
without a copy fallback. Network filesystem failure ambiguity remains OS-defined.

Linux process arguments use the same neutral symbols as Windows. Only units
referencing `arg` or `arg_count` receive `main(int argc, char **argv)` and a private
startup setter. The runtime borrows argv for the process lifetime, validates strict
UTF-8 before copying, and preserves the buffer on invalid encoding, out-of-range
index or insufficient capacity. Empty arguments write nothing. There is no shell,
locale conversion, wildcard expansion or environment fallback.

## Publication and durability

The preparation, exclusion and manual recovery policy from ADR 0411 still applies.
POSIX replacement publishes the destination name atomically on supporting local
filesystems. A file `fsync` does not flush the parent directory entry. This SDK
slice does not expose directory sync or promise recovery after power loss.
Physical power-cut tests, mount changes and disk filling are excluded.

## Validation boundary

Go tests compile the exact checked-in Fairy task manager for both targets. Native
tests run its commands, byte-order oracle, invalid input and preservation cases
in fresh temporary directories at low and optimized profiles on the current host.
Linux-only C probes select production support and check argv, EOF, replacement
and removal with GCC/Clang at both profiles. These tests are included in the
existing Ubuntu Go CI job; no extra runner is added. This Windows checkout passed
Windows native execution and both target emissions. New Linux native execution
has not been observed locally or in hosted CI at this commit.

At this decision, Linux application builds used `experimental build-workspace`.
[ADR 0425](0425-stable-linux-build-and-run.md) subsequently admits stable Linux
`build/run`. Native `test`, target maturity and release distribution remain separate.

Sources: [Linux rename(2)](https://man7.org/linux/man-pages/man2/rename.2.html),
[Linux fsync(2)](https://man7.org/linux/man-pages/man2/fsync.2.html).
