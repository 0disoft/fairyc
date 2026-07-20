# ADR 0128: Preserve resolved package identity through semantic HIR and MIR

- Status: Accepted
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Package source: `docs/package/package-graph-v0.md`
- MIR source: `docs/mir/mir-v0.md`

## Context

The resolved-package-graph fixture already defines module identity as package instance plus module
path. The semantic analyzer previously indexed modules only by source-visible module path, while
user-defined HIR type identities and MIR function names also used only that path. Two isolated
dependency instances that both contained `util` therefore passed graph resolution but could not be
compiled together without a false duplicate or an internal identity collision.

Choosing a public `PackageInstanceId` encoding now would accidentally standardize a lockfile and
compatibility surface before package coordinates, manifests, and the solver exist. Keeping the
identity only in the package fixture runner would leave the actual compiler pipeline incorrect.

## Decision

1. Semantic multi-package input consists of syntax-clean files carrying a deterministic build-local
   `PackageID` plus exact `(from package, import module path, provider package)` bindings produced by
   an already-resolved graph. Every source import requires one binding, including a binding back to
   the current package; missing, duplicate, unused, or absent-package bindings reject the input.
2. Module lookup uses `(PackageID, source-visible module path)`. Equal module paths in distinct
   package instances remain independent; an import resolves only to its exact bound provider.
3. Typed HIR modules retain `PackageID`. User-defined enum, struct, and resource identities and MIR
   function names use a compiler-private qualified module identity for non-default packages.
4. Source spans retain the build-local package discriminator separately from the original relative
   path, as refined by ADR 0130, so equal package-local paths cannot collapse or leak private
   qualification into source-visible fields.
5. `PackageID(0)` preserves the existing explicit single-package closure behavior and its exact HIR
   and MIR names.
6. Build-local IDs and the `$pN` spelling are disposable implementation facts. They must not become
   source syntax, a lockfile key, a serialized `PackageInstanceId`, an Atlas stable identity, a
   cache compatibility key, a native ABI name, or a public diagnostic promise.
7. `internal/packagefixture` assigns nonzero IDs deterministically from validated instance-class
   order, compiles one source representative for each equal instance class, and converts resolved
   lookups into semantic import bindings. It does not perform solving, source acquisition, or public
   package discovery.

## Consequences

- The resolved graph now reaches real semantic analysis instead of stopping at a parallel fixture
  interpretation.
- Isolated equal module paths, equal package-local source paths, user-defined types, and MIR
  functions remain distinct throughout the implemented compiler pipeline.
- A future package manager can replace build-local numbering without migrating source or promising
  the temporary private spelling.
- Package-aware CLI discovery, public diagnostic paths, stable package identities, C symbol
  mangling, caches, and Atlas package-edge identity remain separate gates.

## Verification

- semantic tests compile two roots against different `util` package instances and assert exact
  resolution targets, distinct source identities, and distinct user-defined type identities;
- MIR tests lower and verify both dependency instances and assert distinct private function and
  struct identities;
- the module-path-isolation resolved fixture converts to semantic package input and completes Flow;
- complete Go test, build, vet, and WSL-free Windows host-smoke intents cover regressions.
