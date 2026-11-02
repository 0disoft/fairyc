# Windows Fairy candidate generic transport receipt, 2026-11-02

- Sources: `1e2d5143` (admission and diagnostics), `55981317` (C typing,
  deferred values and artifact restoration), `8d82d119` (22-case fresh-Go
  signature oracle replay, replacing the previous unsupported-scalar expectation).
- Host: Windows amd64, Go 1.27.1, MSVC 14.51.36231.
- Scope: [candidate generic transport v1](../architecture/candidate-generic-transport-v1.md).

## Executed and passed

| Probe | Evidence |
| --- | --- |
| Argument and concrete type admission | Actual Fairy low/optimized execution retains source handles, materializes the seven direct values, forwards arguments, resolves all thirteen concrete scalar names and rejects invalid domains. |
| Closed composites | Seven scalars x six composite signature forms reject through direct type specialization without publishing partial nodes, at both profiles. |
| Signature diagnostics | Actual Fairy low/optimized execution checks stale-source and exact-work limits. The fresh Go CLI supplies a 22-case UTF-8/CRLF oracle; features and original source spans agree for direct, repeated, forwarded, pair, unused and composite cases. Ordinary argument type errors stay distinct from unsupported signatures. |
| Complete restoration and native C | Both native Fairy compiler profiles restore the whole-project artifact against current sources, retain public scalar signature names, and emit partitioned C. Each output runs under MSVC C11 low/optimized: four executions return 42 with empty streams. |
| Value behavior and rejection controls | Independent private C observers supply INT8_MIN, INT16_MIN, signed isize below i32 minimum, UINT16_MAX, UINT64_MAX, 1.25f and -3.5. Values cross two generic functions, mutable local copy/reassignment and deferred generic arguments. Each compiler image also rejects 56 operation/composite/direct-extern-C controls. |
| Go reference | Direct generic signatures pass private, explicit-file and package-aware checks. The repr(C) observer fixture passes C11/C17 low/optimized (four exit-42 executions). Seven types pass memory and fresh-session disk cache checks. |
| Candidate source and hygiene | Isolated candidate `check` and `check --locked` pass: two instances, 579 files, 7,268 MIR functions. Scoped Go vet and diff checks pass. |

The candidate C observers call private generated helpers. They are separate from
the Go repr(C) fixture and do not certify candidate aggregate FFI support. The
candidate proof uses native private compiler APIs, not a full candidate CLI rebuild.

Commands executed from the repository root (the final two admission families were
also rerun separately after widening composite controls and adding the Go oracle):

```powershell
$env:FAIRYC_RUN_PARITY_INTEGRATION = '1'
go test -mod=readonly ./internal/cli -run '^TestCandidateGenericTransportAdmission$' -count=1 -v
go test -mod=readonly ./internal/cli -run '^TestCandidateGenericTransportAdmission/generic_scalar_arguments$' -count=1 -v
go test -mod=readonly ./internal/cli -run '^TestCandidateGenericTransportAdmission/generic_signature_diagnostics$' -count=1 -v
go test -mod=readonly ./internal/cli -run '^TestCandidateGenericTransportPipeline$' -count=1 -timeout 15m -v
go test -mod=readonly ./internal/compiler -run '^(TestGenericSignature|TestBroaderGenerics)' -count=1 -v
go test -mod=readonly ./internal/workspacecompiler -run '^TestBroaderGenericsScalarCache$' -count=1 -v
go vet -mod=readonly ./internal/cli
git diff --check
```

The source check used the existing ignored bootstrap and an isolated copy of the
current candidate source. No checked-in generated artifact owns these results.

## Deferred and unchanged

Native Linux stays in the maintainer's batched queue; the private candidate harness
currently selects Windows MSVC and needs a GNU host replay for Linux. Candidate
C17, interpreter transport, candidate whole-package fresh-session disk-cache
acceptance, full `go test ./...`, hosted CI and clean-machine distribution were not
run. Focused Windows C11 and source-restoration evidence does not complete them.
Static operation interfaces and copied local closure candidate parity remain next
separate slices.

No public CLI/API, SDK ABI, DB, dependency, version, runner or CI configuration
changes. No push, deployment, publication or workflow dispatch. Changes are scoped
source, tests, oracle and documentation; cache/build output remains ignored.

## Later evidence

This 2026-11-02 receipt remains the historical source and native evidence above.
The [2026-11-03 full CLI and persistent-cache receipt](windows-candidate-generics-cli-cache-20261103.md)
supersedes its private-harness-only CLI limitation with a full candidate CLI pass
at both compiler profiles and a whole-package persistent-cache pass at both cache
profiles. The C17, seven-type interpreter-operation, native Linux, full
`go test ./...`, hosted CI, self-hosting and clean-machine limits in this
historical receipt stand.
