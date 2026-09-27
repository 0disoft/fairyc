# Scanner generated-C source attribution

Date: 2026-09-27. Child source HEAD before this diagnostic: `0fe126e6`.
The opt-in compiler test emits the real indexed scanner and the private
three-loop vector candidate for Windows AMD64 C11, without invoking MSVC.
Its [receipt](evidence/scanner-generated-c-source-delta-20260927.json) has the
same indexed, candidate and benchmark-main source hashes as the
[current build-cost comparison](scanner-current-build-cost-20260927.md).

| Generated C span | Indexed bytes | Candidate bytes | Delta |
| --- | ---: | ---: | ---: |
| `scanner::report_work` | 12,043 | 12,139 | +96 |
| `scanner::storage` | 4,185 | 4,695 | +510 |
| SDK generic suffix, 21 functions | grouped | grouped | +227 |
| Preamble, prototypes and `main` | grouped | grouped | +7 |
| **Full C translation unit** | **369,719** | **370,559** | **+840** |

The first 59 emitted function names are identical and can be compared by
source name. The remaining 21 are SDK generic functions whose hashed names
change with the scanner source; the test groups their source lengths instead
of claiming that equal positions represent the same specialization. Both
forms have 80 emitted functions, 21 specializations, and equal foreign-symbol
and C-feature sets. The test checks that every byte of the full delta is
accounted for by those spans.

This locates 606 of the 840 extra **C text bytes** in the two scanner
functions containing the three rewritten loops. It does not establish that
their machine code grew, or that 227 SDK bytes imply new linked helpers.
The prior optimized EXEs are equal at 197,120 bytes. Build-time parity still
has no valid identical-source control on this host, and the cumulative
independent-C 8% admission gates remain separate. The iteration form stays
compiler-private.
