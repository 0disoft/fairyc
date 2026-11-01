# Windows backend review C baseline, 2026-11-01

- Decision: [ADR 0497](../adr/0497-backend-and-target-review.md).
- Harness/source revision: `769d5835ceac9b8c27f41d1130ea02cad28aa42f`; clean worktree at measurement start.
- Host: Windows amd64, go1.27.1, MSVC 14.51.36231.
- Compiler: Visual Studio 18 Community, VC/Tools/MSVC/14.51.36231/bin/Hostx64/x64/cl.exe.
- Compiler executable SHA-256: `dc8426b8760d92cf757df3d10b9f0244a95b454ff43194a58161568a0ec70d53`.
- Local raw report: `.cache/backend-study-final.json`, schema `fairyc.backend-study/v1`.
- Raw report SHA-256: `cd59363cd62bfab7e55c15b8daba05b822f27819392b7bc58d5ccfc15b8f1913`.

The report is an observation artifact, not a backend/target support contract. Its local JSON is
ignored; rerunning creates a new report with new timings and potentially different image hashes.
The committed harness and source revision own the replay inputs. No generated output owns design.

## Executed and passed

Seven workloads x low/optimized x three fresh builds and native executions = 42 executions.
Thirty-six normal returns are 42 with empty stdout/stderr; six checked-overflow executions carry
an independently decoded private integer-overflow receipt. A reserved exit alone is rejected.
All 42 images pass existing PE and AMD64 unwind structural inspection; this is not live unwind,
CodeView/PDB, mixed-stack or full target conformance evidence.

Five workloads pass the MIR interpreter oracle, including the overflow category. Two workloads
contain foreign calls and explicitly skip interpretation: the separate C observer requires reverse
resource cleanup order and exactly two calls, or independently checks aggregate field transport.
The 128-function case calls all 128 functions so C reachability pruning cannot reduce the study
to a single callee. Optimized native code is still free to fold constants or inline these calls.

`go test -mod=readonly ./tools/dev/fairyc-backend-study -count=1`, scoped Go vet, gofmt and Git diff
checks pass. Corpus tests emit both existing target profiles but do not execute Linux. Negative
checks reject a bare trap-like exit, unexpected streams and sample counts outside 1..5.

## Size and frontend observations

MIR counts are functions / blocks / operations before C emission. Source bytes are fixture Fairy
text; C bytes exclude the separately compiled scalar runtime and foreign observer. The common
frontend/MIR and emission observations run once per workload and are reused for both profiles.
Zero milliseconds means below the observed timer resolution, not a claim of zero work.

| Workload | Fairy bytes | MIR counts | C bytes | Frontend+MIR / emission ms |
| --- | ---: | ---: | ---: | ---: |
| scalar-control | 106 | 1 / 11 / 14 | 2,650 | 3.356 / 1.961 |
| inert-record | 144 | 2 / 5 / 5 | 2,128 | 1.014 / 0.000 |
| generic-interface-closure | 298 | 5 / 15 / 10 | 3,422 | 1.629 / 0.000 |
| reverse-resource-cleanup | 324 | 3 / 6 / 7 | 1,803 | 0.576 / 0.000 |
| checked-overflow | 80 | 2 / 5 / 3 | 1,528 | 1.090 / 0.536 |
| aggregate-c-crossing | 261 | 1 / 4 / 5 | 1,935 | 0.568 / 0.000 |
| 128-function-emission | 7,661 | 129 / 1283 / 1411 | 367,629 | 38.244 / 40.794 |

## Native build and launch observations

Each cell summarizes three sequential observations on this host. The shared production C driver
uses fresh temporary inputs, object compilation and linking with its existing C11 low/optimized
MSVC policy. No object cache or LTO is enabled; OS caches and other host activity are uncontrolled.
Build milliseconds include temporary I/O, compiler invocation/setup and linking. Launch/run
milliseconds include writing the image, process launch and output/control capture, so they are
not program throughput. Small optimized fixtures do not imply a compiler-scale speedup.

| Workload | Profile | Build ms median (min-max) | Launch/run ms median | PE bytes |
| --- | --- | ---: | ---: | ---: |
| scalar-control | low | 1030.56 (798.61-1412.18) | 35.23 | 109568 |
| scalar-control | optimized | 724.98 (669.79-781.22) | 32.62 | 109056 |
| inert-record | low | 1226.16 (1100.15-1486.25) | 56.50 | 109568 |
| inert-record | optimized | 907.02 (906.22-959.16) | 44.87 | 107520 |
| generic-interface-closure | low | 859.45 (794.91-862.77) | 30.61 | 109568 |
| generic-interface-closure | optimized | 795.35 (785.34-807.84) | 33.12 | 107520 |
| reverse-resource-cleanup | low | 1027.49 (893.68-1432.87) | 61.96 | 109568 |
| reverse-resource-cleanup | optimized | 1093.28 (1003.46-1394.87) | 38.97 | 107520 |
| checked-overflow | low | 1085.45 (975.68-1092.36) | 40.60 | 109056 |
| checked-overflow | optimized | 1021.12 (909.91-1077.34) | 39.17 | 109056 |
| aggregate-c-crossing | low | 1328.09 (1264.56-1357.74) | 52.19 | 109568 |
| aggregate-c-crossing | optimized | 1005.38 (1001.96-1035.21) | 37.01 | 109056 |
| 128-function-emission | low | 1088.39 (1071.27-1181.68) | 40.74 | 208384 |
| 128-function-emission | optimized | 1511.36 (1481.96-1611.87) | 34.76 | 137728 |

The raw report binds each workload/profile to Fairy, emitted C, runtime/observer and image hashes,
MIR/C sizes, native status/category and summed object bytes. Object/image bytes may contain
path/time-dependent data; this is input identification, not byte-for-byte build reproducibility.
The compiler executable is hashed; SDK, headers, libraries and compiler DLLs are not exhaustively
hashed. This is not an isolated or clean-machine distribution result.

## Replay and deferred evidence

From the repository root, use a new output path; the command refuses to overwrite an existing file:

```powershell
go test -mod=readonly ./tools/dev/fairyc-backend-study -count=1
go vet -mod=readonly ./tools/dev/fairyc-backend-study
go run -mod=readonly ./tools/dev/fairyc-backend-study --output .cache/backend-study-replay.json --samples 3
git diff --check
```

All listed completed commands returned 0. Corpus tests, vet and formatting ran before the harness
commit; the native report ran after that commit with no source changes. A report is only written
after every requested build, structural check and expected native outcome succeeds.

For the later native Linux batch, run the same harness once with `--cc gcc` and a new output, then
with `--cc clang` and another new output. Each run covers both profiles. This Linux driver path is
authored and compiles locally, but native GCC/Clang execution remains unexecuted here.

LLVM/QBE rows remain `not-measured`: neither has a Fairy MIR adapter, and tool lookup on this
host resolved neither candidate. No LLVM/QBE native compilation or new-target execution occurred.
Full repository tests, full compiler/parser workload measurements, sanitizers for this corpus,
new-target ABI/runtime/debug proof, clean-machine validation and hosted CI were not run.
This bounded review does not need those broader checks to record a C reference point; it cannot
establish a replacement-backend benefit. No push, release or deployment was performed.

C remains the product backend. A later LLVM textual-IR experiment needs representative paired
measurements and semantic/ABI/runtime/debug parity before adoption; QBE stays deferred and target
choice stays UNDECIDED. Public API/SDK, DB, runtime, dependencies, target flags, cache policy,
repository hygiene and CI/runner configuration are unchanged. Source-of-truth decision change is
limited to ADR 0497; this receipt and the ordered queue report progress under that decision.
