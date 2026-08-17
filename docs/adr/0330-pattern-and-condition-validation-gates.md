# ADR 0330: Pattern and condition validation gates

- Status: Candidate
- Date: 2026-08-17
- Product source: `docs/product/02-spec.md`
- Grammar source: `spec/grammar/stage0.ebnf`
- Pattern sources: ADR 0017, ADR 0018, ADR 0050, ADR 0136
- Flow sources: ADR 0011, ADR 0012, ADR 0019, ADR 0024, ADR 0217
- Check sources: ADR 0097, ADR 0101, ADR 0102
- Structured-control source: ADR 0329
- Admission source: ADR 0218
- Depends on: ADR 0017, ADR 0018, ADR 0024, ADR 0136, ADR 0217, ADR 0218, ADR 0329

## Context

Fairy C needs a compact way to inspect one value, close the failure path immediately, and leave the
success continuation with stronger type, ownership, and range facts. A spelling that merely shortens
one `switch` is insufficient. The compiler must preserve single evaluation, prevent partial resource
moves, prove that the failure path cannot return to the next statement, and expose only committed
success bindings after the gate.

The current declaration parser treats `let` and `var` as ordinary name bindings. The current switch
parser also owns a dedicated `parseSwitchPattern`, while AST lowering preserves switch patterns
through generic expression-shaped nodes. Adding refutable patterns directly to ordinary `let` would
mix declaration and control-flow syntax while duplicating the existing switch pattern machinery.

The current non-void function check also uses a narrow `blockAlwaysReturns` predicate. A validation
gate cannot define its failure rule in terms of `return` alone: an accepted failure path may break or
continue through a resolved structured control symbol, trap, or diverge. The compiler therefore needs
a general non-fallthrough exit summary rather than another special case in that predicate.

Condition gates add a second requirement. A successful runtime check such as `index < bytes.len()`
should be available as a compiler fact, but only while the exact values and places participating in
that check remain unchanged. Attaching the fact to source variable names would be unsound after
assignment, mutable borrowing, aliasing calls, or length-changing operations.

This ADR records the complete candidate model. It does not reserve `gate`, change the language pin,
or claim implementation.

## Candidate decision

Fairy C should admit **validation gates** in two source forms:

```fy
gate let Pattern = Expression
else Failure;
```

```fy
gate (Condition)
else Failure;
```

A validation gate has one success continuation and one failure branch. The expression or condition
is evaluated exactly once. Success continues to the next statement with committed bindings or
refinement facts. Failure executes a branch that cannot reach the next statement.

The defining contract is:

> Evaluate once, prove or bind on success, and structurally close every failure path.

## Pattern-gate source form

The basic form extracts one refutable pattern:

```fy
gate let .some(user) = find_user(id)
else return .none;

use_user(user);
```

After the gate, `user` is fully initialized. It is not an `Option<User>`, a conditionally initialized
local, or a value requiring a second tag test. The failure path cannot reach `use_user`.

A compound failure path uses a block:

```fy
gate let .some(user) = find_user(id)
else {
    audit_missing_user(id);
    return .none;
}
```

The success bindings begin immediately after the gate and remain in the enclosing lexical scope.
Bindings introduced for failure handling exist only inside the failure block.

## `gate let` remains separate from ordinary `let`

Ordinary `let` stays irrefutable:

```fy
let User user = value;
```

The source should not make readers inspect the pattern to discover whether one declaration performs a
runtime branch. The `gate` keyword states that the construct evaluates a condition, can transfer
control, and changes facts available to the following code.

This also preserves a cleaner compiler boundary. `gate` is a dedicated statement with an explicit
success pattern, scrutinee, optional failure pattern, and failure branch. It is not an expression
hidden inside the existing binding fields.

## Single evaluation

The scrutinee is evaluated exactly once:

```fy
gate let .ok(move socket) = connect(address)
else let .err(error) {
    return .err(classify_connect_error(error));
}

socket.send(request)?;
```

`connect(address)` executes once. Success and failure inspect the same tagged result. The compiler
must not lower the source into two switches, repeat the call, or reconstruct the failure value from a
second evaluation.

If the source scrutinee is not already a stable place, the compiler may materialize one hidden
scrutinee temporary. That temporary is compiler-owned implementation state, not a source binding or
an implicit lifetime-extension facility.

## Complement failure patterns

A pattern gate may bind the value rejected by the success pattern:

```fy
gate let .ok(value) = operation()
else let .err(error) {
    return .err(error);
}
```

The failure pattern must cover the complete complement of the success pattern. For `Result`, `.ok`
and `.err` form a closed pair. For `Option`, `.some` and `.none` form a closed pair.

A user enum may bind the entire unmatched value:

```fy
gate let .authenticated(ref user) = session.state
else let rejected_state {
    audit_rejected_state(rejected_state);
    return .err(.unauthorized);
}
```

A specific failure variant is legal only when it covers every remaining case. The following source is
rejected if the enum also contains `.pending` or `.cancelled`:

```fy
gate let .ready(job) = state
else let .failed(error) {
    return .err(error);
}
```

The first implementation does not add OR patterns merely to make a partial failure pattern complete.
A catch-all whole-value binding handles multi-variant complements.

## Explicit binding modes

Copy values use an ordinary pattern binding:

```fy
gate let .some(count) = optional_count
else return 0;
```

Resource payloads require an explicit ownership mode:

```fy
gate let .some(move connection) = open_connection()
else return .err(.unavailable);
```

Borrowed payloads require `ref` or `ref mut`:

```fy
gate let .some(ref config) = application.config
else return .err(.missing_config);
```

```fy
gate let .some(ref mut user) = users[current_id]
else return .err(.missing_user);

user.score += bonus;
```

The closed candidate binding modes are:

```text
copy
move
ref
ref mut
```

A plain binding is accepted only when the payload belongs to the admitted copy set. A resource
payload without `move`, `ref`, or `ref mut` is a diagnostic rather than an implicit move.

`ref` and `ref mut` require a stable source place whose owner outlives the success continuation. The
compiler must reject borrowing through a temporary tagged container:

```fy
gate let .some(ref user) = find_user(id)
else return .none;
```

If `find_user(id)` returns a temporary `Option<User>`, the source must take ownership instead:

```fy
gate let .some(move user) = find_user(id)
else return .none;
```

Validation gates do not invent a hidden owner whose lifetime is silently extended beyond the
statement.

Failure-pattern bindings use the same ownership modes. A whole unmatched resource value therefore
requires `else let move rejected_state`, while a copy enum may use a plain catch-all name.

## Probe before commit

Pattern execution is divided into two phases.

The **probe phase** evaluates the scrutinee once and checks tags, literals, field presence, and other
non-destructive conditions. It performs no source-visible move, mutable borrow, or binding commit.

The **commit phase** runs only after the entire success pattern matches. It creates bindings and
performs the declared moves or borrows in source order.

For a future nested record pattern:

```fy
gate let .message {
    .kind = .request,
    .body = move body,
    .metadata = metadata
} = packet
else return .err(.invalid_packet);
```

The compiler first proves that `packet` has the correct variant and copy-testable field values. Only
then may it move `body` and create `metadata`.

If the probe fails, the original `packet` remains intact for the failure branch. The implementation
must not require rollback flags or attempt to reconstruct partially moved state.

A successful move updates active-payload and resource-liveness facts exactly once. The consumed
container or payload cannot be dropped again. A failure complement move consumes the original value
only inside the already non-fallthrough failure branch.

## Failure must be non-fallthrough

The failure branch may be one terminating control statement:

```fy
else return .none;
```

```fy
else break search(.none);
```

```fy
else continue outer_loop;
```

It may also be a block whose every reachable path exits the current continuation:

```fy
else {
    log_error();
    return .err(.invalid);
}
```

A block with any fallthrough path is rejected:

```fy
gate let .some(user) = maybe_user
else {
    if (retryable) {
        return .none;
    }

    log_error();
}
```

The required semantic abstraction is a closed exit summary that distinguishes at least:

```text
FallsThrough
Returns
Breaks(ControlID)
Continues(ControlID)
Traps
Diverges
```

The summary may contain multiple non-fallthrough destinations. A failure block is accepted when its
reachable exit set excludes `FallsThrough`.

Control IDs come from ADR 0329. Target resolution and capability checking must happen before the gate
uses the summary. A named `break` or `continue` is not accepted merely because it is syntactically
present; it must resolve to an active compatible control and execute the required lexical cleanup.

This exit model should replace narrow `blockAlwaysReturns` reasoning rather than becoming a gate-only
exception.

## No fallback join

A validation gate cannot assign a fallback and continue:

```fy
gate let .some(user) = find_user(id)
else user = guest_user();
```

That operation is a value join and belongs to `switch`, a future coalescing operator, or ordinary
conditional code. A validation gate guarantees one postcondition only: the success pattern or
condition is true at the following statement.

Because failure cannot return, the compiler does not merge success and failure resource states,
partial initialization states, borrows, or Result obligations after the gate.

## Condition-gate source form

A condition gate turns a recoverable check into a success-path fact:

```fy
gate (index < bytes.len())
else return .err(.out_of_bounds);

let u8 value = bytes[index];
```

The condition is evaluated at runtime. Failure follows the user-selected safe exit. Success may allow
the compiler to reuse the proven relation.

Other initial forms include nonzero and shift-range checks:

```fy
gate (divisor != 0)
else return .err(.division_by_zero);

let i32 result = value / divisor;
```

```fy
gate (shift < 32)
else return .err(.invalid_shift);

let u32 result = value >> shift;
```

A condition gate is not `assume`. If the optimizer does not understand the predicate, the program
remains safe and correct; it simply retains a later redundant check.

Any expression accepted as an ordinary boolean condition may serve as the runtime gate condition.
Only a closed, pure subset produces reusable refinement facts in the first implementation.

## Refinement facts are versioned

A refinement is attached to compiler-private value and place versions, not source names. Conceptually:

```text
index@3 < len(bytes@7)
divisor@2 != 0
shift@4 < 32
```

The existing mutable-local MIR does not need to become a public SSA contract. The compiler may assign
internal revisions to relevant scalar values and places solely for dominance, invalidation, and check
reuse.

The initial refinement vocabulary is intentionally closed:

```text
LessThan(value, length(place))
NonZero(value)
UnsignedUpperBound(value, constant)
```

The first check-elimination consumers are:

- immutable byte-view or admitted array indexing with the same index and length versions;
- checked division or remainder with the same nonzero divisor version;
- checked shifts with the same value version and exact bit-width bound.

A later consumer may extend the vocabulary only with a separately specified proof rule. The source
`gate` construct does not promise that every boolean expression becomes an optimizer theorem.

## Refinement invalidation

A fact is invalidated when any participating value or place receives a new revision. This includes:

- direct or compound assignment to the scalar value;
- mutation of the collection or view owner that can change its length or storage identity;
- creation or use of a mutable alias that may change the relevant place;
- a call whose admitted effects do not prove the relevant state unchanged;
- control-flow joins where predecessor versions or facts disagree.

For example:

```fy
gate (index < bytes.len())
else return .err(.out_of_bounds);

index += 1;
inspect(bytes[index]);
```

The later index uses a new `index` version and must be checked again.

Likewise:

```fy
gate (index < buffer.len())
else return .err(.out_of_bounds);

resize(ref mut buffer, 0)?;
inspect(buffer[index]);
```

The mutable call invalidates the length fact unless its exact effect summary proves otherwise.
Unknown calls conservatively kill refinements involving values or places they may mutate or alias.

The first implementation need not preserve refinements across loop back edges, complex aliases, or
calls without sufficiently precise effect facts. Conservative check retention is correct.

## Relationship to existing constructs

`assert` traps on failure. A validation gate selects a recoverable structured exit:

```fy
assert(index < bytes.len());
```

```fy
gate (index < bytes.len())
else return .err(.out_of_bounds);
```

Result propagation handles the exact error-forwarding case:

```fy
let User user = authenticate(request)?;
```

A pattern gate handles error transformation, `Option`, selected user-enum variants, and failure-value
inspection:

```fy
gate let .ok(move user) = authenticate(request)
else let .err(error) {
    return .err(redact_auth_error(error));
}
```

`switch` remains the construct for multiple continuing branches or multiple value-producing arms.
A validation gate permits exactly one continuation after the checked statement.

## One shared pattern system

The implementation must extract the current switch-only parser into one reusable pattern parser
before adding `gate let`. Switch and gate patterns must share syntax nodes, AST representation,
binding modes, name resolution, exhaustiveness, payload typing, and diagnostics.

A candidate compiler-owned representation is:

```go
type PatternKind uint8

const (
    PatternWildcard PatternKind = iota
    PatternBinding
    PatternLiteral
    PatternVariant
    PatternRecord
)

type BindingMode uint8

const (
    BindCopy BindingMode = iota
    BindMove
    BindRef
    BindRefMut
)
```

The first implementation remains shallow. It admits `.some`, `.none`, `.ok`, `.err`, user-enum
variants, one payload binding or the already admitted named payload fields, and a whole-value catch-all
failure binding. Nested record patterns, OR patterns, guards, and arbitrary destructuring remain
outside the first slice.

`PatternRecord` is reserved in the internal model so the parser does not need a second redesign when
nested patterns are later admitted. Its source grammar is not accepted by this candidate alone.

## AST boundary

`ast.Statement` should gain a dedicated gate node rather than reusing ordinary binding or expression
fields:

```go
type Gate struct {
    Span           diagnostic.Span
    SuccessPattern *Pattern
    Scrutinee      *Expression
    Condition      *Expression
    FailurePattern *Pattern
    Failure        *Block
}
```

Exactly one of `SuccessPattern` and `Condition` is present. A single failure control statement may be
normalized to a one-statement block while preserving its exact source span.

The pattern AST owns names, binding modes, variant and field structure, and exact source spans. AST
lowering must not encode patterns as raw strings or require semantic analysis to reparse source text.

## Typed HIR boundary

Typed HIR should preserve the checked decision and committed consequences without source-pattern
reparsing:

```go
type Gate struct {
    Test            PatternDecision
    Condition       *Expression
    SuccessBindings []PatternBinding
    FailureBindings []PatternBinding
    SuccessFacts    []RefinementFact
    Failure         Block
}
```

The exact Go shape is not fixed by this candidate. The required contract is:

- the scrutinee or condition is represented once;
- probe operations are distinguishable from binding commits;
- binding ownership modes and symbols are resolved;
- failure bindings are scoped only to the failure block;
- refinement facts name typed operands and place revisions;
- the failure block carries a proven non-fallthrough exit summary.

## MIR lowering

MIR lowering has four conceptual regions:

```text
scrutinee or condition evaluation
pattern probe or boolean branch
success binding commit and refinement publication
failure exit or success continuation
```

A pattern gate evaluates its scrutinee before branching. The failure edge sees the unmodified
scrutinee. The success edge commits declared moves, copies, and borrows only after a complete match.

A condition gate branches on the runtime boolean result. Its recognized refinements become available
only in the dominated success continuation.

The failure branch must terminate before the success continuation. MIR must not create a resource-
state join between failure and success. Cleanup generated by `return`, structured `break`, structured
`continue`, Region exit, lexical `defer`, and Result handling remains owned by the existing exit paths.

The source-level `gate`, pattern probe, and refinement publication should be recorded as eliminated or
verified compiler facts. Backends and the interpreter receive ordinary verified CFG operations and
checks, not a source-level gate primitive.

## Flow-analysis requirements

Borrow Flow, Resource Flow, Result Flow, must-use analysis, and backward liveness must agree on the
single success continuation and non-fallthrough failure branch.

The analyses must prove at least:

- failure probes do not consume or partially move the scrutinee;
- success commits produce exactly the declared bindings;
- `ref` and `ref mut` bindings cannot outlive their owner or cross an invalidating mutation;
- resource moves clear the active payload before any cleanup path can drop it again;
- failure complement bindings observe the original unmatched value;
- no Result obligation or borrow created only in failure reaches the success continuation;
- named failure transfers use the resolved `ControlID` and exact cleanup path from ADR 0329;
- condition refinements exist only on dominated success paths and disappear at invalidating joins.

A parser-only, type-checker-only, or optimizer-only implementation is inadmissible.

## Structured-control composition

Validation gates compose with structured control symbols without introducing a new jump category:

```fy
parse_request: {
    gate let .request(move request) = message
    else break parse_request(.err(.unexpected_message));

    gate let .some(ref token) = request.token
    else break parse_request(.err(.missing_token));

    .ok(request)
}
```

A gate failure is a condition selecting an existing structured exit. The selected transfer must run
all lexical `defer` calls, resource drops, Region resets, and Result checks required between the gate
and the target control port.

A `continue` failure targeting a classic `for` reaches the post clause after body-scope cleanup, as
required by ADR 0213 and generalized by ADR 0329.

## Diagnostics

The implementation must provide stable, source-linked diagnostics for at least:

- a gate failure branch with a reachable fallthrough path;
- a success pattern incompatible with the scrutinee type;
- an incomplete or overlapping failure-complement pattern;
- an implicit resource move in either success or failure bindings;
- `ref` or `ref mut` from a temporary owner;
- a mutable borrow that conflicts with an active loan;
- a partial move attempted before the complete pattern is known to match;
- a failure transfer targeting an unknown or incompatible control symbol;
- a condition whose type is not `bool`;
- malformed `else let` syntax or binding mode;
- a refinement consumer whose required fact was invalidated, when a source diagnostic is otherwise
  required rather than silently retaining the runtime check.

Optimization failure by itself is not a diagnostic. The compiler should simply retain the check.

## Delivery slices

### Slice 1: shared shallow patterns and exit summaries

Extract the switch pattern parser and typed pattern representation. Add explicit binding modes and a
general block exit summary. Preserve existing switch behavior and fixtures before reserving `gate`.

### Slice 2: executable `gate let`

Admit shallow Option, Result, and user-enum patterns, complement failure bindings, copy, explicit
`move`, stable-place `ref` and `ref mut`, probe-before-commit lowering, and non-fallthrough failure
blocks. Carry the feature through semantic fixtures, verified MIR, interpreter execution, and
constrained-C execution.

### Slice 3: executable condition gates

Admit `gate (condition)` with the same non-fallthrough failure contract. Publish no reusable proof
outside a closed recognized predicate set.

### Slice 4: bounded refinement reuse

Add compiler-private value and place revisions, dominance and invalidation, then consume facts for
byte-view or admitted-array bounds, nonzero division, and checked shift ranges. Every eliminated check
requires verified-MIR and differential execution evidence showing identical success and failure
semantics.

### Slice 5: pattern expansion

Consider nested record patterns only after probe-before-commit and active-payload behavior are proven
for representative resource-bearing aggregates. Guards, OR patterns, and shared multi-gate failure
blocks require separate admission decisions.

Each executable slice needs an accepted implementation ADR or an accepted revision of this ADR with
exact grammar, diagnostics, language-pin change, fixtures, cache impact, and same-commit execution
evidence.

## Rejected alternatives

- `let Pattern = Expression else Failure`: hides runtime control transfer inside ordinary
  declaration syntax and overloads the existing binding parser.
- lowering to repeated switches or calls: violates exact single evaluation.
- committing moves while matching: requires rollback or permits partial-move corruption on failure.
- implicit resource moves: conceals ownership transfer at the most failure-sensitive point.
- borrowing from hidden temporaries: creates invisible owner lifetime and cleanup rules.
- permitting failure fallthrough: destroys the post-gate success guarantee and forces state joins.
- fallback assignment inside `else`: turns the construct into a value join rather than a gate.
- a gate-specific pattern parser: guarantees drift from switch syntax and diagnostics.
- treating successful conditions as unchecked assumptions: makes optimizer understanding part of
  program safety.
- attaching facts to source variable names: reuses stale proofs after mutation or aliasing.
- pattern guards in the first slice: make move timing and failed-guard ownership substantially more
  complex; two gates express the operation explicitly.
- multiple checks sharing one `else`: makes visible bindings and resource state depend on the last
  successful check and creates a new partial-initialization join problem.
- accepting nested resource patterns before probe-before-commit evidence: expands surface syntax
  before the compiler can prove failure atomicity.

## Admission evidence

Promotion from `Candidate` requires at least one vertical executable slice with all of the following:

- normative grammar and language documentation;
- exact current- and previous-revision keyword behavior for `gate` and any newly reserved spellings;
- lossless CST reconstruction and bounded malformed-input recovery;
- shared switch and gate pattern parser tests proving no grammar drift;
- AST and HIR tests for exact spans, binding modes, complement patterns, and one scrutinee identity;
- semantic fixtures for type mismatch, incomplete complement, temporary borrow, implicit move, and
  failure fallthrough diagnostics;
- Borrow, Resource, Result, must-use, Region, lexical-`defer`, and structured-control fixtures for
  success and failure paths;
- verified MIR fixtures proving probe-before-commit, target cleanup, and no failure-success state
  join;
- interpreter and constrained-C execution of Option, Result, user-enum, resource-move, borrow, named
  break, named continue, bounds, division, and shift examples admitted by the slice;
- mutation tests showing refinements disappear after assignment, mutable aliasing, effectful calls,
  and disagreeing joins;
- compatibility-vector, language-pin, cache-key, starter, and migration updates when syntax is
  actually reserved;
- no backend dependence on a source-level gate or raw source-pattern string.

## Consequences

The candidate gives Fairy C one construct for early pattern extraction and recoverable runtime
validation without weakening ownership or cleanup. `gate let` avoids repeated evaluation and partial
moves. Condition gates turn successful checks into optional, versioned optimization facts without
making optimizer support a safety requirement.

The implementation cost is paid in the correct places: one shared pattern system, target-aware exit
summaries, probe-before-commit ownership semantics, and conservative refinement invalidation. No
current source program, grammar, language pin, cache identity, MIR contract, backend, interpreter, or
target maturity changes under this candidate ADR alone.
