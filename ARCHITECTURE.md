# Architecture

- Status: Stage 0 implementation overview
- Product source: [Product specification](docs/product/02-spec.md)
- Boundary: [System boundary](docs/architecture/00-system-boundary.md)
- Flow detail: [Compilation and runtime flow](docs/architecture/02-runtime-flow.md)

Implementation map for the Go Stage 0 prototype compiler. Normative contracts stay authoritative: `spec/grammar/stage0.ebnf`, `docs/language/`, `docs/adr/*.md`, `docs/mir/mir-v0.md`, `docs/package/*.md`, `docs/abi/*.md`, `docs/runtime/*.md`, `docs/diagnostics/catalog-v0.md`, and the fixture manifests under `spec/fixtures/`. Where this file and a normative document disagree, the normative document wins.

## Frontend and analysis pipeline

```text
source bytes
  -> lexer tokens and trivia           internal/lexer/
  -> lossless compact syntax            internal/syntax/
  -> error-free AST                     internal/ast/
  -> resolve / type / Flow, typed HIR   internal/semantic/, internal/hir/
  -> MIR lowering and verification      internal/compiler/, internal/mir/
  -> MIR interpreter or constrained C11 backend
```

`internal/frontend` composes lexer, syntax, and AST behind one compiler-facing entry. The lexer keeps typed trivia and invalid-input provenance, consumes original UTF-8 bytes, and leaves canonical spans half-open; only an error-free batch parse lowers to AST (`spec/grammar/stage0.ebnf`, `spec/fixtures/parser/manifest-v0.json`).

`internal/semantic` resolves names and builds typed HIR for the implemented subset in resolve, type, and Flow phases. Flow is analysis-only; `internal/hir` holds the typed program and `internal/mir` the executable form (`spec/fixtures/semantic/`, `resource/`, `region/`, `result/`, `mir/`).

`internal/compiler` is the single semantic-to-verified-MIR coordinator. `Check` takes one explicit single-package closure, `CheckPackages` an already-resolved package input, and both publish executable MIR only after `internal/mir` marks it verified. Ordinary source diagnostics stay in the semantic result; internal failures carry an explicit `compiler.Stage` (`package-input`, `mir-lower`, `mir-verify`, `contract-state-mir-lower`).

## Backends

Verified MIR reaches the bounded interpreter (`internal/interpreter/`), the first semantic oracle, or the constrained C11 reference transport (`internal/cbackend/`) driven through `internal/cdriver/` and `internal/cruntime/`. `internal/cbackend` completes reachability, local typing, capability checks, runtime requirements, layout queries, and drop planning before writing C, and each `emitter` owns only a read-only analyzed plan and its output buffer. Target standard-library C sources live in `internal/cstdlib/` behind the closed symbol contract in `internal/cstdlibcontract/`. Generated C is a constrained reference transport, not the Fairy C semantic model.

## Workspace loading and compilation

`internal/workspaceloader` turns one local-exact workspace into a validated package graph with bounded budgets and exact file/line-map identities. `internal/workspacecompiler` coordinates package-aware semantic analysis and MIR reuse on top of a loaded workspace; callers cannot supply a disconnected coordinate map, and ordinary compilation never reads the root `tests/` tree.

`internal/admission` owns the compiler-private typed `Capabilities` and `Policy[Owner]` values shared by source preparation, analysis, and MIR verification. The zero policy admits only the ordinary scalar path and grants no SDK storage authority; storage queries and transfers, nominal arguments, and SDK template ownership use separate typed modes, and invalid combinations fail before expansion.

Per-package caches (`internal/packagecache/`, `internal/packageartifact/`) are derived and disposable, never the only copy of source or specification data. They are keyed by the exact compiler build plus the compatibility vector, and every accepted hit is re-verified through the ordinary verifier before the interpreter or backend observes it; any cache failure falls back to ordinary lowering.

## Test admission

The loader marks test files explicitly in semantic input, and ordinary loading never reads `tests/`. A test path keeps its literal `tests/` prefix and must match its declared module. `@test` is harness-owned and valid exactly once, without arguments, on a private safe nongeneric zero-argument `void` free function. The compiler runs `internal/semantic` declaration validation before generic expansion can erase declarations; invalid markers produce `E9001`. The CLI reads test-entry metadata without changing the loader's AST or removing attributes, so repeated analysis and cache checks retain the original input. A production path or module name cannot grant test admission ([ADR 0181](docs/adr/0181-stable-workspace-semantic-test-v1.md)).

## Windows native toolchain

`internal/cdriver` resolves and invokes the declared constrained-C toolchain. `ResolveWindowsHostMSVC` locates the latest complete x64 MSVC and Windows SDK once per process with `sync.OnceValues`; the first result, including failure and its environment snapshot, persists until process exit, and each caller receives its own `Environment` copy. A process must restart to observe a newly installed toolchain. `WindowsMSVCBuildOptions` selects bounded outputs through one shared builder, with thin named wrappers preserving existing behavior. Discovery supplies local executable and SDK paths; target semantics remain owned by `internal/target`.

## Targets

`internal/target` owns host-independent target semantics: declared IDs, scalar and pointer layouts, per-profile calling conventions, stack facts, and object/debug/unwind contracts. The `x86_64-linux-gnu` and `x86_64-windows-msvc` descriptors contain no local compiler, SDK, or filesystem paths. `internal/layout` applies those facts to size, alignment, offset, and padding, and `internal/cabi` classifies the closed C-representable subset per target. Linux GCC/Clang and Windows MSVC evidence is capability-specific and provisional; overall target maturity is reported separately by `internal/target`.

## Runtime

A compiled program contains only the selected Fairy C runtime and linked platform libraries. The Go runtime belongs to the compiler process and is never linked into compiled Fairy C programs.

## Atlas

`internal/semanticsnapshot` is the only compiler-to-Atlas export boundary: it validates and detaches typed HIR, semantic facts, verified MIR, and exact package metadata into an immutable host-independent snapshot with package-relative source identities, without filesystem, network, cache, backend, or source-text access. Atlas cannot reparse source to invent semantic facts, and its failure does not change diagnostics or build results. [ADR 0203](docs/adr/0203-enforced-one-way-atlas-import-boundary.md) makes the direction executable: only the Atlas/export layer, CLI orchestration, and the Windows evidence harness may import Atlas packages, and no Atlas output feeds back into semantic analysis.

## Failure boundaries

Malformed or unsupported source forms, MIR lowering failures, verifier rejections, and C driver or linker failures stay distinct. `mir.Unsupported` reports a typed source form with no lowering, separate from malformed HIR and internal lowering failures. Verifier failures are compiler defects and never reach a backend; cache entries are disposable and never authoritative.
