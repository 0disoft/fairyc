# ADR 0020: Stage 0 lexical Region MIR and interpreter slice

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-18
- Product source: `docs/product/02-spec.md`
- Language source: `docs/language/0003-regions-and-arenas.md`
- MIR source: `docs/mir/mir-v0.md`
- Follow-up: `docs/adr/0021-stage-0-region-early-exit-cleanup.md`

## Context

Region Flow already proved hidden origins, escape rejection, plain-region element restrictions, and
the package identity of `std::alloc::Arena`, but verified MIR and the interpreter still had no
source-produced region operations. A semantic-only pass could not prove that inner resources drop
before reset or that a normal return cannot leave a live region marker.

The bounded fixture also cannot invent a public Arena constructor merely to make an executable
example. Doing so would freeze standard-library capacity and construction choices before the
runtime contract exists.

## Decision

MIR v0 adds explicit `region-enter`, `region-place`, and `region-reset` operations. Normal lexical
fallthrough drops resources introduced inside the region in reverse declaration order, then resets
the region. Outer resources remain live and are cleaned by their owning scope. A trap remains
non-unwinding and performs no observable reset.

The interpreter models placed inert values and records one reset event containing the lexical
Region span, placement count, and number of resource drops completed before reset. It does not
expose addresses, offsets, capacity, allocation strategy, or an Arena representation.

The executable fixture declares `std::alloc::main` inside the Arena owner module. That module may
initialize its own private field without adding a public constructor. The fixture proves one inert
placement, an inner Token drop before reset, and the outer Arena drop after reset.

This first lowering slice accepts only a direct, side-effect-free `&mut arena_local` source. Calls
or other expressions that produce `&mut Arena` remain a capability error until their evaluation,
borrow extent, and cleanup ordering are represented explicitly; lowering never skips such an
expression.

`V1403` rejects malformed enter/place/reset shape, use of an inactive region, duplicate entry, and
normal return with a region still active. The verifier mutation suite removes one reset from a
verified plain-region seed and requires the isolated `V1403` result.

Early `return`, `?` propagation, `break`, and `continue` across a Region boundary remain outside
this first executable slice. Lowering rejects those paths instead of emitting a function that can
skip reset. They require scope-indexed cleanup edges shared with general nested-exit elaboration.

## Consequences

- MIR and interpreter fixtures now prove normal Region placement and reset ordering from Fairy C
  source rather than verifier-only seed metadata.
- The unstable one-file CLI can execute the same owner-module Region program when its relative path
  matches `std/alloc.fy`.
- No public standard-library constructor, arena capacity model, allocator ABI, or address contract
  is introduced.
- ADR 0021 later closes direct-return and Result-propagation exits, ADR 0022 adds only a
  fixture-authorized in-memory output harness, and ADR 0024 closes nearest-loop `break` and
  `continue` cleanup. Real host I/O remains a separate Phase 1 gate.

## Rejected alternatives

- Add `Arena.new()` only for the fixture: freezes a public API to satisfy test setup.
- Treat reset as interpreter cleanup outside MIR: lets backends and the oracle disagree.
- Allow early exits before cleanup-edge lowering: can silently skip reset.
- Expose arena offsets in fixture output: freezes representation rather than behavior.
