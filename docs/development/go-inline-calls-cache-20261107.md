# Go inline calls: workspace cache receipt (2026-11-07)

Scope: [ADR 0548](../adr/0548-inline-same-module-call-graphs.md), Go workspace
cache integration only. Coordinator owns the ADR, Fairy implementation, lock
refresh, staging and commit. This worker owns only this receipt and
`internal/workspacecompiler/inline_calls_cache_test.go`.

## Verified coverage

The provider's public inline `value` calls forward-declared public inline `leaf`
in an immutable local initializer and its returned expression. The dependent is
a real second workspace package created with the existing `writeCacheWorkspace`
helper. The source uses subtraction so argument reordering changes its result.

- Memory and persistent disk: cold store, all-hit restore with ordinary lowering
  forbidden, and a new persistent session. Exact test-executable build identity
  comes from the existing `genericPersistentConfiguration` helper.
- Fixed identities: v1 leaf, the existing v2 immutable-local vector, and v3
  `bindings(call leaf(48,4), call leaf(local 0,2))`. Expected SHA-256 strings are
  independent literal vectors, not results from the implementation under test.
- Leaf-only edit: outer text and identity remain fixed, result changes 42 to 44,
  and both package keys miss. An unused exported inline body edit likewise
  invalidates both packages while retaining result 44. A private ordinary edit
  leaves one dependent hit and result 44; the refreshed provider's private
  helper separately executes to its new value 2. Retargeting only outer calls to the
  separately exported `spare` changes caller identity and result to 40. Removing
  `spare`'s inline annotation refuses the source before cache reuse or executable
  MIR publication.
- Ordinary/private/other-module targets and self/mutual recursion are refused.
  Every negative has a verified ordinary-source control, so an unrelated import
  or syntax error cannot masquerade as inline graph rejection.
- Exactly 64 distinct functions including the root form a repeated-edge shared
  DAG. Cold admission and warm cache reassembly are tested without executing its
  exponentially large runtime computation; 65 distinct functions are refused.
- A structurally valid forged inner leaf marker is injected through the existing
  lower seam into each real store. Reassembly must fall back, lower, replace both
  records, then restore without lowering. Disk repair is checked across fresh
  sessions, not merely within the writer's address space.
- Native restored products: memory in the parent process, disk in a fresh test
  process with lowering forbidden. A separate C observer checks once-only ordered
  external arguments, then the inner local/return calls, result 42 and empty
  streams. Windows MSVC C11 low and optimized profiles run serially; executable
  and AMD64 unwind inspection are included. Generated C, observer/runtime and
  native image SHA-256 identities are logged by each native probe.

## Actual commands and outcomes

Initial checkout and both execution checkpoints: main,
`1d59bbe887f9c0512eed4101b5a7c8f950b540af` (initially clean). The shared
checkout contains other workers' changes. The first prepared handoff did not
execute tests because the export-class resolver connection was absent. The
coordinator subsequently connected `hir.InlineModuleResolver(m)` once per module
in `internal/inlinebody/hash.go`; actual execution then resumed. No test was run
against the missing connection. These fixtures contain no argument labels.

Both commands below ran on Windows AMD64 with `GOMAXPROCS=1`, `-p=1` and
`-parallel=1`; native probes are serial and do not use `t.Parallel`.

```powershell
$env:GOMAXPROCS='1'
go test -mod=readonly -p=1 ./internal/workspacecompiler -run '^TestInlineCallsCache' -count=1 -parallel=1 -v -timeout 3m
```

**First execution failed overall (package 3.089s; command wall time 8.530s).**
Eight negative subtests hit this worker's incorrect sentinel:
`restored inline call graph invoked ordinary lowering`. Those were annotation
removal in memory/disk, the five closed-target cases, and the 65-function case.
The positive cache sequences and fixed identity vectors reached their expected
results before the annotation-removal sentinel failed. The 64-function DAG,
inner marker repair in both stores, and all four native probes passed.

The existing `Session.Check` intentionally dispatches non-ready semantic analysis
through `session.lower`; `compiler.LowerAnalyzed` returns the diagnostic result
without producing executable MIR. The test incorrectly prohibited entering that
dispatcher for rejected source. Only the negative assertions were corrected:
they now require `OutcomeNotEligible`, unsupported diagnostics, unverified
result, no MIR functions, zero hits/stores and `Lowered=false`. All real warm-hit
tests still install the fatal lower sentinel. No production code was changed.

```powershell
$env:GOMAXPROCS='1'
go test -mod=readonly -p=1 ./internal/workspacecompiler -run '^TestInlineCallsCache(Invalidation|RefusesClosedTargets|GraphBound)$' -count=1 -parallel=1 -v -timeout 3m
```

**Corrected selection passed (package 0.852s; command wall time 6.163s).**
It covers both complete invalidation sequences, all five controlled refusals,
and the 64/65 DAG cases. Native/repair tests were not repeated: their source
inputs and production implementation were unchanged by the assertion-only fix.
Every substantive selected test therefore has passing execution evidence across
these two commands; there is no claim that the first whole selection passed.

The first command's `TestInlineCallsCacheInnerMarkerRepair` passed in 0.20s.
Each store reported `Hits=2`, `ReassemblyFallback=true`, `Lowered=true`,
`Stored=2` during repair, then an all-hit restore without lowering. The disk
repair reader was a fresh session. `TestInlineCallsCacheNativeRestore` passed
in 2.07s, including the fresh child in 0.96s. The standalone child helper was
expectedly skipped in the parent selection; its subprocess invocation executed
both native profiles and printed `inline-calls-new-process-disk-42`.

Observed invalidation results in each store:

| Change | Hits | Misses | Result | Additional observation |
| --- | ---: | ---: | ---: | --- |
| Cold | 0 | 2 | 42 | Both packages stored |
| Unchanged/warm | 2 | 0 | 42 | Lower dispatcher forbidden |
| Leaf only | 0 | 2 | 44 | Outer marker unchanged, leaf marker changed |
| Unused inline export | 0 | 2 | 44 | Existing all-export class invalidated dependent |
| Private ordinary helper | 1 | 1 | 44 | Refreshed helper separately returned 2 |
| Outer call retarget to spare | 0 | 2 | 40 | Caller marker changed |
| Remove spare inline annotation | 0 | 0 | Refused | No executable MIR or stored artifacts |

Each successful edit was followed by an all-hit restore with lowering forbidden.
Persistent disk additionally restored the initial program in a new session.

Final targeted vet passed, 5.283s, no diagnostics:

```powershell
$env:GOMAXPROCS='1'
go vet -mod=readonly -p=1 ./internal/workspacecompiler
gofmt -l internal/workspacecompiler/inline_calls_cache_test.go
```

`gofmt -l` produced no output. An earlier pre-connection vet also passed in
6.694s; final vet above supersedes that preparation checkpoint. Both owned files
pass `git diff --no-index --check` against an empty temporary file. The initial
hygiene wrapper incorrectly treated normal no-index difference exit 1 as a
failure despite empty diagnostics; the corrected wrapper accepts that status
and separately rejects whitespace diagnostics. This was a command-wrapper error.

## Exact identities

Production source SHA-256 values were read before the first execution and after
the corrected selection and were identical at both checkpoints:

```text
internal/hir/inline.go 0880822773a1bfc0c88c4bd1356a76831416f02ed866c8a8a6864375432e81c1
internal/semantic/check_function.go c509175affec6b7c3369185daf98f3f644fcef6dda2f28f440ebe7cf5af7b00d
internal/semantic/check_calls.go 5604bb429135a74741a61ab53cab9594800fc47d2272e892b74192246139f857
internal/semantic/inline.go 54feb93a38573eec32234da5ee2d27ca3d33a50db8c2bb07a5e5925557b90c19
internal/semantic/typecheck.go c13b1f63296b63037ee472e714422fc626cf18d2159666c33293a13c35940194
internal/mir/inline.go 6bffa62274cbc0e2a49e6b13de571688d1707366221862dd77784b74df0a5e6d
internal/mir/verify.go 0d6181c5e598f1d044b07bc2e748be3a810a96c0e1a9b98d4cdd299f624f8e7b
internal/inlinebody/hash.go f3cc91f232e59744cda784483ec71c54e724a7e3ae1173a220231a7b7f4b02de
```

The pre-connection `hash.go` was
`3691a5b79db5b5957ed92081b5fc1ee4af59219703a5ef830b1a7b14c960d458`;
that snapshot had no runtime acceptance claim. Test-file SHA-256 before the
assertion fix was
`e46c8c3198b7414d52043efd213a11861765dbab63a9a320437d23398bd3e881`;
final test-file SHA-256 is
`d13f23aecf0c3906d768d6576aaabe3113d4c974264894886b8502c83b5cd4aa`.
The exact test-executable digests used for persistent build identities were:

| Execution | Executable SHA-256 |
| --- | --- |
| First selection, including native parent/child | `9fd5d9fbc17e84461b1cffc056fcad86755340bdfc0b193407d0fb9775a53daa` |
| Corrected negative/cache selection | `abd2e1306513956562a8964cf9cfb2727eb4da0820650d0916fc12bc8c5387c6` |

Baseline provider source SHA-256:
`9f1a435f4ea611cc5e454ef9313d3ad557e5978bba1d39f9c2523eff93857224`.
Ordinary consumer source SHA-256:
`79d05be86cec146a5b3c5e206345c886a61772ad292dd61f779f7694a962a541`.

Native MSVC **14.51.36231**, C11; all four executions returned **42** with empty
stdout/stderr and passed executable/unwind inspection. Native helper bodies and
fixture inputs were unchanged by the negative-assertion fix. Generated C SHA-256
was `81b468dccba33f8833d725e3a5ed47f9abc5e691682d5a4fa05f54eeba9bce76`
for memory and
`450103491a9646a37336afc6730c2d92be69511843eed0e0e1a3b66094d39665` for
fresh-process disk restoration. Observer/runtime support SHA-256 was
`c42570083357fdbc3a8dcf6a03caef5ad7e162e44acd84cd59ae350ea0c73048`.

| Restored product | Executable image SHA-256 |
| --- | --- |
| Memory low | `91f2024ba790bc4845e0fbe7505711401cea694a455ca59e44ba6ed20f9457b6` |
| Memory optimized | `cc5dc6663d924825a446c4fd2a13ed5008bf6011b23bbc2159f77f0791a1e0a1` |
| Fresh-process disk low | `ea0340d9cb30a1849ba1ca8305864f4e1156d27d5d37e307c286f88ace99bcd8` |
| Fresh-process disk optimized | `3e6a3e30fe4e4abe0280b1d7622163c4874647ded6462b3fa62e8f69f05c1ff3` |

## Scope and deferred validation

No unresolved failure remains in this worker's selected cache/native coverage.
Native Linux, C17 native consumers, full suites, broad stress loops, Fairy
CLI/artifact parity, hosted CI, release and deployment were not run here. The
coordinator runs Go/compiler and Fairy identity separately; their outcomes are
not claimed as this worker's validations. The bounded same-module contract does
not establish support for broader inline bodies or cross-module inline targets.

Only this receipt and `internal/workspacecompiler/inline_calls_cache_test.go`
were changed by this worker. No production, other documents, locks, git index,
commits or remote state were changed. ADR/index/queue integration and commit
remain with the coordinator. No API, database, dependency or runner change.
