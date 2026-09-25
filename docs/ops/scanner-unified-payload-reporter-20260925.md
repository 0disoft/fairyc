# Scanner unified payload reporter

Date: 2026-09-25. Baseline child HEAD:
`f2b262dbd4779f87bda8be5e7701819463425b2f`.
The lexer candidate reporter now selects one immutable payload view: source
bytes for `true`, `false` and `null`, the cooked Buffer view when present, or
an empty view otherwise. One loop emits its byte values. This replaces the
separate indexed source-byte loop and cooked-byte loop without changing the
report format, scanner recognition, language, SDK or runtime behavior.

The configured 38-input C/Fairy lexical-parity test passed. The optimized
four-image smoke checked reporter output, 1,000 completed rounds and checksum
for each lane. Its [receipt](evidence/scanner-payload-reporter-smoke-20260925.json)
records image hashes, MSVC version, generated-source lengths and
specialization counts.

| Image | Before EXE bytes | After EXE bytes | After source bytes |
| --- | ---: | ---: | ---: |
| Up-front C | 184,320 | 184,320 | 52,164 |
| Geometric C | 185,344 | 185,344 | 52,245 |
| Explicit-index Fairy | 202,240 | 202,240 | 395,987 |
| Private vector-loop Fairy | 202,240 | 202,240 | 396,778 |

Generated C shrinks by 801 bytes for the indexed source and 820 bytes for
the private vector source, with 21 specializations in both. The final EXEs
do not shrink; the up-front C-relative size ratio stays 1.09722 and the
8% size ceiling still requires 3,584 fewer bytes. Generated-source size is
not a measured check/build-time result. The one-thousand-round QPC smoke is
not stable runtime evidence. Complete C scanner parity, unused-feature
overhead and the cumulative cost gates remain open; vector iteration stays
compiler-private.
