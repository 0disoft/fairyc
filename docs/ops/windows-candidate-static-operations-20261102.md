# Windows Fairy candidate static operation receipt, 2026-11-02

- Sources: `d2e967ff` (declarations), `95d765a3` (ordinary provider calls),
  `23f59fa5` (provider-bound identities and native pipeline), and `605bb622`
  (root diagnostics and ordinary-provider-body rejection).
- Host: Windows amd64, Go 1.27.1, MSVC 14.51.36231.
- Scope: [candidate static operations v1](../architecture/candidate-static-operations-v1.md).

## Executed and passed

| Probe | Evidence |
| --- | --- |
| Declaration and provider admission | Native Fairy low/optimized execution covers all thirteen scalar families, zero/four parameters, malformed attributes, unsafe/orphan/wrong-signature providers, duplicate tuples and absent-tuple selection. |
| Ordinary call and rejection | Low/optimized execution retains one ordinary call per i32/i64 operation instance with the exact provider/type. A provider returning bool for i32 is rejected by ordinary return typing (E1009). |
| Go identity oracle | The real Go `PreparePackages` API supplies original sources, root IDs, ordered specialization IDs, bindings, dependencies and exported template closures. Fairy independently parses/plans four cross-package graphs: base, selected-provider edit, unused-provider edit and unrelated-private edit. Both native profiles agree with Go; provider edits invalidate dependent identity, unrelated private edits preserve it. |
| Work and missing provider | The identity replay starts catalog work at 1 MiB to verify separate metadata budgets. Missing-provider planning retains the original importing root span and maps G2004 to E4502. |
| Artifact restoration and native C | Native Fairy low/optimized compilers encode/decode and restore the complete current-source artifact, then emit partitioned C. MSVC C11 low/optimized consumers run each output: four executions return 42 with empty streams. Thirteen independent C observers supply scalar boundaries/floating values; i32/i64 provider selection, zero parameters and four-argument order also execute. |
| Go reference | Focused compiler/interface execution and rejection tests, plus workspace memory/fresh-session disk invalidation tests pass. The compiler helper executes Windows C11/C17 low/optimized and emits both reference target profiles. |
| Candidate source | Isolated `check` and `check --locked` pass: two instances, 582 source files and 7,313 MIR functions. The existing ignored Go bootstrap performs these source checks. |
| Hygiene | Focused Go vet and `git diff --check` pass. No generated artifacts are staged. |

Focused reproduction commands from the repository root:

```powershell
go test -mod=readonly ./internal/cli -run '^TestCandidateStaticOperationDeclarations$' -count=1 -v
go test -mod=readonly ./internal/cli -run '^TestCandidateStaticOperation(Identity|Calls)$' -count=1 -v
$env:FAIRYC_RUN_PARITY_INTEGRATION = '1'
go test -mod=readonly ./internal/cli -run '^TestCandidateStaticOperationPipeline$' -count=1 -v
go test -mod=readonly ./internal/compiler ./internal/workspacecompiler -run '^TestStaticInterface' -count=1
go vet -mod=readonly ./internal/cli
git diff --check
```

The complete pipeline uses private partitioned compiler APIs and a 16 MiB host
stack, not a full candidate CLI build. Its two candidate profiles and four C
consumer executions passed in 408.92 seconds. Follow-up metadata budget changes
were checked by the focused identity replay; they do not change canonical facts
or generated C. The declarations/calls probes and the identity replay are separate
from the opt-in complete pipeline.

## Deferred and unchanged

Native Linux remains queued in `ordered-work-20261031.md`; the private Windows
host harness needs a GNU replay before Linux execution is claimed. Candidate
C17, interpreter execution, whole-package persistent cold/warm acceptance, full
`go test ./...`, hosted CI, self-hosting and clean-machine distribution were not
run. Go cache tests are not counted as Fairy candidate persistent-cache evidence.

Source-of-truth changes are ADR 0495, the candidate architecture page, the ordered
work/Linux queue and the package/CLI/SDK index. This receipt records bounded
Windows prototype evidence, not SDK/candidate promotion.

No public Go/CLI API, SDK ABI, DB, dependencies, cache wire version, runner or CI
configuration changes. No push, deployment, publication or workflow dispatch.
