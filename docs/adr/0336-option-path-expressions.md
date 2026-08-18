# ADR 0336: Option Path Expressions

- Status: Candidate
- Date: 2026-08-18
- Product source: `docs/product/02-spec.md`
- Grammar source: `spec/grammar/stage0.ebnf`
- Lexing and recovery source: `docs/language/0006-lexing-parsing-and-recovery-v0.md`
- Result and cleanup source: `docs/language/0004-results-traps-and-control-flow.md`
- Borrow source: `docs/language/0002-borrows-and-return-origins.md`
- Effect source: `docs/language/0008-effects-and-capabilities-v0.md`
- Must-use source: ADR 0219
- Admission source: ADR 0218
- Depends on: ADR 0003, ADR 0009, ADR 0017, ADR 0018, ADR 0019, ADR 0021, ADR 0024, ADR 0136, ADR 0217, ADR 0218, ADR 0219

## Context

Fairy C already treats `Option<T>` and `Result<T, E>` as distinct tagged values. Reading a
resource-bearing active payload by value moves it. Safe-borrow leaves retain their origin and active
loan. `Result` propagation with postfix `?` exits the current function through its error path, runs
normal cleanup for the lexical scopes it crosses, and never turns a trap into an error.

Concise optional navigation is useful, but copying a conventional null-safe operator is not enough.
A naïve implementation of every `?.` as an independent map operation would repeatedly construct and
immediately inspect temporary wrappers:

```text
Option<Account>
Option<Option<Profile>>
Option<Option<Option<ref str>>>
```

That representation would shorten source while making typing, Resource Flow, Borrow Flow, must-use
tracking, MIR, and generated cleanup worse. It would also erase the useful fact that every absent
stage reaches one common path-failure continuation.

The current parser has an additional compatibility hazard. It tokenizes `?` and `.` separately, and
its postfix loop accepts propagation followed immediately by field access or indexing. Existing
source may therefore write:

```fy
result?.field
result?[index]
```

with the legacy meaning of:

```fy
(result?).field
(result?)[index]
```

Adding `?.` and `?[` only to semantic analysis would leave that source spelling dependent on token
grouping and whitespace. The transition must be an exact grammar revision with previous-revision
parsing and migration, not type-directed reinterpretation.

The current Result Flow also recognizes an exact top-level `Result` type when creating and
discharging obligations. Optional calls can naturally produce `Option<Result<T, E>>`. The outer
Option must not erase the inner Result obligation, so an executable optional-path slice needs a
generalized transparent Option envelope for must-use tracking.

This ADR records a complete candidate under the name **Option Path Expressions**. It does not add
tokens, reserve syntax, change the language pin, alter current `Option` or `Result` behavior, change
MIR, or claim implementation.

## Candidate decision

Fairy C should admit one typed control-flow feature containing optional member navigation, optional
index navigation, and lazy Option coalescing:

```fy
let ref str display_name =
    account?.profile()?.display_name_view()
    ?? "guest";
```

The defining contract is:

> Evaluate the root once, execute each path segment only while the path is present, merge every
> path-produced absence into one logical none continuation, preserve ordinary ownership, borrow,
> effect, trap, and must-use rules, and construct an Option only at the final result boundary when
> the expression still needs one.

An Option Path Expression is not a nullable type, a runtime helper protocol, a chain of nested
`Option` values, or a hidden exception mechanism. It is a compiler-owned conditional execution
graph.

The source feature has exactly three operators:

```text
?.      optional member or method segment
?[...]  optional index segment
??      lazy Option coalescing
```

No operator performs automatic Result propagation, bounds recovery, pointer validation, cloning,
allocation, or assignment.

## Core examples

Optional member and method navigation:

```fy
let Option<u32> timeout =
    config?.network()?.timeout_ms();
```

Optional indexing:

```fy
let Option<u8> first =
    packet?.payload()?[0];
```

Lazy default selection:

```fy
let u32 timeout =
    config?.network()?.timeout_ms()
    ?? 5000;
```

Several optional candidates followed by one final default:

```fy
let Endpoint endpoint =
    environment_endpoint
    ?? config?.network()?.endpoint()
    ?? default_endpoint();
```

Conditional command execution:

```fy
cache?.clear();
```

The command form is valid only when the selected operation returns `void`. It produces no
`Option<void>` and is not a value expression.

## One Option envelope for the complete path

Assume:

```text
account: Option<Account>
Account.profile() -> Option<Profile>
Profile.display_name_view() -> ref str
```

Then:

```fy
account?.profile()?.display_name_view()
```

has type:

```text
Option<ref str>
```

The type rule for one segment is:

```text
Option<T> ?. (T -> U)          = Option<U>
Option<T> ?. (T -> Option<U>)  = Option<U>
```

The first rule is map-like. The second is flat-map-like. Source does not name either abstraction and
the compiler does not construct an intermediate wrapper.

Flattening is exactly one level introduced at the current path boundary. It is not recursive
normalization of user data.

If a segment deliberately returns:

```text
Option<Option<U>>
```

the path retains:

```text
Option<Option<U>>
```

after merging the path envelope with only the outer returned Option. The inner Option remains
meaningful user data. A later explicit `?.` segment may choose to navigate that remaining layer.

All absent roots and absent Option-returning segments denote one logical path failure. The feature
does not preserve which stage was absent. Source that needs that distinction must use `switch`.

When no `??` follows, successful completion constructs one final `.some(value)` and every path
failure constructs one final `.none`. No earlier segment constructs an Option.

When `??` removes the envelope, even the final Option construction disappears. Success and fallback
initialize one result local directly.

This is a semantic lowering rule, not an optional optimization. Debug and optimized profiles must
use the same ownership and cleanup graph.

## Source grammar

The candidate grammar is conceptually:

```ebnf
coalesce_expression
  = logical_or_expression
    ("??" coalesce_expression)?
  ;

optional_path_expression
  = postfix_expression optional_segment+
  ;

optional_segment
  = "?." IDENT call_suffix?
  | "?[" optional_index_contents "]"
  ;

optional_index_contents
  = expression
  | expression ".." expression?
  | ".." expression?
  ;
```

The exact grammar integration may factor postfix parsing differently, but it must preserve these
source concepts.

`?.` and `?[` have the same postfix binding power as ordinary member access, indexing, slicing, and
calls. `??` is right-associative and has lower precedence than every existing arithmetic,
comparison, bitwise, and call operation.

The following parses as:

```fy
option ?? (compute() + 1)
```

not:

```fy
(option ?? compute()) + 1
```

Unparenthesized mixing of `??` with `&&` or `||` is rejected:

```fy
flag ?? false || override
```

Source must choose one meaning explicitly:

```fy
(flag ?? false) || override
```

```fy
flag ?? (false || override)
```

This rejection is syntactic. Readability must not depend on whichever parse happens to type-check.

## Multi-character tokens and revision behavior

The current lexer matches symbols in ordered longest-first form. An executable slice adds:

```text
?.
?[
??
```

before the single-character `?`.

The current-revision lexer must emit each as one symbol token. The immediately previous language
revision must continue splitting the same bytes into its historical tokens so the standalone
migrator can parse old source according to old rules.

Whitespace cannot manufacture an alternate meaning. After the new revision, an ungrouped Result
propagation node may not be followed by an ordinary member or index suffix even when trivia appears
between the tokens:

```fy
result? .field
result? [index]
```

Both forms require grouping:

```fy
(result?).field
(result?)[index]
```

Likewise, two nested Result propagations require:

```fy
(result?)?
```

rather than an ungrouped `result??` or `result? ?`.

The previous-revision migrator must rewrite every accepted propagation-then-member,
propagation-then-index, and consecutive-propagation CST shape to the grouped spelling before changing
the exact language pin. The rewrite is syntax-directed and does not require type inference.

This removes a permanent whitespace-sensitive ambiguity and gives formatters one canonical spelling.

## Optional member and method segments

`?.name` first requires a present path payload. It then performs the same resolved member operation
that ordinary `.name` would perform on that payload.

A method segment may include one immediate call suffix:

```fy
user?.calculate_score(load_expensive_policy())
```

The receiver gate is tested before any argument expression. If `user` is absent,
`load_expensive_policy()` is not evaluated.

The method receiver declaration determines ownership:

```text
value receiver     payload may move
ref receiver       shared borrow
ref mut receiver   exclusive borrow
```

`?.` itself has no ownership mode and never inserts a clone or an automatic borrow.

A field segment follows the ordinary field-read rule. Copy fields are copied. Borrow-valued fields
retain their origin. A prohibited resource-field move remains prohibited; optional navigation does
not bypass the existing complete-root ownership boundary.

Every later optional segment is explicit:

```fy
account?.profile()?.display_name_view()
```

An ordinary `.` after an optional path closes that path expression and attempts ordinary member
access on its resulting Option. It does not silently continue inside the payload. This keeps every
conditional step visible.

A segment returning `void` is admitted only as the final segment of an expression statement:

```fy
cache?.clear();
```

The absent path performs no call. The present path performs the command. The expression type is
`void`.

The following is rejected:

```fy
let result = cache?.clear();
```

There is no implicit execution-status boolean and no `Option<void>` value.

## Optional indexing

`?[index]` gates only on path absence:

```fy
let Option<u8> first =
    packet?.payload()?[0];
```

If the payload path is absent, the index expression is not evaluated and the path becomes absent.

If the payload exists, indexing uses the ordinary index and slice rules. An out-of-range index still
traps. Optional indexing must never convert bounds failure into `.none`.

Source that wants recoverable absence from bounds uses an explicit safe API:

```fy
let Option<ref u8> first =
    packet?.payload()?.get(0);
```

This preserves Fairy C's separation:

```text
Option   data-model absence
Result   recoverable operation failure
trap     contract violation
```

Raw-pointer indexing remains outside the safe feature.

## Coalescing closes or preserves the envelope

`??` has exactly two type rules:

```text
Option<T> ?? T          -> T
Option<T> ?? Option<T>  -> Option<T>
```

The first rule removes the envelope by selecting a final fallback:

```fy
let u32 timeout =
    configured_timeout ?? 5000;
```

The second chooses the first present candidate while preserving possible absence:

```fy
let Option<Endpoint> endpoint =
    environment_endpoint
    ?? project_endpoint;
```

After contextual typing, the right operand must have the exact canonical `T` or `Option<T>` required
by the left operand. `??` does not add a new conversion or coercion relation.

The operator is right-associative:

```fy
a ?? b ?? c
```

parses as:

```fy
a ?? (b ?? c)
```

Execution remains left-to-right and lazy. The compiler evaluates `a` first. It evaluates `b` only
when `a` is absent and `c` only when both earlier candidates are absent.

Fallback effects, allocation, calls, traps, and Result propagation occur only on the selected
fallback path at runtime. Their possible effects still belong to the enclosing function's static
effect contract.

`??` is a value selector, not an assignment operator. `??=` is not part of this ADR.

## Result propagation remains explicit

Optional navigation is defined only for `Option`.

The following is a type error when `load_user()` returns `Result<User, LoadError>`:

```fy
load_user()?.name
```

The diagnostic should require explicit propagation:

```fy
(load_user()?).name
```

The distinction is semantic:

```text
Option absence stays inside the current expression.
Result error exits the current function through postfix ?.
```

Changing a type from Option to Result must not silently change local expression control flow into a
function exit.

An Option path may produce a Result value without interpreting it:

```fy
writer?.write(data)
```

If:

```text
writer: Option<Writer>
Writer.write(...) -> Result<usize, WriteError>
```

then the path type is:

```text
Option<Result<usize, WriteError>>
```

It is not:

```text
Result<Option<usize>, WriteError>
```

and the call is not propagated automatically.

Source can state its policy explicitly:

```fy
let usize written =
    (writer?.write(data) ?? .ok(0))?;
```

The meaning is exact:

```text
absent writer       choose successful zero-byte Result
present writer      call write
write error         propagate from the current function
```

No operator in this ADR transposes `Option<Result<...>>` into `Result<Option<...>>`.

## Ownership is ordinary ownership

Option Path Expressions do not clone, retain, reference-count, or allocate.

Copy payloads and fields use ordinary copy semantics.

A by-value receiver or selected move-only fallback performs an ordinary move.

A shared receiver creates an ordinary shared borrow.

A mutable receiver creates an ordinary exclusive borrow only on the present path.

For:

```fy
let Socket socket =
    cached_socket ?? connect(address)?;
```

where `cached_socket` is an owned `Option<Socket>` and `Socket` is move-only, the coalescing
expression consumes the complete `cached_socket` root.

On the present path its payload moves into `socket`. On the absent path the fallback executes. After
the join, `cached_socket` is dead on every reachable continuation:

```fy
use(cached_socket);
```

is rejected.

Keeping the root live only on the absent path would create a hidden path-dependent partial-move
state. The compiler must not repair that state with a runtime flag.

The same rule applies when an optional member segment invokes a by-value receiver. The complete
owned root is considered consumed at the path boundary on every continuation that reaches the join.

When every selected operation only copies or borrows, the root remains live under ordinary copy and
loan rules.

Source that wants to preserve an owner should navigate an Option of a borrow:

```fy
let Option<ref Socket> cached =
    cache.lookup(name);

let ref Socket socket =
    cached ?? ref fallback_socket;
```

The exact legality still follows the existing expression, borrow-origin, and owner-lifetime rules.

## Borrow origins survive gates and coalescing

An optional path does not create a new lifetime category. It preserves existing origin and loan
facts through the present branch and carries no borrow on the absent branch.

A method returning `ref T`, `ref mut T`, `ref str`, or a borrowed view retains the callee's proven
origin. Wrapping that borrow in the final Option does not make the owner live longer.

A borrow into a temporary Option-owned value cannot escape the temporary's cleanup:

```fy
let Option<ref User> user =
    make_optional_user()?.view();
```

This is accepted only if the final borrow origin outlives the expression result. A borrow into a
temporary local owner is rejected by the existing return-origin and owner-lifetime rules.

Coalescing may merge borrow origins:

```fy
fn ref str choose_name(
    Option<ref str> primary,
    ref str secondary,
) {
    return primary ?? secondary;
}
```

The present path originates from `primary`; the fallback originates from `secondary`. Returning that
join remains subject to the existing multiple-origin return restriction and is rejected when no
single valid return origin can be proved.

Using the same join locally is valid when every possible owner outlives the use:

```fy
let ref str selected =
    primary ?? secondary;

consume(selected);
```

`Option<ref mut T>` follows the same rule. Only the selected path's mutable loan survives. The
fallback path does not create its loan unless selected.

Optional paths neither extend lifetimes nor collapse several origins into a false single origin.

## Temporary and cleanup behavior

Every source expression is evaluated at most once.

The root is materialized before its first tag test.

Arguments and indices are materialized only in the present block that needs them.

A segment result needed by a later segment remains live until its last use or until its path edge is
cleaned.

A none edge may leave a different set of path-local temporaries live depending on which gate failed.
The implementation therefore has one logical none continuation but may use gate-specific cleanup
preludes before converging on that continuation.

For a path with `??`, each none edge performs the required lexical defer calls, resource drops, and
Region resets for path-local scopes before evaluating the fallback.

For a path without `??`, each none edge performs its required cleanup before reaching the one final
`.none` construction.

On the successful edge, the final value is evaluated and materialized before cleaning path-only
temporaries. Borrow Flow must reject a final borrow that depends on an owner destroyed by that
cleanup.

A trap retains the existing non-unwinding behavior. It reaches no Option fallback and performs no
language-level cleanup.

## Effects remain unconditional contracts

Calls and argument expressions that may execute on a present path contribute their declared effects
to the enclosing function.

Fallback expressions that may execute on an absent path contribute their effects as well.

Conditional execution does not remove possible authority from the public function contract:

```fy
@effects(filesystem)
fn Option<Config> load_if_present(Option<Path> path) {
    return path?.read_config();
}
```

The function still declares `filesystem`.

Tooling may report that an effect is conditional on a particular gate. The type system uses the
union of every statically reachable path.

## Must-use obligations pass through Option

An Option wrapper must not launder a Result or an `@must_use` call result.

The following types remain must-use when produced by an expression:

```text
Result<T, E>
Option<Result<T, E>>
Option<Option<Result<T, E>>>
```

Likewise, a final optional-path value produced by a callable marked `@must_use` remains must-use even
though the call may be skipped on an absent path.

The dynamic rule is branch-sensitive:

```text
absent path    no inner Result was produced
present path   the produced Result obligation is retained in the Option payload
```

The static use rule is conservative: because a reachable path can produce the obligation, the final
wrapper must be handled.

This is rejected when `flush` returns `Result<void, FlushError>`:

```fy
session?.flush();
```

The diagnostic should report the complete type:

```text
unused Option<Result<void, FlushError>>
the call is conditional, but every produced Result must still be handled
```

Valid policies include returning the wrapper, exhaustively switching it and handling the present
Result, explicitly discarding the complete wrapper, or coalescing to one Result and propagating or
handling it.

An exhaustive Option switch transfers the obligation only into its `.some` payload binding. The
`.none` arm creates no obligation.

An explicit `discard` of the complete wrapper is an explicit policy and discharges the obligation
while dropping only the active payload.

The current top-level `isResultType` use classification is insufficient for this feature. An
executable slice must introduce a transparent Option obligation walk or equivalent expression-level
must-use metadata shared by ordinary calls, Option constructors, switches, coalescing, and optional
paths. This is a general Result Flow correction, not a path-specific exception.

## Optional commands do not manufacture values

A final optional call returning `void` is a command:

```fy
cache?.clear();
```

It is allowed in statement position and runs only on a present path.

It does not produce a boolean indicating whether the command ran, and it does not produce
`Option<void>`.

Source that needs an observable execution result must use a method returning `bool`, `Option<T>`, or
`Result<T, E>`.

A non-void path retains its ordinary value type, usually `Option<T>`, and ordinary expression
statement rules apply. Any embedded must-use obligation still requires handling.

## Assignment targets remain explicit

An optional path cannot be an assignment target:

```fy
settings?.timeout_ms = 5000;
```

Silently skipping a state mutation because an owner was absent hides lost updates.

Source should use `switch`, a validation gate, or a method whose return type makes success or failure
observable.

A conditional command method remains legal:

```fy
settings?.set_timeout(5000);
```

When ignoring absence is unacceptable, the setter should return `bool` or `Result`.

This ADR also excludes:

```fy
cache.value ??= compute();
```

That feature would require single evaluation of a mutable place, conditional movement, assignment,
post-assignment refinement, alias invalidation, and separate failure rules. It is a later decision.

## Raw pointers are not Option

Optional navigation is not defined for raw pointers:

```fy
raw_user?.name
```

A non-null raw pointer does not prove valid storage, alignment, lifetime, initialization, or alias
rules.

Unsafe code must first validate and convert the raw value into an `Option<ref T>` or another safe
abstraction, then use the optional path.

This ADR does not add implicit pointer checks, safe-pointer construction, or bounds-to-none
conversion.

## Lossless CST boundary

The CST preserves each source operator:

```text
KindOptionAccess
KindOptionIndex
KindCoalesce
```

`KindOptionAccess` retains the `?.` span, member name span, and optional immediate call suffix.
`KindOptionIndex` retains the `?[`, index or slice expressions, and closing bracket span.
`KindCoalesce` retains the operator span and both operands.

Lossless reconstruction must reproduce every original byte. Diagnostics and the migrator must be
able to distinguish:

```text
new optional token
legacy propagation followed by a suffix
grouped propagation
ordinary member or index access
```

The parser must recover malformed optional segments without guessing a Result interpretation.

## AST normalization

AST lowering should normalize one contiguous optional chain into one path rather than nested
operator nodes:

```go
type OptionalPath struct {
    Span     diagnostic.Span
    Root     *Expression
    Segments []OptionalSegment
}

type OptionalSegment struct {
    Kind      OptionalSegmentKind
    Span      diagnostic.Span
    Name      string
    NameSpan  diagnostic.Span
    Arguments []*Expression
    Index     *Expression
    SliceEnd  *Expression
}
```

For:

```fy
account?.profile()?.display_name_view()
```

the AST shape is conceptually:

```text
OptionalPath
  root account
  call profile
  call display_name_view
```

A separate AST node preserves coalescing:

```go
type OptionCoalesce struct {
    Span  diagnostic.Span
    Left  *Expression
    Right *Expression
}
```

The AST does not assign types, flatten modes, ownership, or resolved symbols.

## Typed HIR boundary

Semantic analysis resolves and types every segment before Flow:

```go
type OptionPath struct {
    Span             Span
    Root             Expression
    RootOptionType   Type
    PayloadType      Type
    ResultType       Type
    ProducesOption   bool
    Command          bool
    Segments         []OptionPathSegment
}

type OptionPathSegment struct {
    Kind             OptionPathSegmentKind
    Span             Span
    InputType        Type
    OutputType       Type
    PathPayloadType  Type
    Field            string
    Callee           SymbolID
    Arguments        []Expression
    Index            *Expression
    SliceEnd         *Expression
    FlattensOne      bool
    Access           OptionAccessMode
    MustUse          bool
}
```

`Access` records the already-resolved ordinary operation:

```text
copy
move
shared borrow
mutable borrow
void command
```

`FlattensOne` is true only when the selected segment itself returns `Option<U>`. It never means
recursive Option normalization.

Coalescing has an explicit HIR form:

```go
type OptionCoalesce struct {
    Span        Span
    Left        Expression
    Right       Expression
    ElementType Type
    ResultType  Type
    Mode        CoalesceMode
    Access      OptionAccessMode
    MustUse     bool
}
```

The two modes are:

```text
unwrap-with-value
first-present-option
```

HIR retains the complete path until Resource Flow, Borrow Flow, Result Flow, effect checking, and
must-use checking have agreed on the same conditional graph.

The source does not gain a new nullable canonical type. Existing `Option(element)` and
`OptionElement(value)` remain the only type identity.

## Resource Flow

Resource Flow must model the present and absent edges explicitly.

It must prove:

```text
a moved payload was live before extraction
a by-value segment consumes the complete owned root on every continuation
an absent edge never drops a nonexistent payload
a present payload is dropped exactly once when not transferred
path-local resources are cleaned before the fallback
a fallback is evaluated only after the selected none-edge cleanup
success and fallback initialize the result exactly once
no branch join leaves a hidden MaybeLive root
```

A path returning `Option<Resource>` retains one final active-payload drop obligation.

A coalesce returning `Resource` transfers exactly one obligation from either the present payload or
the fallback.

A coalesce returning `Option<Resource>` transfers one wrapper-owned conditional obligation.

The compiler must not implement these states as source-visible flags or backend guesses.

## Borrow Flow

Borrow Flow must prove:

```text
present-only borrows exist only on the present edge
absent edges carry no retained payload loan
segment arguments cannot conflict with the gated receiver borrow
a returned borrow preserves its original owner
a path-local temporary owner outlives every retained borrow
fallback-origin and present-origin joins remain explicit
mutable loans exist only for the selected branch and required extent
```

Optional paths reuse current last-use loan shortening.

A borrow may end before the final closing token when its last use proves that safe. The expression
boundary is only a maximum extent for path-owned temporaries and conditional loans.

No optional path may invent a static origin, erase a parameter index, or convert several origins into
one.

## Result Flow and must-use Flow

Result Flow needs one shared representation of obligations carried under transparent Option
envelopes.

The analysis must prove:

```text
skipped calls produce no dynamic Result obligation
executed calls transfer their obligation into the final wrapper
coalescing transfers the selected obligation only once
Option switches transfer an obligation to the .some binding only
return and call arguments transfer the complete wrapped obligation
explicit discard discharges the complete wrapped obligation
an ignored wrapper remains E4101 or its admitted successor diagnostic
```

A branch join is pending when any reachable incoming path retains an obligation.

Repeated loop execution must not overwrite a pending wrapped obligation.

The implementation must not add a one-off check only for `session?.flush()`. Option construction,
ordinary calls returning Option<Result>, optional paths, coalescing, assignments, parameters, and
switches must use the same obligation model.

## MIR lowering

HIR lowers one Optional Path into one control-flow graph.

For:

```fy
let u32 timeout =
    config?.network()?.timeout_ms()
    ?? 5000;
```

the conceptual graph is:

```text
evaluate config once
test config tag
none -> config-none cleanup prelude -> fallback

call network
test network tag
none -> network-none cleanup prelude -> fallback

call timeout_ms
store result
clean path-only temporaries
goto join

fallback
evaluate 5000
store result
goto join

join
use one initialized u32 local
```

There is one logical fallback continuation. Gate-specific cleanup blocks may precede it.

Without `??`:

```fy
let Option<u32> timeout =
    config?.network()?.timeout_ms();
```

the graph is:

```text
all none edges -> required cleanup -> construct one .none
success edge   -> required cleanup -> construct one .some(timeout)
join           -> one Option<u32> local
```

The lowering may reuse existing Option tag switches, `option-payload`, `option-construct`, ordinary
call terminators, drops, and gotos. A backend receives no `?.`, `?[`, or `??` operation.

MIR should retain source-linked verification facts rather than a runtime optional-path instruction:

```go
type OptionPathFact struct {
    Path                hir.Span
    Root                hir.Span
    Gates               []OptionGateFact
    LogicalNoneTarget   BlockID
    FinalConstructCount int
    IntermediateOptions int
}

type OptionGateFact struct {
    Operator             hir.Span
    InputType            hir.Type
    PayloadType          hir.Type
    PresentTarget        BlockID
    NoneCleanupTarget    BlockID
    ConditionalArguments []hir.Span
    FlattensOne          bool
}

type CoalesceFact struct {
    Operator       hir.Span
    LeftOption     hir.Span
    Fallback       hir.Span
    FallbackTarget BlockID
    ResultType     hir.Type
}
```

Exact fact names remain compiler-private until an implementation decision versions them.

## MIR verification

The verifier must prove at least:

```text
the root is evaluated exactly once
every gate reads the tag of its exact input
a payload extraction is dominated by the matching present edge
conditional arguments and indices execute only on the present edge
every flattened segment removes exactly one Option level
all logical none edges reach the correct cleanup and none or fallback continuation
no intermediate path-created Option remains
the final Option is constructed at most once per concrete path
success and fallback initialize the join local exactly once
a moved root is dead on every continuation
resource drops and drop-flag transitions are neither missing nor duplicated
retained borrows outlive every cleanup that precedes the join
wrapped must-use obligations are preserved
bounds traps remain traps and have no cleanup successor
fallback expressions never execute on a present path
no optional-path source form reaches an interpreter or backend
```

Verifier mutation fixtures should independently duplicate a root evaluation, move an argument before
its gate, bypass cleanup, extract a payload on a none edge, retain an intermediate Option, reuse a
moved root, drop an active payload twice, erase a wrapped Result obligation, or redirect a fallback.
Every mutation must fail closed.

## Diagnostics

An executable slice needs exact source-linked diagnostics for at least:

```text
?. or ?[...] on a non-Option root
?. on Result with the explicit `(expr?).member` repair
?[...] on Result with the explicit `(expr?)[index]` repair
optional navigation on a raw pointer
malformed optional member or index syntax
?? with a non-Option left operand
a right operand that is neither T nor Option<T>
unparenthesized mixing of ?? with && or ||
a void optional command used as a value
an optional path used as an assignment target
a move-only root consumed by navigation or coalescing and later reused
a borrow whose owner dies on a path cleanup edge
an unsupported resource-field move
an unused Option-wrapped Result or @must_use result
a mistaken expectation that ?[...] converts bounds failure to none
```

The Result diagnostic should name both exact types:

```text
`?.` requires Option<T>, but this expression is Result<User, LoadError>
propagate the Result explicitly with `(load_user()?).name`
```

The raw-pointer diagnostic should point to the validation boundary:

```text
optional navigation is not defined for raw pointer `*User`
validate it into `Option<ref User>` inside an unsafe wrapper
```

The bounds diagnostic may suggest `.get(index)` only when a compatible safe API actually exists.

Move diagnostics should relate the operator, consumed root declaration, payload type, and later use.

Must-use diagnostics should display the complete wrapper shape rather than reporting only the hidden
Result.

## Tooling and Atlas

Tools should treat one optional path as one source entity rather than guessing a nested AST chain.

A future Atlas export may include:

```text
root expression
ordered gate spans
resolved member or method symbol per segment
present and none edges
fallback span
final canonical type
copy, move, shared-borrow, or mutable-borrow mode
conditional effects
must-use obligation
borrow-origin relation
```

This supports stable questions such as:

```text
which absence edges reach this fallback
which calls and effects are skipped by this gate
which segment consumes the owner
which segment returns Option
how removing this coalesce changes the final type
which Result obligation is carried under the outer Option
```

The candidate requires no Atlas schema change. Export should follow one executable slice and a
separate versioned transport decision.

## Performance and code generation

Option Path Expressions require no heap allocation, closure, dynamic dispatch, reference count,
exception table, unwinder, or runtime helper stack.

The root is evaluated once.

Every segment adds at most one required tag branch when its presence is not statically known.

A statically known `.some` or `.none` may remove the branch after verified constant reasoning.

No intermediate Option is constructed.

A final `?? T` path constructs no Option.

A final `Option<T>` path constructs one wrapper.

Backends receive ordinary branches, payload reads, calls, indexes, assignments, and cleanup blocks.
They do not reparse or reinterpret optional syntax.

## Delivery slices

### Slice 1: copy and inert value paths

Admit:

```fy
option?.field
option?.method(arguments)
option ?? fallback
```

for copy scalars and admitted inert structs.

Require current and previous-revision lexing, migration of legacy propagation suffixes, lossless CST,
AST normalization, exact root, argument, and fallback evaluation, the one-envelope lowering rule,
typed HIR, verified MIR, interpreter execution, and constrained-C execution.

Reject borrow, resource, wrapped must-use, and optional-index cases with stable unsupported
diagnostics rather than partially lowering them.

### Slice 2: safe-borrow paths

Admit `Option<ref T>`, shared and mutable receiver calls, borrow-returning methods, `ref str`, and
borrowed views.

Add origin propagation, last-use loans, temporary-owner rejection, fallback-origin joins, and
verifier mutation evidence.

### Slice 3: resources and must-use envelopes

Admit move-only Option payloads, by-value receivers, resource-bearing coalescing, and
`Option<Result<T, E>>` or Option-wrapped `@must_use` outcomes.

Add complete-root consumption, edge-specific cleanup, drop-flag verification, transparent Option
obligations, exhaustive Option-switch transfer, and exact ignored-wrapper diagnostics.

### Slice 4: optional indexing and tooling facts

Admit `?[index]` and optional slicing for implemented stable array and view families while preserving
ordinary bounds traps.

Add migration coverage for legacy propagation-plus-index source, Atlas or IDE facts after a
versioned tooling decision, and optimized backend evidence that does not alter trap or cleanup
semantics.

Each executable slice needs an accepted implementation ADR or accepted revision of this ADR with
exact diagnostics, language and compatibility versions, fixture updates, MIR facts, cache impact,
and same-commit native evidence.

## Rejected alternatives

- Treating `?.` as repeated `Option.map`: creates intermediate wrappers and hides one control graph.
- Deep-flattening every nested Option: destroys deliberate user data.
- Defining `?.` for Result: makes a type change alter function-exit control flow.
- Automatically transposing Option and Result: chooses an error policy for the caller.
- Converting bounds failure to `.none`: hides contract violations.
- Allowing raw-pointer optional navigation: treats non-null as a complete memory-safety proof.
- Adding implicit clone or borrow behavior: hides ownership cost.
- Keeping an owned Option root live only on the absent side of a move: creates hidden MaybeLive state.
- Erasing must-use under Option: turns optional calls into an error-suppression mechanism.
- Optional assignment: silently loses state mutations.
- Bundling `??=` into coalescing: adds a separate conditional-place and refinement feature.
- Type-directed reinterpretation of legacy `result?.field`: makes source meaning unstable.
- Depending on whitespace to distinguish Result propagation from optional navigation: breaks
  formatting and migration.
- Immediate backend desugaring without HIR and verifier facts: lets ownership and cleanup drift
  between profiles.
- A runtime optional-navigation helper: adds cost and duplicates compiler-known control flow.

## Admission evidence

Promotion from `Candidate` requires at least one complete vertical executable slice with all of the
following:

```text
normative grammar and ordered multi-character token rules
current and previous language revision lexer behavior
lossless reconstruction for ?., ?[, and ??
migration of legacy propagation-plus-member, propagation-plus-index, and nested propagation
bounded malformed-input recovery
AST normalization into one optional path
exact root evaluation once
present-only argument and index evaluation
right-associative lazy coalescing
unparenthesized logical-mix diagnostics
one-level flattening and preservation of intentional nested Option
proof that no intermediate path-created Option survives
type diagnostics for Result, raw pointer, and non-Option roots
Resource Flow evidence for every admitted access mode
Borrow Flow evidence for origins, loans, and temporary owners
Result and @must_use evidence through transparent Option envelopes
effect evidence for conditional segments and fallbacks
verified MIR facts for gates, cleanup preludes, joins, and final construction
MIR verifier mutation fixtures for ordering, tags, payloads, cleanup, moves, loans, and obligations
interpreter and constrained-C byte-for-byte behavioral agreement
native Linux and Windows evidence for present, absent, fallback, trap, and partial-resource paths
proof that bounds failure remains a trap
proof that no runtime helper, hidden allocation, clone, or Option nesting was introduced
compatibility-vector, language-pin, cache-key, starter, and migration updates when syntax is admitted
```

Resource slices additionally require active-payload cleanup evidence on every none and success edge.

Borrow slices additionally require multiple-origin return rejection and local-use acceptance.

Must-use slices additionally require direct calls returning `Option<Result>`, optional calls,
constructors, assignments, parameters, switches, return transfer, propagation after coalescing, and
explicit discard.

A lexer-only, parser-only, type-checker-only, Flow-only, backend-only, or optimization-only
implementation is inadmissible.

## Compatibility

This candidate changes no current source grammar, token set, language pin, SDK, runtime ABI, MIR
version, public interface, package identity, cache identity, generated code, or target maturity.

An executable slice introduces three new symbol tokens and changes the parse of source bytes that the
previous revision may already accept. The exact language revision must advance.

The immediately previous lexer and parser must preserve legacy tokenization and CST meaning. The
migrator must group old propagation suffixes before updating the exact pin. New optional source has
no previous-revision equivalent and must not be accepted as old source through token fallback.

The formatter must emit grouped Result propagation and canonical optional tokens so formatting never
changes meaning.

Bodies using the new operators change package source digests and exact language-keyed caches in the
ordinary way. Public callable signatures do not change merely because a body uses an optional path.

If verified MIR gains serialized optional-path facts, its artifact version and stable package
artifact compatibility must advance in the implementation slice that admits the encoding. The final
backend ABI need not change because lowering uses existing branches, tags, payload operations, and
calls.

Optional path IDs, gate ordinals, logical none targets, and optimizer thresholds remain
compiler-private unless a later contract versions them.

## Consequences

Option Path Expressions compress nested Option switches without introducing a second null model.
Every explicit `?.` or `?[...]` is one conditional path segment. Every root, argument, index, and
fallback is evaluated only on its required path and at most once.

The design preserves Fairy C's sharp distinctions. Option absence remains local. Result errors remain
explicit function-exit control flow. Bounds and arithmetic violations remain traps. Raw pointers
remain unsafe. Copy, move, shared borrow, mutable borrow, effects, and must-use obligations remain
visible to the compiler and tooling.

The implementation cost is paid in the correct layers: revision-aware tokenization and migration,
lossless CST, normalized path AST, typed HIR, branch-sensitive Resource and Borrow Flow, transparent
Option must-use obligations, one logical none continuation with edge-specific cleanup, and MIR
verification.

Backends receive ordinary verified control flow. No current source program or runtime behavior
changes under this candidate ADR alone.
