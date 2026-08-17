# ADR 0333: Outcome Defer and typed exit protocols

- Status: Candidate
- Date: 2026-08-17
- Product source: `docs/product/02-spec.md`
- Grammar source: `spec/grammar/stage0.ebnf`
- Safety source: `docs/adr/0003-stage-0-safety-boundary.md`
- Result and cleanup source: `docs/language/0004-results-traps-and-control-flow.md`
- Borrow source: `docs/language/0002-borrows-and-return-origins.md`
- Effect source: `docs/language/0008-effects-and-capabilities-v0.md`
- Defer source: ADR 0217
- Flow sources: ADR 0019, ADR 0021, ADR 0024
- Effect decisions: ADR 0116, ADR 0121
- Admission source: ADR 0218
- Depends on: ADR 0003, ADR 0019, ADR 0021, ADR 0024, ADR 0116, ADR 0121, ADR 0217, ADR 0218

## Context

Fairy C already has one bounded lexical `defer` form. Its arguments evaluate immediately from left
to right, only copy-scalar values are retained, registered calls execute in lexical LIFO order on
normal scope exits, and those calls run before resource drops and Region resets. Traps never execute
that cleanup chain.

That contract is useful for unconditional reporting and scalar cleanup, but it cannot observe the
actual `Result` returned by the current function. Repeating success and failure reporting at every
explicit `return` and every propagation edge scales badly, while classifying a block fallthrough,
`break`, assertion trap, or bounds trap as generic success or failure would corrupt the existing
control-flow model.

The difficult part is not parsing `defer ok` and `defer err`. The compiler must evaluate the outgoing
value exactly once, preserve it unchanged while cleanup executes, classify its real active variant,
make an immutable payload view available only for the selected guard, preserve one LIFO order across
all defer forms, and keep the non-unwinding trap boundary intact.

Resource finalization needs a second distinction. Immediate copy capture is sufficient for request
IDs and timestamps, but a resource-like local often needs to remain usable throughout the body and
be available in its latest state when the function exits. Borrowing it for the whole function would
be unnecessarily restrictive, while moving it at registration would make the body unusable.

The useful feature is therefore a typed function-exit protocol rather than two independent pieces of
syntax sugar. This ADR names the source feature **Outcome Defer** and the compiler-internal model
**Exit Protocol**.

This ADR records a complete candidate model and bounded delivery plan. It does not reserve new
spellings, change the language pin, alter current defer behavior, or claim implementation.

## Candidate decision

Fairy C should admit Outcome Defer as a conditional extension of the existing lexical defer stack.

The defining contract is:

> Observe the current function's actual outgoing `Result` after it has been fully evaluated, execute
> matching infallible guards in the ordinary lexical LIFO order, preserve the outgoing value, then
> continue the existing resource and Region cleanup sequence.

The source feature has three layers:

```fy
defer release_lock(lock_id);
```

The existing form runs on every normal lexical exit that crosses its registration scope.

```fy
defer ok record_success(request_id);
defer err(ref error) record_failure(request_id, error);
```

The simple conditional forms run only when the current function exits with the matching `Result`
variant.

```fy
defer outcome(copy request_id, final span) {
    ok(ref value) =>
        finish_ok(span, request_id, value);

    err(ref error) =>
        finish_err(span, request_id, error);
}
```

The grouped form creates one defer registration with explicit captures and at most one arm for each
outcome.

No form introduces exceptions, unwinding, closures, dynamic defer stacks, heap allocation, result
mutation, or a backend-visible runtime protocol.

## Existing lexical defer remains unchanged

The accepted source form remains valid and keeps its current meaning:

```fy
defer record_latency(request_id);
```

Its ordinary arguments evaluate immediately and are retained by value under ADR 0217. It executes
when its lexical scope is crossed by fallthrough, `return`, Result propagation, `break`, or
`continue`. It does not execute on a trap.

There is no separate `defer exit` spelling. The existing form already denotes unconditional normal
lexical-exit behavior, so another keyword would add surface without adding semantics.

Outcome-aware guards join the same registration stack. They do not replace or reinterpret ordinary
defer.

## Simple outcome guards

The first conditional forms are:

```fy
defer ok record_success(request_id);
```

```fy
defer err record_failure(request_id);
```

They are valid only inside a function whose declared result is `Result<T, E>`.

`ok` means that the current function is leaving through an actual `.ok` variant. `err` means that the
current function is leaving through an actual `.err` variant. They do not describe whether a block
fell through, whether an operation looked successful, or whether the process trapped.

Ordinary call arguments retain the existing defer capture rule. In this source:

```fy
let RequestID request_id = current_request_id();

defer err record_failure(request_id);
```

`request_id` is evaluated and copied at registration. A later reassignment of its source local does
not change the retained argument.

A simple outcome guard is still one resolved direct internal call returning `void`. The first slices
retain the current prohibition on foreign calls, constructors, value-returning calls, hidden method
receivers, aggregate capture, and arbitrary blocks.

## Outcome means the actual returned tag

The following returns must select guards from the same semantic rule:

```fy
return .err(error);
```

```fy
let Result<User, LoadError> result = load();
return result;
```

```fy
return load();
```

Only the first form exposes its tag directly in source. The compiler cannot choose a guard by source
spelling. It must materialize the complete outgoing `Result`, inspect the actual active variant when
necessary, and preserve that value across the cleanup chain.

The conceptual order is:

```text
evaluate return expression once
materialize the complete Result return slot
determine the actual ok or err outcome
execute matching deferred actions in lexical LIFO order
execute resource drops and Region resets
move the unchanged return slot to the caller
```

A statically known `.ok` or `.err` edge does not require a runtime tag branch. A return of an
arbitrary Result local or call result may require one tag read and two verifier-checked cleanup
continuations.

The outgoing value exists before every outcome guard. A guard cannot observe a partially evaluated
return expression or force the expression to run twice.

## Result propagation uses the same exit protocol

Propagation is not a separate outcome category:

```fy
fn Result<void, SaveError> save() {
    defer err(ref error) audit_save_failure(error);

    validate()?;
    persist()?;

    return .ok();
}
```

The error edge of `?` already moves the failure payload into an enclosing `Result.err` return path.
That edge therefore selects `err` guards from every crossed scope exactly once.

The success edge of `?` continues normally and selects no outcome guard at that point. A later
function return chooses its own actual outcome.

Lowering should construct the complete propagated return value before invoking an `err` guard. The
guard observes the same payload that will reach the caller, not a second extraction or reconstructed
copy.

## Outcome guards are function-exit guards

An outcome guard registered in a nested block executes only when a matching function exit crosses
that block:

```fy
{
    defer err report_failure();

    perform_step();
}
```

Normal block fallthrough does not call `report_failure`. The registration expires when the block
finishes.

```fy
{
    defer err report_failure();

    perform_step()?;
}
```

If propagation leaves the current function, the `err` guard executes.

```fy
while (condition) {
    defer err report_failure();

    break;
}
```

`break` is an ordinary lexical exit, not a Result error. The `err` guard is removed without running;
an ordinary defer registered beside it still runs.

The same rule applies to `continue` and normal fallthrough. Outcome guards never invent a scope-level
success or failure taxonomy.

## Internal exit classification

The compiler should model normal cleanup edges with one closed exit classification:

```go
type ExitOutcome uint8

const (
    ExitLexical ExitOutcome = iota
    ExitResultOk
    ExitResultErr
)
```

`ExitLexical` covers block fallthrough, `break`, `continue`, and non-Result function return. A trap is
not an `ExitOutcome` because it has no language cleanup edge.

Deferred entries carry a closed matching mask:

```go
type OutcomeMask uint8

const (
    MatchAlways OutcomeMask = 1 << iota
    MatchOk
    MatchErr
)
```

The existing defer uses `MatchAlways`. A simple `defer ok` uses `MatchOk`; `defer err` uses
`MatchErr`. The emitter walks one reverse registration sequence and skips entries whose mask does
not match the selected edge.

This model prevents separate success, failure, and ordinary stacks from reordering source
registrations.

## Immutable payload observation

A guard may bind the selected payload through one read-only borrow:

```fy
fn Result<Response, RequestError> handle(Request request) {
    defer ok(ref response) observe_response(response);
    defer err(ref error) observe_error(error);

    return dispatch(request);
}
```

The payload name and type are inferred from the enclosing function's declared `Result<T, E>`:

```text
ok(ref response)  binds ref T
err(ref error)    binds ref E
```

The borrow begins only when the selected cleanup entry executes and ends when that guard call
returns. It is call-bounded and cannot be stored, returned, copied into a longer-lived safe-borrow
container, converted into a function result, or retained by another local after the call.

The payload is borrowed from the already materialized return slot. A resource-bearing payload may be
observed without transferring ownership. After the guard chain, the complete Result still moves to
the caller.

The following forms are rejected:

```fy
defer ok(move value) consume_return_value(value);
defer err(ref mut error) rewrite_error(error);
```

Outcome Defer may observe the outgoing value but cannot consume or mutate it. Hidden cleanup below a
return statement must not change the function's source-visible result.

If the selected payload type is `void`, a payload binding is invalid:

```fy
fn Result<void, SaveError> save() {
    defer ok(ref value) observe(value);
}
```

The guard may still omit the binding:

```fy
defer ok record_success();
```

The same rule applies to a `void` error payload if such a concrete Result is admitted.

## One lexical LIFO order

All deferred registrations in a scope participate in one order regardless of mode:

```fy
fn Result<Response, Error> execute() {
    let Resource resource = acquire();

    defer first();
    defer err(ref error) second(error);
    defer third();

    return work();
}
```

On an error return, the semantic order is:

```text
evaluate work
materialize Result.err return slot
call third
call second with an immutable error payload borrow
call first
drop resource
reset exited Regions
return the unchanged Result.err
```

On an ok return, the order is:

```text
evaluate work
materialize Result.ok return slot
call third
skip second
call first
drop resource
reset exited Regions
return the unchanged Result.ok
```

The compiler must not execute every ordinary defer, then every err defer, then every ok defer. That
would ignore source registration order.

A grouped outcome registration occupies one position in this sequence. When that position is
reached, at most one matching arm executes.

## Grouped outcome protocol

Success and failure behavior belonging to one logical operation may be grouped:

```fy
defer outcome(copy request_id, copy started_at) {
    ok(ref response) =>
        finish_success(request_id, started_at, response);

    err(ref error) =>
        finish_failure(request_id, started_at, error);
}
```

The group is one defer entry. It does not register each arm independently. On function exit, exactly
one matching arm executes when both arms are present.

A group without `final` captures may omit an arm:

```fy
defer outcome(copy request_id) {
    err(ref error) =>
        audit_failure(request_id, error);
}
```

An ok return reaches the group entry and performs no call.

Each arm is one resolved direct internal call returning `void` in the first implementation. Arm
blocks, loops, propagation, nested defer, tail values, and arbitrary statements remain outside the
candidate's executable slices.

## Explicit capture only

An outcome group is not a closure. Local values used by an arm must appear in its capture list:

```fy
let RequestID request_id = current_request_id();
let User user = load_cached_user();

defer outcome(copy request_id) {
    ok(ref response) =>
        record_success(request_id, user, response);
}
```

The reference to `user` is rejected because it was not captured. Module paths, function names,
constants, and the arm's payload binding remain resolved through their ordinary namespaces; only
local value access is capture-restricted.

`copy` captures evaluate at registration from left to right and are stored in compiler temporaries:

```fy
defer outcome(copy request_id, copy started_at) {
    // ...
}
```

The first executable group slice retains ADR 0217's closed copy-scalar capture family:

```text
bool
i32
u8
u32
usize
```

Broader inert aggregate capture requires separate size, copy, and cache-impact evidence. Safe
borrows, raw pointers, views, Regions, and resources are not `copy` captures.

A trap while evaluating a copy capture prevents registration, exactly as a trap in an existing defer
argument does.

No hidden environment object, reference counting, heap allocation, or runtime capture descriptor is
created.

## `final` capture reserves exit-time ownership

The grouped protocol may reserve a local's final ownership state:

```fy
let TraceSpan span = trace::begin("load-user");

defer outcome(final span) {
    ok(ref user) =>
        trace::finish_ok(span, user);

    err(ref error) =>
        trace::finish_err(span, error);
}

trace::annotate(ref mut span, "cache-miss");
trace::annotate(ref mut span, "database-start");

return load_user();
```

`final span` does not move or borrow `span` at registration. The body may continue to read and
mutate it through ordinary valid borrows.

After registration, the root ownership of `span` is reserved for the exit protocol. The source may
not move it elsewhere, return it directly, explicitly discard it, or reassign the binding:

```fy
return_span(span);
other = span;
span = trace::begin("replacement");
```

Those operations are diagnostics related to the final reservation.

At a matching function exit, the selected arm sees the local's latest state. Its callee signature
determines whether the arm borrows or consumes the local. If the call consumes it, Resource Flow
must clear the local before implicit cleanup. If it only borrows it, the local remains live and the
ordinary resource cleanup that follows deferred calls still owns its final drop.

The first `final` slice accepts only a directly named owned function-local root. Parameters, fields,
array elements, indexed places, pointer dereferences, safe-borrow locals, Region capabilities,
compiler temporaries, and call results are rejected:

```fy
defer outcome(final state.connection) { /* ... */ }
defer outcome(final connections[index]) { /* ... */ }
defer outcome(final get_span()) { /* ... */ }
```

Arbitrary places would introduce partial ownership reservation and alias analysis far beyond the
current root-resource model.

## `final` groups are top-level in the first slice

A group containing any `final` capture must be a direct statement of the function body in the first
implementation. It must declare both an `ok` and an `err` arm.

```fy
fn Result<void, Error> run() {
    let Session session = begin_session();

    defer outcome(final session) {
        ok =>
            finish_session(session);

        err(ref error) =>
            abort_session(session, error);
    }

    return perform_work();
}
```

Every non-trapping normal function exit has exactly one Result variant, so one arm executes. A trap
still bypasses the group and all other cleanup.

Allowing a final reservation in a nested block would create normal fallthrough, `break`, and
`continue` edges on which neither Result arm runs. Supporting that model would require a third
scope-exit arm or a separately accepted target-aware exit taxonomy. It is not bundled into the first
Outcome Defer implementation.

A local may be reserved by at most one active final group. Duplicate or overlapping final
reservations are rejected.

## Guard calls cannot add a failure channel

Every simple guard and every group arm must call a function returning exactly `void`.

The following forms are rejected:

```fy
defer ok commit(transaction)?;
```

```fy
defer err rollback(transaction);
```

when `rollback` returns `Result`, `Option`, another must-use value, or any non-void value.

Outcome Defer does not decide whether a cleanup failure replaces the original error, combines two
errors, converts success into failure, or is discarded. Fallible commit remains explicit source
control flow:

```fy
fn Result<void, TransactionError> update() {
    let Transaction transaction = database::begin()?;

    update_rows(ref mut transaction)?;
    transaction_commit(transaction)?;

    return .ok();
}
```

A resource cleanup may still implement an infallible best-effort rollback under its own accepted
contract.

A guard call can trap through the ordinary language semantics. Such a trap stops the remaining
cleanup chain because Fairy C does not unwind. The compiler or tooling may warn about a statically
obvious trap risk, but it must not invent recovery.

## Trap behavior never changes

`ok` and `err` do not classify traps:

```fy
defer err report_failure();
defer release_lock(lock_id);

let u8 value = bytes[index];
```

If indexing traps, neither call executes. A trap is non-returning and non-unwinding and has no
language cleanup successor.

If an outcome guard itself traps, deferred entries below it, resource drops, and Region resets do
not execute:

```fy
defer first();
defer err dangerous_report();
defer third();
```

On an error return, `third` executes first. If `dangerous_report` traps, `first` and later cleanup are
skipped. This is undesirable application behavior but consistent language semantics.

No implementation may translate a trap into `.err`, run outcome guards during process termination,
or add a hidden catch boundary.

## Effects remain unconditional function contracts

A guard's possible effects belong to the enclosing function's declared effect set even when the
matching outcome is uncommon:

```fy
@effects(network)
fn Result<void, Error> save() {
    defer err log_to_network();

    return persist();
}
```

The enclosing function must declare `network`. Conditional execution does not remove possible
external authority from its interface.

A grouped protocol requires the union of every reachable arm's effects:

```fy
defer outcome(copy request_id) {
    ok =>
        write_metric(request_id);

    err(ref error) =>
        write_audit(request_id, error);
}
```

The type and effect checker uses the union. Atlas or IDE tooling may additionally report which arm
owns each call, but that detail does not weaken the function-level contract.

Copy-capture expressions are ordinary registration-time expressions and retain their own call and
effect checks. A final capture adds no effect by itself.

## Registration semantics

For a group with mixed captures, registration is ordered and failure-atomic:

```fy
defer outcome(copy request_id, final span, copy started_at) {
    // ...
}
```

The compiler validates all capture names and modes, evaluates `copy` expressions from left to right,
and installs final ownership reservations only after every registration-time evaluation completes
without trapping. A trap leaves no partially registered group or final reservation.

`final` names are not evaluated expressions. They resolve to exact local symbols and retain those
symbols for exit-time access.

The source order of captures is preserved for diagnostics and immediate evaluation facts, but final
reservations become active together when the group is registered.

## Relationship to resource and borrow Flow

Outcome Defer participates in existing Flow rather than bypassing it.

Resource Flow must prove:

- every final-captured owned local is live when a matching arm begins;
- no prohibited root move or reassignment occurs after reservation;
- a consuming arm transfers the cleanup obligation exactly once;
- a borrowing arm leaves the obligation for the ordinary cleanup sequence;
- the outgoing Result's active resource payload is not consumed by observation;
- no deferred call creates a second drop obligation;
- every normal function exit crosses each active top-level final group exactly once.

Borrow Flow must prove:

- a payload binding is an immutable call-bounded borrow from the return slot;
- that borrow cannot escape through a callee result, local, Option, Result, or owner move;
- all ordinary borrows of a final local are dead before an exit-time consuming call;
- final reservation is ownership metadata, not a function-long borrow;
- the outgoing Result cannot retain a borrow whose owner is destroyed by the cleanup chain.

Result Flow and must-use analysis must prove:

- the outgoing Result is already a handled return value before guards run;
- a guard returning a value cannot create an ignored obligation;
- propagation selects the err protocol exactly once;
- a deferred arm cannot replace, reconstruct, or discard the outgoing Result.

Region Flow must reject `final` capture of the compiler-owned Region capability and preserve the
existing reset order after deferred calls.

## Source grammar

The complete candidate grammar is conceptually:

```ebnf
DeferStatement
  = "defer" CallExpression ";"
  | "defer" OutcomeSelector PayloadBinding? CallExpression ";"
  | "defer" "outcome" "(" CaptureList? ")" "{" OutcomeArm+ "}" ;

OutcomeSelector
  = "ok" | "err" ;

PayloadBinding
  = "(" "ref" Identifier ")" ;

CaptureList
  = Capture ("," Capture)* ","? ;

Capture
  = "copy" Identifier
  | "final" Identifier ;

OutcomeArm
  = OutcomeSelector PayloadBinding? "=>" CallExpression ";" ;
```

The exact grammar revision may choose reusable nonterminals for calls and names. The semantic
requirements are fixed: simple guards contain one call, arms contain one call, payload bindings are
immutable, and captures are explicit.

`ok`, `err`, `outcome`, `copy`, and `final` should be contextual after `defer`; they do not need to
become globally reserved identifiers.

Parser disambiguation is bounded:

- `defer ok(...);` remains an ordinary always-deferred call when the parenthesized contents are not
  the exact payload-binder prefix followed by another call;
- `defer ok call();` and `defer err call();` are conditional guards;
- `defer err(ref error) call(error);` is a payload guard;
- `defer outcome(...) { ... }` is a grouped protocol and cannot be confused with the existing
  semicolon-terminated call form.

Malformed conditional syntax must recover as current-language syntax rather than being reinterpreted
as a previous language revision.

## CST and AST boundary

The current parser reads one expression after `defer` and produces `KindDefer`. Outcome Defer needs a
structured node rather than packing modes and captures into raw text.

A candidate source model is:

```go
type DeferMode uint8

const (
    DeferAlways DeferMode = iota
    DeferOk
    DeferErr
    DeferOutcome
)

type DeferCaptureMode uint8

const (
    CaptureCopy DeferCaptureMode = iota
    CaptureFinal
)

type DeferPayload struct {
    Name string
    Span diagnostic.Span
}

type DeferCapture struct {
    Mode DeferCaptureMode
    Name string
    Span diagnostic.Span
}

type DeferArm struct {
    Mode    DeferMode
    Payload *DeferPayload
    Call    *Expression
    Span    diagnostic.Span
}

type Defer struct {
    Span     diagnostic.Span
    Mode     DeferMode
    Payload  *DeferPayload
    Call     *Expression
    Captures []DeferCapture
    Arms     []DeferArm
}
```

The exact Go shape is not fixed. CST and AST must preserve mode tokens, payload and capture name
spans, arm order, call source, and exact reconstruction. Semantic analysis must not reparse source
strings.

`ast.Statement` should gain a dedicated `Defer *Defer` field. Continuing to place every new role in
the generic `Expression` field would make payload and capture ownership depend on child ordering.

## Typed HIR boundary

Typed HIR should remove contextual source names and retain resolved symbols, types, scopes, capture
modes, and argument sources.

A candidate model is:

```go
type DeferredArgumentKind uint8

const (
    DeferredCapturedOperand DeferredArgumentKind = iota
    DeferredPayloadBorrow
    DeferredFinalLocal
)

type DeferredArgument struct {
    Kind     DeferredArgumentKind
    Operand  *Expression
    Local    SymbolID
    Type     Type
}

type DeferredCall struct {
    Match     OutcomeMask
    Callee    SymbolID
    Arguments []DeferredArgument
    Span      Span
}

type DeferredAction struct {
    Span     Span
    Scope    Span
    Captures []DeferredCapture
    Arms     []DeferredCall
}
```

The required contract is:

- immediate copy operands are already typed and ordered;
- payload arguments identify the selected Result payload type;
- final arguments identify exact owned local roots;
- every call is resolved, internal, and `void`-returning;
- arm effects are available to the enclosing function checker;
- one grouped action remains one registration entry.

Typed HIR should preserve Outcome Defer until MIR lowering because Flow needs to validate payload
borrows, final reservations, and result-edge selection.

## MIR lowering

The current lowerer keeps one lexical deferred-call list and emits selected calls in reverse order
before function-exit cleanup. Outcome Defer should extend that structure rather than replace it.

Conceptually:

```go
type deferredArgument struct {
    kind     deferredArgumentKind
    captured Operand
    local    LocalID
}

type deferredArm struct {
    match     OutcomeMask
    callee    hir.SymbolID
    arguments []deferredArgument
    span      hir.Span
}

type deferredAction struct {
    scope hir.Span
    span  hir.Span
    arms  []deferredArm
}
```

An existing defer is one action with one `MatchAlways` arm and only captured operands. A simple
outcome guard is one action with one outcome arm. A grouped protocol is one action with up to two
arms.

Return lowering becomes:

```text
lower the return expression exactly once
materialize a typed return local
determine ExitResultOk or ExitResultErr
walk crossed deferred actions in reverse registration order
select at most one matching arm per action
materialize payload borrows and final-local operands at the call site
emit ordinary call terminators
emit resource drops and Region resets
return the unchanged typed local
```

For a runtime-unknown tag, the lowerer creates a verified tag switch before the outcome-specific
cleanup chains. Each execution reaches exactly one chain. Ordinary always-deferred calls may exist in
both CFG branches but execute once on any concrete path.

Propagation lowering already has a dedicated error return edge. It can select `ExitResultErr`
without an extra tag branch after constructing the enclosing Result.

Normal scope fallthrough and loop control use `ExitLexical`. They emit `MatchAlways` entries and
remove outcome-only entries without calling them.

MIR contains ordinary assignments, tag tests, calls, branches, gotos, drops, resets, and returns.
No source-level outcome-defer instruction reaches the interpreter or C backend.

## Return-slot integrity

The hidden return local is a semantic boundary, not a source-visible named return variable.

No guard can assign to it, take a mutable borrow from it, move its payload, or replace its active
variant. Payload observation creates only a shared call-bounded borrow.

The return local remains live until all deferred calls and ordinary cleanup complete. If its active
payload needs drop, that obligation transfers to the caller with the returned value and must not be
included in the callee's local cleanup.

A backend may implement the final transport through registers, hidden memory, or another target ABI
shape only after verified MIR fixes the same semantics. The source feature does not expose a public
return-slot ABI.

## MIR verification

The verifier must check more than call presence. For each outcome-aware return path it must prove:

- the complete return value is materialized before the first outcome guard;
- the selected outcome agrees with the actual Result tag or a dominating exact constructor edge;
- an err path contains no ok-only arm and an ok path contains no err-only arm;
- all matching actions from crossed scopes appear in reverse registration order;
- each grouped action selects at most one arm;
- ordinary always actions preserve their position relative to conditional actions;
- every payload borrow refers to the matching active payload and ends before the next cleanup step;
- every final local is live when referenced;
- a final local moved by an arm is dead before implicit resource cleanup;
- a final local only borrowed by an arm remains available to ordinary cleanup;
- the outgoing Result local is not modified after classification;
- every normal Result exit reaches exactly one return terminator with the original value;
- propagation runs the err protocol exactly once;
- trap terminators retain no cleanup successor;
- no outcome-defer form remains for a backend to reinterpret.

Verifier mutation fixtures should independently remove, duplicate, reorder, or retarget each fact and
require fail-closed rejection.

## Diagnostics

The implementation must provide stable source-linked diagnostics for at least:

- `defer ok`, `defer err`, or `defer outcome` in a non-Result function;
- duplicate ok or err arms in one group;
- malformed outcome selectors, payload binders, captures, or arrows;
- an ok payload binding when the success payload is `void`;
- an err payload binding when the error payload is `void`;
- `move`, `ref mut`, or another unsupported payload-binding mode;
- a payload borrow that escapes the guard call;
- a guard call returning non-void or producing a must-use value;
- a foreign, unresolved, constructor, or otherwise inadmissible guard call;
- an implicit local capture in an outcome group;
- a non-copy value in a `copy` capture;
- duplicate capture names;
- a `final` capture that is not a directly named owned local root;
- a final group outside the function body's direct statement list;
- a final group missing either the ok or err arm;
- a root move, return, discard, or reassignment after final reservation;
- duplicate active final reservations for one local;
- a final local not live on one reachable Result exit;
- a guard whose required effects are absent from the function declaration.

Diagnostics should relate the registration, payload or capture declaration, conflicting move or use,
function return type, selected arm, and callee declaration when applicable.

## Tooling and Atlas

Typed tools may expose the protocol without treating it as runtime closure state:

```text
defer mode outcome
arm ok calls finish_ok
arm err calls finish_err
capture request_id copy
capture span final
payload value shared borrow
```

Effect tooling may display arm-specific calls while the type system retains their union. Control-flow
views may link explicit returns and propagation edges to the selected protocol arm.

This candidate does not require an Atlas schema change. Tooling export should follow only after one
executable slice establishes stable facts and versioned compatibility requirements.

## Performance and code generation

Outcome Defer requires no dynamic runtime stack. The compiler knows every registration and exit edge.

Immediate copy captures use ordinary hidden locals. Final captures retain existing local IDs. A
statically known `.ok` or `.err` return chooses its cleanup chain without a runtime branch. A
runtime-unknown Result adds one read of its existing tag and one branch before ordinary cleanup
blocks.

No closure object, heap allocation, reference count, exception table, unwinder, callback trampoline,
or runtime type reflection is required.

The C backend receives ordinary verified control flow. It must not implement its own outcome
classification, capture semantics, or cleanup ordering.

## Delivery slices

### Slice 1: simple outcome selection

Admit:

```fy
defer ok call();
defer err call();
```

Require Result-returning functions, actual-tag selection, dynamic return-slot materialization,
propagation integration, one LIFO stack, trap skipping, interpreter execution, and constrained-C
differential evidence. Retain the existing immediate copy-scalar argument boundary.

This slice deliberately excludes payload bindings, groups, and final capture.

### Slice 2: immutable payload bindings

Admit:

```fy
defer ok(ref value) call(value);
defer err(ref error) call(error);
```

Add return-slot payload borrows, void-payload diagnostics, borrow-escape rejection, resource-bearing
payload observation, and verifier facts proving the outgoing Result remains unchanged.

### Slice 3: grouped outcome with copy captures

Admit:

```fy
defer outcome(copy request_id) {
    ok(ref value) => finish_ok(request_id, value);
    err(ref error) => finish_err(request_id, error);
}
```

Add explicit capture-only name resolution, one registration slot, arm effect union, duplicate-arm
and missing-capture diagnostics, and no hidden closure environment.

### Slice 4: top-level final ownership reservation

Admit direct function-body groups containing `final local`. Add root reservation, move and
reassignment diagnostics, latest-state exit access, consuming-versus-borrowing arm behavior, both-arm
requirements, Resource Flow integration, and verifier-checked once-only cleanup.

### Slice 5: tooling facts

After executable evidence stabilizes the model, add optional Atlas relations, IDE outcome
breadcrumbs, capture-cost display, and cleanup-chain visualization without changing runtime
semantics.

Each executable slice needs an accepted implementation ADR or an accepted revision of this ADR with
exact grammar, diagnostics, compatibility-vector changes, fixtures, cache impact, and same-commit
native evidence.

## Rejected alternatives

- Separate ordinary, ok, and err stacks: destroys lexical registration order.
- Treating normal fallthrough or `break` as ok: confuses scope completion with a Result variant.
- Treating assertion, bounds, or arithmetic traps as err: violates the non-unwinding trap contract.
- Choosing guards from return-expression syntax: fails for arbitrary Result locals and call results.
- Re-evaluating the return expression for a guard: violates exact single evaluation.
- Mutable or consuming payload bindings: allows hidden code to change the function result.
- Named mutable return variables: exposes the return slot and recreates hidden post-return mutation.
- Fallible guard results: requires a new error-composition policy rather than a defer extension.
- Implicit local capture: hides copy cost, borrow lifetime, and ownership.
- Heap-backed closures or a dynamic runtime defer stack: unnecessary for statically known edges.
- Moving a final local at registration: prevents legitimate body use.
- Borrowing a final local for the full function: creates unnecessary long loans.
- Arbitrary final places: introduces partial ownership reservation and alias-sensitive cleanup.
- Nested final groups in the first slice: requires a third non-Result exit policy.
- Running remaining cleanup after a guard trap: introduces hidden unwinding.
- A backend-visible outcome-defer primitive: duplicates semantics outside verified MIR.

## Admission evidence

Promotion from `Candidate` requires at least one complete vertical executable slice with all of the
following:

- normative grammar and language documentation;
- exact contextual-keyword and previous-revision behavior;
- lossless CST reconstruction and bounded malformed-input recovery;
- AST and HIR tests for modes, payload spans, captures, arms, and resolved symbols;
- semantic fixtures for non-Result functions, void payloads, non-void guards, missing effects,
  implicit captures, invalid final places, and ownership conflicts;
- Result Flow evidence for direct ok, direct err, runtime-unknown return, and every propagation edge;
- Borrow Flow evidence for payload call-bounded lifetimes and escape rejection;
- Resource Flow evidence for consuming and borrowing final arms plus ordinary post-guard drops;
- Region and lexical-defer fixtures proving the required cleanup order;
- verified MIR fixtures for return-slot materialization, tag selection, one LIFO sequence, selected
  arms, unchanged outgoing values, and no trap cleanup;
- verifier mutation fixtures for wrong outcome, missing or duplicated arm, reordered entry, escaped
  payload borrow, dead final local, duplicate drop, and modified return slot;
- interpreter and constrained-C execution for ok, err, propagation, runtime-unknown return, nested
  registration, unmatched guards, guard trap, and mixed ordinary/outcome LIFO order;
- effect fixtures proving every reachable arm contributes to the enclosing function contract;
- proof that no closure allocation, dynamic defer stack, backend outcome primitive, or unwinding path
  was added;
- compatibility-vector, language-pin, cache-key, starter, and migration updates when source syntax is
  actually admitted.

A parser-only, type-checker-only, or backend-only implementation is inadmissible.

## Compatibility

This candidate changes no current grammar, language pin, compatibility vector, runtime ABI, MIR
version, target maturity, or generated code.

An executable slice introduces new contextual syntax after `defer` and therefore requires an exact
language revision even if the new words remain valid identifiers elsewhere. The previous-revision
parser must continue to read historical `defer ok(...);` calls according to its own grammar rather
than reinterpreting them as outcome guards.

Internal outcome masks, hidden return locals, capture IDs, and final reservations are compiler-local
facts. They are not source types, public ABI names, package-interface identities, or stable cache
encodings unless a later contract explicitly versions them.

## Consequences

Outcome Defer removes duplicated success and failure reporting without turning traps into errors or
making return values mutable. It gives guards an immutable view of the exact outgoing payload, keeps
all registrations in one lexical order, and preserves the existing defer-before-drop-before-reset
cleanup contract.

Grouped protocols make capture cost explicit. `copy` retains registration-time scalar values;
`final` reserves one local's exit-time ownership while allowing ordinary body use. Guard calls remain
void and cannot silently create a second error policy.

The implementation cost is paid in the correct layers: typed return-slot materialization,
outcome-aware cleanup selection, call-bounded payload borrows, root ownership reservation, Flow
integration, and MIR verification. Backends still receive ordinary calls and branches.

No current source program or runtime behavior changes under this candidate ADR alone.
