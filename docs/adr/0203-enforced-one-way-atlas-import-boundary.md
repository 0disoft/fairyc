# Enforced one-way Atlas import boundary

- Status: Accepted
- Date: 2026-07-23
- Original boundary: `docs/adr/0004-atlas-semantic-snapshot-boundary.md`
- Exit evidence: `docs/atlas/phase4-exit-evidence.md`
- Implementation: `internal/architecturecontract/`

## Context

ADR 0004 requires compiler frontend, analysis, and MIR packages to produce facts without importing
Atlas. Current code follows that direction, but prose and an observed import search are weak guards:
a later package can import `atlasmap`, `contextpack`, or attributed text and still pass ordinary unit
tests until the cycle or semantic dependency becomes visible elsewhere.

The inverse rule is easier to maintain than a list of every compiler-core package. New compiler,
backend, runtime, or library packages should be protected automatically, while every new Atlas
adapter should require an explicit architecture decision.

## Decision

1. Treat these module roots as the complete current Atlas/export side of the one-way boundary:
   `internal/semanticsnapshot`, `internal/symbolidentity`, `internal/atlasmap`,
   `internal/contextpack`, `internal/symbolcatalog`, `internal/attributedtext`, and
   `internal/evidencebundle`. `semanticsnapshot` belongs on the export side because it consumes
   completed HIR/MIR and must never become a compiler input.
2. Permit direct imports of those roots only from:
   - the Atlas/export roots themselves;
   - `internal/cli`, which selects an explicit experimental operation only after ordinary workspace
     compilation succeeds; and
   - `cmd/fairyc-windows-host-smoke`, which owns bounded local Atlas measurement evidence.
3. Scan every `.go` file below repository `cmd/` and `internal/`, including tests and files excluded
   by this host's build tags. Any other importer of an Atlas/export root is a deterministic contract
   violation. New packages are denied automatically until added to the narrow consumer allowlist by
   an explicit review.
4. Parse Go import declarations with the standard parser. Comments and similar external import paths
   are not matches. Do not invoke `go list`, follow dependency downloads, inspect generated output,
   or infer imports from arbitrary text.
5. Return repository-relative slash-separated file names and sort violations by importer, imported
   package, and file. Malformed Go or an unreadable/missing scanned root fails the check instead of
   producing an incomplete success.
6. Keep the checker repository-internal and run it in the ordinary Go test lane. It changes no
   compiler output, language behavior, CLI surface, artifact format, or runtime dependency.
7. Do not claim more than the structural oracle proves. It prevents direct dependency flow from
   Atlas/export code into unapproved packages. It does not make a binary build when Atlas source is
   itself uncompilable, isolate package initialization, prove downstream LLM safety, or replace the
   stale-map, incompleteness, and text-authority tests owned by their respective stages.

## Evidence

- The repository test parses the current `cmd/` and `internal/` trees and requires zero violations.
- A mutation-sensitive temporary tree introduces `internal/compiler -> internal/contextpack` and
  `internal/mir test -> internal/attributedtext`; both exact violations are required in canonical
  order.
- The same fixture proves that `internal/cli`, Atlas-to-Atlas imports, and the Windows evidence
  harness remain allowed, while comments and lookalike external paths do not create false matches.
- A malformed Go file and missing policy root fail closed with a repository-relative diagnostic.

## Consequences

The first Phase 4 exit condition now has a short stage-local oracle rather than relying on the final
compiler binary or a manual grep. New compiler-core packages are protected without editing a core
package list. Adding another Atlas consumer is intentionally visible as an architecture change.

The CLI remains a statically linked orchestration adapter, so this is not a plugin or process-
isolation boundary. Stable command tests still own their behavior; this decision only prevents Atlas
from becoming an input dependency of the compiler semantic core.

## Rejected alternatives

- List only today's frontend packages: silently leaves every new compiler package unprotected.
- Search source text with a regular expression: confuses comments and strings with imports and misses
  valid alternate formatting.
- Use `go list` as the oracle: obeys host build tags and can omit files that must remain protected,
  while also adding toolchain and module-loading behavior to a simple source contract.
- Forbid Atlas in `internal/cli`: would require a separate executable or plugin boundary that the
  current product has not selected.
- Claim complete runtime fault isolation: the current static binary and this direct-import oracle do
  not prove it.
