# Windows Fairy Atlas generic declaration identity, 2026-11-05

`project_atlas_binders` validates canonical declaration shape, ordered unique
display binders, positional type references, permitted declaration kinds and
aggregate self types before the existing SymbolId encoder consumes arity.
Generic function, struct and resource declarations preserve their original key;
generated implementation names never replace it. Renaming display binders does
not change the ID. Nominal generic arguments and nested borrow/slice/pointer,
array and Option/Result types share the existing 64-level and work limits.

The port follows the existing SymbolId tuple, ADR 0386's positional binders and
ADR 0387's later aggregate-header admission, including
`docs/atlas/fymap-transport-v3.md`. No identity, map, language or runtime schema
changes. It is a common declaration/identity port: it does not admit complete
generic maps, authenticate current source, collect specialization bodies or
broaden the production command. The ordinary semantic reader explicitly rejects
`type_parameters`, including unused binders that are absent from all type uses.

## Windows evidence

The complete source oracle contains generic functions (one and two binders),
nested generic structs, a generic resource, an unused function with composite
signature types, ordinary functions and used scalar specializations. Original
declaration JSON and expected IDs come from independent Go v3 compilation.

`TestCandidateAtlasBinders` runs 31 declaration/identity cases plus an
exhausted-work refusal in each MSVC profile. Positive cases include renamed
binders and the last admitted type depth (63); invalid index 1 for arity 1,
leading-zero/negative/overflow indices, duplicate/invalid names, wrong member
types, unexpected fields, forbidden foreign/enum templates, wrong aggregate
self names/binders, mutability/length and depth 64 reject. Every mutated map is
rechecksummed and checked with Go `DecodeV3` before the native comparison.
Depth-vector IDs are independently recomputed with Go `FromDeclarations` and
their symbol table is re-sorted, so canonical ordering cannot mask the bound.

Executed:

```powershell
go test -mod=readonly -count=1 -timeout 6m -v ./internal/cli -run '^TestCandidateAtlas(Binders|Wire|Validation|Current|Impact)$'
go test -mod=readonly -count=1 -timeout 6m ./internal/cli -run '^TestCandidateAtlas(Binders|Validation)$'
go vet -mod=readonly ./internal/cli
git diff --check
```

The first group passes (83.299s), with both profiles of the existing wire,
semantic validation, independent current-source and reverse-call traversal
checks. The final Binders/Validation group passes (16.090s) after adding the
depth boundary and explicit ordinary-reader generic refusal vectors. Earlier
boundary preparation failed on unsorted replacement symbol IDs; that test
fixture error was corrected before the final run and is not passing evidence.

Native Linux remains batched. Complete generic/specialization map admission,
current-source production, SDK and full CLI acceptance, consecutive self-build
equality, clean-machine distribution and same-commit hosted CI remain open.
Stable public API/SDK ABI, DB, dependencies, repository hygiene and runner
configuration are unchanged. No push, release or deployment.

## Separate v3 specialization transport

The transport reader now shapes v3 function, aggregate and resource-cleanup
specializations, including recursive type arguments, exact call specialization
IDs and the optional specialization count. Member order/types remain strict;
unknown, missing, duplicate-body and null-body forms reject. Optional empty
arrays, empty strings and zero specialization counts reject instead of accepting
a noncanonical encoding. V1 refuses generic fields; v2 remains unsupported.

This is not a semantic promotion. The existing complete validator explicitly
refuses specialization records/counts and specialization calls, as well as
generic declarations, until those independent checks are implemented. A
well-shaped checksummed map with a false implementation ID can pass transport,
but cannot be admitted. Current-source expectations remain independently owned.

`TestCandidateAtlasGenericWire` passes in Windows MSVC low/optimized (9.816s):
used/unused two-package Go maps and the controlled semantic contradiction retain
every original byte. The used fixture requires function implementations, nested
aggregate implementations, empty cleanup, two concrete resource cleanups, borrow
facts and helper specialization call IDs; fixture assertions require empty
cleanup and exactly two executable cleanup records. Nine structural/canonical/version
mutations plus byte/work overruns reject. All original maps and mutations are
checked against the Go codec before native replay. Each transported generic
document is also submitted to the complete reader and must be refused there.

```powershell
go test -mod=readonly -count=1 -timeout 6m -v ./internal/cli -run '^TestCandidateAtlasGenericWire$'
go test -mod=readonly -count=1 -timeout 6m ./internal/cli -run '^TestCandidateAtlas(Wire|Validation|Current)$'
```

The legacy Wire/Validation/Current group passes in both profiles (27.207s).
The new orphan-call vectors compare v1 rejection at decoding and v3 rejection
at complete-workspace joining: a single v3 package does not guess another
package's implementations. Native ordinary admission refuses either form while
generic joins remain unimplemented. Early attempts needed the probe's semantic
module closure and an explicit usize length; an oracle assertion was also moved
to the correct join layer. Those attempts are not passing evidence.

Generic declaration/source production, semantic specialization substitution and
ownership, full joins, SDK/CLI acceptance and final acceptance remain open.
No new schema, stable public API/SDK ABI, DB, dependency, hygiene or runner change;
no push, release or deployment. Native Linux remains deferred.

## Explicit v3 semantic specialization reader

`project_atlas_validate::validate_generics` is now the explicit v3 semantic entry.
The ordinary `validate`/current-source path remains closed to generic declarations
and specialization records. The new entry returns a partition that retains its
specialization table, without creating current-source authority or widening CLI
production. V1/v2 are not guessed or upgraded.

The shared declaration/binder and SymbolId checks apply before implementation
checks. `project_atlas_specializations` enforces canonical ordered `g0_` IDs,
package-local template ownership, exact arity, the admitted scalar argument set,
unique declaration/argument bindings and compatible function/aggregate/resource
cleanup bodies. Aggregate headers match source metadata and recursive type
substitution; fields contain their original spans and remain inside the original
declaration. Cleanup observations remain inside the source resource. Exact local
calls require the matching implementation; cross-package existence still belongs
to the complete workspace join. Ordinary executable summaries cannot stand in for
template bodies. Relation counts include concrete function and cleanup facts.

Template-body coverage stays unavailable; specialization coverage stays exact.
Coverage counts may exceed the local inventory because they describe the source
snapshot, but cannot undercount it. IDs have format/ownership checks here, not
compiler-provenance authentication; an independent current-source reader/producer
must establish those facts. No checksum is treated as authorship.

Executed and passed on Windows MSVC low/optimized:

```powershell
go test -mod=readonly -count=1 -timeout 6m ./internal/cli -run '^TestCandidateAtlasGenericValidation$'
go test -mod=readonly -count=1 -timeout 6m ./internal/cli -run '^TestCandidateAtlas(Binders|GenericWire|Wire|Validation|Current|Impact)$'
go vet -mod=readonly ./internal/cli
gofmt -l internal/cli/candidate_atlas_generic_validate_test.go
git diff --check
```

The final generic semantic suite passes (21.062s); unchanged binder/wire/ordinary
validation/current-source/impact regressions pass (119.684s). Used/unused Go v3
maps, admitted scalar choices, containing field-span and snapshot-wide ledger
boundaries pass. Resealed ID, binding, arity, ownership, summary, nested field,
cleanup, local call, counts and coverage contradictions reject, as does an
exhausted shared semantic work budget. The Go codec independently determines
acceptance before each native comparison. An initial probe needed a non-keyword
local name and immediate handling of a Result; those failed attempts are not
passing evidence.

Full current-source generic joins/production, SDK/full CLI acceptance and final
self-build/distribution/hosted gates remain open. Native Linux remains batched at
the user's request. Stable public API/SDK ABI, DB, dependencies, repository hygiene
and runner configuration are unchanged. No push, release or deployment.
