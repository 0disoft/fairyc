# Go closure-value package-cache coverage, 2026-11-08

## Ownership and current status

Shared detached checkout: `.cache/closure-packages-20261108`, starting HEAD
`94eac0fc97e4c7b513313192aa31d2aa466138fe`. This worker owns:

- `internal/workspacecompiler/closure_value_cache_test.go`
- `docs/development/go-closure-value-cache-20261108.md`
- `internal/moduleinterface/closure_value_test.go` (subsequently assigned by parent)

Normal source admission, memory/disk/new-session/new-process baseline restore,
provider body invalidation, representative Windows low/optimized native execution,
all forged-cache repairs and no-bad-publication controls now pass after the
parent's source-owned verifier fix. Exact private-format gates and feature-free
legacy context bytes also pass. No current blocker remains in this worker's
assigned test scope; the parent's overall assignment is not marked complete here.
No production files, FY sources, locks, queue records, main checkout, commits,
pushes, or CI were changed by this worker. Linux execution remains in the separate
batched lane. This document is evidence and a handoff, not a language contract.

## Deterministic source fixture

The existing `writeCacheWorkspace` creates two actual manifest-backed packages:
provider `answer` and consumer `main`. The provider exports a factory returning
`fn i32(i32)`, an ordinary `Holder` with a direct function-value field, a boxed
factory, and an `apply` function. No AST/HIR/MIR closure facts are patched for
source admission.

The main case exercises four ordered captures, mutation of the original capture
after construction, function-value copies, struct copies and field extraction,
distinct lifted targets with different bodies, creation of an unused trapping
closure, and a side-effecting block argument with an explicit count assertion.
Its expected result is 42. Separate consumers invoke a trapping unused local and
check argument division-by-zero before an overflowing closure body.

Fixed provider SHA-256:
`7419b0289e63b97365487991485fae045e1f5db5f9558c25a40710af69a017a4`.

Current consumer SHA-256 values, in case order (the representative consumer now
evaluates the counted block directly as a closure-call argument, while retaining
the separate cross-package `apply` assertions):

- `d5a01898c98169e280fabcd5fbe428483c51333232184b0b2ebf966bd526c681`
- `0ef453c687965ebde2c275abc23bfed2e436eea29c38994225facf64336a69aa`
- `8a42fcbf59bbcb9ea715ae10a17551f6d5e5a916a7b49eb7a509a1b944c0f322`

## Tests implemented

- `TestClosureValueCacheSourceAdmission`: calls `compiler.CheckPackages` on
  normal workspace semantic input; checks verified MIR, three actual lifted
  targets, interpreter return/trap outcomes, and Linux GNU/Windows MSVC C emission.
- `TestClosureValueCacheRestore`: memory cold/warm and disk cold/warm/new-session
  restore for all three consumers. An all-hit result must never call lowering;
  restored interpreter outcomes must equal fresh `CheckPackages` outcomes.
- `TestClosureValueCacheProviderBodyInvalidation`: edits only the private lifted
  arithmetic body, retaining the public source signatures. The first check must
  lower and publish both packages, produce 43 rather than 42, and then restore
  without lowering. Raw consumer hits may be zero or one; whole-program stale
  execution is forbidden.
- `TestClosureValueCacheNativeRestore` and `TestClosureValueCacheChild`: restored
  representative products run Windows amd64 low/optimized native execution in
  memory and new-session disk lanes; the exact same test executable also restores
  disk artifacts in a new process, compares against fresh source, and runs native
  low/optimized execution. The child must emit its success marker. A 60-second
  context bounds the child, as in the existing inline-call cache test.
- `TestClosureValueCacheNewProcess`: all three consumers independently restore
  in a fresh process without native compilation, including both trap cases.
- `TestClosureValueCacheNoBadPublication`: injects a structurally valid but
  source-inconsistent helper body, target selection, or ordered captures after
  normal lowering, and requires refusal with no executable or retained records.
  Healthy subsequent compilation must then be cold and produce a repaired hit.
  All eight memory/disk cases pass after the parent's source-trust fix.
- `TestClosureValueCacheForgedRestoreRepair`: changes actual retained products,
  not a restore/lower hook. Arithmetic changes, same-type capture reorder,
  target replacement and a literal-return body replacement all pass MIR and
  artifact/codec validation before injection. The eight memory/disk cases must
  lower exactly once, return fresh-source 42, quarantine the forged disk generation,
  and subsequently hit with the ordinary lowering hook forbidden.
- `TestClosureValueCachePrivateFormats`: checks literal exact closure process,
  stable MIR, type-context and lifted-body schemas/versions, refuses legacy
  downgrades, and preserves a fixed feature-free legacy context byte vector.
- `TestClosureValueCacheUnusedCarrier`: memory/disk cold/warm and disk new-session
  scalar execution with an exported, unused ordinary struct containing a Fn field.
  No unnecessary fallback is permitted when the declaration selects no functions.
- `TestClosureValueCacheSourceProofStructure`: unchanged-source control, canceled
  context, forged source-owned carrier field name, and duplicate qualified
  function-name refusal at the proof boundary.
- Moduleinterface `TestClosureValuePublicInterfaceExactBytesAndImmutability`
  and `TestClosureValuePublicInterfaceAdmitsOnlyExactFunctionLeaf`: exact fixed
  v5 wire vectors/hash, unchanged bytes for a helper-body edit, legacy scalar
  bytes, source HIR immutability, and rejection of noncanonical function leaves.

The tests reuse `inlineCallsConfiguration`, `newInlineCallsSession`,
`forbidInlineCallsLowering`, and existing workspace helpers. Budgets remain eight
entries, 32 MiB cache, 8 MiB artifact, and 16 MiB disk record. No new framework or
expanded budget was introduced.

## Executed results and first blocking log

Formatting completed:

```text
gofmt -w internal/workspacecompiler/closure_value_cache_test.go
```

Passed before the parent's concurrent format edit made the package unbuildable:

```text
go test -mod=readonly ./internal/workspacecompiler -run '^TestClosureValueCacheSourceAdmission$' -count=1 -v
--- PASS: TestClosureValueCacheSourceAdmission (0.07s)
    --- PASS: TestClosureValueCacheSourceAdmission/factory-struct-snapshot-targets-unused-trap-argument-once (0.03s)
    --- PASS: TestClosureValueCacheSourceAdmission/invoked-unused-local-traps (0.02s)
    --- PASS: TestClosureValueCacheSourceAdmission/argument-traps-before-body (0.02s)
PASS
ok github.com/0disoft/fairyc/internal/workspacecompiler 0.161s
```

The first dependent cache probe was limited to one memory case:

```text
go test -mod=readonly ./internal/workspacecompiler -run '^TestClosureValueCacheRestore$/^memory$/^factory-struct-snapshot-targets-unused-trap-argument-once$' -count=1 -v
# github.com/0disoft/fairyc/internal/packageartifact
internal\packageartifact\body.go:347:21: undefined: bodyFormat
internal\packageartifact\body.go:390:21: undefined: bodyFormat
internal\packageartifact\stable_artifact.go:230:21: undefined: stableArtifactFormat
internal\packageartifact\stable_artifact.go:277:21: undefined: stableArtifactFormat
FAIL github.com/0disoft/fairyc/internal/workspacecompiler [build failed]
FAIL
```

Exit status: 1. No cache assertion or cache publication was reached by that
probe. Do not describe it as a reproduced cache-runtime failure. The starting
checkout's `packageartifact.ErrClosureValuesUnsupported` deliberately refused
closure values, so source admission was tested separately from cache support.

## Focused resume: public-interface blocker

After the parent reported that `bodyFormat`/`stableArtifactFormat` exist and the
legacy context gate was removed, exactly one focused command was run:

```text
go test -mod=readonly ./internal/workspacecompiler -run '^TestClosureValueCacheRestore$/^(memory|disk)$/^factory-struct-snapshot-targets-unused-trap-argument-once$' -count=1 -v
```

Both cases first passed their explicit `compiler.CheckPackages` source admission
check, then failed at the cold `Session.Check` with the identical error:

```text
closure_value_cache_test.go:167: closure value Session.Check: error=extract public interface for package 2: module "answer": public struct "Holder": field "action": unsupported canonical type "fn i32(i32)" cache={Outcome: Storage: Packages:0 Hits:0 Misses:0 Rejected:0 Stored:0 StoreSkipped:0 StoreFailed:0 Lowered:false ReassemblyFallback:false} diagnostics=[] unsupported=[]
--- FAIL: TestClosureValueCacheRestore (0.11s)
    --- FAIL: TestClosureValueCacheRestore/memory/factory-struct-snapshot-targets-unused-trap-argument-once (0.05s)
    --- FAIL: TestClosureValueCacheRestore/disk/factory-struct-snapshot-targets-unused-trap-argument-once (0.06s)
FAIL
FAIL github.com/0disoft/fairyc/internal/workspacecompiler 0.328s
FAIL
```

Exit status: 1. Production and this workspacecompiler test package compiled;
packageartifact's own tests were not compiled or run. No cache lookup, lowering,
or publication was reached by these two `Session.Check` calls.

Read-only diagnosis: `workspacecompiler.packageInterfaces` calls
`moduleinterface.ComputePackages`. `internal/moduleinterface/interface.go:340`
`stableType` does not admit `hir.TypeClosureI32`; it falls through to nominal
type parsing and the error at line 402. `stableStructFields` exposes the failure
first, but exported factory results and `apply` parameters need the same leaf.

Next needed production change, for the parent or an explicitly aligned scope:
canonicalize only the exact `hir.IsClosureI32` leaf in module interfaces, retaining
legacy feature-free canonical bytes/hashes and existing semantic wrapper/FFI
restrictions. Include the ordinary struct field, public result, and parameter
cases. Do not treat arbitrary function spellings as builtins or alter the fixture
to hide the exported type. No moduleinterface production file was edited by this
worker. This is the next observed blocker, not a claim that later cache gates pass.

After the parent reported successful actual-source packageartifact roundtrips and
new exact helper-format detection, the identical focused memory/disk command was
run once more. It compiled and failed at the **same** `Holder.action` canonical
type error above, at test line 167 in both cases; all report counters remained
zero. Output summary: parent test 0.10s, memory 0.04s, disk 0.06s; package 0.194s,
exit status 1. The artifact fixes therefore do not yet unblock the workspace's
earlier `moduleinterface.ComputePackages` gate. No further dependent run was made.
Required ownership alignment remains `internal/moduleinterface/interface.go`
(`stableType`, exact closure leaf); this worker has not claimed or edited that
production file. The parent can own that narrow fix or explicitly assign it here
before a production edit. No fixture/assertion changes are needed for this error.

## Baseline unblocked; next source-owned trust blocker

After the parent added only `hir.IsClosureI32` to moduleinterface's builtin leaf
set, the focused memory case passed: cold `Hits:0 Stored:2 Lowered:true`, then
warm `Hits:2 Stored:0 Lowered:false Outcome:all-hit`.

These enabled validations then passed:

```text
go test -mod=readonly ./internal/workspacecompiler -run '^(TestClosureValueCacheRestore|TestClosureValueCacheProviderBodyInvalidation)$' -count=1 -v
PASS: memory/disk, all three consumers, warm and disk new-session hits;
      body-only edit 42 -> 43 and healed hits in both storage modes
ok github.com/0disoft/fairyc/internal/workspacecompiler 0.994s

go test -mod=readonly ./internal/workspacecompiler -run '^TestClosureValueCacheNativeRestore$' -count=1 -v
PASS: Windows low/optimized memory, disk new-session, disk new-process
      six executions, each exit 42, empty stdout/stderr, no trap
ok github.com/0disoft/fairyc/internal/workspacecompiler 10.292s

go test -mod=readonly ./internal/moduleinterface -run '^TestClosureValuePublicInterface' -count=1 -v
PASS: closure / closure-body-edit 458 identical bytes,
      fyif5-sha256-bde0c1544468f3fe6d0150dde8c0d14068d65225e1ab9b6d14c3524a6e5aded5
      legacy 167 bytes,
      fyif5-sha256-2a5358d58cba908ccc28bb1758e764a2b13ad69ebd6d713ff4f03eccf738f41f
      source immutability and exact-leaf negatives
ok github.com/0disoft/fairyc/internal/moduleinterface 0.043s

go test -mod=readonly ./internal/workspacecompiler -run '^TestClosureValueCacheNewProcess$' -count=1 -v
PASS: all three consumers, each child Hits:2 Lowered:false Outcome:all-hit,
      fresh-source execution comparison and both C emission profiles
ok github.com/0disoft/fairyc/internal/workspacecompiler 0.775s
```

The native child originally emitted `closure-value-new-process-disk-42`; the
expanded child now uses `closure-value-new-process-disk-ok` for both return and
trap consumers and enables native compilation only with its explicit native flag.
The native evidence above predates only that child dispatch/marker change.

Historical source-trust failure, **fixed by the parent's source-owned verifier**
(exact original focused command and output):

```text
go test -mod=readonly ./internal/workspacecompiler -run '^TestClosureValueCacheNoBadPublication$/^memory$/^body$' -count=1 -v
closure_value_cache_test.go:364: forged body executable/publication escaped: error=<nil> verified=true functions=9 retained=2 cache={Outcome:fallback Storage:memory Packages:2 Hits:0 Misses:2 Rejected:0 Stored:2 StoreSkipped:0 StoreFailed:0 Lowered:true ReassemblyFallback:false}
--- FAIL: TestClosureValueCacheNoBadPublication (0.04s)
    --- FAIL: TestClosureValueCacheNoBadPublication/memory/body (0.04s)
FAIL
FAIL github.com/0disoft/fairyc/internal/workspacecompiler 0.131s
FAIL
```

Exit status: 1. Injection changes only the four-capture helper's return operand
to literal i32 99 after ordinary successful lowering. All signatures/capture
counts remain unchanged and `mir.Verify` explicitly passes before Session sees
the product. The bad product is returned executable and two records are retained.
The assertion remains strict; the test does not execute the untrusted result.

Parent-owned next fix: in `internal/workspacecompiler/session.go`, validate
closure target choice, written snapshot operands/order and lifted-body facts
against the current source HIR before cold memory/disk publication, and apply the
same source-owned validation after warm reassembly/rebind with normal lowering
fallback/repair on a mismatch. Standalone MIR/codec verification proves structural
safety but cannot by itself prove agreement with the source. Parent may introduce
the verifier in an aligned production file; this worker has not modified one.

## Actual retained-cache reproduction and resolution

Before the parent's verifier was wired, the focused actual-cache test reproduced
both same-type capture reorder and changed target arithmetic in memory and disk.
These tests keep the original source, cache key and public interface unchanged.
The disk test reuses the existing `compositeOptionDiskEnvelope`, projects the
well-typed forged MIR with the public artifact APIs, replaces the inner compilation
payload, and writes its valid checksum-addressed generation. It is not torn JSON.

```text
go test -mod=readonly ./internal/workspacecompiler -run '^TestClosureValueCacheForgedRestoreRepair$/^(memory|disk)$/^(arithmetic|captures)$' -count=1 -v
well-typed captures forged execution={Kind:return Value:-99 TrapCategory: ...}
memory/captures: error=<nil> calls=0 cache={Outcome:all-hit Storage:memory Packages:2 Hits:2 Misses:0 Rejected:0 Stored:0 StoreSkipped:0 StoreFailed:0 Lowered:false ReassemblyFallback:false}
disk/captures: error=<nil> calls=0 cache={Outcome:all-hit Storage:disk Packages:2 Hits:2 Misses:0 Rejected:0 Stored:0 StoreSkipped:0 StoreFailed:0 Lowered:false ReassemblyFallback:false}
FAIL github.com/0disoft/fairyc/internal/workspacecompiler 0.386s
```

That initial arithmetic mutation omitted the checked terminator's matching
operator and was rejected by MIR with `V1501 ... trapping operation is not
dominated by its complete checked edge`. This was a test-fixture defect, not the
cache vulnerability. The fixture was corrected to update both the operation and
checked edge; no assertion or MIR check was removed. A focused corrected run gave:

```text
go test -mod=readonly ./internal/workspacecompiler -run '^TestClosureValueCacheForgedRestoreRepair$/^(memory|disk)$/^arithmetic$' -count=1 -v
well-typed arithmetic forged execution={Kind:return Value:62 TrapCategory: ...}
memory/arithmetic: error=<nil> calls=0 cache={Outcome:all-hit Storage:memory Packages:2 Hits:2 Misses:0 Rejected:0 Stored:0 StoreSkipped:0 StoreFailed:0 Lowered:false ReassemblyFallback:false}
disk/arithmetic: error=<nil> calls=0 cache={Outcome:all-hit Storage:disk Packages:2 Hits:2 Misses:0 Rejected:0 Stored:0 StoreSkipped:0 StoreFailed:0 Lowered:false ReassemblyFallback:false}
FAIL github.com/0disoft/fairyc/internal/workspacecompiler 0.406s
```

After the parent added `verifyClosureValueSource` at both warm restoration and
fresh publication boundaries, this complete focused suite passed:

```text
go test -mod=readonly ./internal/workspacecompiler -run '^(TestClosureValueCacheSourceAdmission|TestClosureValueCacheRestore|TestClosureValueCacheProviderBodyInvalidation|TestClosureValueCacheNewProcess|TestClosureValueCacheNoBadPublication|TestClosureValueCacheForgedRestoreRepair)$' -count=1 -v
PASS: admission 0.10s; baseline restore 0.89s; invalidation 0.30s;
      new process 0.81s; no bad publication 1.01s; forged restore repair 1.43s
ok github.com/0disoft/fairyc/internal/workspacecompiler 4.650s
```

All sixteen forgery controls passed. Fresh forged products were refused before
executable return or any retained entry. Actual corrupted-cache products passed
their structural codecs, then caused `Outcome:fallback ReassemblyFallback:true
Hits:2 Stored:2 Lowered:true` with exactly one observed `Session.lower` call.
Fresh-source comparison returned 42, the corrupted disk generation was removed,
and the next reader obtained `Outcome:all-hit Hits:2 Lowered:false` with the hook
forbidden. Target replacement's untrusted execution trapped at division-by-zero;
the other forged results were 62 (arithmetic), -99 (capture order) and 99 (body).

## Source-validation cost and final verification

Read-only audit of the parent's `closure_value_source.go`: every source-bearing
cache request builds a selection of closure target functions, functions with Fn
or ordinary closure-carrier parameters/results/locals, and functions containing a
closure-valued expression. It reconstructs that selection from checked HIR via
`mir.LowerContext`, then compares `packageartifact.ClosureSourceProof` hashes with
local/block/function numbering normalized through qualified target names.

This is additional closure-only lowering/proof work on warm hits as well as before
publication. `Report.Lowered:false` and the forbidden `Session.lower` hook mean
whole-workspace lowering was avoided; they do **not** mean zero reconstruction
work. Unrelated functions are excluded by selection. No separate microbenchmark
was run, so this report makes no per-request time or speedup claim for the proof.
The worker did not alter the verifier, proof, session production code, or reports.

After strengthening the representative fixture to put the counted block directly
inside a closure-value call, the private-format test passed (0.193s), followed by
one final run including baseline, all forgery controls and native execution:

```text
go test -mod=readonly ./internal/workspacecompiler -run '^TestClosureValueCache' -count=1
ok github.com/0disoft/fairyc/internal/workspacecompiler 14.240s
```

This final run includes Windows memory/disk/new-process low/optimized native
execution against the new verifier and final source fixture. The child-only test
skips when invoked without its parent environment and runs in the spawned child.
Linux native execution remains in the parent's batched lane. The unchanged
moduleinterface test evidence is recorded above; no redundant full-suite rerun
was needed. Formatting/whitespace checks pass for all three owned files.

No current worker-scope failure remains. Earlier failed commands and the corrected
fixture error above are retained as historical evidence. No assertion was weakened,
no cache/report flag was faked, no budget increased, and no commit/push/CI or
production/FY/lock/queue/main mutation was performed by this worker.

## Final structural receipt

After the parent added carrier-field source matching, unused-declaration
admission and duplicate-qualified-name refusal, the parent's ongoing trust suite
was observed read-only rather than duplicated. The shared
`.cache/closure-source-trust-attempt1.log` completed with:

```text
ok github.com/0disoft/fairyc/internal/workspacecompiler 2.999s
```

Only the new structural/unused-carrier coverage was then executed:

```text
go test -mod=readonly ./internal/workspacecompiler -run '^TestClosureValueCache(UnusedCarrier|SourceProofStructure)$' -count=1 -v
PASS: unused carrier memory/disk cold/warm and disk new-session;
      normal proof, context cancellation, forged carrier field,
      duplicate qualified function name
ok github.com/0disoft/fairyc/internal/workspacecompiler 0.305s
```

Unused-carrier warm readers retained `Outcome:all-hit Hits:2 Lowered:false`, with
the ordinary lowering hook forbidden. The proof's selection remains limited to
closure-bearing functions: it does not reconstruct a full cached package, and
an unused carrier declaration alone does not select scalar-only functions. The
selected closure reconstruction/proof work described above is still an explicit
validation cost. No additional native or duplicate forgery-suite run was made.

## Latest parent receipt: legacy fast path

Read-only confirmation of `.cache/closure-source-trust-final.log`:

```text
ok github.com/0disoft/fairyc/internal/workspacecompiler 2.691s
```

The parent identified this run as covering `TestClosureValueCacheRestore`,
`TestClosureValueCacheNoBadPublication`, `TestClosureValueCacheForgedRestoreRepair`
and `TestClosureValueCachePrivateFormats`. It was not duplicated by this worker.
The earlier NoBadPublication escape is historical and fixed; both bad-publication
refusal and actual-cache source repair are positive current results.

Read-only audit confirms the final source verifier checks context cancellation
before work, matches carrier field shapes to checked source, and has an explicit
feature-free fast path: when source declarations/locals/targets contain no closure
feature, it avoids expression walking, body reconstruction and function-symbol
indexing. Closure-bearing input still reconstructs only selected closure-bearing
functions for canonical proof, never the full cached package. That bounded
reconstruction remains a validation cost, not ordinary Session lowering.

Current evidence: baseline and exact formats pass in the parent's latest log;
eight publication refusals and eight forged-cache repairs pass; moduleinterface
exact/legacy-byte vectors and structural/unused-carrier controls pass in the
worker runs recorded above. The final full worker run (14.240s) includes Windows
low/optimized memory/disk/new-process native execution; its unchanged emitted/native
inputs remain valid evidence and were not rerun for the legacy fast path. Linux
native remains in the parent's batched lane. The parent's three-package vet was
still reported running when this receipt was requested; no vet pass is claimed.

Exact owned files remain the three paths listed at the top. All are new files;
production changes belong to the parent. No commits, pushes or CI were performed.

## Coordinator integration

The complete changed Go package/cache slice was copied to main after comparing
all 21 implementation, test and receipt files byte for byte. This supersedes the
package-cache deferral in the earlier runtime integration receipt; Fairy
executable closures and whole compiler CLI acceptance remain separate stages.

Final Windows checks on the identical implementation and test bytes:

```powershell
go test -mod=readonly ./internal/packageartifact ./internal/moduleinterface ./internal/workspacecompiler -count=1 -timeout=8m
go vet -mod=readonly ./internal/packageartifact ./internal/moduleinterface ./internal/workspacecompiler
```

All three complete packages passed: packageartifact 2.084s, moduleinterface
0.060s, workspacecompiler 65.940s. The test log remains at
`.cache/closure-packages-20261108/.cache/closure-package-final.log` in the main
checkout. Vet completed with exit 0 and no diagnostics. A direct child-only
invocation skipped as designed without its parent environment; the parent-driven
new-process and native tests ran in the complete package suite.

Stable helper identities, target/capture metadata and feature-exact private
envelopes now survive process and disk restoration. Fresh publication and warm
restoration compare closure-bearing bodies against current checked source,
including target choice, capture order and ordinary carrier field shapes.
Unrelated bodies are not reconstructed. No-closure input keeps the legacy fast
path and existing canonical bytes. This is a validation cost on closure-bearing
cache hits; no benchmark or zero-lowering-work claim is made.

The source contract ADR, public SDK/FFI ABI, database, dependencies, runner,
workflow and repository-hygiene configuration are unchanged. Internal package
signature metadata and private cache formats changed deliberately. No Linux
native execution, complete `go test ./...`, whole-compiler self-build,
distribution, push, release or hosted CI was run in this slice. Linux remains
batched as requested.
