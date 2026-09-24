# Scanner linked unwind interval ownership

Date: 2026-09-25. Source HEAD before this measurement:
`ee063f79c5976e7d6d824150c59014b80f91c49f`.
The optimized Windows AMD64 MSVC four-image scanner smoke retains its
reporter, 1,000-round and checksum checks. It joins every parsed
[MSVC map](https://learn.microsoft.com/en-us/cpp/build/reference/map-generate-mapfile?view=msvc-170)
public symbol address to the exact start of a validated final-image
`.pdata` interval. A function's bytes are assigned to one normalized
object owner only if all matching map rows name that same owner. Zero
matching owners are *unmatched*; conflicting owners are *ambiguous*.
Each interval is counted once. The
[raw receipt](evidence/scanner-unwind-owners-20260925.json) contains the
owner totals but no temporary build paths.

| Image | `.pdata` code bytes | Single-owner bytes | Unmatched bytes | Ambiguous bytes |
| --- | ---: | ---: | ---: | ---: |
| Up-front C | 115,387 | 102,279 | 12,823 | 285 |
| Geometric C | 116,098 | 102,544 | 13,269 | 285 |
| Explicit-index Fairy | 135,509 | 110,054 | 25,193 | 262 |
| Private vector-loop Fairy | 135,509 | 110,051 | 25,196 | 262 |

Against up-front C, the explicit-index Fairy interval total is 20,122
bytes larger: 7,775 in uniquely owned intervals, 12,370 in unmatched
intervals and 23 fewer ambiguous bytes. Its uniquely owned `unit.obj`
intervals total 9,737 bytes across 32 functions; `runtime.obj` totals
2,998 across 21. The C `probe.obj` has 3,317 bytes across 16 uniquely
owned functions. These are **partial linked totals**, not object-wide
function totals. In particular, the unmatched category is too large to
assign the Fairy/C size difference to those three objects or to CRT code.

The private vector-loop image has the same overall `.pdata` bytes as the
explicit-index image. Its uniquely owned `unit.obj` total is three bytes
smaller and its unmatched total three bytes larger. That change in
attribution is not a measured change in final executable size; both EXEs
remain 206,336 bytes. The map cannot prove absence of unnamed/static
functions. Further attribution needs a source of final-image static symbol
identity or another bounded method before a targeted size fix can be
claimed. The cumulative 8% C-relative runtime and EXE gates and complete
C scanner parity remain open.
