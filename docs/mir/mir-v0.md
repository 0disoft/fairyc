# MIR v0

- Status: Draft Stage 0 contract
- Product source: `docs/product/02-spec.md`
- Architecture decision: `docs/adr/0001-initial-architecture-boundaries.md`

## Role

MIR is the only executable semantic contract consumed by backends. It records evaluation order, control flow, moves, cleanup, checks, traps, calls, and unsafe provenance explicitly. C, a future fast backend, a future LLVM backend, and the MIR interpreter must not reconstruct source semantics independently.

MIR v0 is a mutable-local control-flow graph rather than SSA. SSA conversion is a backend or optimization concern after the semantic form is verified.

The current Go prototype implements the source-linked subset exercised by all eleven MIR v0 normalization fixtures: ordered calls, complete-root and tagged-payload moves, RHS-before-drop resource reassignment, explicit normal-exit cleanup, lexical Region placement and reset on fallthrough, direct return, Result propagation, nested exits, nearest-loop `break` and `continue`, `Result` propagation branches, checked division, remainder, and arithmetic traps, and non-unwinding successors. The same verifier passes all ten named baselines and rejects all nineteen isolated v0 mutations with their single expected `Vxxxx` result. The bounded interpreter executes all fifty-six interpreter v0 cases for scalar and tagged control, resource construction and field reads, whole-root and active-payload moves, resource returns and propagation, live/dead/self reassignment, consuming-close ownership, nested and loop-control cleanup, source-linked drop traces, Region reset paths, controlled in-memory test output, exact arithmetic, division, remainder, and right-shift trap and boundary outcomes, and the step and output limits; user drop-body effects, signed left shift, and later runtime behavior remain unfinished gates.

## Lowering pipeline

The required order is:

1. resolve and type typed HIR;
2. normalize expression evaluation from left to right into explicit temporaries;
3. build an analysis-only Flow CFG;
4. lower short-circuit control and `?` into branches;
5. attach lexical scope-exit sets to edges;
6. compute initialization and ownership states to a fixed point;
7. check borrows, return origins, regions, escapes, and must-use results;
8. compute cleanup obligations;
9. elaborate aggregate partial initialization and call-argument temporaries;
10. reject source-visible complete roots that remain `MaybeLive`, then introduce drop flags only for compiler-owned partial construction or another explicitly specified conditional obligation;
11. create cleanup paths for normal exits;
12. merge only identical cleanup suffixes;
13. verify MIR invariants;
14. publish the verified MIR to the interpreter, backends, and `SemanticSnapshot` producer.

Borrow instructions may be erased for machine code after verification, but verified borrow and origin facts remain available to Atlas.

## Core model

### Places and operands

A place names a local and optional projections. Resource move operands may name only a root local in Stage 0.

Operands are:

- `Copy(place)` for copyable values;
- `Move(local)` for a complete resource root or another non-copy value;
- `Const(value)` for a typed constant.

Function signatures retain parameter types in the same order as their parameter locals. A backend
must reject a missing or mismatched type list instead of inferring a scalar, borrow, resource, or ABI
representation from how a local happens to be used.

### Rvalues

MIR v0 requires rvalues for use, unary operations, checked arithmetic, wrapping arithmetic, saturating arithmetic, comparisons, casts, aggregate construction, discriminant read, safe borrow, raw address, region allocation, and explicit unsafe operations.

Each arithmetic family has distinct operations. Backends cannot infer trapping or wrapping from a build profile.

Ordinary `i32` negation, addition, subtraction, multiplication, division, remainder, and signed
right shift use an explicit check block, a two-successor switch, a non-returning trap block, and a
fresh continuation containing the operation. The marker and switch carry the same operands and
operator as the continuation. Addition, subtraction, multiplication, and negation expose
`integer-overflow`; right shift exposes `invalid-shift`; division and remainder expose
`division-by-zero` followed by `signed-division-overflow`. Signed left shift is rejected before MIR.

### Statements

Statements include local storage markers, assignment, drop-flag update, `Drop`, region mark or reset, and explicit provenance markers for unsafe operations. Assignments consume already-evaluated operands; they do not contain nested side effects.

### Terminators

Terminators include `Goto`, integer, boolean, or discriminant switch, `Call`, `Return`, `Trap`, and `Unreachable`. A boolean switch has exactly two successors ordered true then false. Source `&&` and `||` are lowered to boolean switches and branch-local assignments before verification. Source `break` and `continue` become cleanup followed by a goto to the nearest loop exit or header; neither source form remains in MIR. Stage 0 calls have normal successors only. There is no unwind successor.

Fixture-visible trap categories use the canonical identities `bounds`, `integer-overflow`, `division-by-zero`, `signed-division-overflow`, `invalid-shift`, `assertion-failed`, and `unreachable`. A backend may choose a target-specific termination mechanism, but it cannot rename or merge the semantic outcome reported by the interpreter and differential harness.

## Calls

Arguments are fully evaluated into temporaries from left to right before the call terminator. A call records its destination, normal continuation, source span, unsafe scope when applicable, and foreign ABI when applicable.

Foreign calls cannot expose a safe reference in their raw MIR signature. Unknown foreign functions receive no alias, no-unwind, read-only, or escape attributes unless an explicit wrapper contract proves them.

## Verification invariants

The verifier runs before every interpreter or backend handoff.

| Code | Invariant |
| --- | --- |
| `V1001` | Every reachable block ends in exactly one terminator. |
| `V1002` | Every use is dominated by an initialized assignment on all incoming paths. |
| `V1101` | A resource root is moved or dropped at most once. |
| `V1102` | A live resource on a normal exiting edge has exactly one cleanup obligation. |
| `V1103` | No resource projection appears in a `Move` operand. |
| `V1201` | A drop flag is initialized before use and agrees with ownership transfer. |
| `V1202` | `Drop` clears its active flag before invoking cleanup. |
| `V1301` | No safe borrow outlives or conflicts with its verified origin and loan. |
| `V1302` | A safe-borrow return has exactly one permitted origin. |
| `V1401` | A region-derived value does not escape its region. |
| `V1402` | Plain-region allocation never receives a `NeedsDrop` type. |
| `V1403` | Normal control flow enters, places into, and resets each lexical Region exactly once. |
| `V1501` | Every potentially trapping operation has one complete checked predecessor with matching marker, operands, operator, categories, success edge, and non-returning trap edge. |
| `V1502` | A Stage 0 call has no unwind successor. |
| `V1601` | `?`, implicit drop, source short-circuiting, and nested side effects do not remain. |
| `V1701` | Every unsafe instruction carries a source span and unsafe-scope identity. |

A `Vxxxx` failure is a compiler defect, not a user diagnostic. Release builds must report an internal compiler error with a bounded MIR excerpt and reproducible input reference; exact crash-bundle behavior remains UNDECIDED.

## Fixture contract

`spec/fixtures/mir/manifest-v0.json` records source-linked observations after MIR verification. It freezes ordered evaluation, non-unwinding calls, explicit moves, cleanup action order, `Result` branching, trap paths, and eliminated source-only forms without freezing block numbers, local numbers, temporary names, or textual MIR.

The source cases do not test malformed graphs. `spec/fixtures/mir-verifier/manifest-v0.json` covers every current `Vxxxx` invariant through verified named seeds and one isolated mutation per case, without defining arbitrary serialized MIR.

## Interpreter

The first semantic oracle is a small MIR interpreter supporting scalars, tagged `Option` and `Result`, moves, drop traces, region mark and reset, branches, and traps. Its options-only fixture harness may intercept explicitly bound call symbols as bounded in-memory stdout/stderr byte sinks; those hooks are not MIR operations, source-name conventions, standard-library APIs, or host I/O. The interpreter is not required to emulate a complete operating system or C ABI.

The interpreter precedes a second native backend. Differential fixtures compare interpreter outcomes with generated C compiled at multiple optimization levels and, where the target fixture declares them, more than one independent C compiler.

The current Go prototype also emits deterministic ISO C11 for the verified scalar subset in
`spec/fixtures/c-emission/manifest-v0.json`. That runner stops at C text and therefore is backend
emission evidence, not native differential evidence.
