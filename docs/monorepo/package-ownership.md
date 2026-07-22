# Package Ownership

- Status: Draft
- Repository type: monorepo
- Architecture source: `docs/architecture/00-system-boundary.md`

## Package owners

| Boundary | Owns | Must not own |
| --- | --- | --- |
| `cmd/fairyc` | argument parsing, command dispatch, process exit, user-facing assembly | language semantics, target layout, backend-specific lowering |
| `internal/source` | files, spans, line maps, source identity | syntax or semantic decisions |
| `internal/lexer` and `internal/syntax` | tokens, trivia, parsing, recovery, and compact arena-backed syntax | type checking, immutable editor trees, or code generation |
| `internal/ast` | error-free lowering and implementation-private parser assertions | accepting recovered syntax, type checking, or public serialization |
| `internal/parserfixture` | strict parser-manifest execution and exact contract comparison | language rules, fixture rewriting, or production compiler orchestration |
| `internal/resolve` and `internal/types` | names, modules, types, constants, and Stage 0 type properties | backend attributes or deferred generic dispatch |
| `internal/hir` | resolved user-facing semantic representation | machine instruction choices |
| `internal/flow` | initialization, move, borrow, region, escape, effect, and drop facts | source parsing or target instruction selection |
| `internal/mir` | explicit backend-independent executable semantics | C, QBE, LLVM, or object-format details |
| `internal/interp` | bounded execution of verified MIR for semantic fixtures | operating-system emulation or independent semantics |
| `internal/target` and `internal/layout` | target IDs, target schemas, ABI facts, size, alignment, and layout | host Go layout assumptions or local toolchain paths |
| `internal/cabi` and `internal/cabifixture` | host-independent C ABI v0 classification, fixed-signature plans, and versioned fixture execution | source FFI lowering, local compiler probing, or native Fairy ABI promises |
| `internal/toolchain` | local C-driver, sysroot, libc, linker, and capability resolution | semantic target layout |
| `internal/backend/*` | translation from MIR to one backend format | language-policy decisions |
| `internal/diagnostic` | stable diagnostics, rendering, source relationships | control-flow decisions |
| `internal/packagegraph` and `internal/packageidentityfixture` | resolved-instance equality, canonical PackageInstanceId encoding, graph visibility, and versioned identity vectors | dependency solving, source acquisition, public manifest or lockfile syntax |
| `internal/moduleinterface` and `internal/moduleinterfacefixture` | source-visible typed-HIR projection, stable nominal type names, interface hashing, and invalidation fixtures | HIR serialization, private layout, native ABI, cache storage, manifests, or lockfiles |
| `internal/packagecontent` and `internal/packagecontentfixture` | exact canonical package source graph encoding plus per-file content/line-map identities, hashing, and invalidation vectors | source discovery, filesystem authority, source retention, archives, manifests, lockfiles, map publication, or cache storage |
| `internal/packagecache` and `internal/packagecachefixture` | deterministic whole-package compatibility keys, caller-bounded process-local LRU storage, and invalidation fixtures | persistent format, public CLI policy, default limits, remote cache, or source-of-truth semantics |
| `internal/durablefile` | opened-root-contained exclusive temporaries, same-directory immutable publication, identical-writer convergence, corrupt destination repair, and platform-specific post-publication sync | artifact schemas, retention, eviction, mutable replacement, path selection, source truth, authentication, or caller limits |
| `internal/projectmanifest` and `internal/projectmanifestfixture` | bounded strict `fairy.toml` decoding, closed v1 declaration validation, deterministic dependency output, and complete-file fixtures | filesystem discovery, workspace containment, resolved graph construction, lockfile writes, registry, network, or version solving |
| `internal/sdkstdlib` | exact compiler-bundled Stage 0 standard-library Fairy sources, SDK identity, language/runtime requirements, and source identity composition | target C linkage, installation discovery, public API invention, or package solving |
| `internal/workspaceloader` and `internal/workspaceloaderfixture` | explicit local member/source closure, path and symlink enforcement, exact dependency checks, local and bundled-SDK source identities, conditional standard-library injection, feature-distinct graph construction, complete-directory fixtures, and an explicit ephemeral root `tests/` overlay that ordinary loading never reads | test declaration/execution policy, lockfile I/O, target C linkage, registry, network, version solving, or persistent cache writes |
| `internal/lockfile` | canonical generated local/SDK graph encoding, strict identity and source-origin validation, workspace staleness comparison, and atomic replacement | manifest editing, CLI update policy, SDK discovery, registry, network, solving, or cache storage |
| `internal/semanticsnapshot` | immutable deterministic path-free projection of analyzed HIR, semantic facts, verified MIR, exact package metadata, and complete source-file integrity records | `.fymap` encoding, current-source revalidation, SymbolId derivation, Atlas query policy, free text, source reparsing, or compiler feedback |
| `internal/symbolidentity` | provisional source-instance-independent SymbolId tuple/signature encoding, snapshot derivation, strict text forms, and collision rejection | `.fymap` transport, source reparsing, map publication, dependency joins, anchors, query policy, or compiler feedback |
| `internal/atlasmap` | complete workspace-to-snapshot assembly, exact public-interface hash closure, deterministic immutable per-package logical headers and record partitioning, package namespace references, canonical bounded `.fymap` JSON, checksums, strict decoding, contained immutable generations, current-source validation, retention bounds, and semantic revalidation | default paths or quotas, ordinary-command writes, dependency-map discovery, queries, free text, LLM execution authority, embeddings, source reparsing, authentication, or compiler feedback |
| `runtime` | traps, process entry, empty storage, and required low-level memory helpers | allocators, files, strings, collections, logging, unwinding, or garbage collection |
| `std` | `SystemHeap`, `Arena`, `Buffer`, text, Linux file, and safe C wrappers | compiler internals, hidden allocation, deferred generic collections, or portable-path claims |
| `spec` | versioned normative grammar plus parser, semantic, resource, Region, Result, MIR, MIR-verifier, interpreter, C-emission, target-layout, C ABI, resolved-package-graph, project-manifest, workspace-loader, PackageInstanceId, package-source, public-interface, and whole-package-cache fixture schemas and executable contract data | generated compiler output, build-local IDs, textual MIR snapshots, arbitrary MIR exchange, lockfile syntax, or implementation-owned snapshots |
| `tests` | executable language, backend, ABI, and diagnostic contracts | production implementation logic |

## Public-contract policy

Before a stable language revision, internal Go packages are not public APIs. The public surfaces are the documented `fairyc` CLI, language revision, runtime ABI selected for conformant or supported targets, standard-library APIs, emitted schema formats, and declared C interoperability. `pub` source visibility and internal module interfaces are not binary compatibility.

## Cross-boundary review

- A frontend change that affects MIR requires semantic and diagnostic fixtures.
- A lexer or parser change requires the normative grammar, parser manifest, byte-span expectations, and diagnostic catalog to remain synchronized.
- A resolver, type, or Flow change requires the semantic manifest, canonical assertions, diagnostic catalog, and owning language contract to remain synchronized.
- A whole-root resource-state change requires the resource manifest, cleanup-obligation assertions, `E11xx` spans, and drop-elaboration contract to remain synchronized.
- A Region or Result change requires its dedicated manifest, deferred ledger, `E31xx` or `E41xx` spans, and owning language and MIR contracts to remain synchronized.
- A MIR change requires its source-linked fixture observations, verifier, interpreter, every enabled backend, and differential fixtures to be reviewed.
- A verifier-invariant change requires a verified named seed, one isolated mutation, exact expected-code coverage, and harness protocol review.
- A package identity, visibility, or public-interface change requires resolved-graph, PackageInstanceId, and interface-invalidation fixtures, direct module lookup, compatibility, cache identity, and Atlas package-edge review.
- A target-layout change requires ABI fixtures and generated-header review.
- A standard-library ownership change requires drop, allocation, and unsafe-wrapper review.
- An Atlas schema change requires provenance, compatibility, size, and stale-map checks.
- A cache-key change requires invalidation and bounded-storage review.

## Versioning

Compiler release, language revision, MIR, target schema, ABI profile, diagnostics, runtime ABI, standard library, package schemas, `PackageInstanceId`, `.fymap`, `SymbolId`, and interface-hash algorithms are separate compatibility dimensions. Their Stage 0 behavior is owned by `docs/compatibility/versioning-v0.md`.
