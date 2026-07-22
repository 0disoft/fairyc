# ADR 0152: Establish package-stable function identity before package artifacts

- Status: Accepted
- Date: 2026-07-22

## Context

Verified whole-program MIR can be encoded, but it is not a reusable package artifact. MIR calls use
build-local `SymbolID` values and the constrained C backend currently names functions by their index
in one sorted whole-program list. Both values can change when another package changes graph order,
even if the function and its owning package did not.

Persisting those values would make reuse nondeterministic and could silently bind a cached call to
the wrong function. Weakening the whole-program verifier to tolerate unresolved calls would merely
move that failure downstream.

## Decision

Introduce `FunctionID` v1 in `internal/packageartifact`. It hashes one exact
`PackageInstanceId` plus a build-number-independent logical name and canonical signature facts.
Nominal types replace every disposable package number with the referenced full package identity.

`workspaceloader.Workspace.PackageIdentities` owns the exact mapping from the `PackageID` allocation
used by `SemanticInput` to resolved `PackageInstanceId` values. Artifact code must consume that map;
it must not reconstruct identities from module text, package order, paths, or MIR names.

The identifier is compiler-private metadata. It is not source syntax, a HIR symbol, a C ABI name,
or proof that a function body is safe to reuse.

## Consequences

- package-number renumbering no longer changes the identity of an otherwise equal function;
- package, name, signature, effect, or drop-glue changes invalidate the identity;
- incomplete nominal-type mappings and malformed build-local names fail closed;
- a later package artifact can declare imports by stable function descriptor while the assembled
  whole program still receives fresh build-local symbols and passes ordinary MIR verification;
- separate C translation units still require a later backend decision for stable private linker
  names, linkage visibility, and object compatibility.

## Rejected shortcuts

- Serializing HIR `SymbolID` or MIR order was rejected because both are build-local.
- Hashing only source-visible text was rejected because resolved dependency identities and canonical
  types are semantic facts.
- Treating `FunctionID` as a body hash was rejected because body compatibility belongs to the
  package cache key and artifact payload contract.
- Emitting multiple C objects immediately was rejected because current calls and type closure do
  not yet form an independently verifiable package boundary.
