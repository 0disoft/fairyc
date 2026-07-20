# ADR 0114: Interprocedural borrow-return summaries

- Status: Accepted
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Language source: `docs/language/0002-borrows-and-return-origins.md`
- Semantic fixtures: `spec/fixtures/semantic/manifest-v0.json`

## Context

A function type says that a call returns a safe borrow but cannot say which input storage the
result aliases. Treating every safe-borrow argument as a possible result keeps soundness, but it
also keeps owners borrowed when the callee cannot return them. Counting arguments is worse:
`second(left, right)` and a function returning static storage can have the same type and different
provenance.

## Decision

1. After type checking and before ordinary Flow diagnostics, every internal function whose result
   contains a safe borrow receives a compiler-internal return summary. The summary vocabulary is
   `Param(i)`, `RegionParam(i)`, or one exact `Static` source coordinate. It is not source syntax,
   public ABI, or first-class function-type metadata.
2. The summary is inferred from reachable explicit returns and the function tail using the same
   assignment, branch-join, and call evaluation rules as ordinary borrow Flow. A local, raw,
   conflicting, absent, or otherwise unproved origin cannot produce a usable summary.
3. Summaries reach a fixed point through a reverse-call dependency worklist. A changed callee
   reschedules only its borrow-returning callers; the compiler does not rescan every function on
   every round. This resolves forward declarations, imported callees, argument reordering, and
   acyclic wrapper chains independently of source order. A recursive dependency whose result
   source never becomes fully proven remains unknown and fails closed when retained or returned.
4. A call evaluates its callee and every argument exactly once from left to right. `Param(i)` then
   selects only actual argument `i`, preserving that value's origin, loan generations, and pending
   direct borrow. `RegionParam(i)` remaps the region capability at the same position. `Static`
   retains its exact compiler-owned source without borrowing any argument.
5. Safe-borrow arguments not selected by the summary remain call-bounded. They are marked used by
   the call but do not keep their owners protected afterward. Aliases already retained elsewhere
   continue to carry their original loan generation.
6. Summaries do not weaken the single-origin return rule. Multiple reachable return origins remain
   `E2103`; local and local-region escapes remain `E2102` and `E3101`.
7. Typed HIR, MIR, the interpreter, and generated C need no new operation. The summary proves which
   already-lowered non-owning result descriptor the ordinary call transports.

## Consequences

- Multi-argument helpers can return one selected borrow without false owner conflicts at callers.
- Static-returning helpers no longer inherit irrelevant argument provenance.
- Recursive borrow-return inference remains deliberately incomplete rather than guessing.

## Verification

- semantic unit tests cover positional remapping, static returns, selected-owner protection, and
  unselected-owner release;
- the semantic fixture contract grows from 25 to 27 cases with exact `Param(i)` summaries and a
  selected-owner conflict;
- the WSL-free Windows quickstart mutates an unselected fixed-array argument after the call and
  reads the selected result through the generated C and x64 MSVC path.
