# Package cache measurements (Phase 3)

Measured 2026-10-21 at clean commit `b2181e660a78555bc2066ae640d3e80d362d29ab`, Windows 11 Pro build
26300, amd64, Ryzen 5 7430U (6c/12t), 33,654,575,104 bytes RAM, go1.27.1, CGO disabled;
observed clock resolution 518,100 ns.

Verified-MIR package cache measurements on the two-package synthetic `workspace_quickstart` workspace
(65 samples). They do not measure native C object or executable build times, runtime or compiler
throughput, or large-project scaling. Fixture preparation, edit-suite warmup, and the standalone
compiler build are outside the recorded durations; no speedup is claimed at this workload size.

| Suite | Condition | Median | Hits | Misses | Samples |
| --- | --- | ---: | ---: | ---: | ---: |
| In-process | cold fallback | 4.8272 ms | 0 | 2 | 1 |
| In-process | warm all-hit | Below resolution | 2 | 0 | 1 |
| In-process edit | no-op | 1.6179 ms | 2 | 0 | 9 |
| In-process edit | private root | 1.6190 ms | 1 | 1 | 9 |
| In-process edit | private dependency | 1.0978 ms | 1 | 1 | 9 |
| In-process edit | public dependency | 1.2779 ms | 0 | 2 | 9 |
| Persistent | cold fallback | 164.2981 ms | 0 | 2 | 9 |
| Persistent | warm all-hit | 143.4814 ms | 2 | 0 | 9 |

Medians for n=9; single-observation rows show the measured duration. Hits and misses are per sample.
The warm in-process sample reported 0 ns and was marked below clock resolution; treat it as censored.
Persistent rows retained 2 entries and 13,152 bytes throughout.

Retention (entries/bytes): 2/13,152, 3/17,530, 4/21,908, 5/30,722, then 6/41,504 for steps 5–9; evictions 0, 0, 0, 0, 1, 0, 1, 0, 0 (total 2); record rejections 0.

In-process samples execute the interpreter (42 each). Persistent rows time the caller process through
exit, including workspace and cache I/O; they require verified MIR via the cache CLI and do not execute
the interpreter. A 1-hit/1-miss row means one hit and one miss; fallback may lower the full program.

Retained bytes are logical cache record bytes; physical disk, temp files, and compiler outputs are not
measured. The Phase 3 inline-body exit condition remains unresolved.

Raw: [process.json](process.json) · [edits.json](edits.json) · [persistent.json](persistent.json) · [growth.json](growth.json) · [environment.json](environment.json)

Reproduce from the repository root in PowerShell:

```powershell
$env:CGO_ENABLED = "0"
go run -mod=readonly ./tools/dev/fairyc-windows-host-smoke --cache-measurements-only
```
