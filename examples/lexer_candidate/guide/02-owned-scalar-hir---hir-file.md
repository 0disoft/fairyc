[Candidate guide index](../README.md)

## Owned scalar HIR (`--hir-file`)

`lexer-candidate --hir-file <path>` reads the same bounded source file, then
parses, resolves names, checks scalar types and consumes the successful `Typed`
value through `hir_lower::lower`. The returned `hir_model::Program` owns source
bytes, canonical integer text and separate function, binding, expression,
statement, block, edge and signature arenas. It retains no parser or type-check
table. `hir_lower::lower(typed, module_index)` also preserves external function
module/declaration identity when its caller supplies an imported typed input;
the file CLI itself accepts one module without imports.

The private TSV format begins with `fairyc.scalar-hir/v1` and a `P` module-index
record. `F`, `B`, `E`, `S`, `K`, `A` and `T` records describe functions, bindings,
expressions, statements, blocks, ordered edges and signature types. Entity IDs
are one-based; zero means absent. Edge/signature indexes and module indexes are
zero-based. Enum labels and this format are private implementation contracts.

Supported semantics are scalar literals, resolved locals and parameters, unary
and binary operations, ordered calls, blocks and tails, `if`, `while`, classic
`for`, nearest-loop `break`/`continue`, assignment, return, assert, discard and
unevaluated void-call `defer`. Grouping and callee syntax become semantic links.
Integer payload is canonical decimal ASCII, including direct signed minima and
AMD64 `usize` maximum, without overflowing a signed intermediate.

The source cap is 32768 bytes, with at most 512 declarations/functions, 1152
bindings, 4096 expressions, 8192 statements/blocks, 32768 edges/signature entries,
86016 payload bytes and recursion depth 64. The existing parser/resolver/type
budgets apply first. Complete output is buffered under a 2 MiB cap before stdout
publication. Malformed, unsupported, allocation or limit failures return 2 with
empty stdout; scanner/name/type errors retain their existing located messages.
There is no MIR lowering, interpreter, backend or executable compilation of the
input in this mode.

From the repository root, regenerate the independent Go HIR oracle with:

```sh
go run ./tools/dev/fairyc-lexer-file-oracle --hir .cache/hir-file-cli/reference.json
```

Pass that file as `--hir-reference` to `verify.py`, together with the earlier
token/declaration/name/type/module references. `verify_hir.py` reconstructs nested
semantic trees from the native TSV and compares them with Go HIR; it also rejects
cycles and unreachable retained nodes. `tests/hir.fy` exercises the owned API
directly. The Go oracle and Python comparator are test utilities; the candidate
runtime uses Fairy C and public SDK only.

`samples/hir-demo.fy` demonstrates a contextual i64 literal, ordered call
arguments, a conditional result and a classic `for` loop:

```sh
lexer-candidate --hir-file samples/hir-demo.fy
```

Windows development and release pass 1024 CLI executions across the existing
file modes and HIR. Each profile compares 34 positive HIR fixtures and 26
rejections; six direct native API tests pass per profile. The final comparator
checks declaration targets, lexical scope relationships, source spans and error
categories, and rejects 159 deliberate observation mutations per profile.
Check, scoped Go test/vet and Linux/Windows C emission pass; Linux native execution
is deferred to the owner's batch. Compiler/SDK inputs remain unchanged. Evidence:
`docs/ops/evidence/hir-file-cli-20261005.json`.

## Owned scalar MIR (`--mir-file`)

`lexer-candidate --mir-file <path>` reads the same bounded source file, then
parses, resolves names, checks scalar types, lowers the successful `Typed` value
to HIR and consumes that `hir_model::Program` through `mir_lower::lower`. The
returned `mir_model::Program` owns source bytes, canonical literal payload and
separate target, signature, function, local, block, operation and edge arenas; no
parser, type or HIR table is retained. Lowering admits sequential scalar code
(literals, resolved locals and parameters, unary and binary operations, ordered
calls, assignment, checked compound assignment, nested lexical blocks and
assertions) plus value and void conditionals: `if`/`else`, `if` without an `else`,
`else if` chains, arms where both branches return, and short-circuit `&&`/`||` whose
right operand is evaluated only when the left side requires it and may be any
admitted boolean expression, including side effects, calls or a return block. Those
CFG forms use an unconditional jump terminator and a two-way boolean-branch
terminator alongside the earlier checked switch. `while`, classic `for` and
nearest-loop `break`/`continue` now lower to this same CFG. Initializers run once;
an omitted `for` condition is true. `continue` reaches the post clause before the
next condition, while `break` skips post. Post blocks are allocated only when a
body fallthrough or continue reaches them, avoiding orphan blocks for unconditional
breaks or returns. Loop targets use a bounded 64-frame stack of HIR Statement IDs;
the existing recursion budget applies first. Bounded lexical `defer` captures
copy-scalar arguments immediately, once from left to right, and emits ordinary
void internal calls in LIFO order when the registering block exits by fallthrough,
return, break or continue. The return or block-tail value is evaluated before those
calls. Traps skip deferred calls; a trapping deferred call skips remaining calls.
Capture descriptors use a bounded compile-time scope stack, not a runtime queue.
Resource drops, Region resets and Result propagation are outside this scalar MIR
subset; their ordering with defer remains subsequent ownership work.

The private TSV format begins with `fairyc.scalar-mir/v1` and a `P` module-index
record. `T`, `S`, `F`, `L`, `B`, `O` and `A` records describe call
targets, signature types, functions, locals, basic blocks/terminators, operations
and ordered edges. Entity IDs are one-based and zero means absent; edge and
signature indexes are zero-based. Blocks form a mutable-local control-flow graph,
not SSA. A checked operation emits a typed `check` marker, a two-successor
checked-switch terminator and a non-returning trap block whose mask keeps the
type-specific failure categories: integer overflow, division by zero,
signed-division overflow, invalid shift and assertion failure. Nested calls publish
their argument edge range in the shared arena only after every argument completes:
evaluation collects the ordered argument locals in a temporary vector left to right
and appends the contiguous range after the last argument, so an argument that returns
leaves neither orphan argument slots nor a dangling parent call. The branch work adds
no record tag or field: the jump and boolean-branch terminators reuse the existing `B`
layout, and a literal `assert(false)` ends its block with an unreachable,
successor-free success block so no live success path remains. The record labels, trap
masks and this format are private implementation contracts.

Limits are 32768 source bytes, 512 functions/targets, 16384 locals, 16384 blocks,
32768 operations, 65536 edges/signatures and 688128 payload bytes, with recursion
depth 64, at most 256 simultaneously active defer descriptors and 4096 captured
argument references; the existing parser, resolver, type and HIR budgets apply
first. Complete output is buffered under a 2 MiB cap before stdout publication. Malformed, unsupported,
allocation, limit and HIR-lowering failures return 2 with empty stdout and a plain
message; scanner, name and type errors keep their existing located messages.

`mir_model::assemble` checks construction ranges and links. The separate Fairy
`mir_verify::check` then checks the admitted scalar invariants: `V1001` operation,
literal, call, return and table shapes; `V1002` definite initialization on every
reachable predecessor path; and `V1501` exact checked arithmetic, trap edges and
unreachable proofs. Both lowering and rendering require this verification to
succeed. Rendering rejects an invalid graph before appending any bytes, preserving
an existing caller buffer. CLI verification failures exit 2 with empty stdout and
the private `MIR verification V1001`, `V1002` or `V1501` error on stderr.

The verifier requires closed internal function bodies and signatures, unique
parameter/edge/operation ownership, correctly typed operands and destinations,
canonical range-checked integer literals, and no orphan executable table entries.
Checked arithmetic must be the first operation of its success block, match the
guard's frozen operands, and have no unchecked predecessor bypass. Trap blocks
contain no operations or cleanup. An unreachable success block requires a false
assertion on a non-parameter local with exactly one constant-false definition.

Initialization uses heap-backed bitsets and a worklist. Reachability is established
first; predecessor intersections descend from the greatest fixed point, with entry
always seeded only from its declared parameters. Uses are checked after convergence.
Call results become initialized only on the normal successor. Zero-iteration loops
and cyclic predecessor paths cannot manufacture initialized values. Analysis is
bounded to 67108864 charged steps across the program; exhaustion returns the existing
limit code 2, and allocation failure returns 3. Per-function bitset storage is at
most 33554432 bytes under the existing local/block caps, plus bounded graph indexes.

Move/drop discipline, borrow origins, Region placement and unsafe provenance
remain subsequent verifier components because their value forms are not admitted
by this scalar model. There is no candidate interpreter, backend or executable
compilation of the input in this mode.

From the repository root, regenerate the independent Go MIR oracle with:

```sh
go run ./tools/dev/fairyc-lexer-file-oracle --mir .cache/mir-file-cli/reference.json
```

Pass that file as `--mir-reference` to `verify.py`, together with the earlier
token/declaration/name/type/module/HIR references. `verify_mir.py` is a test-only
reader and scalar executor: it reconstructs the tables from the native TSV, checks
the block and edge graph, and executes admitted programs to compare the observed
return value, trap category and fixture-only deferred-call trace with the verified
Go MIR. The trace hook is enabled only for explicitly marked reference fixtures.
It is an independent
observation utility, never a candidate runtime dependency, and it does not natively
compile or execute the input. Every admitted fixture must now match the executed
Go MIR reference; a reference execution error fails the oracle rather than
substituting a fixed expected result. `tests/mir.fy` exercises the owned API directly.

`samples/mir-demo.fy` demonstrates a checked compound assignment across a forward
call, and `samples/mir-branches-demo.fy` demonstrates a conditional result and a
short-circuit assertion:

```sh
lexer-candidate --mir-file samples/mir-demo.fy
lexer-candidate --mir-file samples/mir-branches-demo.fy
lexer-candidate --mir-file samples/mir-loops-demo.fy
lexer-candidate --mir-file samples/mir-defer-demo.fy
```

The current reference set is 106 fixtures, all admitted: 87 returning programs
and 19 trapping programs covering constants, forward and nested
calls, lexical blocks, checked compound assignment, signed `i32`/`i64` overflow and
negation, division-by-zero, signed division/remainder overflow, invalid-shift and
overflowing-shift traps, and assertion traps, extended by value and void conditionals
with and without an `else`, `else if` chains, returning condition operands, an
argument that returns from the first or last call position, a returning `if` used as
an argument, both-returning arms, branch side effects taken and skipped, conditional
fallthrough, and short-circuit `&&`/`||` with side-effecting or trapping right
operands; the unsigned-limit case exercises `usize`, `u32` and `u8` maximum values as
accepted successes rather than traps. Loop cases additionally cover zero iterations,
`i64` counters, side-effecting/calling conditions, nested and mixed loops, loop-local
shadowing, omitted clauses, initializer assignment, return/break/continue in a while
condition, early body returns, body/post overflow and calls skipped by loop control.
Defer cases cover immediate `i64` capture, nested LIFO scopes, skipped branches,
left-to-right argument evaluation, return during registration, return-value-first
cleanup, zero arguments, local shadowing, short-circuit paths, loop fallthrough,
break/continue and traps during registration, body execution or deferred cleanup.
Twenty-one explicitly marked cases compare exact emitted trace bytes as well as
their return value or trap category.

All 106 admitted fixtures match the Go MIR interpreter, with no contract goldens.
The Go lowerer captures source-local scalar reads before subsequent binary operands
or call arguments can change them, and propagates termination through binary and
short-circuit operands, conditions and enclosing returns. Thus
`var i32 x=5;return x+if(true){x=2;3}else{4};` returns 8, and returning operands in
conditions or calls preserve their inner `return 9` without evaluating later calls.
Twelve dedicated Go verifier/interpreter regressions cover these paths, including
the already-correct conditional argument, its fallthrough alternative and the
subsequent while-condition return fix. The same twelve assertions execute in
generated native C under low and optimized MSVC.

The earlier return-order fix passed 1162 CLI executions (581 per profile),
including its 59 MIR fixtures (56 admitted: 42 returning, 14 trapping; 3 deferred).
After restricting snapshots to source locals rather than already-materialized
temporaries, the final Go reference is regenerated and the 59-case MIR comparison
reruns on each retained image. Scoped Go MIR, interpreter, C backend and fixture
suites and vet pass. At that snapshot, `TestStructNestedStorageMIR/array` failed
with both the unchanged baseline lowerer and the revised lowerer. Subsequent
contract review established that its `i32[2]` field is already admitted by ADR 0468;
the failure was an obsolete rejection expectation, not a verifier defect. The
follow-up test now rejects zero-length, oversize and nested arrays and admits
scalar array fields of length 1, 2 and 4096 through source analysis, MIR codec and
both Windows/Linux C emission. Historical evidence retains the observed failure
and its initial classification; this correction does not change language behavior.

That return-order unit intentionally changed Go MIR source. Of the pinned compiler's 619 input
records, only `internal/mir/lower.go` differs; the pinned compiler executable and SDK
remain unchanged. The candidate is still built with that prebuilt compiler, while
the oracle and native regression use current Go source. Example package version
0.11.0 and its Fairy product sources were unchanged in that unit. Evidence:
`docs/ops/evidence/mir-control-order-fix-20261006.json`. Earlier branch and sequential
evidence remains in `mir-branch-file-cli-20261006.json` and
`mir-sequential-file-cli-20261006.json`. Linux native execution is deferred to the
owner's batch, and self-hosting promotion remains blocked.

The preceding loop unit advanced this example to version 0.12.0. Its comparison passed
1212 CLI executions (606 per profile), with all 83 admitted MIR fixtures matching
the executed Go reference and one deferred case. The 12 direct API tests in
`tests/mir.fy` pass under both native profiles. They include structural checks that
while-break reaches exit, for-continue reaches the checked post before the header,
and unconditional break allocates no unreachable post block. Supplementary file
cases reject break/continue outside a loop, a non-bool for condition, an effectful
post clause and a deferred call inside a loop, with empty stdout and unchanged
input bytes. `samples/mir-loops-demo.fy` emits MIR whose test executor returns 2
after an `i64` loop with continue and a while with break. Both target C emission
profiles and the scoped Go workspace/vet checks pass; Linux native execution remains
deferred. The Go while-condition return defect found by this unit was fixed first
in its own commit, with interpreter and low/optimized MSVC regressions.

No TSV tag/field, SDK API, language revision, runtime symbol or stable CLI command
is added. The pinned compiler executable and SDK remain unchanged; current Go
oracle sources are intentionally distinct from the pinned compiler build inputs.
Evidence: `docs/ops/evidence/mir-loop-file-cli-20261006.json`.

The defer unit advances this example to version 0.13.0. Its full comparison passes
1256 CLI executions (628 per profile), including all 106 MIR fixtures. Fifteen
direct API tests in `tests/mir.fy` pass in both native profiles, with new checks
for scalar capture temporaries, nested LIFO calls and absence of deferred-call
argument edges on a trapped path. The sample's test executor returns 7. Both
Windows and Linux target C emission and the scoped Go workspace/vet checks pass;
Linux native execution remains deferred to the owner's batch. A Go lowering defect
found by the registration-return case was fixed first in a separate commit:
an argument that returns must stop further capture and parent-call registration.
Evidence: `docs/ops/evidence/mir-defer-file-cli-20261006.json`.

Version 0.14.0 adds the scalar verifier. Its comparison passes 1256 CLI executions
in both profiles together, retaining all 106 MIR fixtures with unchanged results
and trace bytes. One native test per profile runs a grouped graph inventory:
four valid graphs and 23 invalid graphs that all pass basic construction, plus two
work-budget rejection cases and one invalid-render buffer-preservation case.
The inventory covers call argument/result typing, missing bodies, duplicate target
identities, uninitialized operands and joins, loop entry versus backedges, forged
unreachable proofs, malformed literals, orphan records and checked-operation bypass.
This grouping avoids recompiling the full candidate for every individual mutation.
Windows and Linux C emission and scoped Go workspace/vet checks pass. Linux native
execution remains deferred to the owner's batch; ownership verification and
self-hosting promotion remain subsequent work. Evidence:
`docs/ops/evidence/mir-verifier-file-cli-20261006.json`.

## Recognition and limits

Current recognition: leading UTF-8 BOM, ASCII identifiers and exact current
keywords, longest-match punctuation, horizontal whitespace/LF/CRLF, line and
documentation comments, nested block comments, arbitrary-width 2/10/16-base
integers, decimal fractions and exponents. `src/numbers.fy` follows the reference
numeric recovery envelope and emits `E0004` for malformed numeric tokens.
`src/strings.fy` recognizes ordinary, byte and C strings, simple/hex/Unicode
escapes, exact hexadecimal cooked bytes and C terminators. Invalid escape,
UTF-8, NUL, newline and EOF recovery preserve the reference diagnostic spans.
Invalid code bytes, non-ASCII code points, lone CR and unterminated block comments
retain lossless tokens with `E0001`. `src/utf8.fy` rejects overlong encodings,
surrogates, out-of-range scalars and truncated sequences without skipping bytes.
Token observations are
`kind trivia startByte endByte cookedLength` numeric fields followed by each cooked
byte as a decimal number, separated by spaces. `true`, `false` and `null` retain
their cooked spelling; numbers retain their canonical decimal bytes. EOF appears exactly once.
These numbers are private test observations tied to `internal/lexer/token.go`.
After tokens, a diagnostic count precedes each diagnostic's length-prefixed code,
repair and token bytes followed by its half-open start/end span. The current
diagnostic subset is `E0001`/`E0004`/`E0005`/`E0006`/`E0010`, including comment and string
`exit` repairs and expected closing tokens. The separate diagnostic store uses
stable source/code/repair/token ordering and preserves duplicate records.

Input is bounded to 32768 bytes, each materialized decimal expansion to 16384 bytes
and cumulative retained cooked data to 65536 bytes. Status 0 means a
complete supported token/diagnostic stream (not syntactically valid code);
2 means unsupported input, 3 allocation failure, and 4 candidate-limit exhaustion.
Scanning finishes before output starts, so scan failure produces no partial stream.
The default diagnostic budget admits 256 ordinary diagnostics. The next attempted
diagnostic stops recognition before allocating that diagnostic and adds one
uncharged `E0010`; the complete lossless exhaustion stream is printed normally.
Output failures still use the existing assertion/trap boundary.

`scanner::report_limits` accepts caller-selected source-byte and token limits
within the candidate's independent storage guards. Exhaustion emits `E0010` once,
retains the denied token and remaining bytes as invalid records, and always emits
an uncharged EOF. Byte limits are checked before recognition; token limits count
trivia and the leading BOM. Native tests compare zero, exact and exceeded limits
with the Go reference, including malformed-token diagnostic ordering.
`scanner::report_budget` also accepts a diagnostic limit. The same counter follows
code, numeric and string recognition, including escapes, invalid UTF-8, raw NUL,
newline and final literal validation. Tests compare both the failure position and
the retained diagnostics, including zero allowance and the default 256/257 boundary.
Independent storage guards still apply if callers request more than 256 ordinary
diagnostics. `src/budget.fy` supplies the work meter as an immutable value returned
by each charge. Native tests compare charge sequences against the Go budget and
check exact-limit acceptance, sticky exhaustion and arithmetic at the maximum `usize` value.
`scanner::report_work` connects the meter to BOM, whitespace, comments,
identifiers, punctuation and non-ASCII code bytes. Native comparisons vary work
limits at every source-byte boundary together with token limits, including the
reference's one-byte progress guard when a multi-byte step cannot start.
Numeric recognition also charges source advancement and complete pre-trimming
decimal expansion before materializing output. Exact expansion boundaries,
cumulative numbers, signed 64-bit exponent endpoints and out-of-range exponents
are compared against Go. Independent expansion/storage caps remain in force.
String recognition carries the same work meter through prefixed openers, raw
UTF-8, escapes, malformed-escape recovery and closing quotes. A multi-byte charge
either advances in full or leaves the cursor unchanged, matching the reference.
Native work-boundary tests also combine token and diagnostic limits and mixed
numeric/string input. All recognition paths now participate in work charging;
independent candidate storage caps and the separate compiler-component promotion
gate remain in force.
Status 2 remains reserved for unsupported extensions;
it must not be interpreted as a successful empty token stream.
The Go lexer stays authoritative. The start evidence and separate promotion gate
are in `docs/architecture/self-hosting-readiness-v0.md`.

`TestWindowsFairyLexerCandidate` compares native low/optimized observations against
the Go reference and separately tests unsupported and limit outcomes. The harness
supplies only raw bytes, not precomputed candidate tokens or recognition tables.

## Complete checked-in source corpus

`TestWindowsFairyLexerCorpus` reads every `.fy` file below `spec/fixtures`, without
filtering by pass/fail status. It compares current and immediately previous lexer
revision modes and deterministic NUL insertion, first-byte deletion and invalid-byte
replacement. Every source/mode/mutation case runs with five budget settings:
defaults, half the source-byte allowance, one token, no diagnostics, and half the
source length in work units. The current corpus has 303 fixture source files plus
the real `std_bytes_quickstart` and `std_io_scalar_quickstart` main sources, both
larger than the former 4096-byte limit. Identifier lengths around 4096 and 16384,
a 6000-byte string, and closed/unterminated 12000-byte comments add large-input
boundaries. All 7670 framed observations match Go under low and optimized Windows MSVC.
Comparison retains exact `E0010` positions, invalid ranges, preceding diagnostics
and EOF for exhausted runs. Newly added oversized fixtures
fail the lane rather than silently disappearing. Embedded Go-only seeds and
arbitrary inputs outside the stated candidate limits are not a full conformance claim.

The maintained `src/main.fy` driver reads private binary stdin frames: one mode
byte and a two-byte little-endian source length. Modes 0/1 use default budgets;
modes 2/3 add four unsigned 32-bit little-endian limits in byte, token, diagnostic,
work order. Odd modes use the previous revision. Exact source bytes follow the
header. EOF is valid only between frames. Invalid mode, truncated framing or more
than 2048 frames exits 8; oversized source exits 4. The harness checks invalid
mode, every truncated extended-header length, missing body and a 32769-byte length
at both profiles.

The harness compiles once per profile through the ordinary workspace compiler,
then reuses that artifact for output-bounded batches of at most 2048 frames. Each batch stays below 60KB
of expected output and retains the existing native driver's 64KB output limit and
trap classification. Fixture coverage remains complete; the two project sources
are a bounded workload sample, not all project files. A single observation that
exceeds the output cap fails explicitly, so 32KB source admission does not promise
that every such source fits this private reporting transport. This is exact record
comparison, not a token hash.

## Own-source comparison without increasing process output limits

`TestWindowsFairyLexerStreamSelfCorpus` reads every `.fy` file in this candidate's
`src` directory, including the scanner and observation driver themselves. Input
admission is 32KB so the scanner with its consumer API fits without slicing source
files into lexically unrelated fragments. Boundary witnesses include 32767/32768
byte identifiers, a long cooked string and an unterminated long comment.

`pages::run` is a separate private one-request driver. Its input is a revision byte
(0 current, 1 previous), source length and observation start as u32 little-endian
words, then exact source bytes and EOF. It rejects truncated headers/bodies,
trailing bytes, source lengths above 32768 and observation starts above 1048576.
Each request collects a full Stream, then prints at most 4096 observation integers.
The harness rejoins every page and compares all token metadata and cooked bytes
with Go, under both revisions and native profiles. It keeps the existing 64KB
process-output guard and compares exact values rather than hashes. Diagnostic
records in this API transport use recognition order.

Paging repeats collection per request; it is bounded comparison transport, not a
streaming lexer or a throughput optimization. The source remains owned in full.
The initial run compared eight source files across 148 pages; the expanded local
corpus now compares nine source files and four boundary witnesses across 196 pages.
This is self-lexing evidence, not full parser conformance or bootstrap promotion.
The original Stream API/page transport and first parser passed hosted Linux and
Windows verification at `ecf428a3` in [Frontend run 34441267986](https://github.com/0disoft/fairyc/actions/runs/34441267986).
That historical receipt does not cover later changes; delivery requires successful
CI for the exact pushed commit.

