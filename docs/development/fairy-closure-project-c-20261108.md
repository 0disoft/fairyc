# ADR0551 original-source Complete to project C, 2026-11-08

Independent worker slice after the frozen project-link handoff. Final observed
HEAD `4bb341e6`, with the parent-added mode3 API in the authored source tree.
This worker owns only these three new files:

- `examples/lexer_candidate/tests/closure_project_c.fy`
- `internal/cli/candidate_closure_project_c_test.go`
- This receipt.

No product, prior project-link/cleanup slice, main/index, shared manifest/lock,
Git index or commit is changed. The parent owns concurrent C warning repairs,
the project closure emission API, native CPU and commits.

## Actual compiler route and emission contract

The Fairy producer takes original module source texts, independently scans and
parses them, and resolves their written imports through `module_catalog`.
Every source belongs to the same private test package, owner0. Resolved import
bindings retain actual source/declaration/target-module indices. The package
coordinate is a fixed SHA-256 test namespace independent of source content;
production source stamps and interface hashes remain separate authorities.
No Go type, helper, binding, unit, MIR or generated source is injected.

The producer calls the existing `project_compiler_core::build_complete_sources`
with those exact original sources, owners, bindings and coordinate. A successful
Complete must contain the ordinary identified project and public interface
record. Its project IR must retain `project=true`, `closures=true`, canonical
closure proof, source-qualified lifted helpers and a unique root entry.
Two providers in one control deliberately have equal helper declaration numbers;
their helper module identities must remain different.

The same Complete owner remains alive for both outputs:

1. `module_unit_c_emit::render` renders a single buffer from that Complete's
   checked project IR. It does not lower a separate detached unit.
2. `module_unit_c_names::build_project` derives the real identified project's
   names table, then the new explicit
   `emit_project_closures(raw,named,entry,1,true)` emits selected project parts.

The parent supplied that mode3 API during this task. The unit-only
`emit_closures` mode2 contract remains unchanged and is not used here. Project
and closure flags are never removed to enter a unit API. Mode3's aggregate
policy remains with the parent API; these seven consumers do not introduce
generic closure transport or change wrapper admission. No C product file is
patched or stubbed by this worker.

Both outputs must contain the inline capture representation. The selected
project parts must retain at least one actual closure adapter. Only after
Complete and both emissions succeed does a case publish its name, single C
buffer and `FYCPARTS1` packet, each with an exact decimal byte length. Errors
retain case name, compiler phase/stage/code, source module and original span on
stderr. Failure never falls back to detached linking or precomputed C.

## Seven independent consumer expectations

| Original source case | Expected native result with scalar runtime base80 |
| --- | --- |
| Provider factory passed through a closure parameter | 42 |
| Equal declaration numbers, different provider helpers | 84 |
| Returned struct, copies and captured mutable-i32 snapshot | 42 |
| Factory forwarded through three original modules | 42 |
| Created/copied but unused division-trapping helper | 42 |
| Division by zero inside the provider's invoked helper | 82 |
| Argument division trap before provider body overflow | 82 |

The Go controls parse the same original module texts using their actual module
paths, run `compiler.CheckExplicitFiles`, then execute verified Go MIR with the
Go interpreter. Successful returns and the two independent
`division-by-zero` outcomes are compared with the declared expectations; all
cases must leave stdout/stderr empty. This does not execute Fairy Complete or C.

## Parent-only native harness

The harness reuses the existing unit C exact-byte member reader,
`candidatebuild.Decode`, private compiler partition builder and target C
consumer builders. It builds one low-profile Fairy compiler producer through
`candidateParityBuildParts` and
`BuildWindowsMSVCCompilerHostPartsWithStackReserve`, retaining the existing
one-minute per-part bound and 16MiB stack. The producer must generate all seven
named C packets successfully before any consumer is accepted.

Each case's single and split outputs are then compiled and executed under low
and optimized consumer profiles (28 small consumers total), using the same
`cruntime.ScalarSource(80)` as the unit C harness. Every exit and empty output is
checked; ordered case names, packet counts, exact byte framing and EOF are
required. Producer compilation is a private partitioned path, not evidence of
stable CLI optimized execution or cache/codec acceptance.

## Validation and remaining boundary

Worker commands are source/Go-only:

```powershell
$env:GOMAXPROCS='1'
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureProjectC(Controls|Source)$' -count=1 -timeout=4m -v
gofmt -l internal/cli/candidate_closure_project_c_test.go
```

- Go Controls: **PASS**, 0.01s. Seven original-source Go compiler checks and
  interpreter outcomes agree with the expected five returns and two division
  traps; empty stdout/stderr for every outcome.
- Source: **PASS**, 18.65s; package **PASS**, 18.958s. The actual CLI `check`
  compiled the authored producer and its import closure in a temporary project,
  including the parent's real explicit `emit_project_closures` API. No stub or
  unit-mode fallback was used.
- Go formatting, LF/final-newline and trailing-whitespace checks: **PASS**.
- No product source, frozen slice, shared lock, Git index or commit changed by
  this worker. All three authored files are ready to freeze.

Source checking compiles the authored producer/import closure; it does not
execute the Complete, helper, emission or packet assertions. No remaining
source-level blocker is observed. Actual Complete/identity/C generation and
consumer results remain the parent's native integration gate.

Parent-only reproduction, not run by this worker:

```powershell
$env:GOMAXPROCS='1'
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureProjectCNative$' -count=1 -timeout=15m -v
```

## Parent native acceptance

`TestCandidateClosureProjectCNative`: PASS, 148.87s, in the stable detached
validation worktree with the frozen integration input. The real Fairy producer
completed original source -> Complete -> identified names -> both C emissions.
All 28 generated consumers (seven cases, two profiles, two output forms)
compiled and executed with the independent expected 42/84 return or 82 trap,
empty output, exact ordered packet boundaries and EOF.
Log: `.cache/closure-artifact-project-native-20261108.log` in the main checkout.
The parent-owned `emit_project_closures` mode3 API requires both project and
closure scope, retains ordinary full proof/name/entry checks, and keeps unit
and old project entry points separate. Unknown modes are rejected.

Linux, whole compiler/self-build, stable CLI, persistent cache/codec, hosted CI
and release are unverified by this slice. No normative document, public API/ABI,
DB, dependency, runner or workflow changed.

## Final worker code SHA-256

| Path | SHA-256 |
| --- | --- |
| `examples/lexer_candidate/tests/closure_project_c.fy` | `f6c9bc36f16a5e5045433c771029a11f35cabbe4cae3ae78ba65376a113d4a20` |
| `internal/cli/candidate_closure_project_c_test.go` | `778d5e38c4a5705343ae06c6b77f6f198a79272110305d4b69bc46078e1c0815` |
