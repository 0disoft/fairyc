# Versioned Contract Boundaries

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-17
- Product source: `docs/product/02-spec.md`

## Context

Compiler releases, source semantics, internal MIR, target descriptions, runtime helpers, standard-library APIs, diagnostics, package metadata, caches, and Atlas maps change at different rates. One project version cannot truthfully describe all compatibility.

## Decision

1. Artifacts record a compatibility vector rather than treating the compiler version as universal compatibility.
2. Compiler releases use SemVer. A compiler patch cannot intentionally change valid-source semantics.
3. Pre-1 language revisions are exact. Source compatibility across revisions is not implied.
4. Internal MIR is exact-version, disposable, and not an external backend plugin contract. Old MIR and build caches are discarded rather than migrated.
5. Target schema and ABI profile are separate. Schema describes data shape; an ABI profile describes layout and calling meaning.
6. Diagnostic rule identity and machine-readable diagnostic JSON have separate versions. A published diagnostic code is never reused for a different rule.
7. Pre-1 runtime ABI is an exact compiler match. Standard-library API uses an independent version and declares its language and runtime requirements.
8. Package manifest, lockfile, source archive, and future binary artifact formats are separate contracts.
9. `.fymap`, `SymbolId`, and interface-hash algorithms have separate identifiers. Cross-package joins fail closed when identity algorithms are incompatible.
10. Atlas completeness is relation-specific. Incompatible dependency maps cannot be silently presented as complete.

## Rationale

Disposable compiler state should remain cheap to invalidate, while public source, runtime, diagnostic, and semantic-map contracts need explicit compatibility. Separating versions prevents internal MIR stability from freezing the frontend and prevents Atlas degradation from breaking compilation.

## Consequences

- Bug reports and reproducible artifacts carry more metadata.
- Old cache entries become misses and are collected lazily.
- Atlas can still show local names and spans while marking transitive semantics unavailable.
- Stable language, native ABI, MIR exchange, binary package, and `SymbolId` promises remain deferred.

## Revisit conditions

Introduce compatibility ranges or migrations only when independent released consumers exist and rebuilding from source is no longer an acceptable recovery path.
