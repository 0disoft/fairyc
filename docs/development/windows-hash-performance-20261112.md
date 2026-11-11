# Fairy Hash SHA-256 performance, 2026-11-12

Scope: retain file/hash/UI policy in Fairy C. No native hashing library or C hash
implementation. Preserve SHA-256 reference output, progress and cancellation.

## Baseline and component measurement

Retained the 187,392-byte authored release as
`.cache/windows-hash-performance-20261112/baseline.exe`, SHA-256
`f4c94b3303ea54914c341c5e163a5f847d54a2d7aa736302340fd8d1d13bb8af`,
with its original source alongside it. Compiler remains
`.cache/windows-gui-fairy-cli-20261109/low/fairyc.exe`, SHA-256
`e124d7a921f455f4c3b90ef5e7460959f2338844c930405929fa7b7c5ef157ea`.

`profile_costs.py` copies source/config/native files into an owned temporary
workspace, substitutes `profiling/main.fy`, and links `profiling/clock.c` only
there. The separate entry point calls the production fs/SHA/UI primitives, with
the same 64 KiB chunk and 100ms progress cadence. QPC brackets separate read,
SHA update, message poll/layout, progress and finalization intervals. This is
instrumented wall time, not CPU sampling or the entire production app/dialog
workflow. The production EXE is unchanged by measurement. The pure-memory case
reuses the repeating file's first 64 KiB block. All three hashing cases check
Python's SHA-256; read-only checks total bytes. Fixtures and processes are owned
and removed on completion.

Baseline command, 64 MiB materialized cached local file:

```powershell
python -B examples/win32_hash/profile_costs.py --compiler .cache/windows-gui-fairy-cli-20261109/low/fairyc.exe --source .cache/windows-hash-performance-20261112/baseline-source --mib 64
```

| Mode | Total inner seconds | Read | SHA | Poll/layout | Progress |
|---|---:|---:|---:|---:|---:|
| Read only | 0.024927 | 0.024857 | 0 | 0 | 0 |
| Reused-memory SHA | 12.082316 | 0 | 12.082057 | 0 | 0 |
| File + SHA | 13.351874 | 0.078824 | 13.272756 | 0 | 0 |
| File + SHA + native UI | 14.178003 | 0.072125 | 13.307323 | 0.424990 | 0.372201 |

The file+SHA run spends about 99.4% of its measured loop in SHA update. Native
UI contributes about 5.6% of the GUI loop; 1,024 polls and 129 progress updates
were observed. SHA is the first optimization target. Read-only throughput is
cached-memory/filesystem performance, not storage bandwidth. Wall intervals
include scheduling and concurrent host work; separate runs are not subtracted
to infer costs. Each case used the same Python reference
`281e519df3077b557c6b03f5da83c4e8d397219259615dd7c3308f89cae8f2a6`.
Receipt: `.cache/windows-hash-performance-20261112/baseline-components.json`.

## Fairy-only compression changes

The private word arrays now carry bounded 32-bit values in 64-bit `usize`.
Modular sums mask to 32 bits after at most five words (sum below 2^35); rotations
of a 32-bit word shift by at most 31 bits, fitting within 64-bit intermediates,
and mask before returning. The schedule's four-term sum and round's five-term
sum no longer need nested 16-bit limb additions. Ordinary arithmetic remains
checked. Stream-state validation rejects words above 0xFFFFFFFF, with a focused
test. File digest formatting uses the same representation. The maximum SHA bit
length guard and 64 KiB streaming buffer remain intact.

The schedule and constant table are borrowed once as views. The authored MSVC
backend intentionally protects aggregate owner reads with opaque copies
(`module_unit_c_function.fy`, project prologue). Repeated value-array access
therefore warrants attention. Views avoid those repeated whole-table reads
without weakening the compiler's aliasing protection or disabling checks. No
compiler/backend code was modified. Widening alone gave a single inconclusive
11.631s versus 10.892s pair; that preliminary candidate is not the final result.

## Paired production EXE timing and size

Both files were measured by the same script on the same 64 MiB cached local
fixture. Three pairs alternate baseline/current, current/baseline,
baseline/current. Every process output matches Python's complete report/digest;
there is no GUI observer or instrumentation in either measured production EXE.

```powershell
python -B examples/win32_hash/benchmark_cli.py --baseline .cache/windows-hash-performance-20261112/baseline.exe --pairs 3 --mib 64
```

| Pair | Baseline seconds | Optimized seconds | Speed ratio |
|---|---:|---:|---:|
| 1 | 12.435839 | 6.949838 | 1.789x |
| 2 | 14.117811 | 7.660969 | 1.843x |
| 3 | 16.662005 | 9.263725 | 1.799x |
| Median | **14.117811** | **7.660969** | **1.843x** |

Median elapsed time falls **45.7%**. Absolute times drift upward during these
runs; each pair still improves by 1.79–1.84x. These are host/cached-file results,
not disk bandwidth, a hardware-independent guarantee or a claim that the Fairy
implementation now matches hardware-accelerated SHA. Receipt:
`.cache/windows-hash-performance-20261112/final-pairs-64.json`.

| Authored release | Before | After |
|---|---:|---:|
| EXE bytes | 187,392 | 190,464 |
| KiB | 183 | 186 |

The cost is **3,072 bytes / 3 KiB**. Final SHA-256:
`2e606593333fa80fb6e4a395f295ef02bb6948357f1c97e4c8e8191a7ad13b1a`.
Imports remain USER32, GDI32, COMDLG32, COMCTL32, OLE32 and KERNEL32. No profiler,
hash library, font or verification runtime ships with the EXE. Size receipt:
`.cache/windows-hash-performance-20261112/final-size.json`.

## Final component and actual GUI checks

The same component command was rerun with current source (omit `--source`):

| Optimized mode | Total inner seconds | Read | SHA | Poll/layout | Progress |
|---|---:|---:|---:|---:|---:|
| Read only | 0.041901 | 0.041827 | 0 | 0 | 0 |
| Reused-memory SHA | 6.984742 | 0 | 6.984466 | 0 | 0 |
| File + SHA | 5.554687 | 0.056877 | 5.497562 | 0 | 0 |
| File + SHA + native UI | 5.778969 | 0.052372 | 5.474591 | 0.140320 | 0.111092 |

The three hashing modes still match the independent reference. These sequential
instrumented runs include scheduling variation; the memory-only result being
slower than the file+SHA result is not a measured storage benefit. Use the paired
production EXE comparison for the headline speedup. UI policy is unchanged:
1,024 polls, 53 actual progress updates in this run. Fewer progress updates follow
the shorter job; no separate UI-implementation speedup is claimed. Receipt:
`.cache/windows-hash-performance-20261112/final-components.json`.

Final authored-release validation:

```powershell
# From examples/win32_hash, using the same authored CLI:
fairyc check
fairyc build --locked --release
fairyc test --locked --file tests/sha256.fy --json
python -B verify_cli.py
python -B verify_sha.py
python -B verify_gui.py --job-only --job-mib 64
```

All **19** existing CLI/report/error cases and **24** independent differential
SHA cases pass. New cases include 55/56/63/64/65 and 119/120/127/128/129 padding
boundaries, 65,535/65,536/65,537-byte read boundaries, zero/FF data, one million
`a` bytes and seeded nonrepeating 1,048,833-byte input. The final interpreter
suite passes **3/3**, covering chunked updates, padding, maximum length and
out-of-range word rejection. Native vectors also exercise the MSVC view path.
Receipts: `final-cli.json`, `final-vectors.json`, `final-sha-unit.json`.

The actual production GUI's 64 MiB job completed in an observed **6.362695s**
with the correct digest, **28** coherent progress samples, **145** GUI-thread
pings, p95 **7.412ms**, maximum **26.596ms** and programmatic resize/layout
**32.801ms**. Two real native Cancel-button BM_CLICK operations returned to idle
in **26.271ms / 25.851ms**; the window stayed open, cancelled inputs could be
removed, partial results stayed unavailable and a subsequent `abc` calculation
succeeded in the same window. This observation has its own polling overhead and
excludes initial dialog/setup; it is separate from CLI throughput. The report is
explicitly job-only, `physicalInput:false`, `pixelInspection:false`. Receipt:
`.cache/windows-hash-performance-20261112/final-gui-job.json`.

Go frontend development/release integration also passes:

```powershell
$env:GOMAXPROCS='1'
$env:FAIRYC_RUN_DESKTOP_INTEGRATION='1'
$env:FAIRYC_HASH_JOB_ONLY='1'
Remove-Item Env:FAIRYC_HASH_USABILITY -ErrorAction SilentlyContinue
Remove-Item Env:FAIRYC_HASH_HIGH_CONTRAST -ErrorAction SilentlyContinue
go test -mod=readonly ./internal/cli -run '^TestWindowsHashUtility$' -count=1 -v -timeout 5m
```

PASS **33.771s**, two profile subtests. Each runs CLI 19 cases, differential SHA
24 cases and a 16 MiB actual native GUI job with progress, resize, two cancels and
reuse. Development is **277,504 bytes**, GUI observed busy **8.583s**, ping max
**37.223ms**, cancel **23.427/22.398ms**. Release is **173,056 bytes**, GUI observed
busy **0.332s**, ping max **11.885ms**, cancel **22.372/21.981ms**. These frontend,
profile and GUI-observer measurements are separate from the authored CLI timing;
they must not be substituted into the paired production comparison. Receipt:
`.cache/windows-hash-performance-20261112/final-go-integration.log`.

`git diff --check` passes. No public compiler/language/SDK API, DB, CI workflow,
runner or dependency policy changed. The example's local source lock was
refreshed; SDK identity/digest and compiler are unchanged. Only the example's
private word representation, verification, benchmark and execution documents
changed; no normative source-of-truth document changed. Build/cache output stays
ignored. No version bump, push, publication or deployment.

Remaining limits: synchronous OS reads still defer cancellation until they
return; concurrent same-size writes are not snapshots. No physical keyboard,
different-monitor DPI, screen-reader voice, high-contrast, desktop pixel or drag
suite was rerun for this SHA-only change. Those checks are unrelated to hash
arithmetic and retain the separately recorded environment constraints. Fresh
compiler bootstrap, general compiler suites and hosted CI were not run. The
CLI performance sample is a cached local file, not a cold disk/network workload.
