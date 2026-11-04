# Windows Fairy Atlas verified-type projection, 2026-11-05

`project_atlas_types` projects the checked cleanup-IR type graph into existing
Atlas structured type references. Nominal names use original source modules and
exact package coordinates; import aliases and producer-local type IDs do not
become Atlas identities. Builtins, C ABI scalars, nominal structs/resources/enums,
shared/mutable borrows, scalar and nominal slices, raw pointers (including the
admitted nested opaque pointer), Option/Result and fixed arrays are represented.
String, C string and scalar-slice references preserve Go's separate borrow wrapper.

The projection charges the existing shared work meter and limits type depth to
64, matching the candidate Atlas reader. Invalid IDs and missing, ambiguous or
invalid nominal coordinates reject without returning a partial buffer. SDK
specialization identities remain rejected pending their separate source-fact
projection. This is a common type port, not complete Atlas source-family admission
or integration into the command producer.

Executed on Windows amd64:

```powershell
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestCandidateAtlasTypes$'
```

Passed in 86.55 seconds, MSVC low and optimized. Each native executable freshly
loads and verifies a two-package locked workspace with PATH empty, then compares
all 16 ordinary source function signatures and their ordered structured parameter
types with independently compiled Go Atlas v3 declarations. The native probe
walks every checked ordinary frame, not only the callable/import table. Zero and
past-end IDs, exhausted work and missing nominal coordinates reject in both
profiles. Workspace bytes remain unchanged and no absolute root path appears in
the emitted type facts.

Earlier runs exposed a missing test module closure, integer inference and a
forwarded static-borrow restriction, then incorrect reference spelling and a
probe that enumerated imported callables only. They are failed attempts, not
additional passing cases. The final run above uses the corrected complete input.

Linux native execution remains batched. SDK/generic Atlas production, full CLI
source-family acceptance, self-build equality and hosted CI remain separate.
No public API/SDK ABI, DB, dependency, repository hygiene or runner changes;
no push, release or deployment.
