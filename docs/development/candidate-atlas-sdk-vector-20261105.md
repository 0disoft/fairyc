# Fairy Atlas SDK vector production, 2026-11-05

Scope: complete the remaining authenticated SDK Vec source-production path,
then connect the existing impact/MCP/editor consumers. This is part of the full
implementation queue, not a replacement objective. The current impact/MCP Vec
refusal remains until actual producer and current-source acceptance pass. Ordinary
SDK command evidence belongs to `docs/ops/windows-candidate-atlas-sdk-20261105.md`.

## Ground truth

`internal/cli/atlas_sdk_vec_oracle_test.go` now prepares two real Go v3 workspaces:
an unused `std::vec` import and a root using `empty/push/at<i32>`. The SDK's fixed
ordinary view signatures already require Vec types, even in the unused case.

- Unused import: exactly ten SDK implementations, `Vec<T>` and `take_slot<T>`
  for each of u8, i64, i32, u32 and usize. There are no root-requested generic
  function implementations.
- Root use: the same ten plus `empty<i32>`, `push<i32>`, `at<i32>` and
  `write_slot<i32>`. Original SDK declarations and generic interface hash are
  unchanged by usage.
- Concrete Vec facts retain six physical fields: one mutable void pointer and
  five usize values. The resource specialization owns a separate cleanup body;
  its unsafe/call facts and `at`'s borrow facts remain present in the Go oracle.
- Root calls retain original SDK declaration keys, exact `g0_` implementation
  endpoints and source spans. Every partition preserves canonical v3 bytes
  through codec reconstruction.

These are Go oracle facts, not Fairy/native Vec producer acceptance. The richer
Go relation facts must never become false exact-coverage claims in the candidate;
uncollected relations retain the existing explicit completeness omissions.

## Implementation sequence

- [x] Establish the exact unused/used Go specialization sets and cleanup/type/call
  expectations, while retaining the existing missing-storage-admission refusal.
- [ ] Reproduce the candidate private-API failure for both cases. Static tracing
  suggests `project_atlas_instances::collect_mode` refuses kind-2 Vec cleanup
  frames with code 101. This is not yet a native reproduction. An unused import
  must not be assumed to have zero cleanup or type instances.
- [ ] Retain authenticated SDK Vec source mode separately from user aggregate
  mode. Admit only exact SDK source bytes through the existing source authority;
  map/cache headers, module names and selected symbols cannot grant authority.
- [ ] Connect original Vec template headers, kind-13 nominal type/physical fields,
  existing source specialization identities and cleanup frames through
  `project_atlas_declarations`, `project_atlas_types`, `project_atlas_instances`
  and `project_atlas_aggregate`. Preserve scalar ordering and current-source
  expectations; do not publish compiler-created clones as source declarations.
- [ ] Verify produced v3 partitions/current admission/join against the independent
  Go oracle, including exact retained facts, rejected provenance and explicit
  unavailable coverage. Then remove the command's Vec guard and exercise actual
  impact/MCP/edit commands, keeping SDK declarations noneditable.
- [ ] Complete SDK/user-aggregate combinations and final authored-main replay.
  Linux execution remains batched; self-build/distribution/hosted gates remain
  part of the original queue.

## Windows evidence

```powershell
go test -mod=readonly -count=1 -failfast -timeout 5m -v ./internal/cli -run '^TestAtlas(PrepareSDKVectorWorkspace|SDKVectorSpecializationOracle)$'
```

Final command passes in 2.754s, including both unused/instantiated controls and
the existing SDK storage-authority removal regression. No native Fairy build or
Linux execution occurs in this command. Scoped vet, formatting and diff checks
are recorded with this test-only commit.

The initial raw in-memory equality assertion failed after a successful v3 decode.
The final assertion checks exact canonical wire roundtrip plus explicit semantic
facts, rather than freezing in-memory empty-container representations. The first
attempt is recorded as failed; no product codec defect was established by it.

No public language/CLI, SDK/runtime ABI, map schema, DB, dependency, default
timeout, repository hygiene or runner changes. No push or CI dispatch.
