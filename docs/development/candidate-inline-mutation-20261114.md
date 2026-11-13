# Candidate straight-line inline mutation — 2026-11-14

ADR 0557 is implemented in Go typed identity and both Fairy identity owners:
original-source admission and detached typed-body verification. Value versions
advance after each initializer/assignment; copied immutable locals keep their
earlier values. Mutable local declarations use local kind 3, not borrowed-value
origin metadata. Parameters remain immutable and assignment targets must name
previously initialized mutable local declarations. Plain `=` only is admitted.

Validation executed without desktop/input interaction:

- Go `TestInlineMutation*`, existing inline locals and cross-package inline
  controls pass (1.775 s), including native execution, MIR restore, assignment
  identity changes and an unused overflowing assignment.
- `TestCandidateInlineLocalsIdentity` passes (18.38 s): exact Go/Fairy hashes,
  old canonical identities, refusal controls, local/expression limits, low and
  optimized native source-admission probes.
- `TestCandidateInlineMutationPipeline` passes (247.67 s): two producer and
  two consumer profiles, three independent runtime scenarios (twelve products),
  actual source/body/program/project codecs and restore, private header linkage,
  and five detached-body forgeries. Division/overflow traps retain the expected
  provider source ranges and execution order.
- The signed-Result, nested-Option and inline-mutation CLI fixtures independently
  execute before/after edits through the Go CLI (8.48 s). Full authored CLI/cache
  replay is a separate batched gate and is still pending this receipt.

Branches, loops, cross-module inline calls, composite generic arguments and
other inline scalar types are not implied by this bounded mutation family.
