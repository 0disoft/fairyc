# Parallel Windows acceptance follow-up, 2026-11-07

Two GPT 6.1 Sol High workers prepared disjoint native regression and emitter
changes. The integrated implementation commits are `b4008e0e` (checked CFG drop
projection) and `7237dd67` (shared enum/switch layout lookup). Two subsequent
read-only/microbenchmark workers investigated the longer full-source retry.
Their work did not change production source. All four workers are closed.

## Completed evidence

- Source `check` and `check --locked`: 755 files, 9541 MIR functions,
  14.270s / 13.959s on the combined tree; exact source bytes and lock integrated.
- Drop-projection regression: Windows low/optimized 2/2, with independently
  expected reachable sets, cleanup order, repeated calls/cycles and work bounds.
- Layout-lookup regression: both profiles pass separately and again on the
  combined tree (32.064s). All 2048 queries match an independent scanner,
  reducing 4194304 scan visits to 2048 index charges. C bytes and error/budget
  behavior match the independent expectations.
- The normal build at **b4008e0e**, before layout-lookup integration, passes
  full authored CLI development and release. Its receipt is
  `.cache/current-selfbuild-reach-drop-20261107/receipt.json`; fresh emission
  takes 522.053s and native builds 48.709s / 663.291s. This serial native build
  is not a parallel throughput benchmark.

## Latest-source full builds did not complete

Both attempts below compile revision **7237dd67** with the unchanged 900-second
frontend watchdog in `tools/build-candidate-native-probe.py`. The implementation
of both seed images is based on b4008e0e; their compilation routes differ.

| Seed image | Attempt directory under `.cache/` | Result |
| --- | --- | --- |
| Fairy-emitted native compiler | `current-selfbuild-layout-index-20261107` | Frontend timed out at 900s; no packet or successful receipt published |
| Go-emitted C compiler seed | `current-selfbuild-layout-go-seed-20261107` | Frontend timed out at 900s; no packet or successful receipt published |

The 646 source byte sequences, order, lock and frozen version text are identical
between attempts. Only generated main/version source paths differ, at rows 0
and 583. The complete snapshots and timeout tracebacks are retained in those
directories and their adjacent logs. Each attempt has a diagnostic-only
`failure.json`; neither grants acceptance. Child partial stdout/stderr was
not retained by the helper on timeout, so no zero-output byte count is asserted.

After the attempts, host CPU samples are 100% at 09:47 UTC and 91% at 09:48 UTC,
with approximately 9.8/10.2 GB available memory. An unrelated repository was
also compiling. Those observations do not prove contention caused the failures;
source changes, generated code and host load have not been isolated sufficiently
to assign a slowdown factor. No third full retry is started under that load.

## Bounded runtime investigation

The genuine Fairy microbenchmark is at
`.cache/parallel-selfhost-microbench-20261107/REPORT.md`, with exact commands,
exit codes, input/image hashes and `runs.json`. The Go reference is the frozen
Go compiler for revision 7237dd67; the Fairy candidate is the b4008e0e native
image. This compares the target programs emitted by those compilers, not the
two full-build seed images above.

Both release builds succeed (7.723s / 18.567s). Each runtime path performs five
million iterations in four interleaved runs. All 24 observations exit 0 and
print checksum 2928, with 7.52s total measured execution time.

| Path | Go-emitted median | Fairy-emitted median | Ratio |
| --- | --- | --- | --- |
| Scalar | 284.7ms | 256.7ms | 0.90 |
| Vec len/at | 334.0ms | 351.7ms | 1.05 |
| POD getter | 305.7ms | 372.6ms | 1.22 |

Variation is substantial and unused POD fields can be optimized away. These
results neither reproduce a major loop failure nor locate the full-compiler
timeout. Static inspection finds 60844 `fy_unit_copy` call sites in the prior
81-body packet, including opaque eight-byte field reads. That is a static count,
not a measured dynamic hotspot. The copy is an existing MSVC aliasing workaround;
it has not been removed or narrowed without a regression-backed measurement.
The inspected Vec address calculation and two bounded loops appear consistent,
which does not rule out a separate data-dependent progress defect.

## Remaining acceptance

The latest-source normal build, native/companion fixed points and full current
Go/Fairy comparisons remain incomplete. A successful b4008e0e build is not a
successful 7237dd67 build. Recheck host conditions before another expensive
normal attempt; keep the default failure and do not convert diagnostic runs
into acceptance by changing a watchdog or work bound.

Linux native checks stay batched. No CI, push, deployment or publishing occurs.
Public API/SDK ABI, DB, dependencies, repository hygiene and runners are unchanged.
Only private emitter implementation, focused tests, the lock and the documented
compilation-unit contract changed in the implementation commits.

## Later current-source verification

The later `d0b84329` source checkpoint passes normal full-dispatcher release
construction: authored frontend 512.243s and release native compile/link
633.650s, each within its existing 900s stage watchdog. The subsequent native
fixed point passes with 821.298s frontend and 632.728s fresh native build;
both complete C packets and optimized PE images are exactly equal, with no
excluded image bytes. The authored companion passes three-generation equality
and scratch cleanup. This supersedes the pending Windows release/fixed-point
state above; it does not erase either earlier timeout.

Five Go/Fairy suites execute 342 exact-matching comparisons with no workspace
native-contract failures. Four observations remain explicitly excluded;
therefore only the two CLI suite reports have unqualified `parity=true`.
The unsuccessful entry-lookup optimization was discarded, and only its
regression fixture/evidence is retained. Both Sol High workers and all observed
verification processes have exited without conflicting production edits.

See `current-compiler-acceptance-route-20261107.md` for current receipt paths,
checksums, timings, scope and exclusion reasons. Linux, clean-machine execution,
development self-build, full `go test ./...`, CI and remote actions remain
outside this completed local release slice.
