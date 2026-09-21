# Isolated MIR reuse experiment

Environment: 2026-09-22, Windows AMD64, Go 1.26.4, MSVC 14.51.36231. Source baseline `0465013c0ce5069895cab6a6e39c17611cddda8b`; current implementation/harness hashes verified and recorded in evidence.

Change: `LowerBoundaryPatch` now uses the existing `mir.Clone` of verified baseline MIR instead of re-lowering baseline HIR, after the same caller-coverage and edited-function validation as the relowering control. An independent deep copy is essential to avoid consumer mutations corrupting baseline. Both alternatives use the same checked MIR in tests.

Isolation: new exhaustive scalar-leaf mutation on actual scanner MIR confirms `result->baseline` and `baseline->result` isolation; second caller unaffected.

Timing separation: `MeasureBoundaryPatch` is the optional measurement entry recording stages, separating coverage, validation, and materialization. Coverage includes header indexes, source resolutions, and MIR call coverage. Its args are baseline, patch, and reuse bool. Ordinary `LowerBoundaryPatch` has no clock reads.

Native measurement: 18 runs — scalar/control HIR relowering vs. clone reuse × 2 profiles × 3 rotated reps — all exit 0 with empty outputs. Command `fairyc_nominal_scanner_measure`: 31.285s.

Extra annotation+boundary time, ms:

- Low: relower 10.166–13.992, median 11.006; reuse 3.915–5.149, median 5.001.
- Optimized: relower 9.179–10.723, median 10.720; reuse 3.014–4.856, median 4.652.
- Materialization medians: low relower 9.983, clone 3.634; optimized 9.095, clone 3.319.

Coverage/validation raw phase numbers include zeros; this does not imply free work because of timer resolution, short intervals, and small samples. Extra-overhead reduction is about 55–57%, not total compile speed. About 3–5 ms extra remains; ambient caches have only 3 samples/profile.

Artifacts: all C output 344,257 bytes with same SHA256. Low EXE 203,776, text 150,528, unit.obj 108,671; optimized EXE 138,240, text 85,296, unit.obj 69,021. This is not a binary-identity claim.

Scope: no source-syntax/stable-CLI integration; no full-suite rerun; no runtime-speed, Linux, PDB/debug, or cache measurements. Existing historical evidence remains untouched. New evidence: [raw measurements](evidence/nominal-scanner-reuse-20260922.json).
