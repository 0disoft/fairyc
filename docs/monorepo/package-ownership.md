# Package Ownership

- Status: Draft
- Repository type: monorepo
- Architecture source: `docs/architecture/00-system-boundary.md`

## Planned owners

| Boundary | Owns | Must not own |
| --- | --- | --- |
| `cmd/fairyc` | argument parsing, command dispatch, process exit, user-facing assembly | language semantics, target layout, backend-specific lowering |
| `internal/source` | files, spans, line maps, source identity | syntax or semantic decisions |
| `internal/lexer` and `internal/syntax` | tokens, trivia, parsing, recovery, and compact arena-backed syntax | type checking, immutable editor trees, or code generation |
| `internal/resolve` and `internal/types` | names, modules, types, constants, and Stage 0 type properties | backend attributes or deferred generic dispatch |
| `internal/hir` | resolved user-facing semantic representation | machine instruction choices |
| `internal/flow` | initialization, move, borrow, region, escape, effect, and drop facts | source parsing or target instruction selection |
| `internal/mir` | explicit backend-independent executable semantics | C, QBE, LLVM, or object-format details |
| `internal/interp` | bounded execution of verified MIR for semantic fixtures | operating-system emulation or independent semantics |
| `internal/target` and `internal/layout` | target triples, ABI facts, size, alignment, and layout | host Go layout assumptions |
| `internal/abi` | C ABI v0 classification, `@repr(C)`, boundary signatures, and conformance facts | native Fairy ABI promises |
| `internal/toolchain` | local C-driver, sysroot, libc, linker, and capability resolution | semantic target layout |
| `internal/backend/*` | translation from MIR to one backend format | language-policy decisions |
| `internal/diagnostic` | stable diagnostics, rendering, source relationships | control-flow decisions |
| `internal/cache` | deterministic cache keys, storage policy, collection | source-of-truth semantics |
| `internal/snapshot` | immutable deterministic projection of analyzed HIR and verified MIR | Atlas query policy or free text |
| `internal/atlas` | versioned semantic-map export and bounded queries over snapshots | frontend dependencies, LLM execution authority, embeddings, or source reparsing |
| `runtime` | traps, process entry, empty storage, and required low-level memory helpers | allocators, files, strings, collections, logging, unwinding, or garbage collection |
| `std` | `SystemHeap`, `Arena`, `Buffer`, text, Linux file, and safe C wrappers | compiler internals, hidden allocation, deferred generic collections, or portable-path claims |
| `spec` | versioned normative grammar, parser and semantic fixture schemas, and executable contract data | generated compiler output, build-local IDs, or implementation-owned snapshots |
| `tests` | executable language, backend, ABI, and diagnostic contracts | production implementation logic |

## Public-contract policy

Before a stable language revision, internal Go packages are not public APIs. The public surfaces are the documented `fairyc` CLI, language revision, runtime ABI selected for conformant or supported targets, standard-library APIs, emitted schema formats, and declared C interoperability. `pub` source visibility and internal module interfaces are not binary compatibility.

## Cross-boundary review

- A frontend change that affects MIR requires semantic and diagnostic fixtures.
- A lexer or parser change requires the normative grammar, parser manifest, byte-span expectations, and diagnostic catalog to remain synchronized.
- A resolver, type, or Flow change requires the semantic manifest, canonical assertions, diagnostic catalog, and owning language contract to remain synchronized.
- A MIR change requires every enabled backend and differential fixture to be reviewed.
- A target-layout change requires ABI fixtures and generated-header review.
- A standard-library ownership change requires drop, allocation, and unsafe-wrapper review.
- An Atlas schema change requires provenance, compatibility, size, and stale-map checks.
- A cache-key change requires invalidation and bounded-storage review.

## Versioning

Compiler release, language revision, MIR, target schema, ABI profile, diagnostics, runtime ABI, standard library, package schemas, `.fymap`, `SymbolId`, and interface-hash algorithms are separate compatibility dimensions. Their Stage 0 behavior is owned by `docs/compatibility/versioning-v0.md`.
