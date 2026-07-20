# ADR 0130: Separate package-aware span identity from source display paths

- Status: Accepted
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Package source: `docs/adr/0128-build-local-package-identity.md`
- Diagnostic source: `docs/diagnostics/catalog-v0.md`

## Context

ADR 0128 initially prevented equal package-local source paths from collapsing by qualifying an
internal source string as `$pN/path.fy`. That preserved compiler identity, but the same string flowed
through semantic facts, typed HIR source paths, MIR spans, unsafe provenance, and drop-trace sites.
A future package-aware diagnostic renderer could therefore expose a disposable compiler ID or strip
it inconsistently at one output boundary while other facts still leaked it.

Using only the relative path is also wrong: two resolved packages may both own `util.fy` at the same
byte offsets. Source identity must retain the package dimension even when display paths are equal.

## Decision

1. `hir.Span` stores build-local `PackageID` separately from `Source`, `StartByte`, and `EndByte`.
2. `Source` and `hir.Module.SourcePath` retain the original source-root-relative path. They never
   contain `$pN` qualification.
3. Semantic span ordering, compaction, MIR scope containment, unsafe provenance, and instrumented
   drop-site identity compare both `PackageID` and source path before byte offsets.
4. Compiler-private entity strings may reconstruct a qualified source identity so entities from
   equal package-local paths remain distinct. Those entity strings are internal defect evidence, not
   user diagnostic paths or stable serialization.
5. Existing single-package fixture manifests continue to describe only path and byte offsets. Their
   runners require `PackageID(0)` so an accidental non-default span cannot pass unnoticed.
6. A future package-aware renderer must combine the relative path with a user-facing package label
   supplied by the package layer. It must not render the numeric build-local ID. The label and output
   format remain deferred until a public package command exists.

## Consequences

- Equal `util.fy` paths remain distinct throughout semantic HIR and MIR without private prefixes in
  source-visible fields.
- Diagnostics can preserve accurate internal identity before public package naming is decided.
- Drop-trace sites and unsafe provenance do not merge equal source offsets from different packages.
- Adding `PackageID` to a span does not stabilize diagnostic JSON, MIR serialization, cache keys, or
  package identity.

## Verification

- HIR tests cover private source-identity round trips and malformed private spellings;
- semantic tests assert package-aware use/declaration spans and cross-package related diagnostics
  whose paths contain no private prefix;
- MIR tests assert package-aware function spans while preserving private function identity;
- every existing single-package semantic, Flow, MIR, interpreter, and native fixture still requires
  default-package spans;
- complete Go test, build, vet, and WSL-free Windows host-smoke intents cover integration behavior.
