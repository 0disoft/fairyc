# Scanner MSVC link-map symbol receipt

Date: 2026-09-25. Source HEAD before this measurement:
`0b50f9d6af27629f247b127e0ab37e613fa6453e`.
The optimized Windows AMD64 MSVC four-image scanner smoke now requests
`/MAP` without changing the existing build profile. It still checks each
reporter, 1,000 completed rounds and checksum. The
[normalized symbol report](evidence/scanner-link-map-20260925.json) retains
EXE bytes, map byte count and hash, and selected `fc_` / `fy_fn_` rows with
section offset, preferred address, owner basename and Fairy source name.
Raw map text, including temporary paths, is not published.

| Image | EXE bytes | Map bytes | Selected prefix rows |
| --- | ---: | ---: | ---: |
| Up-front C | 184,320 | 301,225 | 14 |
| Geometric C | 185,344 | 301,385 | 15 |
| Explicit-index Fairy | 206,336 | 307,010 | 40 |
| Private vector-loop Fairy | 206,336 | 307,010 | 40 |

The map contains these five positive `unit.obj` rows in both Fairy images,
with matching section offsets and preferred addresses:

| Symbol | Source function | Section 1 offset |
| --- | --- | ---: |
| `fy_fn_10` | `numbers::decimal` | 0 |
| `fy_fn_28` | `scanner::report_work` | 9,616 |
| `fy_fn_29` | `scanner::scan` | 12,112 |
| `fy_fn_36` | `strings::escape` | 18,672 |
| `fy_fn_41` | `strings::scan` | 22,160 |

These rows establish linked symbol addresses for the five largest
[COFF spans](scanner-function-source-map-20260925.md). The prefix filter
does not enumerate every linked function or CRT contribution, and map rows
do not state exact machine-code lengths. Missing names cannot be treated
as discarded functions. This receipt therefore does not attribute the
20,992-byte final `.text` difference to functions or prove that changing
one will save a particular number of EXE bytes. The next step is bounded
final-image contribution accounting using the map's section layout and
the PE section receipt. The C scanner parity and cumulative 8% gates
remain open.
