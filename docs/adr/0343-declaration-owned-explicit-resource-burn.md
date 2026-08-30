# ADR 0343: Declaration-owned explicit resource burn

- Status: Accepted
- Date: 2026-08-30
- Product source: `docs/product/02-spec.md`
- Language contract: `docs/language/0009-resource-abilities-v0.md`
- MIR contract: `docs/mir/drop-elaboration-v0.md`
- Depends on: ADR 0336, ADR 0341, ADR 0342

## Context

Language 11 made no-drop resources stable but intentionally supplied no escape hatch for destroying
one. That fail-closed baseline prevents accidental asset loss, yet it also prevents a declaration
from implementing a deliberate asset sink. A generic `forget`, unchecked cast, or empty cleanup
would erase the ownership obligation without preserving who authorized the destruction.

## Decision

1. `fairyc-language/12` admits `burn <path>;`. The operand must be one bare local or parameter that
   owns a direct resource value.
2. Burn authority belongs to the package and module that declare the resource. External code can
   transfer ownership to a public function in that module, but cannot burn the imported resource
   directly.
3. Only an explicit no-drop resource with no resource-bearing fields may be burned. Droppable
   resources, projections, temporaries, wrappers, foreign-module resources, and recursive or
   resource-bearing aggregates fail with `E1109`.
4. Flow treats a successful burn as one consuming move and discharges exactly one live ownership
   obligation. Later use, double burn, and fallthrough cleanup remain rejected by existing linear
   analysis.
5. MIR uses a distinct `Burn` operation. `V1104` rejects malformed operands or type mismatches, and
   `V1101` rejects a second discharge. Burn is not encoded as `Drop` and cannot invoke drop glue.
6. The interpreter removes the live value without running cleanup. The C backend emits no cleanup
   call and retains no otherwise unreachable drop glue solely for a burn.
7. The compiler-bundled SDK advances to `0.0.0-stage0.54`; manifests and fixtures use language 12.
   Public-interface v4, MIR artifact schema, runtime ABI `fairyc-runtime/0`, and C ABI identities do
   not change because the operation is compiler-internal and adds no public representation.

## Evidence

Parser, semantic, Flow, MIR lowering/verifier, interpreter, and C-emission tests cover accepted and
rejected operands. `examples/asset_burn_quickstart/` declares a no-drop asset whose cleanup body
traps, burns it in its declaring module, and returns 42. The Windows host smoke builds and runs that
generated-Fairy program under low and optimized MSVC profiles; reaching 42 proves that neither
profile invoked cleanup.

## Consequences

Burn remains deliberately narrower than ordinary destruction. Declaration-owned wrapper APIs are
the cross-module authority boundary. Mint, split, merge, persistent storage, event emission,
contract entry points, gas accounting, and chain-specific ABI generation remain separate vertical
slices. Any later relaxation for resource-bearing values needs a recursive authorization rule and
new MIR/backend evidence rather than broadening `burn` into a generic forget primitive.
