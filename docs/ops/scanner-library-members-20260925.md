# Scanner linked library-member comparison

Date: 2026-09-25. Source HEAD before this comparison:
`e220f670f6a44375b2e9ed594d84dea23ac7b631`.
`TestPrivateVectorSDKCandidateScannerLibraryMembers` builds the same optimized
up-front C, explicit-index Fairy, and private vector-loop Fairy scanner source
with and without bounded MSVC `/VERBOSE` capture. Each pair has the same EXE
byte count, validated unwind interval boundaries, reporter, completed-round
count and checksum. QPC ticks are excluded from output equality. The test
normalizes only the basename of each `Loaded library(member)` record; no
temporary toolchain or workspace path is retained in this report.

| Image | EXE bytes | Selected unique library members | Linker log bytes |
| --- | ---: | ---: | ---: |
| Up-front C | 184,320 | 233 | 1,602,800 |
| Explicit-index Fairy | 206,336 | 232 | 1,608,264 |
| Private vector-loop Fairy | 206,336 | 232 | 1,608,264 |

The two Fairy member sets are identical. Against either Fairy image, the C
image alone selects `libucrt.lib(strlen.obj)`; the Fairy image selects no
additional member. Thus extra archive-member extraction does not explain
Fairy's 22,016-byte EXE excess. This is a member-set comparison, not a retained
code-size measurement: different references can retain different COMDATs from
the same member, and generated unit/runtime code differs from the C probe.
The 8% runtime and EXE gate and complete C scanner parity remain open.
