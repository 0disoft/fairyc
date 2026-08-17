# ADR 0331: Scoped projection borrow windows

- Status: Candidate
- Date: 2026-08-17
- Product source: `docs/product/02-spec.md`
- Grammar source: `spec/grammar/stage0.ebnf`
- Borrow sources: `docs/adr/0003-stage-0-safety-boundary.md`, `docs/language/0002-borrows-and-return-origins.md`
- Flow sources: ADR 0010, ADR 0011, ADR 0012, ADR 0021, ADR 0024, ADR 0217
- Place sources: ADR 0097, ADR 0126, ADR 0135
- Surface source: ADR 0136
- Admission source: ADR 0218
- Depends on: ADR 0003, ADR 0024, ADR 0097, ADR 0126, ADR 0136, ADR 0217, ADR 0218

## Context

Fairy C already shortens local loans to their last proven use. A programmer can also place an
ordinary safe-borrow binding inside a smaller lexical block. A new spelling that only abbreviates
one `ref` or `ref mut` binding would therefore add syntax without adding a meaningful safety or
analysis capability.

The missing capability is precision. The current Stage 0 contract deliberately treats borrows of
different fields of the same root as borrowing the whole root. That rule keeps diagnostics, branch
joins, and loop fixed points bounded, but it makes state-oriented code awkward when one operation
needs independent access to several disjoint fields or two non-overlapping portions of one byte
buffer.

Changing all ordinary borrow analysis to be field-sensitive would impose that cost on every
function. It would also enlarge the existing loan key, branch merge, loop fixed-point, owner-move,
and diagnostic contracts before representative code proves that the extra precision is worth the
complexity.

The useful language feature is therefore not a short-lived alias. It is an explicit declaration of
the alias topology that is valid inside one lexical block. The compiler may perform more expensive
projection analysis only where the source requests it, while ordinary `ref` and `ref mut` retain the
current whole-root rule.

This ADR records that complete candidate model. It does not reserve `with`, change the language pin,
add executable syntax, or claim implementation.

## Candidate decision

Fairy C should admit **Borrow Windows** as zero-cost scoped borrow plans.

A Borrow Window simultaneously guarantees three properties:

1. every selected place and selector expression is evaluated exactly once;
2. every safe-borrow generation derived from the plan has the closing brace as a hard maximum
   lifetime boundary; and
3. overlap is analyzed at a bounded projection granularity inside the window while ordinary code
   remains whole-root.

The defining contract is:

> Select one or more places once, grant verified projection-level aliases to one lexical block, and
> prevent every derived loan from surviving any exit from that block.

A window alias is not a new owner, pointer category, capability object, or runtime allocation. It is
a compiler-proven safe borrow with an additional window identity and place path.

## Source forms

### Single binding

A shared window uses `ref`:

```fy
with ref database = config.database {
    connect(database.host, database.port)?;
}
```

An exclusive window uses `ref mut`:

```fy
with ref mut account = state.account {
    account.score += bonus;
    account.last_seen = now;
}

save_state(ref state);
```

The alias name is always an immutable binding. `ref mut` grants mutation of the referent; it does not
permit rebinding the alias itself.

### Multiple bindings

Several aliases form one plan:

```fy
with (
    ref mut queue = state.queue,
    ref mut cache = state.cache,
    ref metrics = state.metrics,
) {
    process_queue(queue, cache);
    report(metrics);
}
```

The `as` keyword is not used. Fairy C already owns `as` as the explicit cast operator. The
`alias = place` shape also matches the source language's existing declaration direction without
making a window look like a conversion.

### Value-producing body

A Borrow Window is an expression. Its body tail is the result:

```fy
let i32 score = with ref account = state.account {
    calculate_score(account)
};
```

The result may be a copy value or a newly owned value built from borrowed data:

```fy
let UserSummary summary = with ref account = state.account {
    UserSummary {
        .id = account.id,
        .score = account.score,
    }
};
```

A result that retains a window-derived safe borrow is rejected:

```fy
let ref Account escaped = with ref account = state.account {
    account
};
```

This remains invalid even when the underlying ordinary origin is `Param(i)` or `Static`. A Borrow
Window intentionally gives up escape capability in exchange for local projection precision.
Unrelated safe borrows created in the body may still follow the ordinary origin rules; the rejection
is attached to the window loan lineage, not merely to the result type containing a borrow.

## Closing brace is a maximum lifetime

The closing brace is the latest possible end of a window loan, not a requirement to keep the loan
active until that point.

```fy
with ref account = state.account {
    print_account(account);

    state.account = replacement;
}
```

When the existing last-use analysis proves that `account` is dead after `print_account`, the loan may
end there and the later assignment may proceed. The compiler may shorten a window loan, but it may
never lengthen that generation beyond the window boundary.

This preserves the current non-lexical local-loan behavior. The new proof carried by a loan is a
maximum boundary and projection path, not a request to replace last-use analysis with brace-wide
borrowing.

## Projection model

Inside a Borrow Window, a selected place is represented as one root plus an ordered projection path:

```go
type BorrowPlace struct {
    Root        SymbolID
    Projections []BorrowProjection
}
```

The candidate projection vocabulary is:

```text
field(name)
fixed-index(constant)
dynamic-index(value)
slice(start, end)
```

The full model permits the compiler to describe places such as:

```text
state
state.users
state.users[selected_index]
state.cache
state.metrics.current
packet[..header_length]
```

Admission of a projection kind remains separate from its representation. The first executable slice
may support only named fields and the already admitted fixed-array and byte-view families while the
internal path model remains extensible.

## Overlap rules

Overlap is structural and conservative.

Two different named fields of the same aggregate are disjoint:

```fy
with (
    ref mut users = state.users,
    ref mut cache = state.cache,
) {
    clear_users(users);
    rebuild_cache(cache);
}
```

A root and any of its projections overlap:

```fy
with (
    ref state_view = state,
    ref mut users = state.users,
) {
}
```

Two shared aliases may overlap:

```fy
with (
    ref first = state.users,
    ref second = state.users,
) {
    inspect(first);
    compare(first, second);
}
```

If either alias is mutable, overlapping plans are rejected:

```fy
with (
    ref mut first = state.users,
    ref second = state.users,
) {
}
```

Two fixed indices are disjoint when their canonical compile-time values differ. Equal indices
overlap. A dynamic index is disjoint from another dynamic index only when the compiler has an
explicit proof admitted by a later contract. Merely using different source expressions is not proof.

```fy
with (
    ref mut first = values[0],
    ref mut second = values[1],
) {
    exchange(first, second);
}
```

The following remains rejected in the first implementation:

```fy
with (
    ref mut left = values[left_index],
    ref mut right = values[right_index],
) {
}
```

The diagnostic should state that disjointness cannot be proven and recommend constant indices,
different fields, `split`, or separate windows. The source does not silently acquire a fallible
borrow, insert a runtime inequality check, or choose one alias based on header order.

## Ordinary borrows stay whole-root

Borrow Windows do not globally enable projection-sensitive Flow.

An ordinary binding keeps the Stage 0 contract:

```fy
let users = ref mut state.users;
```

That loan protects the complete `state` root for conflict purposes. Projection precision is requested
only by the window form:

```fy
with ref mut users = state.users {
    clear_users(users);
}
```

This boundary is deliberate. Branch joins, loop fixed points, return-origin summaries, and retained
ordinary-borrow diagnostics stay on the existing representation unless a source window is active.
When a direct access or call-bounded borrow is checked against an active window plan, the compiler may
construct a transient projection path for that conflict check. It does not thereby grant a retained
ordinary local a projection-sensitive lifetime outside the window.

## Plan evaluation and simultaneous activation

A multiple-binding header is not evaluated by activating loans from top to bottom. Sequential
activation would make the legality of later place selection depend on source order and would require
partial rollback after a failed selector or overlap check.

The compiler instead performs these phases:

1. evaluate every selector, index, and range bound from left to right exactly once;
2. construct every complete `BorrowPlace` without any window alias in scope;
3. validate stable-place, mutability, type, bounds, and overlap requirements for the complete plan;
4. activate all plan loans simultaneously; and
5. introduce every alias into the body scope.

Alias names are therefore unavailable throughout the complete header:

```fy
with (
    ref user = users[selected_index],
    ref profile = user.profile,
) {
}
```

That source is rejected. A dependent plan is expressed with nested windows:

```fy
with ref user = users[selected_index] {
    with ref profile = user.profile {
        inspect(profile);
    }
}
```

This rule avoids a header dependency graph, partial loan activation, rollback state, and order-sensitive
alias visibility.

## Selector restrictions

The first selector subset should reuse the existing pure scalar header discipline. An index or range
bound may contain literals, paths, grouping, admitted scalar operators, and admitted casts. Calls,
borrows, blocks, propagation, indexing inside selectors, resource movement, and other ownership-
relevant effects remain excluded from the first slice.

A complex selector is computed explicitly before the window:

```fy
let usize selected_index = choose_user_index();

with ref mut user = users[selected_index] {
    update(user);
}
```

The following is rejected because place selection may reorder or mutate the owner while the header is
being constructed:

```fy
with ref mut user = users[choose_and_reorder_users(ref mut users)] {
}
```

The restriction makes evaluation order visible and lets the compiler validate the complete plan
before any alias exists.

## Safe mutable split

The dedicated split form partitions one mutable byte owner at one evaluated cut:

```fy
with split(ref mut packet, header_length) => (header, payload) {
    parse_header(header)?;
    decrypt_payload(payload)?;
}
```

`header` denotes `packet[..header_length]`. `payload` denotes
`packet[header_length..]`. The owner and cut evaluate exactly once. A cut greater than the retained
length follows the ordinary slicing contract and traps with category `bounds`; the syntax does not
invent an `Option`, `Result`, or overlap-error value.

The two aliases are structurally disjoint because both ranges are produced from one owner and one
canonical midpoint. No raw-address comparison or user assertion is needed.

The first split slice should admit only the currently bounded byte families:

```text
u8[N]
ref mut u8[N]
ref mut u8[]
```

Nested split is a block-bounded reborrow:

```fy
with ref mut used = buffer[..used_length] {
    with split(ref mut used, header_length) => (header, body) {
        validate_header(header)?;
        transform(body)?;
    }

    checksum(used);
}
```

While the inner window is active, overlapping access through `used` is unavailable. After the inner
window's last use or closing brace, `used` becomes usable again. This generalizes the existing
call-bounded mutable reborrow to a compiler-checked block boundary.

Arbitrary range partitions, runtime `distinct` acquisition, user-defined splitters, and multiple
unchecked dynamic indices remain outside this candidate's first executable slices.

## Derived borrows and escape tracking

A window alias may create shorter derived borrows:

```fy
with ref mut account = state.account {
    let name = ref account.name;

    print(name);
    account.score += 1;
}
```

The derived `name` borrow carries the same window lineage. Its shared reborrow may end after
`print`, allowing the parent mutable alias to be used again.

A borrow-returning call also preserves the lineage selected by its ordinary return-origin summary:

```fy
with ref account = state.account {
    let name = account.display_name();
    print(name);
}
```

The result is legal while it remains inside the window. Copying the alias into an outer safe-borrow
local does not erase the boundary:

```fy
var ref Account selected = fallback;

with ref account = state.account {
    selected = account;
}
```

This is rejected when `selected` retains the generation at a normal or early window exit. The
compiler should relate the borrow creation, closing boundary, escaping destination, and last use.

The check is dataflow-based rather than a blanket assignment ban. Replacing the outer local with a
non-window borrow before every exit is valid:

```fy
var ref Account selected = fallback;

with ref account = state.account {
    selected = account;
    inspect(selected);

    selected = fallback;
}
```

Wrapping a borrow in the admitted single-borrow-leaf `Option` or `Result` does not bypass the rule:

```fy
var Option<ref Account> selected = .none;

with ref account = state.account {
    selected = .some(account);
}
```

The existing `borrowValueState` already propagates origins, materialized loan IDs, and pending loans
through copies, reassignment, calls, `Option`, and `Result`. A window identity belongs on the loan
generation so all those existing paths preserve the boundary automatically.

## Owner access inside a window

A source access is checked against the active projection plan.

Disjoint owner fields remain accessible:

```fy
with ref mut account = state.account {
    account.score += 1;
    write_metrics(ref state.metrics);
}
```

A whole-root operation conflicts with any active mutable projection:

```fy
with ref mut account = state.account {
    state.reset();
}
```

Moving, consuming, dropping, or replacing an owner while a matching window generation remains live
continues to use the existing owner-lifetime rule. Projection-aware conflict detection narrows the
protected place; it does not weaken `E2105` or cleanup ownership.

## Early exits and cleanup ordering

Window bodies may use the control forms already admitted by their enclosing context, including
`return`, `break`, `continue`, and Result propagation.

```fy
with ref mut account = state.account {
    validate(account)?;

    if (account.disabled) {
        return .err(.disabled);
    }

    update(account);
}
```

Before an exit is accepted, Borrow Flow must prove that the outgoing value and every reachable outer
binding contain no live generation owned by the window.

For a value-carrying exit, the semantic order is:

1. evaluate and materialize the outgoing copy or owned value;
2. execute lexical `defer` calls declared inside the window in reverse order;
3. clean up resources declared inside the window in reverse declaration order;
4. reset Regions exited inside the window after their contained cleanup;
5. end the window loan proof;
6. perform cleanup for outer lexical scopes crossed by the control transfer; and
7. transfer to the resolved return, loop, or other admitted structured target.

Ending a safe loan is not a runtime operation. The proof remains active through inner lexical cleanup
because that cleanup may still rely on values inside the window. A trap follows the existing Fairy C
contract and does not run language-level cleanup.

If later structured-control proposals admit named outward transfers, those transfers must use the
same resolved target and crossed-scope cleanup calculation. A Borrow Window does not create a new
jump category.

## CST and parser boundary

The lossless CST should preserve Borrow Windows as source concepts:

```go
const (
    KindBorrowWindow  Kind = "borrow-window"
    KindBorrowBinding Kind = "borrow-binding"
    KindBorrowSplit   Kind = "borrow-split"
)
```

A single-binding form and a parenthesized multi-binding form normalize to one window node. Binding
nodes retain the exact alias name span, borrow mode, and place subtree. A split node retains the
owner, cut, and both alias spans.

The parser must not immediately desugar the feature into an ordinary block and borrow bindings.
Source-aware diagnostics, reconstruction, formatter support, simultaneous header scope, and future
Atlas facts all require the original roles to remain explicit.

Malformed headers share the parser's bounded recovery budget. Recovery must not introduce aliases
into the body when the complete plan is not syntactically available.

## AST boundary

AST lowering should retain one dedicated structure:

```go
type BorrowMode uint8

const (
    BorrowShared BorrowMode = iota
    BorrowMutable
)

type BorrowWindow struct {
    Span     diagnostic.Span
    Bindings []BorrowWindowBinding
    Split    *BorrowSplit
    Body     *Block
}

type BorrowWindowBinding struct {
    Name     string
    NameSpan diagnostic.Span
    Mode     BorrowMode
    Place    *Expression
}

type BorrowSplit struct {
    Source    *Expression
    Cut       *Expression
    Left      string
    LeftSpan  diagnostic.Span
    Right     string
    RightSpan diagnostic.Span
}
```

The exact Go shape is not fixed by this candidate. The required boundary is that later phases never
reparse a raw header string or infer binding roles from child position alone. An expression-shaped
window may be carried by an explicit `Window *BorrowWindow` field rather than encoding every fact in
generic `Children` and `Text` fields.

## Resolution and type checking

Resolution proceeds in two scopes.

First, every source place and selector is resolved in the enclosing scope. No alias declared by the
same header is visible. Duplicate aliases are diagnosed after the complete header is collected.
Second, the aliases are introduced together in the body scope.

Type checking must establish:

- each source is a stable place rather than a temporary value;
- an exclusive binding targets a mutable place;
- the exact referent and safe-borrow type for each alias;
- every selector belongs to the admitted pure subset;
- each plan overlap is legal for its mutability pair;
- the split owner and cut belong to an admitted byte family; and
- the body tail type is well formed independently of window-escape Flow.

An immutable source requested through `ref mut` continues to use the existing mutable-place
diagnostic. The window feature does not create a second mutability system.

## Borrow Flow changes

The existing loan representation can be extended rather than replaced:

```go
type BorrowWindowID uint32

type borrowLoanRecord struct {
    owner   *symbol
    binding *symbol
    window  BorrowWindowID
    place   BorrowPlace
    fact    BorrowFact
}
```

Pending loans and loan keys must carry the same window and place identity. Copies and reassignment
preserve the generation exactly as they preserve it today.

Window evaluation performs four Flow operations:

1. evaluate the complete header without activating plan loans;
2. materialize all aliases from one validated plan;
3. analyze the body with ordinary last-use, branch, loop, call-summary, and tagged-borrow logic; and
4. inspect the body result, outer environment, and every reachable early-exit environment for live
   generations carrying the window ID.

A failed escape check does not invent a shortened origin or silently copy the referent. It is a
borrow error linked to the exact boundary.

Nested windows create distinct IDs. A child overlapping reborrow temporarily excludes parent access
but does not replace the parent's origin or generation. When the child generation ends, the parent
state becomes usable again if no other retained child loan remains.

## Typed HIR boundary

Typed HIR should preserve resolved symbols and projections rather than source alias strings:

```go
type BorrowWindowID uint32

type BorrowWindow struct {
    ID       BorrowWindowID
    Bindings []BorrowBinding
    Body     Block
    Type     Type
}

type BorrowBinding struct {
    Symbol SymbolID
    Mode   BorrowMode
    Place  BorrowPlace
}
```

A split may either use a distinct typed node or two bindings carrying one shared partition proof.
The required facts are exact owner identity, cut operand, range relation, result borrow types, and
source spans.

## MIR and verifier boundary

A plain Borrow Window has no runtime begin or end instruction. MIR retains verification evidence:

```go
type BorrowWindowFact struct {
    ID       BorrowWindowID
    Bindings []BorrowWindowBindingFact
    Begin    hir.Span
    Ends     []BorrowWindowEndFact
}

type BorrowWindowEndFact struct {
    Exit     hir.Span
    ExitKind string
    Target   BlockID
}

type BorrowWindowBindingFact struct {
    Symbol      hir.SymbolID
    Mode        BorrowMode
    Root        hir.SymbolID
    Projections []BorrowProjection
}
```

A fact contains multiple bindings because one plan may borrow several independent roots. One
root field on the window itself would be insufficient.

The verifier must prove at least:

- every alias use is dominated by the validated window begin;
- no alias or derived generation is live after any edge recorded in `Ends`;
- every normal and structured exit closes the window;
- overlapping mutable binding facts cannot coexist in one plan;
- a nested reborrow suspends conflicting parent access until the child ends;
- split aliases use one owner and one cut, cover the exact admitted ranges, and are dominated by the
  matching bounds check; and
- no source-level window form remains for the interpreter or C backend to reinterpret.

Ordinary aliases lower through the existing pointer or borrowed-view transport. `split` additionally
lowers the already verified bounds edge and two pointer-length view constructions.

## Backend and optimization policy

Borrow Windows are zero-cost in the language sense: entering or leaving a plain window adds no
runtime call, allocation, reference count, flag, or cleanup operation. Selector evaluation and
existing bounds checks still execute exactly as required by source semantics.

The C backend must not emit `restrict`, LLVM `noalias`, or equivalent promises merely because a
window was accepted. The current backend safety contract already forbids deriving those attributes
from safe-looking syntax. Any future alias optimization requires a final verified proof, a precisely
bounded scope, and separate differential evidence.

The first implementation should use Borrow Window facts for safety and diagnostics only. Optimization
is not part of the source contract.

## Diagnostics

Existing diagnostics retain their current roles:

- `E0102` for duplicate alias names in one scope;
- `E1008` when an exclusive alias targets a non-mutable place;
- `E2101` for a conflicting access while a validated loan is active; and
- `E2105` when an owner is moved, consumed, dropped, or replaced before its retained loan ends.

The candidate reserves three additional borrowing categories for implementation:

- `E2106`: a window-derived loan generation escapes its maximum boundary;
- `E2107`: bindings in one borrow plan have a proven overlapping shared/mutable or mutable/mutable
  conflict; and
- `E2108`: the compiler cannot prove that requested mutable projections are disjoint.

These codes are not release-stable until the diagnostic catalog, compiler implementation, and
compile-fail fixtures land together. Diagnostics should show the plan bindings, conflicting or
unproven projections, window boundary, escaping destination, and last use where available.

## Delivery slices

### Slice 1: field windows

Admit single and multiple named-field bindings, shared and exclusive modes, value-producing bodies,
last-use shortening, nested reborrows, and complete escape checks through direct aliases,
reassignment, calls, `Option`, and `Result`.

This slice keeps ordinary borrows whole-root and carries window paths through semantic facts and
verified MIR. It requires representative resource, Region, `defer`, return, `?`, break, and continue
exits before promotion.

### Slice 2: indexed byte projections and split

Admit the bounded existing fixed `u8` array and mutable byte-view families, one-evaluation selectors,
constant-index disjointness, and `split(ref mut bytes, cut) => (left, right)`. Dynamic multi-index
plans remain rejected without a separate checked-selection contract.

### Slice 3: optional verified alias optimization

Only after the verifier proves complete begin/end and overlap facts may a separate proposal consider
scoped backend alias attributes or load/store reordering. The source syntax and safety result must
remain identical when every such optimization is disabled.

Each executable slice needs an accepted implementation ADR or an accepted revision of this ADR with
exact grammar, keyword migration, diagnostics, fixtures, cache impact, and same-commit native
evidence.

## Rejected alternatives

- A short alias-only `with ref` sugar: ordinary lexical blocks and last-use analysis already provide
  short loans; it does not justify new syntax.
- Global field-sensitive borrowing: charges every function for precision requested by a minority of
  code and enlarges branch and loop analysis without a source boundary.
- Activating multi-bindings sequentially: makes legality order-dependent and requires partial
  rollback after later failure.
- Making earlier aliases visible in the same header: creates dependency graphs and partially active
  plans; nested windows express the operation explicitly.
- Allowing arbitrary calls in selectors: hides owner mutation and movement during place
  construction.
- Permitting window-derived safe-borrow results when the underlying origin could otherwise escape:
  destroys the maximum-boundary guarantee and makes projection precision non-local.
- Rejecting every assignment of a window alias: unnecessarily forbids temporary propagation that
  Flow can prove dead before exit.
- Treating `Option` or `Result` as an escape barrier: the existing single-borrow-leaf proof already
  tracks the retained generation through those containers.
- Runtime `distinct` acquisition in the first slice: introduces fallible borrow acquisition,
  overlap-error design, and a second failure category beyond bounds.
- Arbitrary range partitioning or user-defined splitters: requires a larger proof protocol than the
  exact two-way byte split.
- Emitting `restrict` or `noalias` immediately: turns a local safety feature into an unproven backend
  promise.
- Desugaring before CST or AST: loses source roles required for diagnostics, formatting, exact scope,
  and verifier evidence.

## Admission evidence

Promotion from `Candidate` requires at least one complete vertical slice with all of the following:

- normative grammar and language documentation;
- exact current- and previous-revision behavior for `with`, `split`, and any other newly reserved
  spelling;
- lossless CST reconstruction and bounded malformed-header recovery;
- parser tests for single, multiple, expression, nested, and split forms;
- resolution tests proving simultaneous alias introduction and header invisibility;
- typed AST and HIR tests for exact spans, modes, symbols, roots, and projections;
- overlap fixtures for different fields, root-versus-field, shared overlap, constant indices, equal
  indices, and unproven dynamic indices;
- Borrow Flow fixtures for early last use, nested reborrow, direct escape, overwritten escape,
  borrow-returning calls, and `Option` or `Result` propagation;
- Resource, Result, Region, must-use, and lexical-`defer` fixtures for normal and early exits;
- verified MIR fixtures for begin/end facts, every exit edge, nested suspension, and split partition;
- interpreter and constrained-C execution proving selector single evaluation, bounds traps, byte
  split behavior, and unchanged results with optimization disabled;
- verifier mutation fixtures for use-after-window, missing end edges, overlapping mutable facts, and
  malformed split proofs;
- diagnostic fixtures for `E2106`, `E2107`, and `E2108` with role-labelled spans;
- compatibility-vector, language-pin, cache-key, starter, and migrator updates when syntax is
  actually reserved;
- no generated `restrict`, `noalias`, runtime loan flag, rollback flag, or reference-counting state;
  and
- bounded performance evidence showing that projection analysis is paid only inside declared
  windows.

## Consequences

Borrow Windows provide one coherent feature for short aliases, explicit maximum lifetimes,
field-disjoint access, block-bounded reborrows, value-producing bodies, and exact mutable byte
splitting.

The compiler retains the simple whole-root model for ordinary code. More precise analysis is local,
source-requested, and verifier-visible. Window-derived borrows cannot leak precision or lifetime
assumptions into surrounding code, and the first implementation creates no runtime management
mechanism or backend alias promise.

No current source program, grammar, language pin, cache identity, diagnostic catalog, HIR, MIR,
interpreter, backend, or target maturity changes under this candidate ADR alone.
