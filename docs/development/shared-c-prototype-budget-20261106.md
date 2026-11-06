# Shared C prototype header budget, 2026-11-06

Actual Atlas CLI integration after Complete-bound borrow publication produced a
2,099,763-byte common header. Both existing native project drivers cap headers at
2,097,152 bytes, so the Windows invocation rejected before native compilation.
The limit, warnings, timeouts, partition policy and runtime remain unchanged.

Shared ordinary function declarations already omit definition-local parameter
names. Apply the same policy to nested cleanup and foreign function declarations:
retain exact result/parameter types, pointer qualifiers, linkage and symbol names;
omit only names unused by C declarations. Cleanup definitions keep `fy_value`.
Single-translation-unit emission retains its previous spelling.

For the actual compiler, cleanup declaration names cost 1,928 bytes and foreign
declaration names cost another 1,470 bytes. The final common header is 2,096,365
bytes: 3,398 fewer bytes, 787 bytes inside the original bound. Body units remain
26, largest 2,096,243 bytes, and runtime remains 153,115 bytes. These are current
compiler-input measurements, not a general compiler speed or size claim.

The focused partition replay passes in 5.587s on Windows: deterministic build
identity, unchanged small/streaming emission, function-boundary budgets, callback
identity, ordinary callbacks and nested cleanup declarations. Eight native
MSVC executions span low/optimized. The new nested Box/Leaf case independently
requires the parent-then-child C observation 9,4 before returning 42. It also
checks named legacy declarations and named definitions against anonymous shared
cleanup/foreign declarations. Scoped C-backend vet and diff checks pass.

Command:

```powershell
go test -mod=readonly -count=1 -timeout 3m -failfast -v ./internal/cbackend -run '^(TestPartitionedDropHelperPrototypes|TestPartitionedCallbackIdentity|TestPartitionedOrdinaryCallback|TestPartitionedBuildIdentityAndExecution|TestSmallBuildMatchesStreamingUnit|TestPartitionIncludesInByteBudget)$'
go vet -mod=readonly ./internal/cbackend
```

Logs under `.cache/`: `atlas-borrow-shared-prototypes-final-20261106.log`;
`atlas-borrow-cli-optimized-enabled-20261106.log` (original rejection);
`atlas-borrow-cli-optimized-final-20261106.log` (cleanup-only reduction still
683 bytes over); `atlas-borrow-cli-optimized-prototypes-20261106.log` (final
header and separate complete CLI replay). The preceding wider partition replay
passes in 41.544s before the foreign-name correction and is not its final proof.

Actual Atlas command acceptance is recorded separately when its run completes.
Native Linux stays batched; no push or dispatch. No public CLI/API, SDK/runtime
ABI, DB, dependencies, repository hygiene or runner changes.
