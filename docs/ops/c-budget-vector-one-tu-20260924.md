# Vector support translation-unit diagnostic

Date: 2026-09-24. Compiler baseline: `4d5ac983b7562a1f59067cd2eb4e2fbac12ab69e`
with the opt-in one-translation-unit harness branch in this commit.
Windows AMD64, Go 1.26.4, MSVC 14.51.36231, `GOMAXPROCS=1`, optimized C11
`/W4 /WX /MT /O2 /INCREMENTAL:NO`.

## Question and method

The original [checked vector-slot comparison](c-budget-vector-slot-20260924.md)
measured repeated `Vec<u32>.at_mut` at 1.44971 times the C runtime median.
Its generated C calls `fy_private_vector_element_slot_mut_v0` once per loop
iteration across a C translation-unit boundary. How much of that gap remains
if Fairy's generated unit and support source are compiled together?

This diagnostic keeps the committed `vector_slot.fy`, `vector_slot.c` and
`support.c` fixture bytes unchanged. Only the Fairy build path concatenates
its support and generated C into one MSVC translation unit using the existing
probe builder; C retains its independent two-unit build. Both lanes still
execute the same 50-million-update checksum and use the same input, QPC timer
and independent Go result oracle. The one-unit layout is a build experiment,
not a product compiler change or a same-layout build-time comparison.

After one build/run warmup per lane, five alternating pairs were recorded.
All 10 boundary executions passed. [Raw evidence](evidence/c-budget-vector-one-tu-20260924.json)
retains each sample, source and emitted-unit hashes, build times, PE sizes and
the `one_translation_unit` flag. No samples were discarded.

## Observations

| Metric | C median | Fairy median | Fairy/C |
| --- | ---: | ---: | ---: |
| Kernel | 327.921 ms | 376.574 ms | 1.14837 |
| EXE | 167,936 B | 170,496 B | 1.01524 |
| Build | 528.828 ms | 704.918 ms | 1.33298 |

The median of the five paired runtime ratios is 1.15158; the range is
1.14213–1.17937. The EXE is 2,560 bytes larger (+1.524%). Build time is
shown only as an observation because the lanes intentionally use different
translation-unit layouts, and warmed toolchain and OS cache behavior remains
uncontrolled.

The separate-unit [baseline](c-budget-vector-slot-20260924.md) was measured in
a different run. Its larger gap and the optimized object showing a per-iteration
foreign call make the translation boundary a plausible contributor, but this
experiment alone does not isolate the exact instruction cost. Combining units
still leaves every measured runtime pair above the cumulative 1.08 budget.
Source-level borrowing of a fixed slot once before a loop was also tried as a
diagnostic and removed the repeated lookup from the timed loop; that changed
the measured workload, so it is not used as evidence that repeated `at_mut`
meets the budget. The remaining work is to reduce the repeated lookup cost
without weakening bounds, owner provenance, or descriptor validity.
