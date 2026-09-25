# Reuse the decimal normalization buffer

Date: 2026-09-25. Baseline child HEAD: `129a628d`.
`numbers::decimal` formerly copied underscore-free source into `clean`,
then allocated a second `significant` Buffer to omit the decimal point.
The candidate validates the exponent while `clean` is borrowed read-only,
then compacts mantissa digits leftward in the same initialized Buffer.
The number of written digits never exceeds the current read index. Output,
work charging, size limits and cleanup continue through the existing paths.
The removed allocation may change the number at which an injected allocator
fails; exact allocation-call equivalence is not claimed.

The configured 38-input C/Fairy lexical-parity check, candidate decimal
boundary check, source-work check, combined-limit check and Fairy first-
allocation-failure check passed. The optimized four-image smoke verified
reporter output, 1,000 completed rounds and checksums. Its
[receipt](evidence/scanner-decimal-buffer-reuse-smoke-20260925.json) includes
the working-tree source hash, image hashes, sizes and selected COFF spans.

| Image | Baseline EXE bytes | After EXE bytes | After generated/source bytes |
| --- | ---: | ---: | ---: |
| Independent C | 184,320 | 184,320 | 52,164 |
| Geometric C | 185,344 | 185,344 | 52,245 |
| Explicit-index Fairy | 202,240 | 201,728 | 395,860 |
| Private vector-loop Fairy | 202,240 | 201,728 | 396,700 |

The two Fairy EXEs each shrink by 512 bytes. The `numbers::decimal` raw
`unit.obj` span drops from 2,869 to 2,313 bytes, and total Fairy unit raw
code falls from 27,548/27,549 to 26,992/26,993 bytes for indexed/vector
forms. Both still have 21 specializations. The smaller C image gives a
Fairy/C EXE ratio of 1.09444; the 8% ceiling is 198,656 bytes, leaving
3,072 bytes to remove. Raw COFF span reduction is not a final-image
function attribution. Stable runtime, check/build cost, every allocator
failure point and complete C/Fairy scanner parity remain unproven; this
change alone does not admit the vector source form.
