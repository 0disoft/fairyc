# ADR 0129: Centralize semantic analysis through verified MIR

- Status: Accepted
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Architecture source: `docs/architecture/02-runtime-flow.md`
- Package source: `docs/adr/0128-build-local-package-identity.md`

## Context

The experimental CLI, MIR fixtures, interpreter fixtures, C-emission fixtures, and package fixtures
all need the same semantic-to-MIR sequence. Several callers assembled `semantic.Analyze`,
`mir.Lower`, and `mir.Verify` directly. The first package-identity slice proved semantic and MIR
behavior separately, but no shared coordinator guaranteed that an already-resolved package graph
crossed the exact compiler boundary used by product commands.

Copying the sequence is not harmless glue. A caller can forget a phase gate, treat a verifier
failure as an ordinary source rejection, or hand unverified MIR to another layer while every local
unit still looks reasonable.

## Decision

1. `internal/compiler` owns the shared Stage 0 boundary from syntax-clean semantic files through
   verified MIR.
2. `compiler.Check` accepts the existing explicit single-package closure.
   `compiler.CheckPackages` accepts one already-resolved `semantic.PackageInput`; it performs no
   discovery, solving, source acquisition, manifest parsing, or stable package-ID selection.
3. Ordinary semantic diagnostics and deliberately unsupported semantic forms remain in the semantic
   result and produce no executable MIR. They are not Go errors.
4. Invalid package input, MIR lowering failure, and MIR verifier rejection are separate pipeline
   error stages. A verifier rejection remains an internal compiler defect rather than a user source
   diagnostic.
5. A caller may consume `Result.MIR` only when `Result.Verified` is true.
6. The experimental CLI source-closure path, resolved package fixture, and every source-driven MIR,
   interpreter, C-emission, and native-probe fixture producer use this coordinator without changing
   their manifest observations.
7. MIR-verifier mutation fixtures continue to invoke the verifier directly because verifier failure
   is their test subject. Interpreter and backend entry points also retain defensive re-verification
   for MIR received from callers; neither case is a source-to-MIR pipeline bypass.

## Consequences

- Single-package product commands and resolved-package acceptance now share the same semantic,
  lowering, and verification gate.
- Source-driven fixture lanes cannot silently assemble a weaker semantic-to-MIR sequence than the
  product path.
- Backend, interpreter, entry-point, toolchain, target, and artifact decisions remain downstream.
- The coordinator does not make package discovery or a package-aware public CLI exist.
- Expected source rejection stays structurally distinct from compiler failure.

## Verification

- compiler tests cover verified success, ordinary semantic rejection without MIR, and classified
  resolved-package input failure;
- CLI tests cover the existing check, run, emit, build, and target-facing behavior through the new
  coordinator;
- a compiler-boundary architecture test parses every source-driven fixture runner and rejects direct
  `semantic.Analyze`, `mir.Lower`, or `mir.Verify` calls;
- the resolved module-path-isolation fixture completes the coordinator through verified MIR;
- complete Go test, build, vet, and WSL-free Windows host-smoke intents cover integration regressions.
