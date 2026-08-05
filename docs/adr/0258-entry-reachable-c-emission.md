# ADR 0258: Entry-Reachable C Emission

- Status: Accepted
- Date: 2026-08-05

## Context

`cbackend.Emit` emitted every function in the verified `mir.Program`, including SDK functions
unreachable from the selected entry, and emitted a prototype plus macro guard for every foreign
declaration in the program. Generated-Fairy probes therefore linked the full SDK foreign closure
even when the probe used only a few symbols, and dead functions kept C-backend capabilities and
foreign-signature scalar features alive in the generated source. Function values, indirect calls,
exported library roots, and extern callbacks do not exist in Stage 0: the entry is the only root.

## Decision

1. The emission view is filtered inside `internal/cbackend.Emit` to the entry-reachable closure.
   The verified `mir.Program`, package artifact/cache assembly, language grammar, SDK source/API/
   version, runtime ABI/version, and `cstdlibcontract` are unchanged.
2. Root is exactly the selected entry. Edges are collected per reachable basic block:
   - `TerminatorCall` with no foreign ABI names one internal callee;
   - `OperationDrop` on a resource local reaches that resource type's drop glue (direct resource or
     direct resource payloads of an Option/Result discharge), and the drop glue's own body then
     contributes its internal calls and drops;
   - `TerminatorCall` with a foreign ABI records the foreign declaration, which is filtered into
     the emitted view by the normalized LinkName/symbol index. Missing reachable declarations and
     unknown internal callees fail closed with the existing messages.
3. Reachable foreign symbols remain exactly what reachable-function inspection records: foreign
   call LinkNames plus the three Region runtime symbols only when a reachable function performs a
   Region operation. Unreachable foreign and Region usage never enters `unit.ForeignSymbols`.
4. `program.Functions` is name-sorted first, then indexed for duplicate-symbol, duplicate-entry,
   and missing-entry checks. Traversal visits that name-sorted slice and the final emitted
   functions are the visited indices in the same name order, so output order is deterministic and
   independent of traversal order.
5. The filtered function slice and the filtered foreign view rebuild the emitter's symbol maps,
   entry index, locals/types/storage, capability inspection, prototypes, bodies, `unit.Functions`,
   `unit.ForeignSymbols`, and `unit.Source`. The original MIR is never mutated.
6. The generated `main` keeps `(void)&fy_fn_N` only for reachable non-entry functions.
7. Dead functions cannot block emission with unsupported C constructs; reachable functions still
   fail closed exactly as before. The drop-trace profile and the normal profile share the same
   reachable `unit.Functions` contract.
8. Named type definitions (`StructTypes`, `ResourceTypes`, `EnumTypes`) are not dead-code
   eliminated in this decision; full type DCE remains a separate follow-up optimization boundary.
9. `unit.Functions` keeps its meaning as the emitted function list; only its value is minimized.

## Measurement

Counts are observed facts only; no build-time or artifact-size improvement claim is made. The
before counts were observed once in a temporary detached worktree at HEAD `314aa60`; no further
raw measurement runs were made. After counts are the checked-in Windows host-smoke pipeline
measurements after this change.

| Probe | Before ForeignSymbols | Before Functions | Before Source bytes | After ForeignSymbols | After Functions | After Source bytes |
| --- | --- | --- | --- | --- | --- | --- |
| Buffer `with_capacity` | 19 | 54 | 55135 | 2 | 8 | 16500 |
| Buffer `resize_zeroed` | not measured | not measured | not measured | 3 | 11 | 20541 |
| Region failure | not measured | not measured | not measured | 5 | 8 | 18763 |
| String `append` | not measured | not measured | not measured | 3 | 14 | 24708 |

## Consequences

- Generated-Fairy probes now pass `unit.ForeignSymbols` as the entry-reachable minimal set, and
  the host-smoke unit test freezes those sets for Region, Buffer, and String.
- Dead foreign declarations no longer leak prototypes, macro guards, or foreign-signature scalar
  features into generated C, and dead functions no longer keep C-backend capabilities alive.
- This is an emission-view change only. Language remains `fairyc-language/8`, SDK remains
  `0.0.0-stage0.10`, and runtime ABI remains `fairyc-runtime/0`.
- The native Region/Buffer/String generated-Fairy proofs are unchanged behaviorally and continue
  to execute at low and optimized MSVC profiles.
