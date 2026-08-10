# ADR 0330: Acyclic copy-tagged dependency graph

- Status: Accepted
- Date: 2026-08-10
- Depends on: ADR 0326, ADR 0327, ADR 0329

## Context

The constrained C backend could compose structs, restricted user enums, aggregate-bearing tags, and
tags over the two compact scalar records. A generic copy-only tag still could not directly contain
another generic copy-only tag such as `Option<Result<Inner, Failure>>`. The frontend, verified MIR,
and interpreter already preserved those programs, so C emission remained the first rejecting stage.

## Decision

Tagged-type discovery recursively follows direct `Option` and `Result` payloads. A nested tagged
candidate is admitted only when it needs no drop. Tagged layouts are then resolved recursively under
a cycle guard, and the combined aggregate emission graph adds explicit tag-to-tag dependency edges.
The resulting order is:

1. complete scalar, struct, or restricted user-enum payload dependencies;
2. each inner copy-only tagged record;
3. each outer copy-only tagged record;
4. every containing private struct or restricted user enum.

This is private compiler transport for source programs already accepted by the language.

## Boundary

The graph admits only acyclic copy-only tags whose leaves are already admitted Fairy copy scalars,
inert structs, or restricted user enums. Direct compact tags remain the special fixed records.
Nested resource-bearing tags, arrays, borrows, raw-pointer or C-scalar tagged payloads, recursive
dependencies, and every tagged field in `@repr(C)` remain rejected. No implicit drop propagation or
public tagged ABI is introduced.

## Evidence

The 52nd C-emission fixture carries `Option<Result<Inner, Failure>>`,
`Result<Option<Failure>, i32>`, and `Result<Option<Inner>, Failure>` through private struct and
user-enum fields. The interpreter returns 42, unit checks require inner tag definitions before outer
tag fields and outer tag definitions before containing aggregates, and a dedicated Windows
regression compiles and executes the generated Fairy program with low and optimized MSVC profiles
and requires native result 42.

## Compatibility

Language, SDK, runtime ABI, target, MIR, cache, lockfile, public API, and public ABI identities remain
unchanged.
