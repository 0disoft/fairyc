# Scanner function names behind COFF spans

Date: 2026-09-25. Source HEAD before this measurement:
`ac27b5d59dfda17b03111babe27b2696637f4a0f`.
The optimized Windows AMD64 MSVC four-image smoke retains the same reporter,
1,000-round and checksum checks. It maps each `fy_fn_N` COFF start to index
`N` in the C emitter's `Unit.Functions` list. The
[raw source map](evidence/scanner-function-source-map-20260925.json) includes
the largest 20 spans per object; `$p1` is the temporary test package ID.

| Fairy unit symbol | Source function | COFF raw section span |
| --- | --- | ---: |
| `fy_fn_29` | `scanner::scan` | 5,297 |
| `fy_fn_41` | `strings::scan` | 3,134 |
| `fy_fn_10` | `numbers::decimal` | 3,118 |
| `fy_fn_36` | `strings::escape` | 3,071 |
| `fy_fn_28` | `scanner::report_work` | 2,495 |

Both Fairy source forms have these same five largest spans, totaling 17,115
of the 32,116 raw code-section bytes in each `unit.obj`. All four rebuilt
lanes retain the previous object code totals, function-start counts and five
largest symbol spans. Their EXE hashes differ from the
[previous receipt](scanner-function-spans-20260925.md), so this comparison
does not claim byte-identical builds or explain the hash difference.

A COFF span can include padding and can refer to a function the linker later
removes. The C reference uses different function boundaries, so matching a
Fairy source name to a similarly named C function is not a valid size
comparison. The next size decision needs a linker map or equivalent retained
function evidence, then a measured change to the actual source or emission
path. No scanner semantics, language/SDK pin or runtime ABI changed here.
The cumulative 8% C-relative budget and full C scanner parity remain open.
