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

## Complete-source declaration projection

`project_atlas_schema::prepare` rebuilds the signature catalog from the retained
source closure, then compares exact source bytes, owners, imports and package
coordinates with the same successful Complete owner's provenance. This includes
unused foreign signatures that the executable callable table does not inventory.
The type port now also accepts bounded source-resolved records; detached records
alone are not evidence of source freshness or cache admission.

The declaration writer preserves ordinary function/foreign signatures, exact
identifier-only parameter spans, aggregate field and enum payload order/spans,
public/unsafe/C representation flags, C ABI and link names, canonical effects,
and empty/non-empty resource cleanup classification and effects. It retains the
existing Atlas wire shape. Generic and owner-qualified declarations remain
explicitly rejected here; SDK specialization names remain a separate task.

Executed:

```powershell
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestCandidateAtlasSchema$'
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestCandidateAtlasTypes$'
go vet -mod=readonly ./internal/cli
git diff --check
```

The schema test passes in 75.94 seconds on Windows MSVC low/optimized. Each
profile compares the exact canonical JSON bytes of all 27 source declarations
against independently compiled Go Atlas v3 output. The fixture includes two
unused C declarations (one with an explicit link name), a record with arrays,
tags and a pointer, enum payloads, zero-field and nonempty cleanup resources,
and a function with unordered declared effects. Source modules, parameters,
fields, variants and type identities are compared, not just symbol counts.

Six additional invocations per profile reject absent coordinates, a source
overlay differing only by a trailing newline from the Complete owner, exhausted
work, past-end module/declaration IDs, and an unsupported module-header row.
All preserve workspace bytes; native PATH is empty and no host root leaks.
After adding the shared record entry, the original checked-cleanup-IR type test
also passes again in both profiles (67.97s), retaining all 16 signature and four
rejection comparisons. Scoped vet and diff whitespace checks pass.

An earlier run rejected a fixture's fixed-array enum payload in the Go oracle;
that is an existing unsupported source form, not an admitted extension. Another
run exposed parameter spans covering the complete typed declaration instead of
the HIR identifier. The final passing run uses the corrected declared subset
and exact spans; failed attempts are not included in passing counts.

This completes the common declaration port, not the broader production command.
Producer inventory/order, callable-only function counts and summaries, external
call completeness, SDK/generic facts and full command replay still need their
own integration. The existing ordinary-source CLI path is unchanged by this port.
