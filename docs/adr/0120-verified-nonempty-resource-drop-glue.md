# ADR 0120: Execute nonempty resource drop bodies as verified drop glue

- Status: Accepted
- Date: 2026-07-21
- Language source: `docs/language/0001-values-and-resources.md`
- MIR source: `docs/mir/drop-elaboration-v0.md`
- Backend source: `docs/mir/c-backend-safety-v0.md`

## Context

The parser and AST preserved `drop fn(self: &mut Self) { ... }`, but executable lowering admitted
only an empty body. A backend-only text substitution would bypass resolution, typing, effects, MIR
verification, and ordinary call lowering. Treating the resource as an ordinary owning parameter
would instead schedule recursive cleanup of the value while its own destructor was running.

## Decision

1. A nonempty resource drop body is resolved and type-checked as one compiler-private function. It
   is not inserted into the module's source-visible namespace.
2. Typed HIR and MIR link the resource definition to that function with a symbol and mark the
   function with its exact `DropGlueFor` resource type. The internal callable uses one by-value
   resource parameter solely as a lowering representation for source `&mut Self`.
3. The drop receiver does not own a second cleanup obligation. The MIR verifier requires exactly
   one matching `fn(Resource) -> void` glue function and rejects a recursively owned receiver.
4. Interpreter resource state is `live`, `dropping`, then `dropped`. The body may read `self`
   while `dropping`; a second drop is rejected. The body runs before reverse field cleanup.
5. The constrained C backend invokes the verified private function for direct resources and the
   active resource payload of admitted `Option` or `Result` values. Production C does not paste or
   reconstruct source text.
6. This slice admits only a pure drop body under the existing source-effect checker. Resource-level
   drop-effect declarations and implicit-drop effect propagation are deferred. Therefore it does
   not yet authorize a safe File destructor that calls the host filesystem.

## Consequences

- Nonempty drop behavior now survives every compiler stage and executes in the MIR interpreter.
- The Windows target example takes the nonempty path through generated C, local x64 MSVC, and a
  retained-output-free native process without requiring WSL.
- Raw handle release remains unavailable until drop effects and the Windows File runtime are
  connected; pointer-width resource transport alone is not advertised as safe ownership.

## Verification

- semantic tests require the private symbol and exact glue signature;
- MIR verification checks symbol/type/signature/receiver-cleanup consistency;
- an interpreter test observes a byte emitted only from the drop body through a closed test
  intrinsic;
- the 34th C-emission fixture requires `non-empty-resource-drop` and deterministic private glue;
- the Windows host smoke compiles and executes the checked-in nonempty resource with local MSVC.
