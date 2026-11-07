# Go grouped static interfaces: 2026-11-07

Implements [ADR 0549](../adr/0549-grouped-static-operation-interfaces.md) in the
isolated detached checkout `.cache/grouped-static-20261107`, based on
`c5b6c9b430037de0b57cf1d0bc53a3fd520aa7f1`. This note records the Go owner's
local evidence; coordinator-owned Fairy admission, cache and CLI oracles have
separate receipts.

## Implementation

- `internal/genericsource/static_interface.go` admits a single ASCII identifier
  label on public operations, retaining the argument-free legacy path.
- `internal/genericsource/static_interface_group.go` checks group size only
  after operation/provider structural and duplicate checks, then checks complete
  provider bundles by walking the original providers in source order. It reports
  size at the first member's full declaration and partial bundles at the earliest
  provider of an incomplete group/scalar tuple. Empty provider sets are legal.
- Definitions use `DigestText("source-static-operation-group-v1", JSON)` over a
  flat array of original AST functions: operation names sorted, each operation
  immediately followed by its providers sorted by scalar name. All supplied
  tuples and unused members are included. The group digest is shared, while
  capability IDs, provider IDs, provider digest and bound-operation domains stay
  distinct and unchanged. The legacy operation digest recipe is unchanged.
- Sorting uses detached names; source AST functions, spans, attributes and literal
  spellings are neither rewritten nor normalized to compute definitions.

No other production files are changed by this owner. No grammar/ADR, public API
or ABI, database, dependency, runner, limit, cache wire version, queue or source
lock change is introduced. No commit, push or deployment is made by this owner.

## Focused validation

Executed in this checkout on Windows with `GOMAXPROCS=1`, default catalog/plan
limits and no raised bounds:

```powershell
$env:GOMAXPROCS='1'
go test -mod=readonly -count=1 ./internal/genericsource ./internal/compiler -run 'Test(GroupedStaticInterface|StaticInterface|GenericPackageSourceFailuresPublishOnlyDiagnostics)' -v
go vet -mod=readonly ./internal/genericsource ./internal/compiler
```

Final results: genericsource passed in **0.058s**, compiler passed in **6.641s**,
and vet exited **0** without output. Thirteen top-level tests were selected:
five genericsource tests and eight compiler tests. No skips occurred.

Coverage includes:

- 26 exact source diagnostic cases: malformed labels/declarations, private,
  unsafe, extra/duplicate attributes, wrong result, five parameters, two type
  parameters, unused partial bundles, interleaved groups/scalars, duplicate and
  structural precedence over group size/completeness, and missing tuples at
  direct and transitive original root requests. Fixtures retain Korean trivia
  and CRLF, verify original full declaration spans, and publish no interface,
  specialization or HIR on error. Failure and success paths preserve input AST.
- Groups of 2 and 16 with no providers, groups of 2 and 16 with complete i32
  bundles and interpreted requests, rejection of 1 and 17 members, labels of
  1 and 64 bytes accepted and 65 rejected, invalid ASCII/non-ASCII labels,
  differing type parameter names and arities 0 and 4, and imported public
  grouped operations. Identical labels in different modules cannot combine
  singleton declarations into a group.
- Independently enumerated flat AST digest order, exact original package and
  bound-operation digest recipes, distinct capabilities and selected provider
  IDs/digests. Equal-width unused sibling declaration, sibling provider and
  unused-scalar provider edits invalidate both members, public/dependent
  template closures and selected operation/wrapper specialization IDs while
  selected alpha/provider/wrapper ASTs and spans remain byte-for-byte equal.
  An unrelated equal-width private body edit preserves dependent identities.
- Legacy single-operation execution and all existing rejection cases, existing
  generic source-error publication/location tests, plus frozen operation,
  provider, bound and private/package specialization identities.

## Actual native receipts

The existing native helper emitted both target profiles and built and executed
MSVC **14.51.36231** C11/C17 consumers at low and optimized profiles. Each of
these twelve final native executions returned **42**, empty stdout and empty
stderr; none was skipped.

| Root | MIR functions | MSVC C bytes | C11 low image | C11 optimized image | C17 low image | C17 optimized image |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `TestGroupedStaticInterfaceExecution` | 11 | 5512 | 109568 | 107520 | 109568 | 107520 |
| `TestGroupedStaticInterfaceAllScalarTransport` | 66 | 37984 | 116736 | 109568 | 116736 | 109568 |
| `TestStaticInterfaceExecution` (legacy) | 7 | 4033 | 109568 | 107520 | 109568 | 107520 |

The scalar program transports `bool`, `i8`, `i16`, `i32`, `i64`, `isize`, `u8`,
`u16`, `u32`, `u64`, `usize`, `f32`, and `f64` through both members and a generic
relay. Twelve scalars use real C record seeds/observers (including signed minima,
unsigned maxima and nonintegral floats); Fairy `bool` uses both source boolean
values because the existing ABI deliberately disallows Fairy bool in repr(C)
fields. No new bool ABI or scalar-literal admission is assumed. The interpreter
checks grouped i32/i64 arithmetic, provider calls, mixed arities, complete 2/16
bundles and imported calls; the C-seeded thirteen-scalar case is native-only.

Frozen legacy fixture identities:

```text
operation  fa1b8df1e84c65c7c3668a2ba5e6f53123c61e998a1d9c207cd9b03b630fd1f3
bound      3d243be26ca9a43b79a0ab039e5cd3bd6b73e9a06870924e32cc4050b2a2681e
provider   cf2a185654f52ca7ca137c36f6776b9b92fab591fd131ce2cba7491881083b84
package    g0_d9e6a44c9ba21ef02111d8814310d5c3720efbeb3812c48844fd5cfd56491605
private    g0_def8e8b0d61dd115db769c717fea47feb8dceb1d5dc80c11c378ca1d9b31634b
```

## Corrected initial failures and remaining gates

Initial focused attempts exposed test-fixture mistakes, all corrected before the
final passing run: an assertion referenced a nonexistent HIR field; an equal-width
U-to-V declaration edit initially left the result type as U; diagnostic fixtures
used `group.fy` with `module main`, rejected by existing syntax admission; and
the first all-scalar fixture tried to put Fairy bool into a repr(C) field. These
failures were not treated as passes or resolved by extending production bounds.

Full Go/CLI/cache suites, Fairy native/self-build/fixed-point/distribution and
hosted CI were not run by this owner. Actual Linux native execution remains the
coordinator's separate batched gate; emitting the Linux profile here does not
establish Linux execution. Shared oracles and integration/commits remain with
the coordinator, with no outstanding Go-owner validation failure.
