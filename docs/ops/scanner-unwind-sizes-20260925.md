# Scanner final-image unwind function sizes

Date: 2026-09-25. Source HEAD before this measurement:
`5a3f526f2a135eb80108a21739ecbab5f2538ece`.
The optimized Windows AMD64 MSVC four-image scanner smoke still checks the
reporter, 1,000 completed rounds and checksum. It now validates every PE
`.pdata` interval through `InspectAMD64Unwind` and matches selected MSVC
map symbol addresses to exact interval starts. The
[raw receipt](evidence/scanner-unwind-sizes-20260925.json) records those
matches and complete non-overlapping interval totals.

| Image | EXE bytes | `.text` virtual bytes | `.pdata` intervals | Bytes inside those intervals | Other `.text` virtual bytes |
| --- | ---: | ---: | ---: | ---: | ---: |
| Up-front C | 184,320 | 124,320 | 526 | 115,387 | 8,933 |
| Geometric C | 185,344 | 125,024 | 527 | 116,098 | 8,926 |
| Explicit-index Fairy | 206,336 | 144,912 | 589 | 135,509 | 9,403 |
| Private vector-loop Fairy | 206,336 | 144,912 | 589 | 135,509 | 9,403 |

Against up-front C, Fairy's `.text` virtual size is 20,592 bytes larger.
The validated function intervals account for 20,122 bytes of that difference;
the virtual bytes outside them account for 470. The `.text` raw-file
difference is 20,992 bytes because the Fairy image has 400 more bytes of
raw-section padding. These figures reconcile the full raw difference without
assigning the interval bytes to a source object or library.

| Fairy symbol | Source function | Earlier COFF raw span | Linked `.pdata` interval |
| --- | --- | ---: | ---: |
| `fy_fn_10` | `numbers::decimal` | 3,118 | 3,118 |
| `fy_fn_28` | `scanner::report_work` | 2,495 | 724 |
| `fy_fn_29` | `scanner::scan` | 5,297 | 134 |
| `fy_fn_36` | `strings::escape` | 3,071 | 98 |
| `fy_fn_41` | `strings::scan` | 3,134 | 69 |

The COFF section span is not the final linked function length; these
examples show why optimizing the five largest input spans first would be
poorly grounded. The `.pdata` table does not include every executable byte,
such as leaf code or padding. The selected map rows do not assign all 589
Fairy intervals to `unit.obj`, `runtime.obj`, or CRT inputs. The next
attribution step is to join all map-owned public starts to unique
`.pdata` intervals, report unmatched and ambiguous bytes separately,
then select a measured source or emission change. The cumulative 8%
C-relative runtime and EXE budget and complete C scanner parity remain open.
