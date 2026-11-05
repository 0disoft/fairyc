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
