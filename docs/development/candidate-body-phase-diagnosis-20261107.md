# Authored compiler body-phase diagnosis, 2026-11-07

The latest-source normal builds remain unsuccessful. A new diagnostic isolates
where their next investigation should focus; it does not replace a normal build,
prove a fixed point, or attribute the failure solely to shared-host contention.

## Exact-input phase run

`.cache/current-full-phase-probe-20261107` contains the preparation script,
test-only Go overlay, image, provenance, run script and complete output. The
overlay adds a C logger to the authored project dispatcher, per-range compiler
and C emitter. Product sources and work/deadline bounds are unchanged. The
private probe imports the project compiler instead of the full CLI dispatcher.
It must not be presented as full CLI acceptance.

Its Go-emitted optimized Windows image builds in 143.724s with two Go/C workers
allowed. It reads the exact 646-source manifest from
`.cache/current-selfbuild-layout-go-seed-20261107`; all source byte hashes,
manifest, image and original overlay-source hashes match before and after the
run. The version text remains the frozen 7237dd67 text, not a later Git HEAD.

The logger's MSVC CRT clock uses 1000 ticks/second. `run.json` and
`phase-summary.json` retain the raw events and process CPU samples.

| Observed phase | Time in this diagnostic run |
| --- | --- |
| Load, scan and parse 646 inputs | 1.296s |
| Build project signatures | 2.120s |
| Body checking and IR construction | 860.413s |
| Link checked units | 27.505s |
| C emission entered | 891.334s after start |
| Watchdog ends process | 900.012s wall / 786.453s process CPU |

All 646 inputs reach a completed checked range: 153 ranges succeed, with 88
capacity failures causing the existing recursive retry/split. There is no
observed frozen loop. Native exit is 1 after the watchdog kills it; stdout is
empty because bundle publication is transactional. The run script itself exits
0 after saving diagnostics and checking hashes; that is **not** compiler success.
No build receipt, emitted packet, fixed point or parity pass is produced.

The slowest completed range, inputs [445,450), starts at `project_cache_keys`
and takes 23.962s. This is a useful bounded next phase sample. C preparation's
first internal marker is never reached before timeout, so that run cannot
identify a particular C-emitter guard or total C-emission cost. High host load
was observed; no matched low-load baseline or exact slowdown factor is claimed.

## Constant-plan candidate measured and deferred

Static analysis finds that value checking and borrow-provider value checking
rebuild the project constant plan. A minimal optimized native probe in
`.cache/parallel-constant-cost-20261107` parses the same 646 inputs once, builds
signatures once and measures three calls with QPC and process CPU accounting.
All input/source/image checks pass, the stack reserve is checked as 16MiB, and
the same completed work charge is 12733 on every call.

The removable computation median is only **1.6228ms**, separately from
**5.2553ms** source/owner validation. It visits 48978 declarations for 12729
parsed declarations per completed build. At least 153 primary invocations are
known; failed-range and provider invocation counts have not been measured.
The upper structural bound is not an observed count. It would take 6163
redundant computations at that median to save ten seconds. No product cache is
implemented on this evidence: source/owner validation, per-unit work histories
and evaluator limits must remain intact.

The first probe build failed on unsupported casts in the test-only main, after
unnecessarily rebuilding a cold Go cache. Its failed log is retained. The
corrected diagnostic uses existing shared Go caches and passes in 93.962s
overall (51.836s package time); this is not a product-source repair.

## Single checked-range phase isolation

`.cache/parallel-body-stage-cost-20261107` records a single [445,450) range
under an optimized private Windows probe. It loads the same frozen 646 inputs
and adds markers without changing gate order, diagnostics, work or limits.
The shared-cache build takes 221.316s overall (184.029s package time); the native
run takes 26.304s including parse/signatures and completes within its 90s bound.
All source, original overlay-source and image hashes match after execution.

| Range operation | Wall time |
| --- | --- |
| Binding | 20.463ms |
| Closure preparation | 853.152ms |
| Primary value checking | 45.575ms |
| Moves and diagnostics | 10.741ms |
| Origins, including provider checking | **21869.924ms** |
| Origin diagnostics | 0.124ms |
| Cleanup policy | 2.441ms |
| Result flow | 3.174ms |
| Borrow capture and source stamps | 0.242ms |
| Checked return/destruction, lowering and checked-plan destruction | 79.676ms |
| Entire checked range | **22885.588ms** |

Origins accounts for 95.56% of that range, with 19375ms process CPU. Its
constant-build counter increases from 1 to 36: one primary build precedes it,
and 35 distinct provider modules construct value plans inside it. This is not
35 frames or an isolated measurement of provider preparation. The shared
factory and convergence/liveness costs are not separated yet; no cross-range
provider cache is implemented on this observation alone.

All gates succeed and origin diagnostics are empty. Values work is 631959,
move work 96587, origin work 675838, cleanup-policy work 196705 and final IR work
423711. These are independent histories, not additive work. Short CPU samples
are quantized to 15.625ms; a zero CPU delta does not mean no computation.

## Failed invocation evidence preservation

`tools/build-candidate-native-probe.py` now uses
`candidate_native_probe_invocation.run_invocation`. A timed-out/failed command
retains its invocation row and raw binary stdout/stderr before propagating the
original exception. Diagnostic-write failure adds a note without replacing that
exception. Successful invocation records retain the previous exact schema;
timeouts and stderr failures still cannot publish a successful receipt.

Nine focused Python tests pass. A real Windows child writes NUL/0xff stdout
and stderr, times out after two seconds, and is killed/waited by
`subprocess.run`; the original timeout and exact output bytes are preserved.
This improves failed-run evidence, not compiler throughput or acceptance.

## Remaining work

Separate provider preparation from the remaining origin factory/solver work
inside the measured slow range before selecting a compiler change.
The SDK layout index has a finite ownership graph and no newly detected cleanup
cycle; its retention has not been proved to cause the observed delay. Opaque
MSVC copy barriers remain unchanged. Latest-source normal self-build, native
and companion fixed points, and full Go/Fairy behavior comparisons are still
incomplete. Linux remains batched; CI and push are unexecuted.

Public API/SDK ABI, DB, dependencies, repository hygiene and runners are
unchanged. This changes private verification tooling and development evidence,
not a language, target or runtime contract.
