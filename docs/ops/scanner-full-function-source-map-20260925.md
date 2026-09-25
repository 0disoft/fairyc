# Full scanner COFF function-source map

Date: 2026-09-25. Compiler helper HEAD: `c079d737`.
The [full COFF source-map receipt](evidence/scanner-full-function-source-map-20260925.json)
covers every recorded function start in the optimized four-image scanner
smoke. The probe ran the independent C, geometric C, indexed Fairy, and
private vector-loop Fairy images with the existing 1,000-round output and
checksum checks. The first probe invocation completed the test and wrote the
receipt, then failed while summarizing an unnamed function. The corrected
contract revalidated that same receipt and completed successfully.

| Image / object | Raw code bytes | Function starts |
| --- | ---: | ---: |
| Independent C `probe.obj` | 13,653 | 45 |
| Geometric C `probe.obj` | 14,803 | 47 |
| Indexed Fairy `unit.obj` | 27,548 | 81 |
| Vector-loop Fairy `unit.obj` | 27,549 | 81 |
| Each Fairy `runtime.obj` | 8,039 | 59 |

Every raw code byte in the Fairy `unit.obj` and `runtime.obj` objects is
attributed to a recorded span. Each Fairy image has 21 `std::vec` function
spans totaling 1,654 raw bytes; the candidate loop adds no vector function
start or specialization. The one-byte unit raw-size difference appears in
`scanner::report_work` (2,397 indexed, 2,398 candidate). The largest common
unit spans are `scanner::scan` (4,083), `numbers::decimal` (2,869),
`strings::scan` (2,147), and `strings::escape` (1,791) raw bytes. These
numbers identify inspection targets, not retained final-image contributions.

The [preceding four-image size receipt](scanner-unified-payload-reporter-20260925.md)
measured both Fairy EXEs at 202,240 bytes and the smaller independent C EXE
at 184,320 bytes. Its C-relative size ratio is 1.09722, leaving 3,584 bytes
above the 1.08 ceiling. This new COFF report does not remeasure final EXE
bytes or stable runtime. COFF spans include alignment and functions later
discarded by the linker; the C and Fairy objects have different function
boundaries. In particular, the 1,654-byte vector total cannot be subtracted
from the final EXE gap. The next size experiment needs retained final-image
function ownership or a measured, semantics-preserving change to a common
scanner path. The 8% gate and stable source admission remain open.
