# Authored compiler body-phase diagnosis, 2026-11-07

The earlier normal builds timed out. The later current-source optimized-seed
attempt completes successfully; see the final section. The intermediate
diagnostics below neither prove a fixed point nor attribute the earlier failure
solely to shared-host contention.

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

## Provider preparation and closure isolation

Two Sol High workers independently inspect interface/generic preparation and
body/name/call preparation; the parent owns profiling and product integration.
The test worker subsequently owns only the new entry-lookup fixture and Go
harness. Neither worker edits production sources, the lock, or this report.

The provider/factory/solver probe in
`.cache/current-origin-provider-split-20261107` builds in 154.701s and succeeds
with unchanged source/image hashes. Its 35 providers spend 246.929ms binding,
11000.325ms preparing closures, and 536.759ms checking values. Factory provider
discovery/preparation takes 11807.861ms; storage/statics, allocation and liveness
take 2.305ms, 0.486ms and 4.779ms. The origin solver's input factory takes
11815.694ms, validation 5.353ms and inference/final checks only 15.867ms.
Optimizing solver leaf construction is therefore not the measured priority.

The next diagnostic, `.cache/current-closure-stage-split-20261107`, adds markers
only around closure collection stages. It builds in 132.641s and executes the
same frozen 646 inputs and [445,450) range successfully in 16.673s. Source and
image hashes remain unchanged. Its 35 providers perform 70 collection passes:

| Provider closure operation | Total wall time |
| --- | --- |
| Interfaces | 6504.853ms |
| Concrete bodies | 4477.031ms |
| Concrete names | 504.999ms |
| Ordinary names | 927.265ms |
| Calls | 556.516ms |
| Drops | 43.612ms |

These successive probes ran under different host conditions. Their different
absolute times are not an optimization comparison. All are private diagnostic
probes, not normal CLI acceptance.

## Entry lookup candidate measured and rejected

`module_unit_body_ast::mapping` visits every source entry for every projected
body. `syntax::entry_at` searches from the start for each logical ID.
For an ordered arena with E entries and B projections, that mapping alone makes
B*E*(E+1)/2 ID comparisons. The proposed direct-position path makes B*E comparisons;
ordered owner publication verifies the ID sequence once in O(E), without an
allocation or work charge. The 408-entry/24-function fixture therefore changes
the mapping count from 2002464 to 9792 comparisons. This is a structural count,
not a measured instruction count or whole-compiler speedup.

In the diagnostic variant, direct access is guarded by a private proof flag: successful parser results
have a unique 1..len ID permutation; ordered assemblers prove that property
with one scan; copies retain the proof. Nested initializers can reserve the
outer ID before appending an inner entry, so position mismatches retain the
original search. Arbitrary assembled duplicates, gaps and permutations also
retain the original search and first-match behavior. Bounds assertions, record
order, all Entry fields, work charges and error precedence are unchanged.

The first flag-only diagnostic left every assembler untrusted and regressed
the sampled range wall median by 34.4%. Its four observations and matching work
are retained in `.cache/current-closure-stage-split-20261107/ab-summary.json`;
that implementation is not accepted as a performance improvement. The second
variant additionally verifies ordered publication. The regression does
not isolate bool-copy overhead or establish a sole cause.

The native fixture checks ordered and nested source records, projected origins,
exact work 36/40 and 37200, exact/one-over work limits, invalid-declaration error
precedence, duplicate first-match behavior and copied/ordered assembled owners.
With the second variant it passes both Windows profiles (low 3.15s, optimized
7.72s; package 12.906s). The copied-closure syntax/call regressions also pass in
69.427s. The original
fixture draft omitted return entries and failed before any product edit; the
corrected baseline passes before the implementation is applied.

The second variant's optimized probe builds in 130.556s. A separate four-run
before/after/before/after comparison is retained in
`.cache/current-entry-ordered-20261107/ab-summary.json` and its raw outputs.
Both images read the same 646 source byte sequences in the same order; only
the frozen syntax source's path is redirected to its unchanged private copy.
All runs exit 0 with identical range work and body counts. The diagnostic
variant differs from the proposed product source only by the test markers and
an explanatory field comment; it is not normal-product acceptance.

| Same-input diagnostic median | Original | Ordered-proof variant |
| --- | --- | --- |
| Entire checked range | 12850.870ms | 13006.410ms |
| Range process CPU | 12718.750ms | 12929.688ms |
| Ordinary-name preparation | 810.982ms | 719.235ms |

There is no demonstrated whole-range improvement: wall time increases 1.21%
and process CPU 1.66% in this small sample. The ordinary-name reduction does
not justify claiming a full compiler speedup. Both production variants are
discarded; `syntax.fy` and the lock are restored to their original bytes.
Only the regression fixture and measured development evidence are retained.
After restoration the complete new native fixture passes again on low 3.53s
and optimized 7.17s (package 12.675s); 750 source/test/lock file hashes match
before and after execution. The restored syntax source SHA-256 is
`afc3e2aa3be27d8debc8f691525c4f6151d8d47ec09297139491bfa3ab3ff91c`.
The transient variants' complete source checks pass (755 files, 9542 MIR
functions, final check/locked 13.905s/12.856s), but do not establish a current
normal self-build.

## Current-source normal release build

At `d0b84329`, a fresh Go-emitted optimized full-CLI seed passes
`TestCandidateInlineLocalsFullCLI/optimized`: 205 C parts and 14 independent
CLI/cache/trap processes, with six exact Go-matching test reports. The wrapper
takes 175.735s. Evidence is under
`.cache/current-source-optimized-seed-20261107`.

The subsequent normal, uninstrumented build in
`.cache/current-selfbuild-d0b84329-optimized-20261107` completes with:

- 646 current sources and a freshly frozen version module;
- 512.243s authored frontend, exit 0, empty stderr, 55063498 C packet bytes;
- 633.650s release C compile/link, exit 0 and empty stderr;
- successful version equality, source/tool/image hashes and 16MiB PE stack;
- `passed=true`, full `file_cli::run`, no command/compiler-only scope, no frontend
  reuse, no Go frontend, no runtime Go builder and no product timeout change.

The packet SHA-256 is
`351f2170ccdfa98d5497ab9757efa649d04ec371077a3851d2850ed3a5a8e753`;
the release image SHA-256 is
`8a5e2e0d0522dc12a6c0c90c54c5d0b5efd85ce50131bf20aa9050a93128cc40`.
The authenticated Fairy companion/runtime assets are reused only after all 36
Fairy and 23 runtime source hashes match. Source trees are clean before and
after the 1151.378s wrapper. Each frontend/build stage retains its 900s bound;
this is not a claim that the complete wrapper takes less than 900s.

Host CPU samples vary from 10% to 80%, so this successful run does not establish
that host contention alone explains the earlier failures. The previous Go seed
was also optimized; this is not a low-to-optimized profile comparison. Neither
rejected entry-lookup variant contributes to this successful product build.

## Remaining work

Run the native/companion fixed points and current Go/Fairy comparisons against
the successful release receipt, using disjoint output directories. Do not infer
throughput from a structural micro-optimization that did not improve the range.
The SDK layout index has a finite ownership graph and no newly detected cleanup
cycle; its retention has not been proved to cause the observed delay. Opaque
MSVC copy barriers remain unchanged. Normal release self-build now passes;
native and companion fixed points and full Go/Fairy behavior comparisons remain
incomplete. Linux remains batched; CI and push are unexecuted.

Public API/SDK ABI, DB, dependencies, repository hygiene and runners are
unchanged. This changes private verification tooling and development evidence,
not a language, target or runtime contract.
