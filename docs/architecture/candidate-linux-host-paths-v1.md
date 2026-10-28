# Private Fairy Linux host paths v1

This is the first Linux candidate-host slice. `linux_host_paths.fy` exposes private
`current`, `temporary` and `executable` queries returning owned UTF-8 byte buffers.
It does not connect the Linux candidate CLI, process launcher or native build bridge.
The Windows host implementation and default Go compiler are unchanged.

`internal/candidatehost/linux_paths.c` implements the single private
`fairyc_private_linux_host_path_v1` ABI. Kinds 1, 2 and 3 query `getcwd`, `TMPDIR`
and `/proc/self/exe`, respectively. It returns zero or native `errno`, with a separate
output length excluding NUL. The output length is reset before each operation and
remains zero on failure. A full-capacity `readlink` result is rejected as `ERANGE`;
it cannot establish whether the link was truncated. The environment scan stops at
the supplied capacity and copies only a complete NUL-terminated value.

Fairy owns the 4096-byte scratch buffer, validates the returned length and NUL
terminator, rejects empty paths, embedded NUL and malformed UTF-8, and copies at
most 4095 bytes into the published result. Linux filenames with arbitrary non-UTF-8
bytes are outside this private slice. Backslashes and relative `TMPDIR` values are
preserved. An unset or empty `TMPDIR` selects `/tmp` in Fairy, matching Go's Linux
temporary-directory selection; this query does not create or validate a directory.
Executable discovery requires `/proc/self/exe`; no argv or PATH fallback is added.

Problems distinguish malformed paths (1), allocation (2), native errors (3, with
`errno`) and the private path bound (4). Native `ERANGE` remains a native error.
No partial path is returned after a native, validation or allocation failure.

`candidatehost.LinuxSource` selects the embedded adapter for the admitted private
symbol and rejects unknown Linux or Windows host symbols. It is an internal
selection seam for subsequent Linux bridge work, not a public linker extension.

## Validation boundary

- `TestPrivateLinuxHostSymbolSelection` checks exact selection and rejection.
- `TestLinuxPathAdapterBoundaries` compiles the production C adapter with only its
  POSIX calls replaced by deterministic test functions, in MSVC low/optimized
  profiles. It covers errors, truncation, absent/empty `TMPDIR`, exact capacity,
  output sentinels and invalid arguments. This is not native Linux execution.
- `TestLinuxHostPathsEmissionAndPolicy` emits the actual Fairy module for Linux
  and Windows, then executes its validation/fallback/error policy against a private
  ABI stub in both native profiles on the current Windows/Linux amd64 host. The
  Linux policy branch requires GCC.
- `TestLinuxPathAdapterNative` compiles the unchanged adapter with GCC and Clang,
  in low/optimized profiles, and compares OS queries with independent libc calls.
  It runs only on Linux amd64. Native Linux execution remains pending from this
  Windows development host.

This slice changes no public language/SDK ABI, target maturity flags, dependencies,
workflow jobs or distribution status. Process execution, rooted filesystem access,
Linux candidate dispatch and consecutive Linux self-build proof remain open.
