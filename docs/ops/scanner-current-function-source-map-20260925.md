# Scanner current COFF function spans

Date: 2026-09-25. Current child HEAD:
`99fdd5e425566b2173e6721fdb114c54f3a5e3e7`.
The optimized Windows AMD64 MSVC four-image scanner smoke checked reporter
output, 1,000 completed rounds and checksum, then mapped bounded `fy_fn_N`
COFF starts to emitted source-function names. The
[current raw report](evidence/scanner-current-function-source-map-20260925.json)
contains image hashes and the largest 20 spans per object. Its four lanes and
object accounting passed the configured source-map checks.

| Fairy source function | Earlier raw span | Current raw span | Difference |
| --- | ---: | ---: | ---: |
| `scanner::scan` | 5,297 | 4,083 | -1,214 |
| `numbers::decimal` | 3,118 | 2,869 | -249 |
| `scanner::report_work` | 2,495 | 2,495 | 0 |
| `strings::scan` | 3,134 | 2,147 | -987 |
| `strings::escape` | 3,071 | 1,791 | -1,280 |

The [earlier source map](scanner-function-source-map-20260925.md) measured
`unit.obj` at 32,116 raw code-section bytes; the current object has 27,646,
a reduction of 4,470. Its mapped function count remains 81. The C `probe.obj`
code sections remain 13,653 bytes in the up-front lane and 14,803 bytes in
the geometric lane. The [current four-image smoke](scanner-string-state-borrow-20260925.md)
measured both Fairy EXEs at 202,240 bytes versus 206,336 bytes in the earlier
source-map period.

These are object-section spans, which may contain padding and functions that
the linker discards or folds. They do not allocate the final 3,584-byte
remaining EXE size gap to particular functions. The unchanged reporter is a
bounded next audit target because its source path is exercised by the smoke,
but a source edit still needs direct C-relative size and runtime evidence.
The 8% gate, complete C scanner parity, paired check/build time and stable
runtime measurements remain open.
