# Workspace Boundaries

- Status: Draft
- Repository type: monorepo
- Ownership source: `docs/monorepo/package-ownership.md`

The repository is a toolchain monorepo, not a collection of independently deployed services. The compiler, specification, runtime, standard library, fixtures, and Atlas schema evolve together until a later compatibility policy explicitly separates them.

## Planned top-level boundaries

```text
cmd/fairyc/        Go executable assembly
internal/          Go compiler implementation packages
runtime/           Fairy C runtime sources and target adapters
std/               Fairy C standard library
spec/              normative grammar plus parser and semantic fixtures; later ABI and diagnostic contracts
tests/             future compile, run, trap, runtime, differential, compatibility, and ABI fixtures
docs/              product and engineering rationale
```

The `spec/` grammar, parser fixtures, and semantic fixtures now exist as design-owned executable data. Implementation directories remain planned and are created only when their phase starts.

## Dependency direction

- CLI driver depends on compiler orchestration, never the reverse.
- Parser and syntax layers do not depend on type checking, HIR, MIR, or backends.
- Typed HIR depends on resolved syntax and target-independent type contracts.
- Flow analysis consumes typed HIR and produces facts required by MIR lowering.
- MIR is backend-independent and cannot import C-, QBE-, LLVM-, or object-format-specific concepts.
- The MIR interpreter consumes verified MIR and cannot define independent language behavior.
- Backends depend on MIR and target descriptions.
- ABI classification depends on `TargetSpec`; toolchain resolution depends on `ToolchainSpec` and cannot redefine layout.
- Runtime and standard library may expose contracts consumed by the compiler, but do not import compiler implementation packages.
- A compiler-owned `SemanticSnapshot` projection consumes analyzed HIR and verified MIR.
- Atlas serialization and queries consume `SemanticSnapshot`; frontend and analysis packages cannot import Atlas.

## Build ownership

The Stage 0 compiler is one Go module unless measurement proves a need for multiple modules. Fairy C runtime and standard-library artifacts are target-specific toolchain components, not Go packages. A directory boundary does not imply an independently versioned package.

## Change coordination

A syntax change coordinates EBNF, parser fixtures, diagnostics, and language docs. A resolver, type, or Flow change coordinates the semantic manifest, diagnostics, owning language contract, and future implementation. A later executable semantic change may additionally coordinate HIR or MIR, affected backends, runtime or standard library, and Atlas schema. Documentation-only rationale does not force machine-code regeneration.

## Deferred boundaries

Plugin ABI, native Fairy ABI, binary package format, package registry, remote binary cache, self-hosted compiler package layout, standalone Atlas server, direct debugger integration, portable path package, generic collections, and custom allocator packages remain deferred.
