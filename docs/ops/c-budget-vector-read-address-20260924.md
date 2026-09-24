# Direct-address shared vector C comparison

Date: 2026-09-24. Exact Fairy C source:
`604329ef2d28a8dc64bdcaede25c85538525002b` (SDK `0.0.0-stage0.67`).
Windows AMD64, Go 1.26.4, MSVC 14.51.36231, `GOMAXPROCS=1`, optimized
C11 `/W4 /WX /MT /O2 /INCREMENTAL:NO`.

The [indexed](c-budget-vector-read-index-20260924.md) and compiler-private
[iteration candidate](c-budget-vector-read-candidate-20260924.md) use the same
unchanged Fairy workload sources and independent checked C source as before.
Only the authenticated SDK `Vec.at<T>` implementation changes: after the safe
`index < length` check, it derives a typed address and returns an owner-bound
shared borrow without calling the private C slot lookup. The C reference still
uses its own checked descriptor and does not link Fairy support. The old
runtime helper validates more descriptor fields per read than the new SDK
path; safe vector construction maintains those invariants, while malformed
unsafe descriptors are outside this equivalence claim.

Each workload ran one warmup pair, then five alternating C/Fairy build-and-run
pairs. Every lane passed the five boundary inputs, and all results matched the
independent Go recurrence. Disassembly of the optimized Fairy units contains
no call to `fy_private_vector_element_slot_v0`. The [indexed raw report](evidence/c-budget-vector-read-address-index-20260924.json)
and [candidate raw report](evidence/c-budget-vector-read-address-candidate-20260924.json)
contain all samples, source hashes, artifacts and phase-independent timings.
The C and support fixture hashes are unchanged across both before/after
comparisons. Ambient OS caches, CPU scheduling and frequency were uncontrolled.

| Workload | C kernel median | Fairy kernel median | Paired runtime median | EXE Fairy/C | Build Fairy/C |
| --- | ---: | ---: | ---: | ---: | ---: |
| Indexed | 1,267.1168 ms | 1,339.6690 ms | 1.05726 | 1.01824 | 1.23333 |
| Private candidate | 1,301.1311 ms | 1,326.5149 ms | 1.04905 | 1.01824 | 1.57946 |

Indexed paired ratios were 1.05830, 1.05726, 1.01883, 1.04176 and 1.08803.
Candidate paired ratios were 1.04905, 1.03508, 1.08950, 0.99804 and
1.06629. Both paired medians and both EXE ratios are below the cumulative
1.08 limit on these samples, but one pair in each workload exceeded it.
The earlier [indexed](c-budget-vector-read-index-20260924.md) and
[candidate](c-budget-vector-read-candidate-20260924.md) paired medians were
1.38544 and 1.37255. Their separate-run timing cannot quantify the isolated
effect of this SDK change. The removed per-element call is directly visible
in generated code and is a plausible major cause of the improvement.

Build time is an observation marker, not an 8% acceptance gate. The candidate
build ratio is especially variable. These are small four-element kernels;
they do not establish representative lexer performance, cold CLI startup,
Linux behavior, unused-feature overhead or stable source eligibility. A
prebuilt-artifact execution comparison and full lexer workload are still
needed before a promotion verdict.
