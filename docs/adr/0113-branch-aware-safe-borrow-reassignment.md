# ADR 0113: Branch-aware safe-borrow reassignment

- Status: Accepted
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Language source: `docs/language/0002-borrows-and-return-origins.md`
- Diagnostic source: `docs/diagnostics/catalog-v0.md`
- Semantic fixtures: `spec/fixtures/semantic/manifest-v0.json`

## Context

Stage 0 already permits safe borrows in local variables, but the first Flow pass treated each
binding as if it had one immutable origin for the whole function. Allowing assignment under that
model would either retain a dead old loan, lose the new owner's protection, or select whichever
branch happened to be visited last. All three outcomes are wrong.

## Decision

1. A safe-borrow local may be reassigned outside a loop. The right-hand side is evaluated first;
   only then does its proven origin and retained-loan set replace the target's previous state.
2. Each retained direct borrow creates a distinct loan generation. Copying a borrow preserves that
   generation, while replacing one local ends only that local's route to the old loan. Other aliases
   continue to keep the owner protected.
3. Flow clones borrow environments for `if` and `match` branches and joins only reachable exits.
   Equal origins compact to one fact. Different possible origins remain an explicit disjunction;
   returning the merged value is `E2103`, never an arbitrary branch choice.
4. Loan conflict checks use branch-reachable backward liveness. An owner assignment conflicts only
   when a borrow value that can reach a later use on that same control path still carries the
   owner's loan. A use in the mutually exclusive branch does not create a false `E2101`.
5. Direct safe-borrow returns still require one `Static`, `Param(i)`, or `RegionParam(i)` origin.
   `Local(name)` remains `E2102`, and a local-region origin remains `E3101`.
6. A borrow-returning internal call with exactly one safe-borrow argument propagates that
   argument's origin exactly. Calls with multiple safe-borrow arguments conservatively retain their
   union until function summaries encode the selected parameter. A call with no provable borrow
   argument remains unsupported rather than inventing `Static` provenance.
7. Loop-carried safe-borrow reassignment remains closed. Its environment needs an explicit
   iteration-carried fixed point and break/continue edge contract rather than reusing branch rules.
   Backward loop liveness is still a bounded finite-source fixed point for already admitted loans.
8. Typed HIR and MIR need no new assignment operation. Once Flow proves the source assignment,
   ordinary MIR local assignment transports the non-owning value descriptor. Interpreter and C
   backends therefore consume the same verified MIR assignment.

## Consequences

- A local can intentionally switch from one safe owner to another without keeping the old owner
  borrowed after its last reachable use.
- Branch joins preserve uncertainty instead of erasing it, making diagnostics deterministic and
  fail-closed.
- General interprocedural return-origin summaries and loop-carried borrow phis remain later work.

## Verification

- the semantic fixture contract grows from 23 to 25 cases with a passing same-origin branch
  reassignment and a failing different-origin return;
- unit tests cover sequential replacement, new-owner protection, old-owner release, mutually
  exclusive branch liveness, same-branch conflict, and the loop gate;
- interpreter and constrained-C tests execute reassigned fixed-array byte views;
- the WSL-free Windows quickstart reassigns one `&[u8]` descriptor and observes the new array.
