# ADR 0115: Loop-carried safe-borrow provenance

- Status: Accepted
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Language sources: `docs/language/0002-borrows-and-return-origins.md`, `docs/language/0007-name-resolution-and-typing-v0.md`
- Semantic fixtures: `spec/fixtures/semantic/manifest-v0.json`

## Context

ADR 0113 admitted safe-borrow reassignment across straight-line and branch control flow but kept
loops closed. Removing that gate without an iteration model would either forget a new owner's loan
at the next header or retain one freshly numbered loan per analysis round and never converge.
`break` and `continue` also have different destinations and cannot share one exit merge.

## Decision

1. A mutable safe-borrow local may be reassigned inside `while`. Right-hand-side-first replacement,
   alias preservation, single-origin return checks, and owner-conflict rules remain unchanged.
2. Forward borrow Flow computes a loop-header fixed point. The initial entry, reachable body
   fallthrough, and every nearest-loop `continue` environment feed the next header. Each iteration
   joins origins, retained loan generations, and known/unknown state for bindings that existed at
   loop entry.
3. A direct borrow retained at the same source position in the same binding uses one canonical
   static loan-site ID across analysis rounds. This ID abstracts any dynamic iteration's generation;
   aliases still preserve it and a later assignment still replaces only that binding's route.
4. Fixed-point probes suppress diagnostics, unsupported reports, and escaping-return facts. After
   convergence the header and body are evaluated once with observable effects, preventing repeated
   diagnostics or duplicate fixture-facing borrow facts.
5. The loop may execute zero times. The converged header therefore represents condition-false
   exits before or after any iteration. Every nearest-loop `break` environment is joined separately
   into the post-loop environment. `continue` never exits the loop.
6. Backward loan liveness uses its existing finite-source fixed point. A `continue` targets header
   liveness and a `break` targets post-loop liveness, so owner conflicts require a later reachable
   use on the corresponding control path.
7. Returning a post-loop borrow that may originate from different entry or iteration sources
   remains `E2103`. Flow does not use constant conditions, scalar-value correlation, or an
   at-least-once loop proof to erase the zero-iteration path.
8. Typed HIR and MIR need no borrow-specific phi operation. Ordinary local assignment and existing
   loop CFG edges already transport the non-owning descriptor after Flow verifies provenance.

## Consequences

- Safe byte-view and ordinary borrow locals can switch owners in real iterative code.
- `continue`-carried loans protect the selected owner in later iterations, while overwritten owners
  are released on the path after reassignment.
- Cross-iteration analysis is conservative when scalar predicates correlate paths; it never guesses
  that a loop executes or that two iterations are mutually exclusive.

## Verification

- semantic unit tests cover admission, old-owner release, new-owner protection, `continue`
  back-edges, and `break` exit-origin joins;
- the semantic fixture contract grows from 27 to 29 cases with a same-origin loop pass and a
  multi-origin `break` return failure;
- the WSL-free Windows quickstart reassigns a fixed-array byte view inside a one-iteration loop and
  reads byte 42 through generated C and x64 MSVC.
