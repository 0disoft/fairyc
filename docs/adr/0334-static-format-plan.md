# ADR 0334: Static Format Plans

- Status: Candidate
- Date: 2026-08-18
- Product source: `docs/product/02-spec.md`
- Grammar source: `spec/grammar/stage0.ebnf`
- Parser source: ADR 0009
- Safety source: ADR 0003
- Output sources: ADR 0215, ADR 0258, ADR 0262, ADR 0303
- Buffer and text sources: ADR 0246, ADR 0247, ADR 0263, ADR 0318
- Effect source: ADR 0116
- Compile-time source: ADR 0220
- Admission source: ADR 0218
- Depends on: ADR 0003, ADR 0009, ADR 0116, ADR 0215, ADR 0218, ADR 0220, ADR 0246, ADR 0247, ADR 0258, ADR 0262, ADR 0263, ADR 0303, ADR 0318

## Context

Fairy C already owns exact allocation-free output primitives for `ref str`, raw byte views, `i32`,
`u32`, `usize`, and `bool` on stdout and stderr. Those primitives append no newline, use canonical
locale-independent text, and retain the exact byte prefix completed before a console failure.
`std::bytes::Buffer` separately owns capacity and initialized length, while `std::text::String` can
exchange storage with Buffer through explicit ownership-preserving operations.

A format literal that immediately desugars into several ordinary output calls would reuse those
pieces, but it would leave the important contract undefined. Later hole expressions could fail or
trap after an earlier fragment has already been printed. The compiler could not reason about total
length before a fixed-buffer or Buffer write. A backend would lose the opportunity to coalesce a
small plan, use borrowed text without copying, or choose a target-specific segmented write.

Turning interpolation into a String first is also the wrong default. It would allocate even for
console output, hide the exact ownership cost, and erase the distinction between a formatting plan
and materialized UTF-8 storage. A runtime formatter or `printf` transport would add dynamic format
parsing, type tags or C varargs, locale state, and a second source of evaluation-order semantics.

The missing concept is a compiler-owned output plan. Source should state typed fragments once. The
compiler should retain the literal bytes, typed expressions, encoders, length bounds, evaluation
order, and chosen destination until verified lowering can select the cheapest valid execution.

This ADR records that complete candidate model under the name **Static Format Plan**. It does not
reserve syntax, add public SDK operations, change the language pin, change MIR, or claim an
implementation.

## Candidate decision

Fairy C should admit Static Format Plans with the compact source prefix `fmt`:

```fy
let usize written = std::io::write_fmt(
    fmt"user={user_id} score={score:+d} active={active}\n"
)?;
```

The plan is not a `str`, `String`, Buffer, aggregate, closure, or runtime descriptor. It produces no
bytes merely by being parsed or evaluated. Conceptually the compiler retains:

```text
literal "user="
encode user_id as canonical decimal
literal " score="
encode score as signed decimal with an explicit sign
literal " active="
encode active as true or false
literal newline
```

The defining contract is:

> Evaluate every typed hole once from left to right, prove the complete output geometry before the
> first destination mutation, then execute one verified sink-specific byte plan without runtime
> format parsing or implicit allocation.

A Static Format Plan is compiler-owned and non-first-class. It survives through typed HIR and
verified MIR only long enough to select and prove one concrete sink execution.

## Existing output operations remain unchanged

The accepted operations retain their exact behavior:

```fy
std::io::write(text)
std::io::write_bytes(bytes)
std::io::write_i32(value)
std::io::write_u32(value)
std::io::write_usize(value)
std::io::write_bool(value)
```

The stderr family remains equally unchanged. No overload resolution is added to `write`. A plain
`write` call continues to accept exact text only and reveals no formatting behavior.

Format sinks use distinct names:

```fy
std::io::write_fmt(fmt"value={value}\n")
std::io::write_stderr_fmt(fmt"error={code}\n")
```

Those names make compiler-owned plan execution visible in source and avoid adding a source-spellable
universal formatting type to ordinary call resolution.

No format sink appends a newline. Every emitted byte remains visible in the format literal or in an
explicit nested template.

## Contextual `fmt` syntax

`fmt` is contextual only when immediately followed by a double quote in an expression position:

```fy
fmt"value={value}"
```

Whitespace between `fmt` and the quote is not accepted as a format opener. Outside this adjacent
form, `fmt` remains an ordinary identifier. The candidate therefore does not reserve `fmt` across
the language.

The format body uses the existing UTF-8 string escape rules for literal text. Literal braces use
doubling:

```fy
fmt"object={{id={id}}}"
```

`{{` contributes one literal `{` byte and `}}` contributes one literal `}` byte. A lone unmatched
brace is a format diagnostic, not an ordinary string byte.

## Holes contain ordinary Fairy C expressions

Each `{...}` hole contains one ordinary Fairy C expression:

```fy
fmt"user={user.id}"
fmt"next={values[index + 1]}"
fmt"score={calculate_score(ref user)}"
fmt"value={load_value()?}"
```

The hole parser reuses the ordinary expression grammar, name resolution, typing, effects, Result
propagation, borrow checking, and source spans. It does not introduce a second expression language.

The first executable grammar excludes expression forms that require an unescaped top-level `{` or
`}` inside the hole, including blocks and record literals. Such work remains explicit before the
plan:

```fy
let UserSummary summary = calculate_summary(ref user)?;

std::io::write_fmt(
    fmt"id={summary.id} score={summary.score}\n"
)?;
```

Parentheses, calls, paths, fields, indexing, slices, prefix and binary operators, casts, and Result
propagation retain their ordinary meanings. Delimiter-aware parsing must distinguish nested `()` and
`[]` from the top-level `:` that begins a format spec and the `}` that ends the hole.

## Every hole evaluates before output begins

All hole expressions evaluate exactly once from left to right. Every successful result is retained in
a hidden typed local until the complete plan finishes. Only after all hole expressions have
completed does the compiler calculate exact fragment lengths and begin destination execution.

```fy
std::io::write_fmt(
    fmt"a={load_a()?} b={load_b()?}\n"
)?;
```

The semantic order is:

```text
evaluate load_a once
evaluate load_b once
retain both typed results
calculate every exact encoded length with checked usize arithmetic
prepare the selected destination
begin writing bytes
```

If `load_b()` propagates an error, no part of `a=` reaches the sink. If a later hole traps, no earlier
literal or value has been written. This matches the ordinary expectation that call arguments finish
before the call body begins.

A preflight length overflow is a recoverable format failure with zero destination progress. It is not
an arithmetic trap and it starts no console write, fixed-buffer mutation, Buffer reserve, or String
allocation.

## Hidden values obey ordinary borrow and ownership rules

A retained hole result remains live until its last fragment use. The compiler must not shorten it
before plan execution merely because source evaluation has finished.

```fy
std::io::write_fmt(
    fmt"before={buffer_view(ref buffer)} after={mutate(ref mut buffer)}"
)?;
```

If the first hole returns a borrow that remains needed by the plan, the second mutable access
conflicts with that active loan. Existing Borrow Flow rejects the source. Static Format Plans do not
copy borrowed text or invent a hidden owner to bypass the conflict.

The format plan never performs an implicit resource move. Unsupported values are rejected before
MIR rather than consumed, dropped, reflected, or dynamically dispatched.

## Closed canonical value set

The first complete type vocabulary is:

```text
bool
u8
i32
u32
usize
ref str
```

Copy scalars are read by value. `ref str` contributes its exact retained UTF-8 bytes and preserves
its ordinary borrow origin. The default encoder set is closed:

```text
bool    ASCII true or false
u8      unsigned base-10 ASCII
 i32    signed base-10 ASCII
u32     unsigned base-10 ASCII
usize   unsigned base-10 ASCII
ref str original UTF-8 bytes
```

The leading space before `i32` in the table is presentation only; it is not output syntax.
Implementations must use the canonical spelling without locale, grouping, or target-specific text.

Resource values, user structs, user enums, `Option`, `Result`, raw pointers, exact `c::*` scalars,
foreign aggregates, Regions, function values, and arbitrary safe borrows have no implicit format.

```fy
fmt"value={result}"
```

When `result` is a `Result`, compilation fails. Source must handle the Result and explicitly select a
supported payload. This prevents output from silently choosing an error policy or consuming a
resource-bearing variant.

Raw pointers are also excluded. Automatic address output creates accidental ASLR and sensitive
layout disclosure. A future explicit unsafe address formatter may produce a supported scalar or
nested plan, but raw pointers never gain a default encoder.

## One locale-independent canonical form

Default formatting produces the same bytes on every target and build profile. It does not consult a
locale, terminal, environment variable, process encoding, or host C library formatter.

Thousands separators, localized decimal points, date formatting, terminal colors, Unicode display
width, and implicit labels remain outside the default contract. The source must invoke a separately
specified operation or template when it needs those policies.

## Closed format-spec grammar

The candidate supports a closed static format spec after one top-level colon:

```fy
fmt"id={id:d}"
fmt"id={id:08x}"
fmt"mask={mask:#010x}"
fmt"bits={flags:016b}"
fmt"delta={delta:+d}"
fmt"name={name:>16}"
```

The numeric kinds are:

```text
d decimal
x lowercase hexadecimal
X uppercase hexadecimal
b binary
o octal
```

The admitted flags are:

```text
+ explicit positive sign for i32
# canonical radix prefix for x, X, b, or o
0 zero padding for a numeric field
```

Width is one nonnegative decimal literal in the format spec. Alignment is one of `<`, `>`, or `^`.
The default alignment is right for numeric values and left for text or bool. Space is the only
ordinary padding byte. `0` changes numeric padding to ASCII zero and places zeros after any sign and
radix prefix. Sign and prefix bytes count toward width.

For centered alignment with an odd number of padding bytes, the left side receives the smaller half
and the right side receives the remaining byte. This rule is target-independent.

`#` produces exactly `0x`, `0X`, `0b`, or `0o`. It is rejected with decimal output. `+` is rejected
for unsigned values, bool, and text. Numeric kinds are rejected for bool and text. Zero padding is
rejected for text and bool.

The implementation must parse spec fields into typed data. It must never retain an unvalidated raw
specifier for a backend or runtime to reinterpret.

## Text width is UTF-8 byte width

Text alignment uses the exact UTF-8 byte count reported by `str.len`. It does not estimate terminal
columns, Unicode scalar count, grapheme clusters, East Asian width, or font rendering.

Text precision and implicit truncation are excluded. A format plan never cuts a UTF-8 sequence or a
user-visible grapheme to satisfy a field. Source that needs truncation must first call an explicit
byte or Unicode operation with its own failure and boundary contract.

Dynamic width is excluded:

```fy
fmt"{value:{width}}"
```

All spec fields are source literals. This keeps maximum encoded size and planner work bounded and
prevents formatting from becoming a second runtime expression language.

The lexer, parser, and format planner use independent bounded budgets for fragment count, nesting,
template expansion, literal bytes, and width. An implementation guard is not a stable language or
ABI limit and must be reported as such.

## Static Format Plans are not first-class values

The following forms are rejected:

```fy
let plan = fmt"user={user_id}";
return plan;
store.plan = plan;
send_to_other_thread(plan);
```

A plan cannot be stored, returned, assigned, placed in Option or Result, captured by a closure,
converted to a pointer, compared, or sent across a function boundary. Making it first-class would
require a heterogeneous environment containing typed values, borrows, lifetimes, encoder metadata,
and dynamic dispatch.

A plan is legal only as:

```text
the direct plan argument of one exact compiler-known format sink
a nested plan produced by an admitted format template
the body result of an admitted format template
```

The sink itself cannot become an ordinary function value because its plan parameter has no
source-spellable first-class type. Compilation eliminates every plan before runtime linkage.

## Compiler-known sink resolution

Format sinks are exact compiler-known call forms. Name resolution still verifies the canonical module
and operation, but ordinary user code cannot declare a substitute function that receives a hidden
plan type.

The initial console sinks are:

```fy
std::io::write_fmt(fmt"...")
std::io::write_stderr_fmt(fmt"...")
```

Later materializing sinks belong to `std::fmt`:

```fy
std::fmt::into(destination, fmt"...")
std::fmt::append(buffer, fmt"...")
std::fmt::string(fmt"...")
```

A plan passed to any other call is rejected even when that call uses the same source name in another
module or package instance.

## Console sink contract

`write_fmt` and `write_stderr_fmt` allocate no heap storage and append no newline. On success they
return the complete plan byte count. All holes and exact lengths are ready before the first console
operation.

The first executable backend may dispatch verified fragments through the existing exact
`write`, `write_i32`, `write_u32`, `write_usize`, and `write_bool` families after completing global
evaluation and preflight. `u8` may use a compiler-owned scalar encoder or a value-preserving `u32`
transport. No bool-specific, u8-specific, or generic raw formatter symbol is required.

A later backend may coalesce or segment the same plan without changing source semantics.

## Format write failures

Existing `ConsoleError` cannot describe a preflight overflow or identify which verified fragment was
active when a destination operation failed. Console plan execution therefore needs a distinct
`FormatWriteError` contract.

The observable information must include at least:

```text
failure kind
raw destination status when one exists
exact plan-wide byte prefix completed before failure
zero-based fragment ordinal
```

The compiler retains the exact source span for every fragment. The runtime value carries the stable
fragment ordinal rather than embedding source paths and byte offsets in the runtime ABI. Diagnostics,
traces, and tooling can map that ordinal back to the compile-time span when the build retains the
plan map.

A preflight overflow reports zero written bytes and performs no output. A console failure includes
all complete earlier fragments and the exact partial prefix of the current fragment.

```text
literal "user="          5 bytes complete
user_id                  3 bytes complete
literal " score="        4 bytes of 7 complete
failure

written = 12
```

A logical format plan does not promise an atomic operating-system write or isolation from another
thread or process. Source requiring atomic publication must first materialize the plan into owned or
caller-provided storage and use a separately synchronized sink.

## Fixed-buffer materialization

A later sink writes into a caller-provided mutable byte view:

```fy
fn Result<ref str, FormatBufferError> render(
    ref mut u8[] scratch,
    u32 id,
    i32 score,
) {
    return std::fmt::into(
        scratch,
        fmt"id={id} score={score}"
    );
}
```

On success the returned `ref str` borrows the exact initialized result prefix from `scratch`. It
cannot outlive the destination origin.

The sink computes the exact required byte count before modifying the destination. If capacity is
insufficient, it returns a destination-capacity failure and leaves the destination bytes untouched.
Once capacity has been proved, encoding itself has no recoverable failure path; it writes each byte
and returns the exact prefix view.

When a fixed array's canonical size proves sufficient for the plan's maximum result, the verifier may
eliminate the runtime capacity branch. When the maximum required result cannot fit and the plan has
no unbounded text fragment, compilation may reject the call before execution.

Fairy C spells fixed-array types as `u8[N]`, not C- or Rust-shaped declarators. Contextual conversion
from a mutable `u8[N]` place to the expected `ref mut u8[]` sink parameter preserves the existing
array borrow rules.

## Buffer append materialization

`std::fmt::append` accepts an exclusive Buffer borrow and appends one complete plan:

```fy
std::fmt::append(
    ref mut buffer,
    fmt"user={id} score={score}\n"
)?;
```

The compiler evaluates every hole, computes exact added length, checks old-length plus added-length,
and requests the final capacity once. If reserve fails, the existing initialized prefix remains
unchanged. On success, fragments fill the newly reserved tail and the Buffer length commits once.

The current public `Buffer.view_mut` exposes only the already initialized prefix. An executable
format append slice therefore requires a compiler/SDK-private verified tail-fill and commit boundary;
it must not expose uninitialized capacity as a general public mutable view. The private operation
must preserve the same allocation, ownership, and failure-atomicity rules as existing Buffer
operations.

No geometric growth loop or per-fragment public append sequence is required after successful
preflight.

## Owned String materialization

`std::fmt::string` is the only plan sink that necessarily creates a new owned allocation:

```fy
let String message = std::fmt::string(
    fmt"user={id} score={score}"
)?;
```

It evaluates and preflights once, allocates the exact final capacity once, fills the bytes, and
returns one owned `String`. It does not repeatedly grow a String or Buffer.

Every admitted fragment is valid UTF-8: literals and `ref str` are validated text, while scalar
encoders emit ASCII. The compiler can therefore mark the completed Buffer storage as UTF-8 by
construction. The implementation may use a compiler/SDK-private trusted Buffer-to-String ownership
transfer and skip a redundant full UTF-8 scan. The public `std::text::from_buffer` validation
contract remains unchanged for arbitrary user-provided bytes.

Allocation remains an explicit operation rather than a hidden effect. Allocation failure uses the
selected owned-storage error contract and leaves no partially returned String.

## Length planning

Each typed plan carries three classes of length information:

```text
static literal bytes
maximum bytes known from type and static spec
bytes known only from a retained runtime value
```

For:

```fy
fmt"id={id} active={active} name={name}"
```

with `id: u32`, `active: bool`, and `name: ref str`, the compiler knows every literal length, at most
10 decimal bytes for `id`, at most 5 bytes for bool, and no type-only maximum for `name`.

At runtime preflight reads the retained `name.len`, computes the actual scalar digit counts and
padding, and adds every fragment with checked `usize` arithmetic. This exact total drives console
strategy, fixed-buffer capacity, one Buffer reserve, or one String allocation.

An internal plan may expose:

```go
type FormatPlan struct {
    Span             hir.Span
    Fragments        []FormatFragment
    StaticBytes      uint64
    MaximumBytes     uint64
    HasMaximum       bool
    HasUnboundedText bool
}
```

`MaximumBytes` is meaningful only when `HasMaximum` is true. Width is a minimum output width; because
text is never truncated, a text field with static width can still be unbounded.

## Lexer and lossless CST boundary

`fmt"..."` must be tokenized as actual format syntax. The compiler must not emit one ordinary String
token and reparse its cooked bytes during semantic analysis.

The lexer enters a bounded format mode and preserves distinct source spans for:

```text
format opener
literal text
hole opener
ordinary Fairy C expression tokens
format spec
hole closer
format closer
```

Literal fragments retain the same escape and UTF-8 validity rules as ordinary strings. Hole tokens
retain their original bytes and ordinary token kinds. Format mode tracks nested parentheses and
brackets so a top-level colon and closing brace are unambiguous.

The CST adds dedicated structure:

```text
KindFormatPlan
KindFormatText
KindFormatHole
KindFormatSpec
```

Lossless reconstruction must still reproduce every original byte. A malformed spec points to the
exact unsupported character, and a hole type error points to the ordinary expression span rather
than the whole literal.

## AST boundary

AST lowering preserves format concepts without reparsing raw source text:

```go
type FormatPlan struct {
    Span      diagnostic.Span
    Fragments []FormatFragment
}

type FormatFragment struct {
    Kind       FormatFragmentKind
    Span       diagnostic.Span
    Text       string
    Expression *Expression
    Spec       FormatSpec
}
```

`FormatSpec` stores parsed flags, static width, alignment, and numeric kind. Literal source spelling,
cooked UTF-8 bytes, expression span, and spec span remain separately available for diagnostics and
formatting tools.

## Typed HIR boundary

Semantic analysis resolves every hole before selecting an encoder. HIR retains one typed plan and
never stores an unresolved raw format spec:

```go
type FormatPlan struct {
    Span             Span
    Fragments        []FormatFragment
    StaticBytes      uint64
    MaximumBytes     uint64
    HasMaximum       bool
    HasUnboundedText bool
}

type FormatFragment struct {
    Kind       FormatFragmentKind
    Span       Span
    Expression *Expression
    Encoder    FormatEncoder
    Width      uint32
    Alignment  FormatAlignment
    Nested     *FormatPlan
}
```

Adjacent literals are merged. A scalar expression already proven compile-time constant may be
encoded and folded into a literal fragment. Constant folding consumes existing typed constant facts;
it does not move format parsing or runtime interpolation into the scalar const evaluator.

The plan itself has a compiler-only type that is unavailable to source type syntax. HIR sink nodes
retain the exact resolved sink kind and destination rather than representing the operation as an
ordinary call with an erased plan argument.

## Separate format-planning phase

Static Format Plan is not compile-time evaluation of runtime values. Compile time selects syntax,
hole order, canonical encoders, static padding, length bounds, and sink strategy. Actual hole values
remain runtime values.

The current scalar const evaluator has its own closed value set, call-depth limit, and node budget.
Format syntax, string fragments, template expansion, and sink planning must not be added to that
evaluator. A separate `internal/formatplan` phase should own plan validation, normalization, constant
fragment folding, expansion budgets, and deterministic strategy facts.

That phase may consume already-computed constants but cannot invoke arbitrary compile-time source
execution to manufacture format fragments.

## Verified MIR boundary

Verified MIR retains plan identity until sink-specific lowering. A candidate model is:

```go
type FormatPlanID uint32

type FormatPlan struct {
    ID        FormatPlanID
    Span      hir.Span
    Fragments []FormatFragment
}

type FormatExecute struct {
    Plan        FormatPlanID
    Sink        FormatSink
    Operands    []Operand
    Destination Operand
    ResultLocal LocalID
}
```

Literal bytes and encoder metadata live in the plan table. Hole expressions lower in source order to
ordinary typed operands before `FormatExecute`. No MIR fragment contains a runtime format string,
source identifier, dynamic type tag, or locale handle.

The initial backend may expand one verified execute node into the existing primitive calls. Later
backends may use stack coalescing or segmented output while consuming the same verified plan.

## MIR verification

The verifier must prove at least:

```text
every hole operand is evaluated exactly once in source order
all hole evaluation dominates preflight and every sink mutation
operand type agrees with the selected canonical encoder
no implicit resource move or mutable payload observation occurs
all length additions use checked usize semantics
preflight failure reaches no destination mutation
fixed-buffer success is dominated by sufficient capacity
Buffer append reserves the exact committed total before tail mutation
String construction owns one complete allocation before publication
console prefix accounting includes partial progress in the active fragment
fragment ordinal and plan identity agree on every failure edge
all retained borrows remain live through their final fragment use
no plan value escapes, joins an ordinary aggregate, or reaches a call boundary
no raw format string or source-level plan reaches a backend or runtime
```

Verifier mutation fixtures should remove, duplicate, reorder, or retarget each fact independently and
require fail-closed rejection.

## Backend strategies

A small plan with a known bounded maximum may encode into one fixed stack buffer and invoke the
existing write-all path once. A provisional threshold such as 256 bytes is a backend heuristic, not
a language contract.

A plan containing long borrowed text may retain literal and text spans without copying. Numeric
fragments use small fixed stack scratch storage. A POSIX backend may choose a `writev`-shaped private
transport; a Windows backend may coalesce small fragments and reuse its existing `WriteFile` loop.
The number of operating-system calls is not source semantics.

Every strategy must preserve the same fragment order, byte result, exact prefix count, and failure
ordinal. A backend may not introduce:

```text
runtime format parsing
C varargs
runtime type switches
locale state
heap formatter objects
implicit String construction
backend-specific evaluation order
```

A new segmented target primitive, if later justified, is private target-support compatibility data
and requires its own component, failure, and native evidence. It is not implied by this candidate.

## Explicit format templates

User-defined formatting should be explicit rather than an implicit global `Display<T>` relation:

```fy
fmt fn user_log(ref User user) {
    fmt"id={user.id} role={user.role}"
}
```

Use sites name the template:

```fy
std::io::write_fmt(
    fmt"selected={user_log(ref user)}\n"
)?;
```

A format template is not an ordinary function. It has no runtime symbol, function pointer, dynamic
dispatch, allocation, effect, or first-class result. The planner expands it into a nested typed plan
at the use site.

The first template slice should admit only private same-package templates with explicit parameters,
no recursion, no effects, no resource movement, and one Static Format Plan body. Cross-package or
public templates require their normalized body or another versioned expansion identity to become
part of the public interface and package-cache contract; this candidate does not silently define
that encoding.

Multiple named templates may format one type differently for logs, UI, protocols, or redaction.
There is no implicit selection and no coherence conflict between libraries.

## Effects

Constructing a Static Format Plan performs no output and adds no effect by itself:

```fy
fmt"user={id}"
```

Hole expressions retain their ordinary effects. The enclosing function must declare every effect
needed to evaluate them.

`std::io::write_fmt` and `std::io::write_stderr_fmt` require `console`. `std::fmt::into` and
`std::fmt::append` add no external capability effect. `std::fmt::string` may allocate, but allocation
remains an explicit ownership and failure operation rather than a Stage 0 external effect.

An explicit format template is effect-free; effectful computation must occur in an ordinary hole
expression before or around the template call.

## Diagnostics

An executable slice should reserve a dedicated format-plan diagnostic family and provide exact
source-linked errors for at least:

```text
an unmatched or unescaped brace
an empty or malformed hole
an unsupported or duplicate format flag
an invalid flag, kind, width, or alignment for the hole type
a hole type without a canonical encoder
an implicit resource, Result, Option, aggregate, C scalar, or raw-pointer format
a plan used outside an exact compiler-known sink or template
a dynamic width, precision, truncation, locale, or unsupported Unicode-width request
a format template with effects, recursion, an inadmissible body, or an implicit capture
planner budget exhaustion
an exact fixed destination that can never hold a bounded plan
```

The primary span should be the smallest responsible text. Related spans should identify the hole
expression type, sink declaration, destination capacity, template declaration, or conflicting borrow
when relevant.

A runtime console failure remains data in `FormatWriteError`; it is not a compile-time diagnostic.

## Delivery slices

### Slice 1: canonical console plans

Admit `fmt"..."`, escaped braces, ordinary expression holes, the default encoders for `bool`, `u8`,
`i32`, `u32`, `usize`, and `ref str`, plus `std::io::write_fmt` and
`std::io::write_stderr_fmt`.

Require complete hole evaluation before output, exact preflight, plan-wide prefix accounting, a
separate FormatWriteError, lossless CST, typed HIR, verified MIR, interpreter execution, and
constrained-C execution. The first backend may sequence existing exact output primitives after
preflight. It must add no general runtime formatter or C varargs.

### Slice 2: plan-preserving output strategies

Add adjacent-literal and constant-fragment folding, bounded stack coalescing, long-text segmented
execution, strategy facts, and verifier mutation coverage. Preserve identical bytes and failure
semantics across strategy choices.

### Slice 3: materializing sinks

Add `std::fmt::into`, `std::fmt::append`, and `std::fmt::string` with fixed-buffer origin retention,
one Buffer reserve and commit, one exact String allocation, private initialized-tail handling, and
trusted UTF-8-by-construction storage transfer.

### Slice 4: static numeric presentation

Add `d`, `x`, `X`, `b`, `o`, `+`, `#`, `0`, static width, and byte-based `<`, `>`, `^` alignment.
Keep dynamic width, precision, truncation, floats, locale, and Unicode display width closed.

### Slice 5: explicit format templates

Add private nonrecursive effect-free `fmt fn` templates, nested plans, expansion budgets, and exact
capture-free name resolution. Public template identities and cross-package expansion remain a
separate compatibility decision.

Each executable slice needs an accepted implementation ADR or an accepted revision of this ADR with
exact grammar, diagnostics, language and SDK versions, cache impact, fixtures, MIR invariants,
private runtime symbols when any, and same-commit native evidence.

## Rejected alternatives

- Immediate desugaring to sequential writes: permits a late hole failure after an early output prefix
  and erases global length and strategy facts.
- Interpolation that always constructs String: hides allocation and copies data for console output.
- Overloading `std::io::write`: complicates ordinary call resolution and conceals plan execution.
- A runtime format string: adds dynamic parsing and makes malformed specs a runtime concern.
- C `printf` or varargs: loses exact types, evaluation guarantees, partial-prefix accounting, and
  target-neutral behavior.
- Parsing one cooked String token during semantic analysis: produces inaccurate spans and leaves
  formatter, IDE, and lossless CST unaware of holes.
- Merging the feature into scalar const evaluation: confuses runtime values with compile-time
  planning and turns one bounded evaluator into a general formatting subsystem.
- First-class plan values: require heterogeneous closure-like environments, lifetime erasure, and
  dynamic dispatch.
- Implicit `Display` or reflection: creates coherence conflicts, hidden cost, and accidental
  sensitive-data output.
- Implicit resource moves: hides ownership transfer inside presentation code.
- Automatic raw-pointer formatting: creates an address-disclosure path.
- Dynamic width, precision, or truncation: makes cost less predictable and introduces unsafe or
  ambiguous text slicing.
- Locale and terminal-column formatting in the language core: makes output target- and
  environment-dependent.
- Implicit newline variants: add API-name micro-semantics while hiding emitted bytes.
- Claiming console atomicity: confuses one logical plan with operating-system interleaving
  guarantees.
- Exposing Buffer uninitialized capacity as a public mutable view: weakens initialization and
  ownership boundaries merely to optimize one sink.
- Implementing templates, writev, Unicode layout, floats, every sink, and every specifier in the
  first slice: delays the useful canonical console feature behind unrelated complexity.

## Admission evidence

Promotion from `Candidate` requires at least one complete vertical executable slice with all of the
following:

```text
normative grammar and exact contextual fmt behavior
current- and previous-revision lexer and migration behavior
lossless token reconstruction and bounded malformed-input recovery
exact spans for literal text, holes, and specs
AST and HIR tests for fragment order, cooked bytes, encoders, and static length facts
single-evaluation tests for effectful holes
Result propagation and trap tests proving zero output before sink entry
Borrow Flow tests for retained ref str values and later mutation conflicts
type diagnostics for every excluded value family
checked total-length overflow with zero destination progress
console failure injection inside each fragment family with exact global prefix counts
one lexical fragment ordinal mapped to the correct compile-time span
verified MIR fixtures for evaluation, preflight, sink execution, and no plan escape
MIR verifier mutation fixtures for missing, stale, reordered, duplicated, or mistyped facts
interpreter and constrained-C execution with byte-identical stdout and stderr
native Linux and Windows evidence for exact bytes, counts, and injected partial failures
no runtime format parser, C varargs, locale dependency, hidden allocation, or bool/u8 generic symbol
effect fixtures proving only hole effects and selected sink effects are present
compatibility-vector, language-pin, SDK, interface-hash, cache-key, starter, and migration updates when syntax and public sinks are admitted
```

Materializing slices additionally require fixed-buffer no-mutation failure evidence, Buffer
reserve-and-commit failure atomicity, exact one-allocation String evidence, borrow-origin retention,
and proof that UTF-8 validation is skipped only for compiler-proven plan bytes.

A lexer-only, parser-only, semantic-only, SDK-only, backend-only, or runtime-only implementation is
inadmissible.

## Compatibility

This candidate changes no current source grammar, reserved-word set, SDK interface, language pin,
runtime ABI, MIR version, package interface, cache identity, backend output, or target maturity.

An executable `fmt"..."` slice changes the grammar even though `fmt` remains contextual, so the exact
language revision must advance. The previous-revision lexer and migrator must preserve historical
identifier and String behavior and must not reinterpret malformed current format source as valid old
source.

Adding `write_fmt`, `write_stderr_fmt`, or materializing `std::fmt` sinks changes the compiler-bundled
SDK source and public-interface identity. A verified MIR plan table changes serialized MIR and
package-artifact compatibility only when its exact versioned encoding is admitted.

The first console slice can reuse current target output primitives and need not change the runtime
ABI. Later segmented private operations require explicit target component and native evidence and do
not become a public C ABI merely because a backend uses them.

FormatPlan IDs, fragment ordinals, encoders, hidden plan types, planner budgets, and backend strategy
thresholds are compiler-private unless a later contract explicitly versions one of them.

## Consequences

Static Format Plans give Fairy C concise typed output without turning formatting into a hidden
allocation, a dynamic runtime language, or a `printf` wrapper. Every hole is evaluated once before
output, canonical encoders are selected from exact types, and the compiler knows enough geometry to
preflight fixed storage, reserve Buffer once, allocate String once, or choose an allocation-free
console strategy.

The design preserves the current explicit cost model. Plain `write` remains plain text. Console
sinks own `console`; fixed and Buffer sinks are effect-free; String creation exposes ownership and
allocation failure. Plans cannot escape or acquire a runtime representation.

The implementation cost is paid in the correct layers: format-aware lexing, lossless CST, typed plan
HIR, a separate bounded planner, verified sink MIR, precise prefix errors, and backend strategy
selection. Existing scalar output, Buffer ownership, String transfer, Borrow Flow, Result Flow,
effects, and native adapters remain the primitives rather than being replaced.

No current source program or runtime behavior changes under this candidate ADR alone.
