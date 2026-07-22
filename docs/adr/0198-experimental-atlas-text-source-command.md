# Experimental Atlas source-text command

- Status: Accepted
- Date: 2026-07-23
- Product source: `docs/product/02-spec.md`
- Encoding prerequisite: `docs/adr/0197-canonical-attributed-text-json-v0.md`
- CLI contract: `docs/cli/command-contract.md`

## Context

ADR 0197 defines a canonical attributed-text artifact but intentionally leaves command selection,
argument policy, stdout behavior, and failure classification undecided. Atlas Context Pack v1 must
continue to contain compiler facts only, so exposing exact source text by filling its empty text
arrays would silently change an accepted schema. A command also cannot infer text, output, or count
budgets without hiding resource policy from automation callers.

## Decision

1. Add only the unstable command `fairyc experimental atlas-text source`.
2. Require `--package-uid`, one canonical visible or full `--symbol`, explicit
   `--origin workspace-authored`, positive `--max-fragments`, `--max-fragment-bytes`,
   `--max-text-bytes`, and `--max-output-bytes`; admit optional `--locked` and no other option.
3. Fix `source` as the explicit source-excerpt selection. Do not add a redundant `--kind`, infer a
   symbol by name, accept compiler-bundled SDK text, or provide any numeric default.
4. Reuse the manifest-owned exact workspace, compiler identity, current Atlas partition, and lock
   boundary used by the existing Atlas commands. Ordinary mode may maintain `fairy.lock` before a
   later selector or budget failure; `--locked` verifies the exact lock without writing.
5. Write exactly one compact canonical `fairyc.attributed-text-json/v0` value to stdout with no
   trailing newline. Accept visible and full forms of the same current SymbolId and canonicalize
   both to byte-identical output.
6. Treat a source fragment that exceeds the explicit text budget as successful, checksummed output
   containing one `fragment-byte-limit` omission and no source bytes. Treat a final envelope that
   exceeds `--max-output-bytes` as an atomic invocation failure with no stdout.
7. Use provisional code `2` for invalid or duplicate options, an unknown package or exact symbol,
   and an insufficient final-output bound; `3` for source diagnostics; `4` for internal invariant,
   build-identity, or output-write failures; and `7` for manifest, workspace, graph, or lock failure.
8. Keep `.fymap` v1 and Context Pack v1 unchanged. Add no decoder, input file, output path, text
   store, documentation or generated-text attachment, dependency acquisition, model call, token
   estimate, permission, authentication, safe-change inference, or stable CLI promise.

## Evidence

- CLI tests discover the exact `helper` SymbolId through the catalog command, then require visible,
  full, and locked invocations to emit identical compact JSON with no host path or trailing newline.
- The selected source contains prompt-like printable text; tests require it to remain exact and
  explicitly `untrusted-text`, not to become a command or authority boundary.
- Budget tests require a successful checksummed omission with no source body, while invalid flags,
  unsupported origin, malformed or absent SymbolIds, unknown packages and symbols, guard overflow,
  incompatible text limits, final-output overflow, and writer failure preserve their distinct
  process classes.

## Consequences

Callers can now compose the exact SymbolId catalog with one explicit source-text request without
changing compiler-fact formats or selecting a persistent store. The command is intentionally
awkward for humans because every resource and origin choice stays visible. A future documentation
command, multi-fragment producer, decoder, composite pack, or stable CLI requires another decision.

## Rejected alternatives

- Add text to Context Pack v1: changes an accepted compiler-fact-only schema.
- Infer `workspace-authored` or numeric limits: hides trust and resource policy.
- Return code `2` for an admitted source omission: confuses a complete policy result with command
  failure and discards checksummed omission evidence.
- Accept output paths: creates persistence, overwrite, containment, and cleanup policy that this
  command does not own.
