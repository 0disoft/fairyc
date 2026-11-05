# Owner-qualified Fairy Atlas declarations, 2026-11-05

Scope: bring the existing Go owner-qualified function source family into Fairy
Complete-bound Atlas production, current-source admission and impact/MCP/body-edit
commands. This is one slice of `implementation-queue-20261105.md`, not completion
of that objective. Native Linux remains batched; no push or dispatch.

## Required facts and sequence

1. Go snapshot/codec/query prerequisite: preserve the written declaration path
   `Cell::get` exactly once, the nominal owner with its defining module/package
   coordinate, the exact receiver form and ordered parameters including `this`.
   Local and imported call endpoints must resolve the same declaration.
2. Fairy Complete prerequisite: inspect and extend signature, body-call and public
   interface owners for all eight source cases below. The existing signature
   check admits borrowed receivers only; zero-parameter associated functions,
   consuming receivers and public owner-qualified interface extraction must not
   be mistaken for already supported Complete inputs.
3. Fairy Atlas production: retain those exact checked facts, qualified keys and
   canonical SymbolId v1 identities through production, reader/current validation
   and complete workspace joins. Preserve existing depth/work/admission bounds.
4. Actual impact/MCP/body editing and a fresh full authored-main replay. Keep all
   existing groups and independent current-source Go comparisons. Historical
   closure/pre-closure image results cannot attest a changed image.

The source matrix is associated/no parameter, associated/one parameter, shared
receiver, exclusive receiver, consuming record, consuming resource, imported
shared receiver and unused associated function. All remain requirements; an
implementation accepting only the already-supported borrowed subset is partial.
Generic owner methods and first-class closures are separate expansions.

## Go prerequisite evidence

`TestAtlasOwnerQualifiedSourceOracle` reproduces the source family against the
normal checked workspace pipeline, canonical v3 encode/decode, complete joins and
exact qualified-name catalog lookup. It independently checks owner path/package,
receiver, receiver-parameter retention and expected resolved call count; unused
associated functions retain declarations without inventing calls.

Two production problems were reproduced: snapshot construction doubled the owner
prefix (`Cell::Cell::get`), and the codec rejected contract-valid qualified keys.
The codec and query now reuse the existing nonempty Stage 0 path validator from
the SymbolId v1 contract, without changing its identity encoding or map version.
`not::an_identifier` is a valid path, so its old negative query control is replaced
with `not:::an_identifier`; a positive empty-result control preserves the valid
unknown-path behavior. Eleven malformed paths still reject at local/call keys.

Windows checks:

- Full `internal/semanticsnapshot` package: pass, 0.209s.
- Full `internal/atlasmap` package: pass, 2.022s. The initial run preserved the
  obsolete exact-name negative-control failure before that control was corrected.
- Eight source oracles: pass, 0.546s package time, including shared-method docs.
- Scoped Go vet: pass for snapshot, map and CLI packages.
- Formatting and `git diff --check`: pass.

Logs are local `.cache/atlas-owner-qualified-go-before-20261105.log`,
`.cache/atlas-owner-qualified-go-final-20261105.log` and
`.cache/atlas-owner-qualified-oracle-final-20261105.log`.

Fairy production, executable methods, actual commands, new-image full CLI,
consecutive self-build, clean-machine distribution, same-commit CI and native
Linux remain unproved here. No source language, stable CLI flag, SDK/runtime ABI,
map wire version, dependency, DB, runner or repository policy is changed.
