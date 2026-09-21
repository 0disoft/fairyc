# Nominal scanner cost experiment

Date: 2026-09-22. Host: Windows AMD64, Go 1.26.4, MSVC 14.51.36231.
Compiler baseline: `1fae6f0fa1388a0a70f3b8657acf8ccdfe72b6c1`.
Test: `fairyc_nominal_scanner_measure` passed in 36.123 seconds.

## Design

3 variants scalar/struct/nominal × 2 profiles low/optimized × 3 repetitions = 18 successful build/executions.

There is no source syntax yet. The experiment explicitly annotates compiler-verified HIR, validates nominal uses, then requires erased HIR/MIR equal the original.

Timing breakdown:

- `check_ns`: workspace load/lock plus existing check + MIR.
- `annotation_ns`: test JSON clone + annotations.
- `boundary_ns`: full reflective validation + erasure + MIR equality.
- `native_ns`: already checked MIR -> C/MSVC, excluding final EXE write/run, preceded by separate C inspection.

Nominal extra cost beyond ordinary check and native compilation: annotation generation 105.55–136.87 ms, validation/erasure 62.50–78.42 ms.

Variant order was rotated. Caches were ambient and there were only 3 samples, so no speedup claim is made.

## Results

Scalar and nominal C output: 344,257 bytes with same SHA256.

- Low: EXE 203,776, `.text` 150,528, unit.obj 108,671.
- Optimized: EXE 138,240, `.text` 85,296, unit.obj 69,021.

Struct C output: 345,271 bytes.

- Low: EXE 204,288, `.text` 150,624, unit.obj 108,767.
- Optimized: EXE 138,240, `.text` 85,328, unit.obj 69,200.

All scalar/nominal PE VirtualSize section values match. `runtime.obj` varies by 4 bytes across samples, so no binary identity claim is made.

All 18 runs exited 0 with no stdout/stderr. No PDB was requested. Runtime performance, Linux, and debug/cache size were not measured.

Evidence: [raw measurements](evidence/nominal-scanner-20260922.json).

## Decision

Measured generated-code size criterion met. Compilation no-regression is NOT met/proven; the prototype adds work. Stable feature admission remains deferred.
