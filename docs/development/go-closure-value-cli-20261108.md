# Go closure-value whole CLI acceptance — 2026-11-08

Scope: detached `329ebcf5` worktree `.cache/closure-interfaces-20261108`,
ADR 0551. This sidecar owns only this receipt and
`internal/cli/closure_value_cli_test.go`. This sidecar made no production, ADR,
tracked-lock, main, commit, remote-state or CI changes. Concurrent candidate
FY/test/lock edits are outside this sidecar and were left intact.

## Current implementation and validation state

The parent implemented the backend fix in `internal/cbackend/terminators.go:291`.
For checked i32 division/remainder, the existing `_MSC_VER` private volatile
divisor barrier now applies when the divisor is `mir.OperandI32`, the function
is a parameter-free inline body, or it is a `ClosureTarget`. Both zero and
MIN/-1 guards read that snapshot. The previous inline behavior is retained;
ordinary non-closure runtime-input guards remain unchanged. No compiler warning,
optimization flag, timeout or language contract was relaxed.

The parent's new `internal/cbackend/closure_value_msvc_guard_test.go`,
`TestClosureValueMSVCCheckedDivisionGuards`, covers `/` and `%`, literal and
runtime-input closure divisors, both guard reads under `_MSC_VER`, the ordinary
runtime-input control, and the ordinary literal-zero barrier. This sidecar read
the implementation and test. The parent reports the three focused backend tests
passed in **0.073s**, and `go vet -mod=readonly ./internal/cbackend ./internal/cli`
passed. This sidecar did not repeat those checks.

**Final real Go CLI acceptance: PASS, 25.792s**, confirmed by reading the parent's
`.cache/go-closure-cli-final-20261108.log`:

```powershell
go test -mod=readonly ./internal/cli -run '^TestClosureValueCLI' -count=1 -timeout=180s -v
```

The log confirms source/interpreter checks, all low/optimized build/run/test
lanes, direct native trap checks, cold/warm/new-process disk restoration,
corruption recovery, and all 52 refusal commands passed. Both native profiles
retain exact status **5** for the four-case test report, ordered division-by-zero,
division-by-zero, integer-overflow, then passed; captured streams are empty.
Normal `run` still returns **42**. Final matrix native-test timings were
**4.4633899s low / 7.9211144s optimized**, with live test contexts.

The earlier parent-reported isolated optimized native acceptance also passed in
**5.896s** (native-test
wall time **5.7559782s**), using the unchanged acceptance sources/assertions and
the existing isolated optimized/test command below. All four cases ran with
exact categories: division-by-zero, division-by-zero, integer-overflow, then
passed; original CLI status **5**, one passed/three deliberately trapped tests.
The initial timeout and subsequent status-6/C4702 failure are historical failures,
not the current optimized result. The ClosureTarget-only barrier removed the
stall; the additional literal-divisor condition fixed MSVC C4702 at
`unit.c(7986)` in an ordinary literal-zero argument guard.

**Final Windows native backend regression: PASS, 16.454s**, confirmed by reading
the parent's `.cache/go-closure-msvc-regression-final-20261108.log`:

```powershell
go test -mod=readonly ./internal/cbackend -run '^(TestClosureValueSourceNative|TestPartitionedZeroParameterInlineTraps)$' -count=1 -timeout=180s
```

`TestClosureValueSourceNative` covered five source cases in two profiles
(10 executions), and `TestPartitionedZeroParameterInlineTraps` covered five
cases in two profiles (10 executions): **20 native executions passed**.
The Windows Go CLI unit has no pending implementation or verification work;
its two sidecar files are stable for the parent's four-file integration/commit.
This sidecar did not rerun compilation during cleanup/receipt updates. No Fairy
compiler executable, Linux, self-build, distribution or CI result is claimed.

## Source and command coverage

The test writes real `fairy.toml` root/member manifests, a source dependency at
`libs/answer`, root source, and discovered `@test` source. It reuses
`writeWorkspaceSessionFixture`, `writeWorkspaceFile`, `runForTest`,
`workspaceCacheArguments`, `decodeWorkspaceCacheReport`, `portableNativeReport`
and `practicalFileEquals`. These invoke the Go CLI dispatcher and real native
toolchain; they do not build or execute a Fairy candidate compiler. There are no
AST/HIR mutations, invented lifted facts, replacement lowerers or fake receipts.

- Returned factories with four ordered copied i32 captures and a different
  zero-capture helper pass through the same typed parameter. Noncommutative
  arithmetic checks capture order. Independent factory instances return 42/52;
  changing the creator's mutable source to 999 does not change the snapshot.
- Immutable closure copies, returned ordinary structs, struct copies and a
  field extracted to a named local remain callable after factory return.
- Creating/copying an unused trapping closure does not execute its body.
  A side-effecting argument runs exactly once. Tests distinguish an unused body
  local's division trap, an argument's division trap before body overflow, and
  actual checked body overflow. A final passing test proves continuation/order.
- Stable `check`, locked check, interpreter `test`, Windows native `build`,
  `run`, and `test --native` are exercised. Development is native **low**;
  `--release` is **optimized**. Builds must publish PE bytes. Runs preserve the
  original program result 42 and empty streams. Locked native commands preserve
  the disposable fixture's lock bytes.
- Thirteen refusal sources each reach all four stable commands: mutable
  construction, reassignment, unsupported signature, direct field invocation,
  Option transport, repr(C), foreign transport, pointer-to-carrier transport,
  closure capture, five captures, seventeen locals, sixty-five expression nodes,
  and generic forwarding. Every command must retain status **3**, the relevant
  original diagnostic, empty stdout and no build output. No limits are raised.

## Persistent cache controls

Stable product commands do not consume the package MIR disk cache. The existing
Go CLI `experimental check-workspace-cache` is therefore exercised separately,
using the unchanged stable helper's cache limits. Cold publication must store
two packages; warm and child-process checks require two verified hits, zero
misses/stores and `lowered=false`. The child is the same Go test executable,
spawned through `os/exec` with a 60-second bound and the same compiler identity.
It also runs a fresh-source stable interpreter success test.

Both actual generation files are hashed; warm/new-process restoration must not
rewrite them. Corrupting one real generation must produce one hit/one miss,
one rejected disk record, lowering and two replacement stores, followed by an
all-hit check. Invalid original source must return **3/E1009**, publish nothing
over the good generation bytes, and recover to all-hit after source restoration.
Observed retained seed: **2 records, 105948 bytes, 9 MIR functions**.

## Historical Windows verification before the backend fix

Host: `go1.27.1 windows/amd64`, installed MSVC, 2026-11-08 KST. Commands ran
from the detached worktree, always with `-mod=readonly`:

```powershell
go test -mod=readonly ./internal/cli -run '^TestClosureValueCLI(StableSource|PersistentCache|Refusals)$' -count=1 -timeout=120s -v
go test -mod=readonly ./internal/cli -run '^TestClosureValueCLI' -count=1 -timeout=180s -v
go test -mod=readonly ./internal/cli -run '^TestClosureValueCLI(StableSource|PersistentCache|Refusals)$' -count=1 -timeout=120s
go test -mod=readonly ./internal/cli -run '^TestClosureValueCLIPersistentCache$' -count=1 -timeout=120s
```

1. Initial source/cache/refusal smoke: **PASS**, 1.325s, before adding corruption
   and the final two refusal cases.
2. Complete bounded sidecar: **FAIL**, 142.798s. Source/interpreter tests,
   all 52 refusal commands, persistent cache including its real child, corruption
   repair, low native build/run/test, optimized build/run, and both direct trap
   programs in both profiles **passed**. Optimized native test compilation
   returned original CLI status **6**, not the expected test-trap status 5:

   ```text
   fairyc: build failed: compile generated unit: MSVC host command exceeded 1m0s
   ```

   Low native test JSON reported 1 passed/3 deliberately trapped tests, exact
   ordered categories and empty captured stdout/stderr. Direct run traps retain
   status **5** and source locations: provider `answer.fy:9:82` for the unused
   local, root `main.fy:1:101` for argument-before-body ordering.
3. Final narrowed run after strengthening diagnostic assertions initially could
   not compile: concurrent `candidate_closure_value_interfaces_test.go:91:8`
   reported `declared and not used: i` through vet. The other owner subsequently
   removed that unused variable; the same command then ran in 1.366s. Source
   and all 52 refusal checks passed. Cache restoration/corruption checks passed,
   but an incorrect test expectation `E1006` failed against the actual original
   return-type diagnostic `E1009`. The expectation was corrected to `E1009`.
4. Only the persistent-cache test was rerun after that correction: **PASS**,
   0.852s, including cold/warm/new-process, actual corruption recovery, exact
   **3/E1009** refusal and restoration to all-hit. The expensive native matrix
   was not repeated.

The child-only entry skips when invoked without the parent environment; its
actual subprocess invocation passed. `gofmt` was applied to the owned Go file.

## Historical isolated optimized test diagnosis

The native matrix now has nested `build`, `run`, `test` subtests so the failing
command can be selected without repeating other native work. Assertions, source
constants and deadlines were preserved. Exactly one isolated native invocation:

```powershell
go test -mod=readonly ./internal/cli -run '^TestClosureValueCLIWindowsNative$/^optimized$/^test$' -count=1 -timeout=180s -v
```

Result: **FAIL**, original CLI status **6**, Go package elapsed **62.577s**,
native-test elapsed **62.3898161s** (including Go compilation/observer wall time:
80.2284631s). Stable CLI arguments were exactly
`test --json --locked --native --capture-output --release`. Four tests were
selected with original expected status 5, one pass and three traps. Compilation
stalled on the **first**, `tests::closure::a_body_trap`; later tests never ran.

Evidence is retained under
`.cache/go-closure-cli-optimized-20261108-1/`:

- `workspace/`: original manifests, source and fixture lock under normal names.
- `generated/unit.preprocessed.c`: actual CL input, **14209 bytes**, SHA-256
  `2b73779657cf1310b9e860db1015c8a02175837d62f65866b87b049bda0817ac`.
  Runtime source/preprocessed snapshots are also retained (2894/3085332 bytes).
- `stdout.txt`, `stderr.txt`, `run.json`, `inputs.json`, `provenance.json`, and
  `cl-commands-owned.json`. The last file filters the raw process observations
  to test PID 41728; the unfiltered log also contains another worker's CL calls.

The passive observer changed only child `TMP`/`TEMP` to an owned capture
directory. No Go overlays, alternate diagnostic sources, compiler wrappers,
product hooks, flag changes or limit changes were used. Effective `GOFLAGS`
was empty. Both this file's provider and the workspacecompiler cache provider
are the same **624-byte noncommutative** source, SHA-256
`7419b0289e63b97365487991485fae045e1f5db5f9558c25a40710af69a017a4`.
The `workspace=` log text was added before the isolated run to identify its
actual directory; it is not evidence of a different fixture.

Actual compiler: MSVC `14.51.36231`, owned CL PID 21472:

```text
cl.exe /nologo /std:c11 /TC /W4 /WX /MT /O2 /c unit.c /Fo:unit.obj
```

The failing cdriver route is `compileNativeWorkspaceTest` →
`prepareNativeProgram`/`nativeBuildPlan.compile` → `windowsNativeToolchain.Build`
→ `BuildWindowsMSVCTargetCached` → `buildWindowsUnitCached`
(`internal/cdriver/unit_cache_windows_target.go:47`) →
`runWindowsHostCommandWithLimitAndTimeout`
(`internal/cdriver/windows_host.go:942`). That runner starts its own 60-second
`context.Background()` deadline. At failure, `t.Context().Err()` was **nil**,
and the outer test deadline still had **117.5555605 seconds** left. The original
complete run also used **180s**, not 120s; the 120s invocations excluded native.
Thus test cleanup/outer cancellation did not cause this CL timeout. Deferred
directory removal runs after the command returns; compiler cancellation targets
that command's owned PID tree.

Source-level explanation for the original approximately 124-second failure:
`useRuntimeObject` can retry a failed runtime-hit link callback after recompiling
runtime. That callback includes generated-unit compilation, so a unit timeout
can be attempted twice. This explains a possible doubled deadline; the initial
run's individual CL invocations were not captured, so this remains an inference.

For the requested **pre-preprocessor UnitSource**, the test-only diagnostic
`TestClosureValueCLIEmitNativeInputs` loads the retained workspace, byte-checks
all three original sources against the acceptance constants, and uses
`WithPackageTests` → `prepareWorkspaceTests` → `workspacecompiler.Check` →
`nativeWorkspaceTestProgram` → `prepareNativeProgram`. It saves
`plan.options.UnitSource`/`RuntimeSource` and never called Build/Execute. This
diagnostic seam has since been removed from the acceptance file and archived as
data; it is not a runnable acceptance test now.

Result: **PASS**, 0.222s. `plan-inputs/plans.json` contains all four original
test identities, sizes and hashes. `a_body_trap.unit.c` is **3597 bytes**, SHA-256
`4895e53440deab8dee5333827193f15ff801bd087b63e09227ebb0f97d8ce094`.
The other UnitSources are 6628, 5389 and 13394 bytes. These files are explicitly
**reconstructed exact native plans**, while `generated/unit.preprocessed.c` is
the **live captured failing compiler input**. No further 60-second compile was run.

## Historical alternate-C noinline controls

At the parent's request, only two variants of that exact captured input were
compiled; neither is acceptance source or a production change. Raw UnitSource
variants are retained alongside the actual preprocessed compiler inputs under
the evidence root's `noinline-diagnostic/`. Original files/hashes are preserved.
Adapter-only inserts `__declspec(noinline)` on `fy_closure_2`'s definition;
target-only inserts it on `fy_fn_2`'s prototype and definition. Eventual `_MSC_VER`
guarding belongs to a product fix, not these Windows-only diagnostic variants.

The opt-in `TestClosureValueMSVCNoInlineDiagnostic` seam is now archived as data,
not part of the acceptance file. Its historical result follows.

Each CL command retained `/nologo /std:c11 /TC /W4 /WX /MT /O2 /c unit.c
/Fo:unit.obj`, with an independent **10-second** diagnostic bound and owned-PID
tree cancellation. No warning, optimization or product timeout relaxation.

| Diagnostic variant | Raw bytes | Actual input bytes | CL wall time | Result |
| --- | ---: | ---: | ---: | --- |
| adapter-noinline | 3618 | 14230 | 10.3183002s | diagnostic deadline; no object |
| target-noinline | 3639 | 14251 | 10.3218415s | diagnostic deadline; no object |

Actual input SHA-256: adapter
`4154fcdc79f0cd4daab0803fb9c09d68eb9c1603b45b8dbb1ed607a709a45fc3`;
target `67b90854e3df708f92465578a4ba390d28e7b82cdf4bf506e43b8ae412b5b84c`.
Raw variants: adapter
`79aa3fa08b0346277642493ced6050f7df7d988d8511bb3f8cf3fb02f086b3aa`;
target `23f34815b5be67165f4bde8e231a32c358e3d1d174ea0263be1712032321b1ce`.
`noinline-diagnostic/results.json` retains exact commands, hashes and output.
The diagnostic Go test correctly **failed**, 20.838s; neither variant is a
confirmed fix within this short control. No baseline or 60-second compile was
repeated, and no runtime semantic claim is made for these alternate C files.

## Historical alternate-C volatile-guard control

The parent's next hypothesis matched the baseline MSVC i32 guard snapshot in
`internal/cbackend/terminators.go:291`, whose original condition covered only
zero-parameter inline bodies. Exactly **one** retained-C variant was compiled:
assign the existing private `fy_checked_divisor_i32` before the first literal-zero
division guard, and use it in **both** zero and MIN/-1 guards. The later division
operation and every other byte were preserved. No noinline changes were included.

The opt-in `TestClosureValueMSVCVolatileGuardDiagnostic` seam is now archived as
data, not part of the acceptance file. Its historical result follows.

Result: **PASS**, Go package 0.435s; actual CL compile **0.1432633s**, status 0,
under the same 10-second diagnostic bound and unchanged `/O2 /W4 /WX` flags.
The generated COFF object passed `pecoff.InspectObject`.

| Artifact in `volatile-guard-diagnostic/` | Bytes | SHA-256 |
| --- | ---: | --- |
| `raw.unit.c` | 3662 | `c46875b9cf8c8e039041ae6e9ebd907c9bea1bb1d151c654aab6db7101b8a36b` |
| `unit.c` (actual CL input) | 14283 | `dc8ef8a385e78a77a102a42307f728483951f14615df561acd2aa90c5612943c` |
| `unit.obj` | 4156 | `00518ecd14115862e3ba338d748cdd16db975d46d71d07cf7427b5cb0989f946` |

`results.json` retains command, compiler, timings and hashes. This is explicitly
an **alternate-C compile diagnostic**, not an acceptance run or production fix.
It motivated the parent's extension of the existing `_MSC_VER` i32 guard-snapshot
condition; the final implementation also covers literal i32 divisors as described
above. No production file, original fixture or assertion was edited by this
sidecar.

## Scope limits and retained diagnostics

Cleanup note: the three opt-in diagnostic tests above and their diagnostic-only
imports were removed from the acceptance Go file after preserving them as data
under `.cache/go-closure-cli-optimized-20261108-1/diagnostic-seams/`:
`diagnostic-tests.go.txt`, `closure_value_cli_test.go.before-cleanup.txt`, and
`cleanup.json` (before/after hashes and removal inventory). Their commands and
results above are historical; removed-test invocation commands are no longer
listed as runnable commands. Original acceptance fixtures, assertions and native
timing remain intact. Cleanup ran no compilation/native tests.

- All scoped Windows Go CLI checks are complete: final full CLI acceptance and
  the 20-execution native backend regression passed; the parent also reports
  focused backend checks and vet passed. Existing MSVC 60-second command
  deadlines and acceptance statuses/assertions remain intact. There is no
  pending work in this Go CLI unit.
- Disk restoration here verifies/reassembles MIR through the cache CLI. Stable
  native execution and the child's interpreter test independently compile fresh
  source; they do not prove execution of cached MIR through product commands.
  Forged-artifact/source-binding and restored-native probes remain owned by
  `internal/workspacecompiler/closure_value_cache_test.go`; they were not rerun.
- Linux remains batched and unrun; no Fairy candidate execution, Go/Fairy invalid
  closure status-parity claim, self-build, distribution or CI/release claim.
  No API, database, dependency, runner or source-of-truth contract was changed.
