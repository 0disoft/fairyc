# ADR 0332: Access-typed range `for`

- Status: Candidate
- Date: 2026-08-17
- Product source: `docs/product/02-spec.md`
- Grammar source: `spec/grammar/stage0.ebnf`
- Safety source: `docs/adr/0003-stage-0-safety-boundary.md`
- Flow sources: ADR 0010, ADR 0011, ADR 0012, ADR 0024, ADR 0217
- Array and view sources: ADR 0097, ADR 0126, ADR 0135
- Surface source: ADR 0136
- Classic-loop source: ADR 0213
- Admission source: ADR 0218
- Depends on: ADR 0003, ADR 0024, ADR 0097, ADR 0126, ADR 0136, ADR 0213, ADR 0217, ADR 0218

## Context

A range loop that only abbreviates repeated indexing would add familiar syntax without defining the
ownership, aliasing, cleanup, or bounds proof needed by Fairy C. The useful contract is deeper. One
header must state how the element may be accessed, how long the source remains leased, how the
current index is proven valid, and who owns cleanup when control leaves before the final element.

Fairy C already has an accepted classic C-shaped `for` with separately preserved initializer,
condition, post, and body regions. Its header subset is deliberately scalar and excludes calls,
borrows, indexing, resource movement, and other ownership-relevant behavior. Reusing that
`ForLoop` representation for range iteration would force the element binding into `Initializer`,
the source into `Condition`, and the hidden cursor into `Post`, losing the source concepts needed by
Flow, diagnostics, MIR verification, and tools.

The current borrow contract also treats ordinary borrows conservatively. A source range may need one
loop-wide stable lease plus one short element reborrow per iteration. Those two levels must not be
collapsed into a new independent borrow on every back edge, and an element borrow must not silently
survive into the next iteration.

Resource-bearing iteration adds another boundary. Moving elements one by one must not expose a
user-visible partly moved collection. Early `break`, `continue`, `return`, or Result propagation must
clean the current element, the unvisited owned elements, and container storage in one specified
order.

Finally, the feature should improve generated code rather than merely hide a manual loop. A range
loop can serve as a verified bounds-proof boundary when the compiler retains the source, length,
cursor, stable lease, and dominating condition through MIR verification.

This ADR records the complete candidate model. It does not reserve new syntax, change the language
pin, add a public iterator protocol, or claim implementation.

## Candidate decision

Fairy C should admit **Access Typed Range `for`** as a structured iteration contract whose element
access mode is mandatory in source.

The four source forms are:

```fy
for (copy byte in bytes) {
    checksum ^= byte;
}
```

```fy
for (ref user in users) {
    render(user);
}
```

```fy
for (ref mut pixel in pixels) {
    pixel.alpha = 255;
}
```

```fy
for (move job in jobs) {
    execute(job)?;
}
```

An optional logical iteration index is declared separately:

```fy
for (ref user, index i in users) {
    print_user(i, user)?;
}
```

The header therefore declares an access mode, an element binding, an optional `usize` ordinal, and
one range source evaluated exactly once.

A mode-less form is rejected:

```fy
for (item in items) {
    use(item);
}
```

The compiler must not infer whether this copies, borrows, mutably borrows, or consumes. The required
mode is part of the source contract and remains visible during review.

## Access modes

### `copy`

`copy` reads one element value into an immutable local binding.

```fy
for (copy value in values) {
    consume_copy(value);
}
```

The binding type is the range element type `T`. The mode is legal only when `T` belongs to the
admitted copy set. A resource element is rejected:

```fy
for (copy file in files) {
    inspect(file);
}
```

A diagnostic should identify the non-copy element type and recommend `ref` or, for an owned range,
`move`.

Although the element is copied, the source retains a stable shared range lease for the whole loop.
That lease prevents structural mutation that could change length or backing storage:

```fy
for (copy value in values) {
    values.append(3);
}
```

The element binding itself remains immutable. Code needing a mutable local copy creates an ordinary
`var`:

```fy
for (copy value in values) {
    var i32 adjusted = value;
    adjusted += 1;
    consume(adjusted);
}
```

Access mode and local rebinding policy remain separate concepts.

### `ref`

`ref` creates one shared lease for the source and one shared element reborrow per iteration.

```fy
for (ref user in users) {
    inspect(user);
}
```

The element binding type is `ref T`. The loop-wide lease stabilizes the source. The iteration loan
normally ends at its last use before the advance edge, and it may never pass through fallthrough or
`continue` into the next element.

Conceptually:

```text
shared range lease
    iteration 0 element reborrow
    iteration 1 element reborrow
    iteration 2 element reborrow
range lease release
```

Compatible shared observations of the source remain legal when they do not structurally mutate it:

```fy
for (ref user, index i in users) {
    let usize total = users.len;
    print_position(i, total, user)?;
}
```

A mutable borrow or structural mutation of the source conflicts with the active shared range lease.

### `ref mut`

`ref mut` creates one exclusive range lease and one exclusive element reborrow per iteration.

```fy
for (ref mut pixel in pixels) {
    pixel.red = clamp(pixel.red);
    pixel.green = clamp(pixel.green);
    pixel.blue = clamp(pixel.blue);
}
```

The element binding type is `ref mut T`. The alias name is immutable; only the referent is mutable.
The previous element loan must be dead before the advance edge enters the next element.

Structural mutation is rejected:

```fy
for (ref mut item in items) {
    items.append(create_item());
}
```

A second mutable access through the source path is also rejected:

```fy
for (ref mut item, index i in items) {
    items[i + 1].enabled = false;
}
```

Algorithms requiring two adjacent mutable elements must use a separately verified non-overlapping
view such as a future `windows_mut(2)` or `split_at_mut` result. The range loop executes a proven
view; it does not guess whether arbitrary dynamic projections overlap.

### `move`

`move` consumes an owned range source and transfers each element to the loop body.

```fy
for (move job in jobs) {
    execute(job)?;
}
```

At loop entry, the complete source moves into one compiler-owned range owner. The source name is dead
after that move, so it remains unusable after the loop.

The user-visible collection never enters a partly moved state. The hidden owner tracks the current
cursor and all remaining cleanup obligations:

```text
move the complete source into the range owner
move the current element into the iteration binding
execute the body
clean the current element if it remains live
advance to the next element
clean range storage after all elements are discharged
```

Passing the element by value transfers ownership normally:

```fy
for (move job in jobs) {
    scheduler.submit(job)?;
}
```

If the body leaves the element live, the element is dropped at the iteration boundary.

A borrowed range cannot use `move`:

```fy
fn void process(ref Job[] jobs) {
    for (move job in jobs) {
        execute(job)?;
    }
}
```

The source must own its elements. Emptying a collection while retaining the collection itself is a
separate operation exposed through an owned drain view:

```fy
for (move job in jobs.drain()) {
    execute(job)?;
}

assert(jobs.is_empty);
```

`move in jobs` consumes the source. `move in jobs.drain()` consumes the drain object. The distinction
must remain explicit in source and types.

The complete `move` contract belongs to this design, but its first executable implementation is
deferred until Fairy C has an admitted owned sequence with per-element cleanup evidence.

## Source evaluation and materialization

The range source is evaluated exactly once before loop entry.

```fy
for (ref item in select_bucket(key)) {
    process(item);
}
```

`select_bucket(key)` runs once, not once per element. The conceptual order is:

```text
evaluate the source expression
materialize an owner or range descriptor when required
resolve the stable range capability
read the logical length once
acquire the range lease
enter the loop header
```

Calls, propagation, and other ordinary expression behavior may occur in the source expression when
the resulting value has an admitted stable range capability. This differs intentionally from the
closed scalar header of classic `for` and requires a dedicated checker.

A temporary owner may be retained until range close only when the compiler can prove the view's
owner relation. This is loop-bounded materialization for exact single evaluation, not a general
hidden lifetime-extension rule. A borrowed element still cannot outlive that temporary owner.

Raw or foreign-derived views without a proven safe owner remain subject to the existing unsafe and
borrow-origin rules.

## Stable length and backing storage

Range `for` has a stable-length contract. The logical length is read once at loop entry and remains
valid until range close.

For a direct range, the cursor visits logical ordinals from zero through `length - 1`. Dynamic
containers may participate only when their lease prevents reallocation, length changes, and backing
storage replacement during the loop.

The optional `index` binding is always an immutable `usize` logical visitation ordinal:

```fy
for (ref item, index i in items[10..20]) {
    inspect(i, item);
}
```

Here `i` ranges from zero through nine. It is not the original collection index. Future reverse,
chunk, window, and zip views retain the same rule: `index` means visit order, independent of the
view's physical access plan.

The cursor increment is defined only on the body-success and `continue` paths. Because the advance
edge executes only while `cursor < length`, incrementing the final valid cursor cannot overflow
`usize`: when `length` is the maximum representable value, the final cursor is at most one less than
that maximum and advances exactly to `length`.

## Iteration loans and the back edge

A `ref` or `ref mut` element loan may not survive through the range back edge.

```fy
var ref User saved = fallback;

for (ref user in users) {
    saved = user;
}
```

If `saved` remains live at fallthrough or `continue`, the compiler rejects the loop because the next
iteration would create another element loan while the prior generation still exists.

The rule applies through aliases and the existing single-safe-borrow leaf inside `Option` or
`Result`. It is a Flow property, not a syntactic ban on assignment.

A loop-exiting path is different. An element borrow may be transferred through `return` or another
structured outward exit only when all of the following hold:

- the transfer does not reach the range advance edge;
- the range source has one safe origin that outlives the destination;
- the source owner is not the loop-bounded temporary materialized for range entry;
- the transferred borrow retains an ordinary loan on the source after the range lease is discharged;
- mutable transfers keep the source exclusively borrowed until the escaped borrow's last use.

Outside the range loop, the first implementation may conservatively collapse the exact element
projection to the existing whole-root source loan. That preserves current ordinary borrow precision
while allowing a proven element borrow to escape a completed loop safely.

A future structured-control implementation may route a value-bearing exit from the loop into a named
outer block. That integration must use the same resolved target and cleanup path; it is not required
for the first range-loop slice.

## `continue`, `break`, and cleanup

Range `continue` targets a compiler-owned advance block, analogous to the post block of classic
`for`.

The normal iteration order is:

```text
header condition
create the element binding
execute the body
run body-scope cleanup
end the element loan or discharge the moved element
advance the cursor
return to the header
```

`continue` first exits the complete iteration scope. It runs lexical defers, drops body-local
resources, discharges the current moved element when still live, ends the element loan, and only then
advances the cursor.

`break` performs iteration cleanup and transfers to range close without advancing.

`return` and Result propagation perform iteration cleanup, range cleanup, and every outer lexical
cleanup required by the existing exit contract before leaving the function.

For an owned `move` range, an early exit has the fixed order:

```text
evaluate the exit value
run body defers in reverse order
drop body-local resources in reverse declaration order
drop the current element if it remains live
drop unvisited owned elements in reverse source order
drop range-container storage
run outer lexical cleanup
perform the control transfer
```

Reverse cleanup of unvisited elements matches ordinary aggregate teardown and avoids making cleanup
order depend on the cursor direction of a future view.

A trap follows the existing Fairy C rule: it does not perform language-level unwind cleanup.

## Stable range capabilities

Range `for` does not accept an arbitrary user-defined `next()` protocol.

An unconstrained iterator can hide a call per element, allocation, locking, I/O, fallible advance,
dynamic length, mutation during traversal, or cleanup that is not visible in the loop header. That
model conflicts with Fairy C's bounded execution and explicit ownership goals.

A range source must instead provide a compiler-known or later separately admitted stable range
capability containing at least:

```text
one stable logical length
one pure logical-index-to-element access plan
the supported access-mode set
the source owner and borrow origin
the lease kind required by each mode
the close and remaining-element cleanup plan
```

The initial source families are deliberately closed:

```text
T[N]
ref T[]
ref mut T[]
compiler-known owned sequence
compiler-known stable range view
compiler-known owned drain view
```

Only source families admitted by an implementation slice are executable. Listing the full family in
this candidate does not claim that owned sequences, drain views, or transformed views already exist.

A future user-defined range must expose a sealed, verifier-readable descriptor or equivalent
compiler facts. Implementing an arbitrary method named `next` is never sufficient.

## Views remain separate from loop syntax

Traversal shape is expressed by range-producing views, not by adding clauses to `for`.

Future examples include:

```fy
for (ref item in items.reverse()) {
    inspect(item);
}
```

```fy
for (ref chunk in bytes.chunks(64)) {
    consume(chunk);
}
```

```fy
for (ref window in samples.windows(4)) {
    analyze(window);
}
```

```fy
for (copy pair in zip_exact(left, right)?) {
    consume(pair);
}
```

`zip_exact` and `zip_shortest` remain distinct because mismatched lengths must not acquire an implicit
policy from the loop syntax.

Each view owns its stable length, physical access plan, supported modes, source origin, and cleanup
contract. Range `for` executes that plan and supplies the verified iteration structure.

## Contextual keywords and parsing

`copy`, `move`, and `index` are contextual in a range header. They remain ordinary identifiers in
other source positions:

```fy
let i32 copy = 3;
let i32 move = 4;
let i32 index = 5;
```

`ref`, `mut`, and `in` already participate in Fairy C syntax. No new globally reserved keyword is
required by the candidate.

Classic and range forms are distinguished by bounded delimiter-aware lookahead inside the top-level
`for (...)` header.

A classic header has exactly two top-level semicolons:

```fy
for (var usize i = 0; i < count; i += 1) {
    consume(i);
}
```

A range header has one top-level `in` separator and no top-level semicolon:

```fy
for (ref item in items) {
    consume(item);
}
```

Lookahead must track nested parentheses, brackets, and braces so delimiters inside a source
expression do not misclassify the header. Malformed or ambiguous headers produce bounded syntax
recovery rather than falling from one grammar into the other after arbitrary consumption.

## CST and AST boundaries

Range iteration receives its own lossless CST kind:

```go
const KindRangeFor Kind = "range-for"
```

The node preserves the mode, element binding, optional index binding, source expression, and body as
separate source regions. It must not be represented as a malformed classic `KindFor`.

The AST should contain an explicit shape similar to:

```go
type RangeMode uint8

const (
    RangeCopy RangeMode = iota
    RangeRef
    RangeRefMut
    RangeMove
)

type RangeFor struct {
    Span        diagnostic.Span
    Mode        RangeMode
    ElementName string
    ElementSpan diagnostic.Span
    IndexName   string
    IndexSpan   diagnostic.Span
    Source      *Expression
    Body        *Block
}
```

`ast.Statement` should retain separate classic and range fields. Immediate desugaring to classic
`ForLoop` is rejected because it erases the access mode, range source identity, lease, and bounds
proof before ownership-sensitive analysis.

Element and index bindings enter scope only for the body. The source expression resolves in the
outer scope and cannot refer to either header binding.

## Type checking and HIR

The type checker needs a dedicated `checkRangeFor` path. It evaluates and types the source once,
selects one admitted range capability, checks mode compatibility, creates the element and optional
index symbols, registers the range lease, and checks the body under the iteration-loan rules.

Typed HIR should preserve a shape similar to:

```go
type RangeLeaseID uint32

type RangeFor struct {
    Span        Span
    Mode        RangeMode
    Source      Expression
    SourceType  Type
    ElementType Type
    Element     SymbolID
    Index       SymbolID
    Lease       RangeLeaseID
    Body        Block
}
```

The exact Go layout is not fixed by this candidate. The required contract is that HIR retains the
source expression once, the selected capability, mode, canonical element type, symbols, lease
identity, source origin, and body.

Mode checks include:

- `copy` requires an admitted copy element;
- `ref` requires a shared or exclusive source capability that can yield a shared element reborrow;
- `ref mut` requires one mutable stable source place or exclusive view capability;
- `move` requires an owned source with an explicit remaining-element cleanup plan;
- `index` always has canonical type `usize`;
- source and body effects continue to obey the enclosing function's existing effect contract.

A range source may be effectful to evaluate once. Per-element access itself must follow the pure
range access plan and cannot conceal an arbitrary call.

## Flow-analysis requirements

Borrow Flow, Resource Flow, Result Flow, must-use analysis, backward liveness, Region handling, and
lexical defer must agree on the same range frame.

A conceptual active frame contains at least:

```go
type activeRange struct {
    lease         hir.RangeLeaseID
    mode          hir.RangeMode
    source        hir.SymbolID
    element       hir.SymbolID
    index         hir.SymbolID
    body          hir.Span
    advanceTarget ControlTarget
    exitTarget    ControlTarget
}
```

The actual semantic and MIR representations may differ. The required behaviors are:

- source evaluation occurs once before the frame becomes active;
- the loop-wide lease survives every ordinary back edge;
- each element loan is a distinct generation bounded to one iteration;
- fallthrough and `continue` require every element-derived loan to be dead;
- `break`, `return`, and propagation route through exact iteration and range cleanup;
- an escaping element borrow retains the source origin and a post-loop whole-root loan;
- `move` tracks the current element and all unvisited cleanup obligations without exposing partial
  source state;
- zero-iteration and all loop-exit states are included in joins;
- an invalid ownership, borrow, Result, or Region state fails before MIR.

Modeling a range loop as a scalar `while` is insufficient once element loans or move cleanup are
present. Shared fixed-point utilities may be reused, but the range frame and its exit edges must
remain explicit.

## MIR lowering

Range lowering can reuse the verified CFG skeleton of classic `for` while preserving a distinct
advance block:

```text
open range
read stable length
initialize cursor to zero

goto header

header:
    branch cursor < length to body or close

body:
    create verified element value or reborrow
    execute body
    cleanup iteration scope
    goto advance

advance:
    end element generation
    increment cursor
    goto header

close:
    release lease or discharge owned range
    continue after loop
```

The source-level form is eliminated before the backend, but MIR retains verifier-readable range
operations or equivalent facts. A minimal conceptual set is:

```text
range-open
range-element-copy
range-element-ref
range-element-ref-mut
range-element-move
range-close
```

The operations need not become a public MIR serialization contract under this candidate. They must
remain distinct long enough for the verifier to validate bounds, lease, ownership, cleanup, and
origin facts.

## Verified bounds proof

A range element access is not an ordinary unchecked index. The MIR verifier admits it only when all
of the following hold:

- one `range-open` dominates the element operation;
- the current path is dominated by the true edge of `cursor < length`;
- `length` was read from the same opened range source;
- cursor and length use canonical `usize` semantics;
- no operation invalidates source length or backing storage while the lease is active;
- the access mode matches the opened lease and source capability;
- the element generation is closed before every advance edge;
- every normal exit reaches the matching `range-close` or a verified early-exit cleanup path;
- a move range accounts for the current and every unvisited owned element exactly once.

Mutation fixtures must independently break each invariant and require verifier rejection.

Once verified, the interpreter and C backend receive ordinary CFG plus a proven element access. They
do not need to emit one source-visible bounds trap per element. The generated C may use a plain
pointer or array loop after preserving the verified source and length:

```c
uintptr_t length = range.length;
for (uintptr_t index = 0; index < length; index++) {
    const item_type *item = &range.data[index];
    /* lowered body */
}
```

This is an illustration, not a frozen emitted spelling or public layout.

The initial C backend must not emit `restrict`, LLVM `noalias`, or stronger alias metadata solely
from the range syntax. Such promises require proof from the final borrow analysis and target
lowering. A verified range may guide local compiler decisions without creating a public or foreign
alias contract.

## Diagnostics

The first implementation should reserve stable categories for at least:

- `E1014`: the range header has no access mode or has a malformed mode and binding shape;
- `E1015`: the source does not provide the stable capability required by the selected mode;
- `E2109`: an element-derived loan reaches range fallthrough or `continue` and therefore crosses the
  back edge.

Existing diagnostics remain authoritative where they already fit:

- `E1008` for `ref mut` targeting a non-mutable place;
- `E1012` for loop control outside an enclosing loop;
- `E1101` for use of a source consumed by `move`;
- `E1102` for duplicate cleanup obligations;
- `E1105` for an invalid ownership back edge;
- `E2101` for conflicting shared or mutable access during a lease;
- `E2105` when an escaped element borrow outlives or moves its source owner.

Diagnostics should relate the range declaration, source owner, element binding, conflicting access,
loan creation, last use, and back edge where applicable. Missing-mode diagnostics should list the
four source modes without inferring one.

## Initial executable slice

The first implementation should be narrow and executable.

It admits:

```text
copy, ref, and ref mut over admitted scalar T[N]
copy and ref over ref u8[]
copy, ref, and ref mut over ref mut u8[]
optional index binding for every admitted source
```

The fixed-array element set initially matches the existing executable copy-scalar family:

```text
bool
u8
i32
u32
usize
```

`move` is parsed and represented by the complete candidate design but remains unsupported for these
sources until an owned sequence and exact remaining-element cleanup contract are admitted.

The first implementation does not include arbitrary user ranges, reverse/chunk/window/zip views,
drains, resource elements, dynamic structural mutation, value-producing loops, async streams, or
concurrency.

The highest-value executable evidence is concentrated on these contracts:

- source expressions evaluate exactly once;
- stable length is read once and source mutation is rejected during the lease;
- `continue` closes body cleanup and the element generation before advance;
- a borrow retained through the back edge is rejected;
- an outward borrow transfer keeps the source origin and cannot outlive a temporary source owner;
- `ref mut` prevents a second source-path mutable access;
- the verifier rejects a range element without the matching dominating bounds proof;
- `break`, `return`, and propagation execute required body and range cleanup;
- interpreter and constrained-C execution agree on visited values, index ordinals, and traps outside
  the verified element operation.

A small number of cross-layer fixtures proving those invariants is more valuable than a large matrix
of superficial header spellings.

## Delivery slices

### Slice 1: distinct syntax and copy execution

Add bounded classic/range header disambiguation, contextual modes, `KindRangeFor`, AST and HIR range
nodes, exact source evaluation, stable length, optional index, and executable `copy` over fixed arrays
and byte views.

### Slice 2: shared and exclusive element reborrows

Add loop-wide shared and exclusive leases, iteration loan generations, back-edge rejection, source
mutation conflicts, `ref`, `ref mut`, interpreter execution, and constrained-C transport.

### Slice 3: outward borrow transfer

Preserve source origin through element reborrows, allow only loop-exiting transfers whose source
outlives the destination, collapse post-loop precision to an ordinary whole-root loan, and add
integration with any accepted target-aware structured-control contract.

### Slice 4: owned `move` ranges

Admit one owned sequence and drain view, hidden range ownership, per-element movement, current and
remaining-element cleanup, early exits, and exact once-only drop verification.

### Slice 5: stable transformed views

Admit separately proven reverse, chunk, window, and zip descriptors without adding new loop clauses
or opening an arbitrary iterator protocol.

Each executable slice needs an accepted implementation ADR or accepted revision of this ADR naming
exact grammar, diagnostics, compatibility changes, fixtures, cache impact, MIR invariants, and
same-commit native evidence.

## Rejected alternatives

- Inferring `copy`, `ref`, `ref mut`, or `move` from the element type: hides ownership and mutation
  rights from source review.
- Reusing classic `ForLoop` fields for a range loop: loses mode, source, lease, and bounds identity.
- Immediate AST desugaring into classic `for` or `while`: erases the facts needed by Flow and MIR
  verification.
- Reborrowing the whole source independently on every iteration: obscures one stable loop-wide lease
  and complicates back-edge reasoning.
- Allowing structural mutation while iterating: invalidates stable length and backing storage.
- Allowing an element borrow through fallthrough or `continue`: permits overlapping iteration
  generations.
- Exposing a partly moved source collection: makes cleanup path-dependent in user-visible state.
- Allowing `move` from `ref T[]` or `ref mut T[]`: consumes elements the source does not own.
- Using arbitrary `next()` objects: hides cost, failure, mutation, and cleanup behind a protocol call.
- Adding reverse, stride, filter, chunk, or zip clauses to `for`: turns the header into a query
  language instead of composing explicit views.
- Treating range success as an unsafe unchecked-index assumption: makes safety depend on optimizer
  behavior rather than verified MIR.
- Emitting `restrict` or `noalias` directly from source syntax: can exceed the final proven alias
  contract.
- Implementing resource-bearing `move` before owned-sequence evidence: pays the hardest cleanup cost
  before a real executable source family exists.

## Admission evidence

Promotion from `Candidate` requires at least one complete vertical slice with all of the following:

- normative grammar and language documentation;
- bounded classic/range lookahead and malformed-header recovery;
- exact current- and previous-revision compatibility behavior;
- lossless CST reconstruction with contextual-keyword tests;
- AST and HIR tests for separate mode, source, element, index, and lease identities;
- semantic fixtures for missing mode, incompatible source capability, mutation conflicts, and
  back-edge loan escape;
- Borrow, Resource, Result, must-use, Region, and defer fixtures for normal and early exits;
- verified MIR fixtures for open, bounds domination, element access, advance, close, and cleanup;
- verifier mutation fixtures for wrong source, wrong length, missing lease, stale lease, missing
  element close, and duplicate move cleanup;
- interpreter and constrained-C execution for every admitted mode and source family;
- exact single-evaluation observations for effectful source expressions;
- differential evidence that verified range elements introduce no per-element source bounds trap;
- proof that the backend emits no `restrict` or `noalias` promise in the initial slice;
- compatibility-vector, language-pin, cache-key, starter, and migration updates when syntax is
  actually admitted;
- no public iterator, range ABI, descriptor layout, or source-level MIR operation implied by the
  implementation.

## Compatibility

This candidate changes no current grammar or compatibility vector. When an executable range form is
accepted, the normative grammar and exact language pin must advance even though `copy`, `move`, and
`index` remain contextual rather than globally reserved.

The previous-revision parser and migrator must continue to distinguish historical classic `for`
input. A malformed current range header must not be reinterpreted as valid historical syntax and
written back with changed semantics.

Range capability identities and internal lease IDs are compiler-private. They are not source type
spellings, public ABI names, package-interface identities, or serialized cache keys unless a later
contract explicitly promotes a versioned representation.

## Consequences

Access Typed Range `for` gives one compact header a precise semantic payload. `copy` creates an
immutable element copy under a stable shared lease. `ref` creates a shared element reborrow. `ref mut`
creates an exclusive element reborrow under one exclusive range lease. `move` consumes an owned
source and owns every remaining cleanup obligation.

The design keeps source evaluation single, length stable, iteration loans off the back edge, cleanup
structured, and bounds elimination verifier-backed. It also keeps arbitrary iterators and query-like
loop clauses out of the language.

No current source program, grammar, language pin, cache identity, MIR contract, backend, interpreter,
standard library, or target maturity changes under this candidate ADR alone.
