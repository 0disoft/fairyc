# Result direct-return adoption comparison

Date: 2026-09-24. The [lexer-candidate diagnostics module](../../examples/lexer_candidate/src/diagnostics.fy)
now binds the success payload from `bytes::copy` and `Vec.push` outside their
`switch` blocks and returns directly on failure. The previous ADR 0398 form
kept the success continuation inside nested branches. Both were built from
the same workspace with only that function body changed. The
[raw comparison](evidence/result-return-adoption-20260924.json) records exact
source hashes and every sample.

`TestResultBranchReviewReturnAdoptionCost` ran on Windows AMD64 with Go 1.26.4,
`GOMAXPROCS=1`, and MSVC 14.51.36231. Five alternating previous/current
frontend pairs followed one warmup pair. A pair measures workspace loading,
verified compilation, and C emission in the same Go process; it does not
include an MSVC build. The native comparison separately built each source in
low and optimized profiles. Four fixed parser inputs produced matching
declaration and diagnostic output across all four builds. The focused native
test also retained success, copy-allocation-failure, push-allocation-failure,
and limit behavior in both profiles, with no live heap blocks.

| Observation | Previous form | Direct-return form |
| --- | ---: | ---: |
| Median workspace lower duration | 966.812 ms | 949.516 ms |
| Median C emission duration | 520.933 ms | 537.070 ms |
| Generated C bytes | 1,114,718 | 1,114,970 |
| MIR functions | 407 | 407 |
| Low-profile EXE bytes | 410,624 | 411,136 |
| Optimized EXE bytes | 221,696 | 221,696 |
| Low-profile unit `.text` bytes | 286,078 | 286,142 |
| Optimized unit `.text` bytes | 109,349 | 109,285 |

The median paired duration ratios are 0.997 for workspace lowering and 1.044
for C emission. The five emission ratios range from 1.021 to 1.143, so these
samples do not establish a stable emission-speed change. Generated C grew by
252 bytes; the low-profile EXE grew by 512 bytes (about 0.125%), while the
optimized EXE stayed the same size. These are same-program source-form
comparisons, not Fairy-versus-C measurements. The complete MSVC build and
runtime speed were not timed here. Ambient OS caches and concurrent machine
load were uncontrolled. The cumulative C-relative 8% budget therefore remains
unverified for this feature; the earlier [vector build comparison](c-budget-vector-sdk-reference-cache-20260924.md)
still reports a build ratio above 1.08 for its separate workload.
