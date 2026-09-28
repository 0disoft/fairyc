# Linux Vec iteration native revalidation (2026-09-28)

An external tester ran the supplied `compiler.test` binary for source commit
`f939b390c5c5a7e72f1590ae18f45ce9c1f2f9e4` once, selecting
`TestPrivateVectorSDKCandidateLinuxVectorLoopEmission` with `-test.count=1`.
The supplied report, raw log, results JSON and evidence ZIP were reviewed
locally; this repository's Windows host did not execute the Linux binaries.
The supplied test binary's SHA-256 is
`6c557f420367e0d06d518bf8bde48263091b668c03ae1aed90dc56eebc3b3838`.

The tester reported Debian 13.3, x86_64, GCC 14.2.0 and glibc 2.41. All
four builds retained `-std=c11 -pedantic-errors -Wall -Wextra -Werror
-fno-common`; low used `-O0` and optimized used `-O2`.

| Source form | Profile | GCC | Native exit | stdout / stderr |
| --- | --- | ---: | ---: | --- |
| Explicit index | Low | 0 | 7 | 0 / 0 bytes |
| Explicit index | Optimized | 0 | 7 | 0 / 0 bytes |
| Vec iteration | Low | 0 | 7 | 0 / 0 bytes |
| Vec iteration | Optimized | 0 | 7 | 0 / 0 bytes |

The Go test process passed without failed or skipped subtests. The fixture
requires `vector_live == 0` and `vector_allocations == vector_frees` after
function return. The tester also ran byte-identical copies of the four ELF
files once each to capture separate stdout and stderr. The two optimized
ELFs had the same SHA-256,
`a673ebf6211487458c52e77acb3e9603e5aaf9d3c717b8813973a25b7b68d769`.
That identity applies to this fixture and these build settings only.

The supplied evidence archive was
`fairyc-f939b390-linux-vec-revalidation.zip` (SHA-256
`fbd80b95faa2641da99f960ffa4ec9f690229ea41aa2af12e087b89bb090dd74`).
The companion raw log was `fairyc-f939b390-linux-vec-native.log` (SHA-256
`ccf8a55bd125c0aca809d085ac721b29e4b81091ec58c311dfe4cebd6e26e5d0`).
The archive contains the four generated `unit.c`/`runtime.c` pairs, GCC
invocation records, ELF files, raw output and test logs. These artifacts were
provided externally and are not vendored into the source tree.

This closes the Linux native correctness check for this one private Vec
fixture. It does not establish lexer parity on Linux, build/check cost,
general zero-cost lowering, or the cumulative C-relative 8% runtime and
executable-size gates. The earlier failed null control remains failed; no
performance samples or C-relative size comparison were taken in this run.
