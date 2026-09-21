# Function-scoped nominal scanner experiment

Date: 2026-09-22, Windows AMD64, Go 1.26.4, MSVC 14.51.36231. Baseline `76bbc5479c2a4b35d1469293cde30e5c4283d531` plus new patch; hashes in evidence.

4 variants — scalar, struct, nominal full-closure, nominal-patch — across low/optimized profiles, 3 rotated repetitions, 24 successful builds/executions. `fairyc_nominal_scanner_measure` passed in 46.590s.

All nominal variants produced exactly the same C SHA256 as scalar. Scalar C: 344,257 bytes; low EXE 203,776, text 150,528, unit.obj 108,671; optimized EXE 138,240, text 85,296, unit.obj 69,021. Struct C: 345,271; low EXE 204,288, text 150,624, unit.obj 108,767; optimized EXE 138,240, text 85,328, unit.obj 69,200.

Implementation patches exactly 2 functions, `token_info` and `inspect`, copying headers/body only there. It validates source-resolution coverage, including uses eliminated from MIR, and MIR callers. It rejects omitted callers/callee, raw/wrong-domain args, missing source resolutions, and semantic changes, and verifies exact edited-function erasure in the implementation.

Whole-MIR comparison is only a test oracle for the new path; baseline HIR is still ordinarily lowered for fresh MIR. A small reflective walk remains inside edited functions; there is no whole-HIR deep clone/traversal. Indexes scan headers, resolutions, and MIR call blocks.

Annotation+boundary median, low: full 175.523ms, patch 12.549ms; optimized: full 179.078ms, patch 10.629ms. Ranges, low: full 161.819–176.829ms, patch 9.184–13.195ms; optimized: full 157.308–254.355ms, patch 8.512–13.650ms. Reduction is ~93–94% of extra experimental work, not total compile time. Patch annotation recorded 0ns in 4 samples and 0.511/1.242ms in others; zero is timer granularity, not proof work is free. Boundary includes coverage and lowering; JSON cloning is no longer on the patch path. Ordinary check and native compilation times are separate raw data.

Evidence: [raw measurements](evidence/nominal-scanner-patch-20260922.json). Original full-closure evidence report unchanged.

No new syntax/stable CLI integration, no runtime/Linux/debug/cache measurement, and no full-suite rerun because this was an isolated experiment. No speed-no-regression claim: ~9–14ms remains added, with only 3 samples/group under ambient caches. No exact binary-identity claim. Next admission remains deferred.
