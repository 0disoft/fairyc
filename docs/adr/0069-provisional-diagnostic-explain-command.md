# ADR 0069: Provisional diagnostic explanation command

- Status: Accepted
- Date: 2026-07-19

## Context

Diagnostic codes already identify prototype parser, semantic, Atlas, and MIR-verifier rules across
the Stage 0 contracts. Developers can search the repository for a code, but that is slow, depends on
the checkout layout, and encourages the short render-time error message to become an overloaded
documentation surface.

The planned CLI already reserves `fairyc explain <diagnostic-code>`. Implementing that lookup does
not require package selection, target selection, a backend, a source root, or any of the still-
undecided stable build syntax. It must not imply that every reserved v0 code is implemented or that
the pre-release diagnostic catalog is release-stable.

## Decision

1. `fairyc explain <diagnostic-code>` accepts exactly one canonical uppercase `Edddd`, `Adddd`, or
   `Vdddd` code and performs an exact lookup. It does not case-fold, fuzzy-match, list nearby codes,
   or silently reinterpret malformed input.
2. A successful lookup writes three deterministic, ANSI-free lines to stdout: code and meaning,
   category, and `fairyc.diagnostic-catalog/v0 (provisional Stage 0)`. It reads no source file and
   performs no directory scan, toolchain discovery, process execution, network request, or write.
3. Missing, extra, malformed, and unknown codes are invocation failures with provisional code `2`,
   no stdout, and bounded stderr plus usage. Output-write failure uses internal code `4`.
4. The Go catalog is the executable lookup copy. Exact `E` and `A` meanings remain owned by
   `docs/diagnostics/catalog-v0.md`; exact `V` meanings remain owned by `docs/mir/mir-v0.md`. A test
   parses both Markdown tables and rejects missing, extra, or textually drifting executable entries.
5. Ordinary source diagnostics may use the catalog meaning as a fail-closed human fallback when a
   shorter contextual renderer is absent. Dynamic render details such as an expected token remain
   owned by the existing renderer.
6. A successful explanation says only that the code is reserved by the prototype catalog. It does
   not claim that the rule is currently emitted, fixture-backed, released, or stable. Machine-
   readable `explain --json`, code listing, search, localization, and richer repair guidance remain
   separate decisions.

## Consequences

- A diagnostic code can be explained on Windows or any other compiler host without a source tree
  operation, C toolchain, WSL, or supported target.
- Documentation stays authoritative while executable lookup cannot silently drift from its exact
  one-line meanings.
- The top-level spelling is implemented early, but Phase 3 still owns release stabilization of the
  broader CLI, positional syntax, exit taxonomy, and machine-readable output.
