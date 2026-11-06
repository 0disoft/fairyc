# Fairy Atlas SDK vector production, 2026-11-05

Scope: complete the remaining authenticated SDK Vec source-production path,
then connect the existing impact/MCP/editor consumers. This is part of the full
implementation queue, not a replacement objective. Impact/MCP Vec admission now
has actual producer, current-source and Windows command acceptance. Ordinary
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
- Pop use: fourteen implementations too, with `pop<i32>` instead of `at<i32>`.
  Here `take_slot<i32>` also has an ordinary callable frame. Its cleanup inline
  observation must resolve to that same implementation without duplication.
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
- [x] Reproduce the candidate private-API failure. After kind-2 Vec cleanup
  admission, the native producer retained five of ten planned implementations:
  the compiler lowers cleanup's five `take_slot<T>` calls to checked storage
  intrinsics, with no separate callable frames. The producer must retain their
  actual inline observations, not assume source-plan nodes prove execution.
- [x] Retain authenticated SDK Vec source mode separately from user aggregate
  mode. Admit only exact SDK source bytes through the existing source authority;
  map/cache headers, module names and selected symbols cannot grant authority.
- [x] Connect original Vec template headers, kind-13 nominal type/physical fields,
  existing source specialization identities and cleanup frames through
  `project_atlas_declarations`, `project_atlas_types`, `project_atlas_instances`
  and `project_atlas_aggregate`. Preserve scalar ordering and current-source
  expectations; do not publish compiler-created clones as source declarations.
- [x] Finish the low/optimized produced-v3/current/join acceptance against the independent
  Go oracle, including exact retained facts, rejected provenance and explicit
  unavailable coverage. The full unused/used/pop matrix passes both profiles;
  the [Windows receipt](../ops/windows-candidate-atlas-sdk-vector-20261105.md)
  records the exact scope and remaining command gate.
- [x] Remove the command's Vec guard after that acceptance and exercise actual
  impact and read-only MCP commands. Both Windows profiles pass; SDK declarations
  remain noneditable and MCP retains no write tool.
- [x] Exercise actual Vec user-body preview/apply, including newly requested
  bool element specializations, without changing SDK declaration authority.
- [ ] Complete SDK/user-aggregate combinations and final authored-main replay.
  Linux execution remains batched; self-build/distribution/hosted gates remain
  part of the original queue.

The combined user-resource-cleanup/SDK Vec case now has actual production-loaded
map and root/SDK impact acceptance in both Windows profiles; see
`candidate-atlas-sdk-aggregate-20261107.md`. Existing bounded combination admission
was already present. This further coverage retains exact fourteen SDK/three user
implementations and current source/provenance; it does not close the final
authored-main or broader combination/consumer gates in the row above.

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

The private producer/current/join follow-up is recorded in
`docs/ops/windows-candidate-atlas-sdk-vector-20261105.md`. It does not complete
the command and source-family combination items above.
