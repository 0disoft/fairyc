# Scanner COFF function-span receipt

Date: 2026-09-25. Source HEAD before this measurement:
`35864cbac88169771544ea017cf2d905745b24d0`.
The optimized Windows AMD64 MSVC four-image scanner smoke checks the reporter,
1,000 completed rounds and checksum, then reads the retained COFF objects.
The [raw report](evidence/scanner-function-spans-20260925.json) records the
owning EXE hash, every object's code-section total and the largest 20
function-symbol spans per object.

| Image and object | Function starts | Code-section raw bytes | Bytes covered by function spans | Largest span |
| --- | ---: | ---: | ---: | --- |
| Up-front C `probe.obj` | 45 | 13,653 | 13,653 | `fc_scan_string` 3,291 |
| Geometric C `probe.obj` | 47 | 14,803 | 14,803 | `fc_scan_string` 3,291 |
| Explicit-index Fairy `unit.obj` | 81 | 32,116 | 32,116 | `fy_fn_29` 5,297 |
| Explicit-index Fairy `runtime.obj` | 59 | 8,039 | 8,039 | `fy_private_vector_reserve_v0` 585 |
| Private vector-loop Fairy `unit.obj` | 81 | 32,116 | 32,116 | `fy_fn_29` 5,297 |
| Private vector-loop Fairy `runtime.obj` | 59 | 8,039 | 8,039 | `fy_private_vector_reserve_v0` 585 |

The next four largest Fairy `unit.obj` spans are `fy_fn_41` (3,134),
`fy_fn_10` (3,118), `fy_fn_36` (3,071), and `fy_fn_28` (2,495).
Those five starts cover 17,115 of 32,116 raw code-section bytes in each
Fairy unit object. The two source forms have identical object-level function
counts and five largest spans in this build. Their top-20 lists differ at one
318-byte tie. This is a candidate list for source and link-map attribution,
not evidence that any particular function is wasteful.

A span runs from one function symbol start to the next start in the same COFF
section, or to the section end. It can include alignment and other non-function
bytes. Full coverage means every code section has a function start at offset
zero; it does not establish exact instruction sizes or that the linker retained
every function. Object totals must not be added into the final PE `.text`
difference. Mapping `fy_fn_N` to source functions and a linker map remain the
next steps before any size optimization. The 8% C-relative runtime and EXE
gate, complete C scanner parity, and check/build cost remain open.
