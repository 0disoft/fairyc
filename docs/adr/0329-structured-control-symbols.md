# ADR 0329: Structured control symbols

- Status: Candidate
- Date: 2026-08-17
- Product source: `docs/product/02-spec.md`
- Grammar source: `spec/grammar/stage0.ebnf`
- Flow sources: ADR 0011, ADR 0012, ADR 0019, ADR 0024
- Control-flow sources: ADR 0021, ADR 0024, ADR 0213, ADR 0217
- Admission source: ADR 0218
- Atlas source: ADR 0004
- Depends on: ADR 0021, ADR 0024, ADR 0213, ADR 0217, ADR 0218

## Context

Fairy C currently supports nearest-loop `break` and `continue`. The type checker records only loop
nesting depth, while MIR lowering chooses the last entry in a loop stack. That representation cannot
identify an outer loop, a named non-loop exit, a value-producing local control expression, or the
exact lexical scopes crossed by a non-nearest transfer.

Adding names only as parser labels would be unsafe. Borrow Flow, Resource Flow, Result Flow, lexical
`defer`, Region reset, and MIR cleanup all need the same resolved target. If any phase treats an outer
transfer as a nearest-loop transfer, reachability and cleanup facts no longer describe the executed
program.

The useful abstraction is therefore not a callable name for an `if` or `for`. A function name
identifies an entry point. A structured control symbol identifies one or more outward control ports:
an exit port, a continuation port, and eventually a typed value-exit port.

This ADR records the complete candidate model so the implementation can be delivered in bounded
vertical slices. It does not reserve syntax, change the language pin, or claim implementation.

## Candidate decision

Fairy C should admit **structured control symbols** as lexical, non-first-class identities attached
to structured control constructs.

The common source form is a C-like label prefix:

```fy
scan_bananas: for (var i32 i = 0; i < banana_count; i += 1) {
    for (var i32 j = 0; j < crate_count; j += 1) {
        if should_skip(i, j) {
            continue scan_bananas;
        }

        if found_enough(i, j) {
            break scan_bananas;
        }
    }
}
```

Existing nearest-loop forms remain unchanged:

```fy
break;
continue;
```

A named transfer resolves to an active enclosing control symbol. It never enters a block, resumes a
completed block, jumps sideways, or crosses a function boundary.

The word `as` is not used for this feature. Fairy C already owns `as` as its explicit cast operator,
and overloading it for control labels would make two unrelated concepts compete for one spelling.

## Control capabilities

Each control kind exposes a closed capability set.

| Control kind | Exit | Continue | Value exit |
| --- | --- | --- | --- |
| block | yes | no | yes |
| `if` | yes | no | yes |
| `switch` | yes | no | yes |
| `while` | yes | yes | no in the first implementation |
| `for` | yes | yes | no in the first implementation |
| `unsafe` | yes | no | yes |
| `region` | yes | no | yes |

A source `break` selects an exit or value-exit port. A source `continue` selects a continuation port.
Capability mismatch is a type error rather than a generic label error. For example, targeting a
named `if` with `continue` must report that the control has no continuation port and that only
`while` and `for` provide one.

Loops deliberately do not produce values in the first value-exit slice. A named value block around
the loop is clearer and keeps loop fixed-point analysis separate from expression result joining:

```fy
let i32 found = search: {
    for (var i32 i = 0; i < values.len(); i += 1) {
        if matches(values[i]) {
            break search(i);
        }
    }

    -1
};
```

## Value-producing controls

A later slice may admit a named block as a value-producing local control expression:

```fy
let i32 banana_index = find_banana: {
    for (var i32 i = 0; i < items.len(); i += 1) {
        if is_banana(items[i]) {
            break find_banana(i);
        }
    }

    -1
};
```

`break find_banana(i);` evaluates `i` and supplies it to the named block's result join. The transfer
expression has type `never`. Every normal fallthrough result and every value exit must join to one
canonical type.

Parentheses are mandatory for value transfer:

```fy
break;
break scan_bananas;
break find_banana(index);

continue;
continue scan_bananas;
```

Unnamed `break(value);` is rejected because the source does not identify which result join receives
the value. `break find_banana index;` is also rejected; call-shaped parentheses make the value-exit
port visible and leave the grammar extensible.

This facility is a local structured return without function extraction. It has no call, capture,
allocation, address, or callable identity.

## Identity and scope rules

A control symbol is visible only in the body of its construct and nested bodies. A loop cannot refer
to its own symbol from its initializer, condition, or post clause. Those regions execute before or
outside the body capability and must not gain a runtime control value.

An active control name cannot be shadowed by another control name. A control name also cannot collide
with a local binding in the same function. The compiler may maintain separate internal namespaces,
but source-level collisions are rejected because they add ambiguity without expressive power.

The symbol is not a first-class value. Source code cannot assign, pass, return, store, call, take the
address of, export, or reference it from another function. The only legal use is an outward transfer
from the current point to an active enclosing control.

Renaming a source control name does not change its semantic identity after resolution. HIR and MIR
must use a function-local numeric `ControlID`, not a retained source string.

## Structured transfer and cleanup

A named transfer is not lowered as an unchecked `goto`. The compiler computes the exact lexical
scopes between the transfer site and target port and performs the same verified obligations as every
other early exit.

For a value exit, the required order is:

1. evaluate the value expression;
2. move or materialize the result into a hidden temporary;
3. compute the lexical scopes crossed by the transfer;
4. execute lexical `defer` calls in reverse order;
5. drop live resources in reverse declaration order;
6. reset exited Regions after their contained cleanup;
7. discharge or diagnose outstanding Result obligations;
8. transfer to the target control join;
9. use the hidden temporary as the control expression result.

Value evaluation must precede cleanup. A result may use a resource that is still live at the transfer
site, but it may not retain a borrow or resource reference whose owner dies in a crossed scope.

A named `continue` for a classic `for` targets the post block, not the condition block. Cleanup for
scopes exited from the body occurs before the post clause, preserving ADR 0213.

## Compiler representation

The lossless CST should use one generic wrapper rather than adding label fields independently to each
control node:

```text
KindControl
  Text       source name
  NameSpan   exact name span
  Children   exactly one wrapped control node
```

The wrapped node remains the existing `KindWhile`, `KindFor`, `KindBlock`, `KindIf`, `KindSwitch`,
`KindUnsafe`, or `KindRegion`. The wrapper centralizes source identity, attributes, documentation,
formatter behavior, and future Atlas export.

AST lowering should preserve a source-facing `ControlLabel` containing the name and exact name span.
A dedicated control-resolution step should assign one function-local identity and resolve every
transfer before ownership-sensitive Flow analysis.

Typed HIR should expose a closed control table similar to:

```go
type ControlID uint32

type ControlCapabilities struct {
    Exit      bool
    Continue  bool
    ValueExit bool
}

type ControlSymbol struct {
    ID           ControlID
    Name         string
    Kind         syntax.Kind
    ExitType     hir.Type
    Capabilities ControlCapabilities
}

type ControlTransfer struct {
    Target ControlID
    Value  *hir.Expression
}
```

Source names and name spans remain available for diagnostics and tools, but MIR transfer edges use
only `ControlID` and concrete CFG targets. A rename must not perturb MIR shape beyond source spans or
non-semantic tooling facts.

The current `loopDepth int` plus nearest `loopStack` model must be replaced by a shared active-control
model containing at least the resolved ID, control kind, exit block, optional continuation block,
lexical scope depth, Region depth, and optional result local.

## Flow-analysis requirement

Control resolution must happen before Borrow Flow, Resource Flow, and Result Flow. Each analysis
maintains a target-aware active-control stack and routes a transfer to the frame selected by its
resolved `ControlID`.

An outer transfer encountered inside nested loops is recorded on the outer frame. The inner loop must
not reinterpret it as its own break edge, merge it into its local exit state, or make subsequent inner
or outer body code reachable. Backward liveness must use the same resolved target when selecting
break and continue live-out sets.

The first implementation is incomplete unless all of these agree:

- syntax reconstruction and recovery;
- AST control ownership and exact spans;
- name resolution and capability checks;
- Borrow Flow reachability and loan state;
- Resource Flow liveness, moves, and cleanup obligations;
- Result Flow handled-state joins;
- MIR target selection and lexical cleanup;
- verifier, interpreter, and constrained-C execution.

A parser-only or type-checker-only implementation is explicitly inadmissible.

## Diagnostics

The implementation must provide stable, source-linked diagnostics for at least:

- an unknown or inactive control name;
- a transfer outside every compatible control;
- duplicate active control names;
- control and local binding name collisions;
- `continue` targeting a control without a continuation port;
- value exit targeting a control without a value-exit port;
- incompatible normal and explicit result types;
- a value exit whose result retains a borrow from a crossed scope;
- a reference from a loop initializer, condition, or post clause to its own control symbol;
- malformed value-transfer parentheses.

Diagnostics should identify the transfer site, the selected or conflicting control declaration, and
the missing capability where applicable.

## Stable external anchors

A local control name is optimized for human reading and lexical resolution. It must not also serve as
a durable external identity for Atlas, IDE automation, refactoring tools, or AI agents.

A later tooling slice may add an explicit stable anchor:

```fy
@anchor("checkout.count-bananas")
count_bananas: if x > 3 {
    // ...
}
```

Renaming `count_bananas` leaves the external anchor unchanged. Atlas may then export relationships
such as function contains control, control kind, anchor, transfer targets control, exited scope,
join block, and result type. The anchor remains metadata; it does not make the control first-class or
addressable at runtime.

## Formatter and IDE behavior

An `end name` closing syntax is not admitted. It duplicates the opening name, creates mismatch errors,
and adds parser surface without changing semantics.

A formatter may instead add or update a non-semantic closing comment for sufficiently long named
controls:

```fy
count_bananas: if x > 3 {
    // ...
} // end count_bananas
```

IDE folding and breadcrumbs may display concise identities such as `count_bananas: if`,
`scan_crates: for`, or `find_banana: block -> i32`.

## Delivery slices

### Slice 1: named loops

Admit named `while` and `for`, named `break`, and named `continue`. Replace loop-depth and nearest-loop
assumptions with resolved `ControlID` targets across all three Flow analyses and MIR. Preserve bare
nearest-loop forms and existing classic-`for` post behavior.

This slice requires positive nested-loop transfers, unknown-name and capability diagnostics,
cleanup-crossing fixtures, interpreter execution, and constrained-C differential evidence.

### Slice 2: named value blocks

Admit named ordinary blocks and `break name(value)`. Add hidden result locals, result-type joins,
`never` transfer typing, move-before-cleanup behavior, and borrow-escape rejection.

### Slice 3: remaining structured controls

Lift the same wrapper and capability model onto `if`, `switch`, `unsafe`, and `region`. No new transfer
primitive or MIR identity model should be introduced in this slice.

### Slice 4: tooling identities

Add optional stable anchors, Atlas relationships, formatter closing comments, IDE breadcrumbs, and
control-aware refactoring support. Tooling identity remains separate from the lexical source name.

Each slice needs its own accepted implementation ADR or an accepted revision of this ADR with exact
grammar, compatibility cut, diagnostics, fixtures, and same-commit execution evidence.

## Rejected alternatives

- `if (...) as name` or `for (...) as name`: conflicts with the existing cast operator and obscures
  the fact that the name denotes an outward port rather than a value conversion.
- `end name`: duplicates identity at the closing delimiter and creates avoidable mismatch syntax.
- first-class control values: turns structured exits into continuation or `goto` capabilities and
  invalidates lexical ownership assumptions.
- arbitrary inward or sideways jumps: bypass initialization, cleanup, borrow lifetime, and Result
  handling.
- adding independent label fields to every CST control node: duplicates parser, formatter, tooling,
  and Atlas logic instead of preserving one wrapper contract.
- retaining source strings in MIR: couples renaming to backend identity and delays target validation.
- attaching a name map to `loopDepth`: cannot model capabilities, value joins, exact cleanup scopes,
  or non-loop controls.
- value-producing loops in the first value slice: increases fixed-point and syntax complexity while
  a named value block already expresses the operation clearly.
- implementing syntax before target-aware Flow: produces a compiler that parses programs whose
  ownership and cleanup model it cannot faithfully analyze.

## Admission evidence

Promotion from `Candidate` requires one vertical implementation slice with all of the following:

- normative grammar and language documentation;
- exact current- and previous-revision lexer behavior where a spelling becomes reserved;
- lossless CST reconstruction and bounded malformed-input recovery;
- AST and typed-HIR tests proving source spans and resolved `ControlID` identity;
- semantic fixtures for scope, collision, unknown target, and capability diagnostics;
- Borrow, Resource, Result, Region, and lexical-`defer` fixtures for inner and outer transfers;
- verified MIR fixtures proving target blocks, cleanup paths, and eliminated source forms;
- interpreter and constrained-C execution of nested named loop examples;
- compatibility-vector and cache-key updates when the syntax is actually reserved;
- Atlas schema changes only in the separate tooling slice.

## Consequences

The candidate gives Fairy C one coherent path from familiar labelled loop control to typed local
value exits without importing unrestricted jumps or first-class continuations. The implementation
cost is intentionally paid in semantic resolution and Flow correctness rather than hidden in parser
sugar.

No current source program, grammar, language pin, cache identity, MIR contract, backend, formatter,
Atlas schema, or target maturity changes under this candidate ADR alone.
