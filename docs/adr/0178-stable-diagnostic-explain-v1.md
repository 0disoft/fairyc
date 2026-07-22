# ADR 0178: Stabilize diagnostic explanation v1

- Status: Accepted
- Date: 2026-07-22
- Diagnostic sources: `docs/diagnostics/catalog-v0.md`, `docs/mir/mir-v0.md`
- CLI sources: `docs/cli/command-contract.md`, `docs/cli/output-and-exit-codes.md`
- Prototype prerequisite: ADR 0069
- Implementation: `internal/cli/run.go`, `internal/diagnostic/catalog.go`

## Context

`fairyc explain` already provides exact lookup across every reserved parser, semantic, Atlas, and
MIR-verifier code without consulting a workspace or target. Its executable catalog is checked
against both owning Markdown tables. Keeping the top-level spelling provisional no longer protects
an unresolved semantic choice; it only withholds a useful compatibility boundary that is already
fully closed.

Stabilizing the command must not imply that every reserved diagnostic is emitted, fixture-backed,
or part of a released language. Human explanation wording also needs room to improve without
reassigning a code.

## Decision

Stabilize `fairyc explain <diagnostic-code>` as v1.

- The command accepts exactly one canonical uppercase `Edddd`, `Adddd`, or `Vdddd` code. It performs
  exact lookup only and rejects missing, extra, malformed, lowercase, and unknown values.
- Success writes exactly three ANSI-free lines: `<code>: <meaning>`, `Category: <category>`, and
  `Catalog: fairyc.diagnostic-catalog/v0 (provisional Stage 0)`.
- The line shape, code identity, category identity, catalog identity, and trailing newline are the
  stable human contract. Meaning prose may improve under the diagnostic compatibility policy, but a
  code is never silently reassigned.
- The command reads no source, manifest, lock, configuration, current directory, environment,
  target, toolchain, or network state and writes no file.
- Stable codes are 0 for a successful lookup, 2 for invalid invocation or unknown code, and 4 for
  output failure. Failure emits no partial stdout.
- `--json`, listing, search, localization, repair guidance, and claims that a rule is implemented or
  fixture-backed remain outside v1.

Command stability is independent from the catalog's explicit provisional Stage 0 lifecycle and
from future diagnostic JSON or span compatibility.

## Evidence

- catalog tests prove canonical unique codes, immutable lookup copies, and exact one-line meaning
  parity with the two authoritative Markdown tables;
- CLI tests execute every catalog entry through the exact three-line shape and cover all three code
  namespaces, malformed and unknown values, help, missing source roots, and output failure;
- full Go test, build, vet, and strict scaffold checks cover the shared command boundary.
