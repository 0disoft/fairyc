# Process-local package MIR artifact v0

- Status: Executable provisional boundary
- Decision: `docs/adr/0153-process-local-package-mir-artifacts.md`
- Process codec decision: `docs/adr/0154-process-bound-package-artifact-codec.md`
- Implementation source: `internal/packageartifact/artifact.go`, `internal/packageartifact/codec.go`, `internal/workspacecompiler/session.go`

## Scope

This boundary splits one already verified whole-program MIR value into package-owned function
slices. Each slice carries:

- its build-local `PackageID` and exact `PackageInstanceId`;
- the exact package-number-to-instance map used by the compilation;
- package-owned MIR function bodies paired with `FunctionID` v1 descriptors;
- explicit body-free signatures for called functions owned by other packages;
- the exact workspace-wide type and foreign-function context from that compilation.

The last item is intentionally conservative. Type definitions still contain build-local package
numbers and resource drop links. Sharing the complete immutable context lets the ordinary MIR
verifier check each function slice now without pretending that type metadata already has a stable
persistent encoding.

## Extraction and verification

Extraction accepts only a closed whole program that passes `mir.Verify`. Every function receives a
freshly recomputed `FunctionID`. A package slice owns exactly the functions whose source span carries
its `PackageID`. Calls to other packages become explicit imports. Because v0 retains the whole type
context, non-empty resource drop functions referenced anywhere in that context are also explicit
owned functions or imports.

Artifact verification rejects inconsistent owner maps, executable functions hidden in the context,
unsorted or duplicate bindings, changed descriptors or IDs, missing imports, and corrupt bodies. It
then creates unreachable verifier-only stubs from imported signatures and reruns the ordinary closed
MIR verifier. Those stubs are never returned to a backend or interpreter.

## Reassembly

Reassembly requires every slice to have the same exact process-local identity and type context.
Every import must resolve to exactly one owned `FunctionID` with the same descriptor. The assembler
allocates internal symbols strictly above every original internal and foreign symbol, rewrites
internal calls and resource drop links, sorts functions by name, and reruns `mir.Verify` before
returning executable MIR.

The checked-in multi-package workspace test executes the reassembled program and requires result
`42`. Separate negative tests corrupt a body and ID, omit an import provider, and drift the shared
context.

## Explicit exclusions

Artifact v0 has a bounded canonical process codec solely so `packagecache.MemoryStore` can own and
account for the complete payload. Every codec instance embeds a random process identity. Bytes from
another codec fail closed, so this is not a disk or cross-process format. There is still no disk
path, cross-process lock, default size limit, cache lifetime, CLI surface, remote transport, or
compatibility promise across compiler processes. The shared type context prevents package-granular
storage savings. The reuse coordinator still performs workspace loading and semantic analysis on
every request; only MIR lowering and its first whole-program verification can be skipped.

Persistent package reuse requires a later stable type/context encoding, corruption-safe disk policy,
cross-process coordination, and measured clean/warm evidence. A mismatch remains a safe miss
followed by ordinary compilation.

## Bounded cache integration

`packageartifact.Cache` now connects the process codec to `packagecache.MemoryStore`. A verified hit
requires the expected output-interface hash, exactly one MIR payload, same-process canonical decode,
and full artifact verification. Misses and rejected payloads are distinct. Rejected entries are
removed only if the complete stored record still matches the failed copy, preserving a concurrent
replacement.

The cache reports verified hits separately from raw byte-store hits. `workspacecompiler.Session`
runs package semantic analysis, computes every public interface and whole-package key, and bypasses
MIR lowering only when every function-owning package is a verified hit and the complete set
reassembles. Any miss, rejected payload, or incompatible mixed-generation context falls back to the
ordinary compiler boundary and refreshes current artifacts. Oversized cache admission is skipped
without failing compilation.

Tests prove the control-flow fact directly: the cold and immediately repeated two-package build
both execute to `42`, while the lowering hook is called once. A private-context source edit causes a
partial miss and ordinary lowering. Reverting that edit can expose individually valid artifacts
from different generations; exact-context reassembly rejects the mixture, ordinary lowering runs,
and the next request becomes a complete hit. These are deterministic reuse checks, not elapsed-time
speedup evidence.
