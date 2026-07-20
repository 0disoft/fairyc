# MIR v0

- Status: Draft Stage 0 contract
- Product source: `docs/product/02-spec.md`
- Architecture decision: `docs/adr/0001-initial-architecture-boundaries.md`

## Role

MIR is the only executable semantic contract consumed by backends. It records evaluation order, control flow, moves, cleanup, checks, traps, calls, and unsafe provenance explicitly. C, a future fast backend, a future LLVM backend, and the MIR interpreter must not reconstruct source semantics independently.

MIR v0 is a mutable-local control-flow graph rather than SSA. SSA conversion is a backend or optimization concern after the semantic form is verified.

The current Go prototype implements the source-linked subset exercised by all eleven MIR v0 normalization fixtures: ordered calls, complete-root and tagged-payload moves, RHS-before-drop resource reassignment, explicit normal-exit cleanup, lexical Region placement and reset on fallthrough, direct return, Result propagation, nested exits, nearest-loop `break` and `continue`, `Result` propagation branches, checked division, remainder, arithmetic, and right-shift traps, and non-unwinding successors. The same verifier passes all ten named baselines and rejects all twenty-one isolated v0 mutations with their single expected `Vxxxx` result. The bounded interpreter executes all seventy-two interpreter v0 cases for scalar and tagged control, typed scalar comparisons, checked byte indexing, value-preserving integer casts, and checked `i32`, `u8`, `u32`, and `usize` arithmetic, division, remainder, bitwise operations, and right shift, resource construction and field reads, whole-root and active-payload moves, resource returns and propagation, live/dead/self reassignment, consuming-close ownership, nested and loop-control cleanup, source-linked drop traces, Region reset paths, controlled in-memory test output, exact trap and boundary outcomes, and the step and output limits; user drop-body effects, left shift, and later runtime behavior remain unfinished gates.

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

MIR v0 requires rvalues for use, unary operations, checked arithmetic, wrapping arithmetic, saturating arithmetic, comparisons, casts, aggregate construction, discriminant read, borrowed-view length, safe borrow, raw address, region allocation, and explicit unsafe operations.

A scalar `binary` operation records its exact common operand type in `OwnerType` and exact result in
`Type`. Equality admits matching `bool`, `i32`, `u8`, `u32`, `usize`, or raw-pointer operands and
returns `bool`; ordering admits matching operands from the four Fairy integer types and returns
`bool`. The interpreter executes raw-pointer equality only when both values are typed null; native
  non-null pointer identity remains backend evidence. Binary `+`, `-`, and `*` also admit matching
  `i32`, `u8`, `u32`, or `usize` operands and return that same type. `/` and `%` admit the same
  matching integer set with an exact typed checked-division marker and switch. `&`, `^`, and `|`
  admit the same matching set without a trap; `>>` admits it through an exact typed checked-shift
  marker and switch. `V1001` rejects missing or forged type metadata,
invalid immediate families, mixed types, aggregate equality, pointer ordering, and boolean ordering.

Built-in `Option` and `Result` construction records the concrete constructed type. Inert struct,
restricted user enum, and resource construction use distinct operations backed by separate
program-level definitions, so
copyable layout values never acquire cleanup semantics by being mistaken for resources. Field reads
name the owner type and are verified against exactly one aggregate definition. Payload extraction
records the concrete extracted type. Missing type identity is `V1001`; an interpreter or backend
cannot infer it from source syntax, local numbering, or use sites.

Each restricted user enum definition preserves declaration-ordered variants and declaration-ordered
named payload fields. `enum-construct` records the exact variant and a complete unique field set;
`enum-payload` records owner type, active variant, field, and extracted type. An `enum` switch
records the concrete enum type and a variant table in declaration order, with one successor per
variant. The verifier rejects missing definitions, forged variants, incomplete or duplicate fields,
payload type drift, and reordered or incomplete switch tables as `V1001`. Resource and safe-borrow
payloads are rejected before MIR.

The `view-length` operation accepts exactly one operand whose verified owner is a safe borrow of
`str` or `[u8]`. It records that owner type, the exact `len` property, and a `usize` destination.
The operation copies the existing byte-length metadata and has no call, allocation, scan, trap, or
ownership effect. A mismatched operand family, owner, property, or result is `V1001`; backends may
not reinterpret the count as Unicode scalars, graphemes, capacity, or a C terminator-inclusive size.

Immutable byte-view indexing uses a `view-index-check` marker, a two-successor
`checked-view-index` switch, a non-returning `bounds` trap block, and a fresh continuation containing
the `view-index` load. The marker, switch, and load carry the same already-evaluated `&[u8]` receiver
and `usize` index in source order. The load records the exact borrowed owner and a `u8` destination.
The verifier rejects a mismatched family, index type, result type, operator, marker, trap category,
or non-dominating check as `V1001` or `V1501`. It never infers safety from generated-C layout.

The implemented raw-address operation accepts only a direct local place produced by an exact safe
borrow-to-raw-pointer cast. It records pointee type, source borrow mutability, destination pointer
type, source span, and containing unsafe scope. `V1701` rejects missing or forged scope provenance;
`V1001` rejects a mismatched place, pointee, mutability, or operation shape. The C backend emits
`&local`; the interpreter reports a native-only capability boundary instead of fabricating an address.

Every user resource type has one program-level definition containing its canonical type identity,
source span, declaration-ordered field names and resolved field types, and drop classification.
Resource construction names that definition and lists exactly its declared fields in order. A field
read records both the owning resource type and result type. Missing definitions, reordered or unknown
fields, mismatched result types, duplicate definitions, invalid fields, and a drop classification
other than the currently executable `empty` kind are `V1001`. The MIR lowering derives `NeedsDrop`
from these definitions; callers cannot supply a separate AST-built resource-type side channel.

Each arithmetic family has distinct operations. Backends cannot infer trapping or wrapping from a build profile.

Ordinary `i32` negation and matching-type `i32`, `u8`, `u32`, or `usize` addition, subtraction,
multiplication, division, remainder, and right shift use an explicit check block, a two-successor switch, a non-returning trap block, and a
fresh continuation containing the operation. The marker and switch carry the same operands and
operator as the continuation. Addition, subtraction, multiplication, and negation expose
`integer-overflow`; right shift exposes `invalid-shift` and carries the exact operand width;
division and remainder expose `division-by-zero`, with `signed-division-overflow` additionally
present only for `i32`. Left shift is rejected before MIR. Matching-type `&`, `^`, and `|` lower as
typed non-trapping binary operations.

### Statements

Statements include local storage markers, assignment, drop-flag update, `Drop`, region mark or reset, and explicit provenance markers for unsafe operations. Assignments consume already-evaluated operands; they do not contain nested side effects.

### Terminators

Terminators include `Goto`, integer, boolean, or discriminant switch, `Call`, `Return`, `Trap`, and `Unreachable`. A boolean switch has exactly two successors ordered true then false. A restricted user-enum switch has one successor per declaration-ordered variant and carries that exact table. Source `&&` and `||` are lowered to boolean switches and branch-local assignments before verification. Source `break` and `continue` become cleanup followed by a goto to the nearest loop exit or header; neither source form remains in MIR. Stage 0 calls have normal successors only. There is no unwind successor.

Fixture-visible trap categories use the canonical identities `bounds`, `integer-overflow`, `division-by-zero`, `signed-division-overflow`, `invalid-shift`, `assertion-failed`, and `unreachable`. A backend may choose a target-specific termination mechanism, but it cannot rename or merge the semantic outcome reported by the interpreter and differential harness.

## Calls

Arguments are fully evaluated into temporaries from left to right before the call terminator. A call records its destination, normal continuation, source span, unsafe scope when applicable, and foreign ABI when applicable.

Internal calls may pass exact `&str` and `&[u8]` operands to matching parameters. The semantic
borrow origin and permission are checked before MIR; MIR retains the exact parameter type and does
not lower the argument to a raw address. The interpreter passes its private view value, while a
native backend selects private storage transport only after verification. Foreign signatures still
reject safe borrows.

Source C-scalar transport preserves all sixteen TargetSpec C-scalar identities as distinct foreign
result and parameter types. A producer call may initialize a local of any exact type and a later
consumer call may use the local as its argument. MIR does not encode those values as Fairy
fixed-width, pointer-width, boolean, or floating immediates, and the verifier's shared
foreign-signature rule therefore treats mismatched identities as incompatible. Full call-operand
type checking remains a backend responsibility in this prototype.

The `scalar-cast` operation accepts one already-evaluated operand and records exact source and result
types for one of the four value-preserving integer pairs. It has one destination and no trap,
ownership, allocation, or unsafe provenance. The verifier rejects missing type identity, a forged
pair, a mismatched immediate family, an extra operand, or a missing destination as `V1001`.
Interpreters and backends must preserve the mathematical value rather than reinterpreting bits.

A checked ordinary `+`, `-`, or `*` lowers one typed `binary`, one typed `checked-arithmetic` marker,
and a `checked-arithmetic` switch with the same `SwitchType`. The continuation may use the result
only after the non-trapping edge; the other edge reaches the canonical non-returning
`integer-overflow` block. `V1501` rejects a missing marker, operand/operator drift, or a marker/switch
type that differs from the binary operation. This applies to `i32`, `u8`, `u32`, and `usize`.

A checked `>>` lowers one typed `binary`, one typed `checked-arithmetic` marker whose operator is
`>>`, and a `checked-shift` switch with the same `SwitchType`. The trap edge is exactly
`invalid-shift`; the continuation is reachable only for an amount inside the selected type width.
`V1501` rejects marker, operand, operator, category, or type drift. `i32` continuation semantics are
arithmetic right shift, while `u8`, `u32`, and `usize` continuation semantics are logical right shift.

`@repr(C)` struct definitions may retain any of those exact C-scalar identities or another acyclic
`@repr(C)` struct as fields. Their struct construction and field-read operations keep the exact
field types; target layout, dependency ordering, and C spelling remain backend-owned. Raw `*void`
and `*mut void` retain opaque pointer identity and carry no dereference or conversion operation.
One nested pointer-to-opaque-pointer type may carry the verified direct-local address of a pointer
output slot; MIR still contains no raw load, store, arithmetic, or pointee conversion.

Foreign declarations preserve separate source and external link identities. Declarations mapped to
one link identity must have the same ABI, result type, and ordered parameter types or verification
fails as `V1001`. Foreign calls cannot expose a safe reference in their raw MIR signature. Unknown
foreign functions receive no alias, no-unwind, read-only, or escape attributes unless an explicit
wrapper contract proves them.

## Verification invariants

The verifier runs before every interpreter or backend handoff.

| Code | Invariant |
| --- | --- |
| `V1001` | Every reachable block ends in exactly one terminator; each operation has its required shape and concrete type identity; every executable aggregate use agrees with one valid program-level definition; and foreign declarations mapped to one link name have compatible signatures. |
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

The first semantic oracle is a small MIR interpreter supporting scalars, tagged `Option` and `Result`, restricted inert user enums, borrowed UTF-8 and byte views with byte-length reads, moves, drop traces, region mark and reset, branches, and traps. It constructs user enums, selects declaration-ordered variants, extracts named payload fields, and also constructs inert structs and resources and resolves field reads from their distinct verified program-level definitions; only the resource definition participates in reverse-field cleanup. The only executable drop classification is currently `empty`; source user drop-body effects are not interpreted. Its options-only fixture harness may intercept explicitly bound call symbols as bounded in-memory stdout/stderr byte sinks; those hooks are not MIR operations, source-name conventions, standard-library APIs, or host I/O. The interpreter is not required to emulate a complete operating system or C ABI.

The interpreter precedes a second native backend. Differential fixtures compare interpreter outcomes with generated C compiled at multiple optimization levels and, where the target fixture declares them, more than one independent C compiler.

The current Go prototype also emits deterministic C11 for the verified scalar and flat-resource
subset in `spec/fixtures/c-emission/manifest-v0.json`, including private `Option<i32>` and
`Result<i32, i32>` construction, calls, matching, and propagation. Direct nonempty all-`i32`
resources with empty drop bodies may be constructed, read, moved through calls, returned, assigned,
and discharged. Resource-bearing `Option` and `Result` values are also accepted when every payload
is exactly `i32` or one of those direct resources and at least one payload is a resource. Their
private numeric C struct layout is computed from `TargetSpec` and guarded by C11 size, alignment,
and offset assertions. Tagged construction and payload operations carry their concrete MIR type into
the backend. Result propagation extracts the error and reconstructs the enclosing function's
declared Result type before cleanup; it never returns a differently typed operand container. That
runner stops at C text and therefore is backend emission evidence, not native differential or
cleanup-trace evidence. The separate Linux runner re-emits the same verified MIR with a closed
`drop-trace-v0` observer profile and compares exact source/type/span site sequences with the
interpreter; product emission remains uninstrumented and hosted execution is still unobserved.
