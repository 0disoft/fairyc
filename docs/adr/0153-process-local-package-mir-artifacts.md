# ADR 0153: Establish process-local package MIR artifacts before persistence

- Status: Accepted
- Date: 2026-07-22

## Context

The compiler has a bounded process-local cache store, a verified whole-program MIR codec, and a
package-stable `FunctionID`. It still has no independently checkable package payload. Directly
placing the whole-program codec into one cache entry per package would duplicate work while calling
it incremental compilation. Serializing sliced MIR immediately would instead freeze build-local
type identities and resource drop links as accidental compatibility facts.

## Decision

Add a process-local `packageartifact.Artifact` boundary. It owns only one package's function bodies
and lists every cross-package function dependency by stable descriptor and body-free MIR signature.
For v0, all slices retain the same cloned workspace-wide type and foreign-function context plus the
same package identity map.

Each artifact independently re-enters `mir.Verify` with verifier-only unreachable import stubs.
Reassembly requires exact context equality, resolves every import to one owned `FunctionID`, assigns
a new collision-free internal symbol space, rewrites calls and resource drop links, and re-enters
the closed-world verifier. The reassembled checked-in workspace must still execute to `42`.

## Consequences

- package function ownership and cross-package imports are executable contracts rather than inferred
  from function order;
- corrupt bodies, changed identities, missing providers, conflicting descriptors, and context drift
  fail before execution or backend emission;
- reassembly proves cached function symbols never need to be trusted or reused;
- v0 is useful for bounded process-local reuse experiments but is intentionally inefficient because
  every slice carries the same type context;
- persistent encoding and storage remain deferred until type context, limits, invalidation, and
  measurements have their own evidence.

## Rejected alternatives

- One whole-program MIR blob per package was rejected because it contains no package reuse boundary.
- Allowing unresolved calls through `mir.Verify` was rejected because it weakens the executable MIR
  contract.
- Persisting build-local `PackageID`, `SymbolID`, or C function indexes was rejected because graph or
  declaration order can change them.
- Splitting type metadata immediately was rejected because nominal type closure, public aggregate
  shape, resource drop glue, and foreign declarations need a separate stable import contract.
