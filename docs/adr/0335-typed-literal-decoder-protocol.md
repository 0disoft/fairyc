# ADR 0335: Typed Literal Decoder Protocol

- Status: Candidate
- Date: 2026-08-18
- Product source: `docs/product/02-spec.md`
- Grammar source: `spec/grammar/stage0.ebnf`
- Lexing source: `docs/language/0006-lexing-parsing-and-recovery-v0.md`
- Safety source: ADR 0003
- Versioning source: ADR 0008
- Effect source: ADR 0116
- Admission source: ADR 0218
- Compile-time source: ADR 0220
- Preprocessing source: ADR 0322
- Public-interface source: `docs/package/public-interface-hash-v3.md`
- Package-cache source: `docs/package/whole-package-cache-v2.md`
- Depends on: ADR 0003, ADR 0008, ADR 0009, ADR 0116, ADR 0218, ADR 0220, ADR 0322

## Context

Fairy C already has bounded scalar compile-time evaluation. `@const_eval` is deliberately private,
module-local, effect-free, non-recursive, scalar-only, and limited to one explicit return expression.
It admits no bindings, mutation, loops, arrays, aggregates, Result construction, allocation, target
inspection, host access, or public cross-package evaluator transport. That boundary keeps constants
cheap, target-neutral, cache-honest, and removable before MIR.

Typed literals need a different execution shape. A useful duration, IPv4, UUID, semantic-version,
magic-byte, regular-expression, route, or glob decoder needs bounded input traversal, local state,
checked arithmetic, structured diagnostics, and sometimes compiler-owned static output. Expanding
`@const_eval` until it can implement those decoders would turn one intentionally narrow scalar
facility into a general `comptime` language and make its budget, interface, and cache rules unstable.

A spelling such as `duration"250ms"` would also solve only syntax. The important contract is that a
library can publish a deterministic decoder, the compiler can verify and execute it without loading
native code, a failure can point into the original literal source, and a success becomes an ordinary
typed constant or read-only artifact before Flow and MIR.

The decoder must remain an ordinary resolved Fairy C symbol. There must be no global literal-suffix
registry, expected-type search, compiler plugin, procedural macro token stream, runtime parser
fallback, or target-dependent memory snapshot.

This ADR records the complete candidate model under the name **Typed Literal Decoder Protocol**. It
does not reserve expression syntax, add literal SDK types, change the language pin, change public
interface v3, add a cache domain, or claim implementation.

## Candidate decision

Fairy C should admit typed literal invocations whose source form is one resolved decoder path, one
`@` execution delimiter, and one core literal carrier:

```fy
import std::time as time;
import std::net as net;
import app::protocol as protocol;

let Duration timeout = time::duration@"250ms";
let IPv4 loopback = net::ipv4@"127.0.0.1";
let Magic png = protocol::magic@b"\x89PNG\r\n\x1a\n";
```

A later raw-literal slice may admit forms such as:

```fy
let Regex decimal = regex::compile@r#"[0-9]+(\.[0-9]+)?"#;
```

The exact raw text and raw byte token spellings are not currently part of the normative grammar.
They remain dependent on a separate core raw-literal contract. Typed literals do not invent a raw
string lexer privately inside semantic analysis.

The defining contract is:

> Resolve one library decoder, verify its bounded Literal IR, execute it deterministically over one
> core literal payload, map any declared error back to the original source, and replace a successful
> invocation with one target-independent Canonical Value or compiler-owned static artifact.

Typed literals are neither macros nor compiler plugins. The decoder receives only the cooked text or
byte payload selected by the core lexer. It never receives surrounding tokens, source files,
imports, compiler state, target layout, or host capabilities.

## Source grammar

The complete conceptual grammar is:

```ebnf
typed_literal
  = decoder_path "@" text_literal
  | decoder_path "@" raw_text_literal
  | decoder_path "@" byte_literal
  | decoder_path "@" raw_byte_literal
  ;

decoder_path = path ;
```

The first executable slice admits only the existing cooked UTF-8 string token:

```ebnf
typed_literal = path "@" string_literal ;
```

The construct is one primary expression, not a general postfix operator. The left side must be a
path. The right side must be one admitted core carrier token. These are rejected:

```fy
select_decoder()@"250ms";
(decoder)@"250ms";
decoders[index]@"250ms";
```

Ordinary trivia may occur between tokens, while the formatter uses the canonical compact spelling:

```fy
time::duration@"250ms"
```

`@` already introduces item attributes at declaration boundaries. Expression-position `path @
literal` is syntactically distinct and requires an exact language revision when admitted. It does
not make `@` a general user-definable operator.

Interpolation is not admitted inside a typed literal carrier:

```fy
time::duration@"{amount}ms"
```

A dynamic value uses an ordinary runtime call. The entire typed-literal payload must be fixed by the
core lexer before decoder selection begins.

C-string tokens are not typed-literal carriers. Their implicit terminating NUL and C interop meaning
are not a neutral text or byte protocol. A decoder that needs a terminator must validate or append it
explicitly in its own target-independent value model.

## Decoder paths use ordinary name resolution

`time::duration` is an ordinary Fairy C symbol path. Existing module aliases, package instances,
visibility, duplicate-name diagnostics, and definition navigation apply without a new namespace.
Two packages may both define `duration`; only the imported path at the invocation selects one.

The compiler must not search every decoder by expected result type:

```fy
let Duration value = @"250ms";
```

That form is rejected. Adding a dependency must not silently change which decoder interprets source
bytes. The selected decoder remains visible in code review, search, rename, and Atlas facts.

There is no suffix spelling such as:

```fy
"250ms"_duration
```

A global suffix namespace would bypass ordinary import and package rules and make collisions harder
to diagnose.

## Compile-time invocation and runtime invocation are distinct

`@` requests deterministic compile-time decoding:

```fy
let Duration timeout = time::duration@"250ms";
```

Parentheses perform an ordinary runtime call:

```fy
let literal::Text input = literal::Text::from(config_value)?;
let Result<Duration, literal::Error> timeout = time::duration(input);
```

A typed-literal failure is a compile-time diagnostic. A runtime call returns `Result` according to
the function signature.

The compiler never falls back from a failed typed literal to a runtime parser. A successful typed
literal causes zero runtime parse executions. Source that needs runtime parsing must write the
parenthesized call explicitly.

## Value decoder declaration

A fixed-size value decoder uses `@literal_value`:

```fy
@literal_value(text)
pub fn Result<Duration, literal::Error>
duration(literal::Text input) {
    return decode_duration(input);
}
```

The declaration contract is exact:

- one safe free Fairy C function;
- exactly one `@literal_value(<carrier>)` attribute;
- exactly one parameter whose compiler-known literal view type agrees with the carrier;
- exactly `Result<T, literal::Error>` as the result;
- no declared effects;
- no unsafe or foreign boundary;
- a success type `T` that satisfies the structural Literal Stable predicate;
- a source body that lowers both to ordinary runtime MIR and verified Literal IR.

The attribute does not overload ordinary calls. The function remains callable at runtime with its
written parameter and Result type.

Private helper functions reachable from a value decoder need no public annotation. The compiler
discovers the closed helper call graph, requires every helper to satisfy the same safe Literal IR
subset, lowers that complete closure, and includes it in the decoder IR hash. A reachable ineligible
helper makes the decoder declaration fail closed.

## Package-local first boundary

The first executable slice accepts typed invocation only when the decoder and invocation belong to
the same resolved package instance. This avoids pretending that current public-interface v3
transports decoder behavior.

A decoder may still be `pub` so other modules in the same package can resolve it and downstream
packages can call it normally at runtime. Until the public decoder transport slice lands, a
downstream `decoder@"..."` use is rejected even though `decoder(input)` remains an ordinary public
call.

Same-package decoder or helper changes already alter the package source digest and invalidate the
provider build. Cross-package source acceptance opens only after a new public-interface version
records the complete decoder contract.

## Literal input views

A decoder does not receive an ordinary `str` or byte slice. It receives one compiler-known bounded
view whose indexing and range geometry are target-neutral.

The text profile exposes a closed API conceptually equivalent to:

```fy
input.len
input.byte(index)
input.slice(start, end)
input.range(start, end)
input.starts_with("0x")
```

`literal::Text` is valid UTF-8 because it comes from a cooked core string token. `byte(index)` is
checked byte access. `slice` creates another view into the same immutable payload. `range` creates a
source-relative error range and does not expose a pointer.

The source-visible length and index domain is a fixed-width compiler-known coordinate, initially
`u32`. The compiler enforces a hard literal-input byte ceiling below that domain. Runtime conversion
from `ref str` checks the same ceiling before constructing `literal::Text`. A decoder cannot observe
host `size_t`, Fairy `usize`, pointer width, or target layout through the view.

Later carrier types are distinct:

```text
literal::Text
literal::RawText
literal::Bytes
literal::RawBytes
```

The raw profiles preserve raw-token payload semantics established by their future core literal
contract. They do not permit the decoder to inspect source delimiters or token trivia.

## Value literals and artifact literals are separate profiles

One execution profile cannot serve fixed values and input-sized static programs without either
hiding allocation or forcing every parser into fixed arrays. The protocol therefore has two
profiles.

### `@literal_value`

Value decoders produce fixed-size Canonical Values such as:

```text
Duration
IPv4
UUID
Port
SemVer
permission mask
protocol magic
```

The profile permits no compiler-time allocation whose result survives evaluation. Scratch storage is
bounded and discarded. The result must be a Literal Stable value. The same source function remains
runtime-callable.

### `@literal_artifact`

Artifact decoders produce input-sized immutable static data such as:

```text
regular-expression automaton
URL route matcher
glob program
protocol dispatch table
static format plan
```

A declaration is conceptually:

```fy
@literal_artifact(raw_text)
pub fn Result<Regex, literal::Error>
compile(literal::RawText input) {
    return compile_regex(input);
}
```

Artifact evaluation may allocate only inside a compiler-owned bounded literal arena. The result is
frozen into target read-only storage. The declaration has no ordinary runtime-call meaning; a
library wanting runtime regex compilation must expose a separate ordinary parser API with an
explicit allocator and Result contract.

Artifact results may contain compiler-known immutable handles:

```fy
struct Regex {
    StaticBytes program;
    u32 state_count;
    u32 capture_count;
}
```

`StaticBytes` and `StaticText` are copyable inert values referencing compiler-owned read-only data.
They have no cleanup, mutation, allocator identity, or source-visible raw pointer. Their exact source,
HIR, package-artifact, ABI, and backend contracts must be admitted with the artifact slice.

A value decoder never gains the artifact arena. A simple Port or IPv4 parser must not pay for or
silently depend on compiler-time allocation.

## Literal Stable is a structural predicate

Fairy C does not add a trait system solely to mark literal results. The compiler determines Literal
Stable structurally over canonical HIR types.

The candidate admits:

```text
bool
explicit target-independent fixed-width integers
inert structs whose fields are Literal Stable
inert enums whose payloads are Literal Stable
Option and Result whose payloads are Literal Stable
bounded fixed arrays of Literal Stable elements
StaticText and StaticBytes in the artifact profile
```

The current first value slice starts with the fixed-width executable set already present in Fairy C:

```text
bool
u8
i32
u32
```

It may additionally admit inert structs, restricted inert enums, and bounded fixed arrays composed
only from that set when the complete vertical implementation proves their Canonical Value and HIR
materialization paths.

The structural predicate rejects:

```text
resource types
safe borrows
mutable borrows
raw pointers
function pointers
Region capabilities
usize or another target-width integer
c::* scalars
foreign aggregates
uninitialized storage
runtime allocator or owner handles
```

`usize` is intentionally excluded from decoder results even though it is an ordinary Fairy C type.
A cached literal result must not change with pointer width. A source decoder can use the bounded
literal coordinate type internally, but the final Canonical Value remains target-independent.

## Canonical Value, not host memory

The Literal VM never returns Go struct bytes, C layout bytes, host pointers, or backend ABI records.
It returns one target-independent value tree:

```text
LiteralValue
    Scalar
    Aggregate
    Variant
    FixedArray
    StaticText
    StaticBytes
```

An aggregate records canonical nominal type identity, field identity, and child values. A variant
records canonical enum identity, variant identity, and payload values. Integer values retain their
exact declared width and mathematical value.

For example, a duration may be retained as:

```text
Aggregate std::time::Duration
    seconds       u32 2
    nanoseconds   u32 500000000
```

It is not retained as the memory bytes of one target struct. Windows, Linux, and later backends
materialize the same Canonical Value through their selected target layout after semantic evaluation.

The value tree contains no source path, line number, host endianness, pointer width, padding, target
triple, optimization profile, or build-local package number. Nominal identities use stable package
coordinate and module/type identities where cross-package transport is admitted.

Value-tree validation runs after Literal IR execution. A decoder cannot forge a field, variant, type,
or static handle that does not agree with its declared success type.

## A separate versioned Literal IR

Decoder source must never execute as native code inside the compiler process. The compiler does not
build or load a provider `.so`, `.dll`, object file, C shim, JIT page, or plugin. User source cannot
crash the compiler through native execution or access the build machine through host APIs.

Eligible source lowers to a versioned **Literal IR**. The verifier accepts only a closed deterministic
instruction and type set before the interpreter executes anything.

The candidate Literal IR supports enough structure to write real parsers:

```text
local bindings
fixed-width scalar mutation
if and switch
bounded classic scalar loops
checked arithmetic, comparison, bitwise, and shifts
fixed arrays and bounded scratch storage
literal input byte access and slicing
Literal Stable aggregate and variant construction
calls within the verified helper closure
static error construction
```

The first slice excludes recursion. It also excludes general while loops unless lowering proves an
explicit bounded form. Classic loops may range over fixed constants or the bounded input coordinate.
Independent execution fuel remains mandatory even for verifier-approved loops.

Literal IR categorically excludes:

```text
unsafe
foreign calls
raw pointers
safe-borrow escape
resources and cleanup
Regions
filesystem
console
environment
clock
randomness
network
processes
threads
target layout or ABI queries
compiler filesystem or package graph access
dynamic linking
runtime reflection
```

The decoder and every helper declare no effects. Compiler-known literal kernel operations are not
ordinary external capabilities and remain closed by IR version.

## Runtime MIR and Literal IR must agree

A value decoder has one Fairy C source body and two lowerings. Runtime calls lower through the normal
typed HIR and MIR pipeline. Typed invocations lower through Literal IR after the same resolution and
type checking.

The two paths must share declared-width arithmetic, branch, indexing, aggregate, enum, and Result
semantics. An implementation cannot maintain an unrelated host parser for compile time and a Fairy
parser for runtime.

Promotion requires differential fixtures that execute the same admitted decoder and input through:

```text
Literal IR interpreter
ordinary MIR interpreter
constrained C execution for the runtime call
```

The success value or structured `literal::Error` must agree. This evidence catches semantic drift
between the compiler kernel and the language runtime path.

Artifact decoders have only the Literal IR path and therefore require separate verifier, determinism,
serialization, and backend materialization evidence.

## Deterministic resource budgets

Every Literal IR execution is bounded independently of source trust.

The kernel applies:

```text
base fuel
input-byte-scaled fuel allowance
per-invocation hard fuel ceiling
per-package cumulative hard ceiling
scratch-memory ceiling
maximum error-catalog entries
maximum Canonical Value nodes
artifact arena byte ceiling
final static-output byte ceiling
```

The compiler owns absolute ceilings. A decoder contract may request or publish a lower budget class;
a dependency can never raise the consumer compiler's hard maximum.

Fuel is charged by versioned Literal IR operations, not host timing. Identical IR, input, contract,
and kernel version therefore reach the same value, declared error, or budget diagnostic on every
machine.

Budget exhaustion is a stable compile-time failure for that invocation. It does not hang, crash, or
silently switch to runtime parsing. The diagnostic identifies the decoder declaration, invocation,
consumed fuel, applicable ceiling, and input byte count.

## Structured decoder errors

The key product value is not merely converting a runtime error into a build error. The library must
identify the exact invalid part of the literal.

A decoder may return:

```fy
return .err(literal::error(
    input.range(unit_start, input.len),
    "missing-unit",
    "expected ns, us, ms, s, m, or h"
));
```

The error contract contains only bounded deterministic data:

```text
input-relative half-open range
static error key
static message
optional static replacement
optional static expected-token list
```

Keys, messages, replacements, and expected tokens must originate from static literals reachable in
the verified decoder artifact. A decoder cannot construct an unbounded dynamic message from input or
read another source file.

The rendered stable identifier is automatically namespaced by provider package coordinate, module,
decoder, and key:

```text
<package-coordinate>/std::time/duration/missing-unit
```

Two dependencies may both use `overflow` without colliding.

A replacement is valid only when its range belongs to the same carrier input and the replacement
bytes satisfy that carrier's core lexical rules. Tooling may expose a verified replacement as a quick
fix. The decoder cannot edit outside the literal or return an arbitrary file span.

A typed invocation whose decoder returns `.err` produces no HIR value, MIR, static data, or runtime
fallback.

## Literal source maps

Current literal tokens preserve one complete source span and one cooked value. That is insufficient
for decoder errors after escape processing. Four cooked UTF-8 bytes may all originate from one source
escape:

```fy
emoji::fruit@"\u{1F34C}"
```

The lexer must add a sparse `LiteralSourceMap` side table keyed by token index. It maps cooked byte
ranges to the source byte ranges that produced them:

```text
cooked 0..4 -> source range covering \u{1F34C}
```

The source map is created during the existing cook step. Semantic analysis must not reconstruct it
by reparsing cooked bytes.

Identity text may use one base offset or compact identity segment. Escape, normalization, and
non-identity segments receive explicit entries. Ordinary tokens do not gain per-token heap fields;
only literals needing nontrivial mapping allocate side-table entries.

Decoder errors use cooked input coordinates. The compiler maps them through the side table to the
smallest exact original source range. A range crossing multiple mapped segments covers their minimal
contiguous source extent and retains related cooked-range data for IDE tooling.

The first typed-literal slice must ship this map with precise decoder diagnostics. Pointing every
error at the entire string is not an acceptable partial implementation.

## Optional canonicalizers

A decoder may later name one compiler-verified canonicalizer:

```fy
@literal_value(text)
@literal_canonical(canonical_duration)
pub fn Result<Duration, literal::Error>
duration(literal::Text input) {
    return decode_duration(input);
}
```

The canonicalizer is compiler-only and deterministic. It returns bounded compiler-owned canonical
text, not a borrowed `literal::Text` view into temporary storage. A later exact SDK contract may name
that result `literal::CanonicalText`.

For one invocation the formatter rewrite is enabled only after the compiler verifies:

```text
decode(original) = value
canonicalize(value) = canonical
decode(canonical) = the same value
canonicalize(the same value) = the same canonical text
```

A failed round trip does not change the typed literal's decoded value. It disables automatic rewrite
for that invocation and reports the canonicalizer defect against the decoder declaration when the
formatter or validation command requests canonicalization.

This permits equivalent spellings such as:

```fy
time::duration@"1500ms"
time::duration@"1.5s"
net::ipv6@"2001:0db8:0:0:0:0:0:1"
```

to format into one project-selected canonical spelling without trusting arbitrary text rewriting.

Canonicalizers are a later delivery slice. The first decoder implementation does not delay on them.

## Public decoder behavior is a public source contract

Current public-interface v3 intentionally excludes function bodies and private helpers. That is
correct for ordinary calls because changing a body does not normally change whether downstream
source type-checks.

A public typed-literal decoder is different. Changing its executable decoder body can make yesterday's
literal fail, change its Canonical Value, change an error range, or change canonical formatting. The
verified decoder artifact is therefore part of downstream source acceptance.

Cross-package typed invocation requires a new public-interface version rather than adding hidden
fields to v3. The new interface records at least:

```text
decoder qualified identity
ordinary callable identity
accepted carrier kinds
value or artifact profile
input view type
success type and canonical materialization schema
budget class and declared lower ceilings
static error key/message/replacement catalog
canonicalizer identity and hash when present
verified Literal IR hash
complete verified helper-closure hash
Literal IR version
literal kernel minimum version
```

The provider publishes verified Literal IR data, not native code. The consumer validates all length
frames and hashes, checks the IR version, re-runs the verifier under its own hard ceilings, and fails
closed when unsupported.

A private helper change that alters the normalized decoder IR changes the public decoder contract and
therefore the public-interface hash. That downstream rebuild is intentional. A private change that
does not alter the verified decoder artifact need not change the decoder contract hash.

The ordinary runtime-call signature remains part of the existing callable interface facts. The typed
contract adds source-decoding behavior rather than replacing ordinary function identity.

## Literal evaluation cache

Literal evaluation uses a dedicated immutable cache domain:

```text
fairyc.literal-eval/v1
decoder_ir_hash
decoder_contract_hash
carrier_kind
cooked_input_hash
literal_kernel_version
```

The key excludes:

```text
source path
line and column
invocation span
build profile
target triple
target layout
backend
host operating system
```

Identical verified decoder behavior and identical payload produce the same target-independent result
wherever invoked. Source mapping and diagnostic rendering remain invocation-local and are applied
after a cached decoder error is loaded.

Successful Canonical Values and decoder-declared structured errors may be cached. Corrupt entries,
unknown versions, type mismatches, invalid ranges, invalid static handles, and values exceeding the
current compiler ceilings are rejected and recomputed or reported fail closed.

Target layout materialization and read-only section emission belong to later target-dependent cache
stages. They are not folded into the evaluation key.

Artifact output may be deduplicated by the digest of its validated Canonical Value and static blobs.
Different source spellings that decode to the same canonical artifact need only one read-only data
instance in the final program when linkage identity permits it.

## Compiler pipeline

Typed literals extend the existing literal set with one explicit primary expression rather than
registering an unbounded family of lexer token kinds.

The candidate pipeline is:

```text
source

lexer
    core carrier token
    cooked payload
    LiteralSourceMap

CST
    KindTypedLiteral
    decoder path
    @ delimiter
    carrier token

AST
    decoder path
    carrier kind
    cooked payload
    carrier source span and source-map identity

name resolution
    ordinary module and package lookup

semantic checking
    resolve exact decoder declaration
    validate profile, carrier, effects, input, Result, and success type

literal expansion
    load or lower Literal IR
    verify artifact and helper closure
    read evaluation cache or execute bounded kernel
    validate Canonical Value or structured error

HIR
    ordinary scalar, aggregate, enum, array literal
    or compiler-owned static-data reference

Flow and MIR
    normal typed value only

backend
    target layout materialization
    optional read-only blob emission
```

`KindTypedLiteral` remains lossless. The CST retains the decoder path, `@`, carrier source spelling,
and all trivia. AST lowering consumes structured children and never reparses the full source text.

Value literal expansion completes before Borrow, Resource, Result, Region, and must-use Flow. The HIR
value participates exactly like an ordinary constant of its success type.

MIR and generated C contain no decoder path, Literal IR, error catalog, evaluation budget, source
spelling, or cache metadata. Artifact HIR retains only verified static-data identity needed for
read-only emission.

## HIR boundary

A candidate pre-expansion HIR node may retain:

```go
type TypedLiteral struct {
    Span         Span
    Decoder      SymbolID
    Carrier      LiteralCarrier
    PayloadHash  [32]byte
    SuccessType  Type
    ContractHash [32]byte
}
```

It is a transient semantic node. Successful value expansion replaces it with existing HIR literal,
aggregate, variant, or fixed-array concepts. Artifact expansion replaces it with a compiler-owned
static-data reference.

No source-spellable `TypedLiteral`, `LiteralValue`, decoder object, or Literal IR handle enters the
ordinary type system. Typed literal invocations cannot be stored as unevaluated computations, passed
to functions, returned for later execution, or converted into function values.

## Literal IR artifact verification

The verifier checks more than opcode validity. It must prove or enforce at least:

```text
all values use admitted fixed-width or compiler-known literal types
all control-flow targets are in one verified function body
all calls target the closed verified helper closure
no recursion appears in the first version
all input views derive from the one invocation carrier
all range values remain within the carrier coordinate domain
all error ranges refer to the same carrier
all error text and replacement data are static and bounded
all aggregate and variant construction agrees with canonical type definitions
all fixed arrays obey admitted bounds
all artifact handles refer only to the current bounded arena
no unsafe, foreign, effectful, address-bearing, resource, Region, or target query exists
fuel and memory charging points agree with the Literal IR version
all success and failure exits produce the declared Result shape
```

The interpreter validates dynamic bounds and checked arithmetic during execution. A verifier proof
never turns an invalid input into undefined behavior.

Dependency artifacts are untrusted input. Malformed length frames, duplicate IDs, hash mismatch,
unknown opcodes, excessive declarations, unsupported versions, and verifier failure must be rejected
without executing the artifact.

## Effects and capabilities

A decoder declaration and every helper have the empty effect set. Typed literal evaluation adds no
external effect to the enclosing function because it completes during compilation.

A runtime call to a value decoder also remains effect-free under the same declaration. Artifact
decoders have no runtime call path.

Allocation inside the future artifact arena is compiler-owned bounded construction, not a Stage 0
external effect and not permission to use the runtime allocator. Filesystem, environment, clock,
random, process, network, console, and foreign capabilities remain unavailable.

## Diagnostics

An executable implementation should reserve a dedicated typed-literal diagnostic family rather than
reusing `E4301` through `E4305` for a different execution engine.

Stable source-linked diagnostics are required for at least:

```text
malformed path @ carrier syntax
an unresolved, invisible, or non-decoder path
duplicate, misplaced, or malformed literal attributes
carrier and decoder-profile mismatch
wrong parameter or Result signature
declared or reachable effects
an ineligible helper or recursive helper graph
non-Literal-Stable success type
unsupported public decoder transport or Literal IR version
Literal IR verification failure
fuel, scratch, Canonical Value, or artifact-output budget exhaustion
a decoder-declared structured error
an error or replacement range outside the carrier
invalid static error text or replacement
Canonical Value/type mismatch
corrupt evaluation-cache entry
canonicalizer round-trip failure when rewrite is requested
```

A decoder-declared error uses the decoder's package-scoped stable identifier while the compiler's
own diagnostic code identifies the general typed-literal failure category. The primary span is the
mapped literal subrange. Related spans identify the invocation, decoder declaration, helper,
attribute, budget contract, or provider artifact as appropriate.

Errors caused only by an earlier poisoned decoder declaration should be suppressed rather than
repeated at every invocation.

## Tooling and Atlas

Because the decoder is an ordinary resolved symbol, definition navigation, rename, references, and
package ownership work without a compiler-global registry.

A typed-literal hover may show:

```text
type Duration
canonical value 250000000 nanoseconds
decoder std::time::duration
profile value
evaluation 184 steps
static output 0 bytes
cache hit true
```

A decoder error may publish a bounded expected-token set for completion at the cursor.

After the executable contract stabilizes, Atlas may expose one invocation entity with:

```text
decoder symbol
carrier kind
input digest
canonical value digest
evaluation steps
static output bytes
cache-hit state
source span
```

This candidate does not alter the current Atlas schema. Tooling export follows a separate versioned
transport decision or an accepted implementation revision.

## Delivery slices

### Slice 1: package-local cooked text value decoders

Admit:

```fy
decoder@"cooked text"
@literal_value(text)
```

Keep typed invocation inside one package instance. Add `literal::Text`, `literal::Error`, sparse
LiteralSourceMap data, exact mapped diagnostics, the narrow Literal IR verifier and interpreter,
fixed fuel and scratch ceilings, dual runtime-MIR lowering, and Canonical Values for the first fixed
scalar and structurally proven inert aggregate, enum, and fixed-array subset.

Do not add raw carriers, cross-package typed invocation, canonicalizers, artifact allocation,
StaticText, or StaticBytes.

### Slice 2: public decoder transport and evaluation cache

Add the next public-interface version, versioned decoder contract encoding, verified helper-closure
transport, consumer-side re-verification, unsupported-version failure, package-cache invalidation, and
the `fairyc.literal-eval/v1` immutable cache.

Cross-package typed invocation remains closed until all of those parts land in one vertical slice.

### Slice 3: bytes, raw carriers, and canonical formatting

Add the existing cooked byte-string carrier, then separately admitted core raw text and raw byte
carriers. Add canonicalizer artifacts, invocation round-trip verification, formatter rewrite, and
precise raw/cooked source mapping.

C-string carriers remain excluded.

### Slice 4: artifact literals

Add `@literal_artifact`, compiler-owned bounded literal arena, StaticText, StaticBytes, Canonical Value
static handles, read-only backend emission, static-output ceilings, artifact serialization, and
content-addressed deduplication.

The slice should prove one real workload such as a small regex, route, or glob program rather than
introducing a general compiler-time allocator without a consumer.

### Slice 5: tooling facts

Add optional Atlas invocation relations, IDE evaluation-cost display, expected-token completion,
cache observability, and static-output growth diagnostics after the underlying facts are versioned.

Each executable slice requires an accepted implementation ADR or accepted revision of this ADR with
exact grammar, language and SDK versions, diagnostic codes, interface and cache versions, fixture
manifests, budgets, artifact formats, and same-commit native evidence where runtime behavior exists.

## Rejected alternatives

- `duration"250ms"` with no delimiter: collides with existing and future literal prefixes and makes
  decoder selection harder to search.
- C++-style user-defined suffixes: create a global suffix namespace outside ordinary imports.
- Expected-type decoder search: permits dependency changes to alter source meaning invisibly.
- Native compiler plugins, shared libraries, or JIT execution: expose the build host and compiler
  process to untrusted package code.
- Procedural-macro token streams: turn a bounded payload decoder into arbitrary source generation.
- Passing the full source token stream to a decoder: bypasses the core lexer and source grammar.
- Extending `@const_eval` with loops, mutation, aggregates, Result, and arenas: destroys the narrow
  scalar facility's cost and compatibility boundary.
- Running generated MIR, C, or target executables during decoding: makes evaluation depend on host
  and target toolchains.
- Saving host or target memory bytes as the result: bakes padding, endianness, pointer width, and ABI
  into a supposedly portable literal.
- Automatic runtime fallback after a compile-time failure: hides parsing cost and changes failure
  timing.
- Dynamic error messages or arbitrary source spans: make diagnostics unbounded and permit decoders
  to forge unrelated edits.
- Allowing resources, raw pointers, `usize`, C scalars, or foreign aggregates in Canonical Values:
  makes evaluation target- or lifetime-dependent.
- Giving value decoders the artifact arena: introduces hidden allocation into fixed-size values.
- Making artifact decoders ordinary runtime callables: hides allocator and ownership contracts.
- Quietly extending public-interface v3: creates two incompatible meanings for one versioned hash.
- Trusting provider-verified Literal IR without consumer verification: treats dependency artifacts as
  privileged compiler input.
- Making typed-literal computations first-class values: requires delayed environments, lifetime
  capture, dynamic dispatch, and a runtime decoder representation.
- Shipping the first slice without cooked-to-source mapping: produces whole-literal diagnostics and
  loses the main usability advantage.

## Admission evidence

Promotion from Candidate requires at least one complete executable vertical slice with all of the
following:

```text
normative grammar and exact path @ carrier precedence
current- and previous-revision parser and migration behavior
lossless CST reconstruction including trivia and @
lexer LiteralSourceMap identity and escape-segment fixtures
AST retention of decoder path, carrier, payload, and spans
ordinary multi-package name resolution and visibility behavior
exact decoder attribute, input, Result, effect, and success-type validation
closed helper-graph discovery and hashing
Literal IR codec, verifier, interpreter, and independent operation budgets
negative verifier fixtures for every forbidden opcode, type, call, range, and handle
single-evaluation and deterministic-fuel evidence
structured decoder errors mapped to exact escaped and unescaped source ranges
invalid replacement and out-of-range error rejection
Canonical Value structural validation and stable hashing
cross-target materialization evidence proving equal semantic values on Linux and Windows
runtime-MIR versus Literal-IR differential fixtures for value decoders
ordinary runtime decoder execution through interpreter and constrained C
proof that no native plugin, JIT, host I/O, target query, or runtime fallback executes
HIR folding to ordinary values and absence of decoder metadata in MIR and generated C
language-pin, SDK, compatibility-vector, cache-key, starter, and migration updates when syntax lands
```

The public transport slice additionally requires:

```text
new public-interface fixtures and historical-version rejection
private-helper change invalidation through normalized decoder IR hash
consumer-side decoder artifact re-verification
unsupported Literal IR and kernel version failure
immutable evaluation-cache hit, miss, corruption, and type-mismatch fixtures
source-location-independent and target-independent cache-key evidence
```

The artifact slice additionally requires:

```text
bounded arena and final-static-output exhaustion
read-only static section emission
zero runtime parser and zero runtime allocation for the chosen workload
StaticText and StaticBytes immutability and no-cleanup evidence
content deduplication and target-layout materialization evidence
```

A syntax-only, lexer-only, semantic-only, native-plugin, provider-trusted, runtime-only, or whole-
literal-diagnostic implementation is inadmissible.

## Compatibility

This candidate changes no current grammar, reserved-word set, language pin, SDK, public-interface
hash, package-cache key, Literal IR, MIR, runtime ABI, target maturity, backend output, or Atlas
schema.

The first executable `path@literal` slice changes expression grammar and therefore advances the exact
language revision. The immediately previous parser must preserve historical attribute and literal
behavior and reject new expression-position typed literals rather than reinterpret them as recovered
old source.

Adding `literal::Text`, `literal::Error`, decoder attributes, or runtime decoder APIs changes the
compiler-bundled SDK and its public-interface/source identities. Public decoder transport advances the
public-interface version and every dependent cache vector. Artifact handles and read-only data add
separate versioned package-artifact and backend compatibility only when implemented.

Literal IR versions, kernel versions, contract hashes, budget classes, Canonical Value encodings,
StaticText, StaticBytes, error catalogs, and evaluation-cache domains are compiler or SDK contracts;
they do not become a stable public C ABI merely because the compiler serializes them.

## Consequences

Typed Literal Decoder Protocol gives Fairy C libraries an official way to distribute small safe DSLs
without macros, native compiler plugins, runtime parsing, or a global suffix registry.

The source remains explicit. `time::duration@"250ms"` names the decoder, fixes the carrier, and
requests deterministic compile-time execution. `time::duration(input)` remains an ordinary runtime
Result-returning parser.

Decoder failures become package-scoped diagnostics on the exact original literal range. Successful
value decoders disappear into ordinary target-independent constants before Flow and MIR. Artifact
decoders can later freeze verified input-sized programs into read-only data with no runtime parser or
heap construction.

The implementation cost is paid in the correct layers: sparse literal source maps, a separate bounded
Literal IR, Canonical Values, structured errors, public decoder contract hashing, consumer-side
verification, and a dedicated evaluation cache. The existing scalar `@const_eval` evaluator remains
small and does not become a general compile-time language.

No current source program or runtime behavior changes under this candidate ADR alone.
