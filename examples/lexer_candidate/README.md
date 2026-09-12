# Experimental Fairy lexer candidate

This is the first Fairy-written recognition slice, built by the Go stage0
compiler for Windows AMD64 MSVC. It is not a compiler replacement or stable CLI.
`src/scanner.fy` recognizes original bytes directly; Go does not supply its tokens.

## Owned consumer API

`scanner::collect` consumes an input `bytes::Buffer` and the same explicit revision
and byte/token/diagnostic/work limits as recognition. It returns
`Result<scanner::Stream, i32>` without console output. Stream owns the original
source, token storage, cooked bytes and diagnostics; input is consumed on both
success and failure. Status 3/4 means allocation/storage failure and publishes no
Stream. Reference-budget exhaustion instead returns a complete lossless Stream
containing `E0010`.

`scanner::release(stream)` explicitly consumes and destroys the result. Ordinary
scope exit also destroys it; the API does not use `discard` as an early-release
operation.

`token_count`, `token_info`, `token_raw`, `token_cooked`, `source_view`,
`diagnostic_count`, `diagnostic_info` and `diagnostic_token` provide read-only
access. Metadata is copied; returned byte views borrow the Stream and cannot
outlive it or coexist with its move/drop. Indexed access retains the vector bounds
trap. Diagnostic access uses recognition order, while the legacy report sorts for
display. Numeric kinds/codes and cooked encoding remain experimental implementation
contracts, not a stable SDK or public ABI. Native consumers exercise transfer and
budget exhaustion; compile-fail tests cover local escape and use after/drop during
a live borrow.

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

## Bounded syntax parser

`syntax::parse` consumes a Stream and an item limit no greater than 512. It returns
an owned `Parsed` containing declarations, expressions, binding entries,
statement trees, written types, struct/resource fields and item attributes, retaining the entire original Stream. This
candidate recognizes `module path;`, `import path as alias;`, and functions with
optional `pub` visibility (retained in `public`, `public_start`, `public_end`;
private declarations use false and zero spans). The modifier is not discarded
from unsupported declaration forms. Functions preserve written parameter and
result types using ordinary Fairy spelling (`fn i32 f(i32 x) { return x; }`).
Parameter lists allow a trailing comma. Bodies admit empty blocks,
`let`/`var` bindings with optional type annotations and required initializers,
and `return` with an optional expression. Statement-only nested blocks, expression
statements, ordinary and compound assignments, and `if (...) { ... } else if (...)
{ ... } else { ... }`, `while (...) { ... }`, `break;` and `continue;` are admitted.
Classic `for` loops and other statements
remain outside this subset.
It does not resolve names or imports, validate types, module-to-file mapping or
return obligations, admit the complete Stage 0 grammar, or produce executable MIR.

Ordinary functions, structs and resources accept ordered attributes before optional `pub`.
`attributes.fy` retains qualified names, source spans and ordered path, integer
or string arguments (including empty argument lists and trailing commas).
Declaration `attribute_start`/`attribute_count` selects the global attribute range.
`attribute_at` exposes each attribute's name span and argument range;
`attribute_argument_at` and `attribute_argument_text` expose argument kind
(1 path, 2 integer, 3 string), span, token and original spelling.
The parse shares limits of 256 attributes and 1024 arguments, checked before
allocation; exhaustion returns 4 without a tree. Failed items roll both tables
back, and recovery does not attach a rejected prefix to a later declaration.
Attribute semantics, effect validation and permissions remain outside this parser.

The private `types` module recognizes named/qualified types, `ref`, `ref mut`,
grouping, unsized slices and fixed arrays. It preserves `ref u8[]` versus `(ref u8)[]` without
resolving names. `Declaration.result_type` and `Entry.type_root` are one-based
roots; an inferred binding uses zero. `type_at`, `type_count` and `type_text`
expose postorder nodes and exact source slices. Node kinds are 1 path, 2 reference,
3 mutable reference, 4 slice and 5 group; wrappers point to `left`, `right` is zero.
All declarations share 4096 type nodes and a 32-active-call type-depth limit.
Failed declarations roll back type nodes with the other tables.
Raw pointers and foreign function types remain unsupported.

Named types also accept nonempty generic argument lists, trailing commas and
nesting. Kind 6 retains the base path in `left` and a reverse argument-link tail
in `right`; kind 7 links an argument root in `left` to its predecessor in `right`.
`type_argument_count` and `type_argument` expose source order. Generic nodes and
links share the same type-node budget, and argument nesting shares type depth.
The type cursor consumes each half of `>>` at its actual byte boundary without
rewriting source tokens or changing expression shifts. Stray closing halves fail.
This does not admit generic declarations, specialization or type resolution.

Ordinary `struct Name { ... }` declarations use declaration kind 4, with optional
`pub` on the struct and individual fields. `field_start`/`field_count` select a
zero-based source-ordered range in the separate `field_at`/`field_count` table.
Each field preserves its full/name/public spans, visibility and `type_root`.
Structs have no body or result-type root. All structs share a 1024-field limit;
failed declarations roll back fields and types together. Duplicate-field legality,
layout and ownership are not checked. Non-generic resources use declaration kind 6
and share the same field/type arenas and 1024-field budget. Optional `cleanup`
bodies reuse the statement parser and are retained by `cleanup_body` in the shared
statement arena. `cleanup_start`/`cleanup_end` span the keyword through closing
brace; all three zero means unwritten. Empty cleanup has a real empty block.
Cleanup entries use the declaration's source-ordered entry range. Resources still
have no function `body` or result type. Body nodes, entries and nesting share the
same budgets as functions, and malformed cleanup rolls back the whole declaration.
Declaration attributes and `pub` spans are preserved as for structs; imports
cannot follow resources. Generic declarations, attributed cleanup,
and generic record expressions remain unsupported. No resource ownership or
destruction behavior is implemented by the candidate. Malformed resources roll
back fields, types and attributes before recovering to the next declaration.

Expressions admit integers, `true`/`false`, strings, byte strings, qualified identifier paths, calls,
nested parentheses, prefix
`+ - ! ~ *`, and the grammar's binary operators from multiplicative through
logical-or. C strings and other literals
remain outside this subset. Precedence follows `spec/grammar/stage0.ebnf`;
binary operators associate left, while unparenthesized comparison chains fail.
The private postorder node kinds are 1 literal, 2 path, 3 group, 4 prefix,
5 binary, 6 call, 7 argument link, 8 field, 9 index, 10 record and 11 record-field link. Field/index receivers are
in `left`; index expressions are in `right`. Field text is the exact member token.
Calls, fields, single indices and range slices form repeatable postfix chains.
Kind 26 retains a slice receiver in `left` and a kind-27 bounds node in `right`.
Bounds retain optional start/end expression IDs in `left`/`right`; zero means a
written omission, not a synthesized numeric value. The bounds token is `..`;
its span excludes brackets, while the slice spans the complete receiver and suffix.
All four forms `a[x..y]`, `a[..y]`, `a[x..]`, `a[..]` share existing node/depth
budgets. No bounds evaluation, range ordering or borrow validation occurs.
One-based child IDs (zero absent), half-open source-byte spans and
source token spelling preserve structure without evaluating or resolving names.
Declaration `expression` is the last top-level return root, or zero if absent/empty.
`expression_count`, `expression_at` and `expression_text` expose read-only views.
The independent `bodies.fy` module owns recursive expression, statement and block parsing;
`parse_tokens.fy` owns exact token spelling, trivia skipping and operator
classification. Existing body helpers delegate to it; cursor mutation and mutual
recursion remain in `bodies.fy`, without an import back from the token module.
`syntax.fy` retains declaration rollback and the existing query facade.
`literal_kind` exposes the source token kind. `literal_cooked` borrows the
scanner's canonical cooked representation: string/byte-string values are lowercase
hexadecimal bytes, not a newly decoded buffer. Raw spelling and byte spans remain
unchanged. Lexical diagnostics still reject the entire parse with status 5.
Path text is its exact source slice, including intervening trivia. Calls retain
the callee in `left` and a reverse-linked argument tail in `right`; each argument
link points to its expression in `left` and its predecessor in `right`.
`call_argument_count` and `call_argument` expose the original argument order.
Empty calls, trailing argument commas and repeated call suffixes are admitted.
All call/argument nodes share the existing node limit, and argument recursion
shares the expression-depth limit. Generic specialization execution remains unsupported.
Ordinary records retain the path root in `left` and the reversed field-link tail
in `right`. Field links retain the value in `left`, previous link in `right`,
member token and full initializer span. `record_field_count` and `record_field`
expose field-link IDs in source order; `expression_text` returns each field name.
Empty records, nesting and trailing commas are admitted. Duplicate fields,
field types and layout are not checked. Record and field-link nodes consume the
same expression budget; no separate record arena or unbounded side table exists.
Failed declarations roll back their nodes before recovery. A shared 4096-node
ceiling and 64-active-expression-call limit return status 4 without a result.

Value-block expressions have kind 12: `left` selects a statement-block ID and
`right` is absent. The block's `expression` field is its optional tail root,
separate from its ordered statement links. A semicolon keeps an expression in
the statement list; an unwritten tail remains zero. Function bodies still require
statement terminators. Nested value blocks share the 32-block-depth and
64-active-expression bounds instead of resetting them at the expression/body edge.
Entry IDs are reserved before parsing an initializer, so nested entries cannot
overtake their source owner or bypass the 1024-entry budget. Completed records
may be stored in postorder; `entry_at` resolves the source-order ID with a bounded
scan. Statement entry references use those same IDs.

Conditional expressions use kind 13 (`left` condition, `right` branch record).
Kind 14 retains the then statement-block ID in `left` and the optional else
expression ID in `right` (kind 12 block or kind 13 nested conditional).
Conditions, branch order, written tails and full byte spans are retained.
An unwritten else stays zero; branch type agreement and value availability are
semantic questions. Else-if recursion remains bounded by the expression limit.

Switch expressions use kind 15 (`left` subject, `right` reversed case-link tail).
Kind 16 is a shallow leading-dot variant pattern with an optional kind-17 binding
in `left`; kind 17 retains an identifier or `_` spelling. Kind 18 is a case
(`left` pattern, `right` statement-block ID), and kind 19 links cases in reverse.
`switch_case_count` and `switch_case` expose case IDs in written order.
Variant names and binding names retain their own tokens; pattern, case and switch
spans remain distinct. Empty and duplicate cases are syntax-only observations:
exhaustiveness, reachability and payload/type validity are not checked.
Record patterns, whole-value wildcards and multiple positional bindings remain
unsupported. All pattern/case/link nodes use the same 4096-expression-node budget.
Case bodies share block and expression depth limits with their enclosing tree.

Contextual variant names such as `.ok`, `.err` and `.none` use kind 28, with no
children. The full span includes the leading dot and intervening trivia; the
token and `expression_text` retain the identifier. Existing call nodes wrap this
callee for `.ok(value)`, preserving ordered arguments and shared budgets. Type
inference and variant validity are not checked. Contextual record construction
`.Variant { ... }` remains unsupported.

Explicit type calls are recognized only after a path and when the closing angle
list is followed by a call, using read-only lookahead capped at 256 significant
tokens. Trivia does not consume that token allowance. Kind 20 retains the path in
`left` and a type-argument link tail in `right`; the ordinary kind-6 call owns
this specialization node as its callee. `specialization_argument_count` and
`specialization_argument` expose written type roots in source order. Type
arguments share the declaration type arena and its limits. Nested `>>` is split
only by the existing type cursor; ordinary comparisons and shifts are unchanged.

Reference expressions use kind 21 for `ref` and 22 for `ref mut`, with the
operand root in `left` and absent `right`. Full spans retain intervening trivia
and the mutable marker; the source token retains `ref`. Postfix operations bind
inside the reference, while binary operators remain outside. These nodes consume
the existing expression-node and active-depth budget. Borrow validity, operand
mutability and lifetime rules are not checked by this syntax candidate.

Kind 23 is a written `as` cast: `left` is the operand and `right` is the target
type root. Casts associate left, below prefixes and above multiplication; their
operator tokens and full spans remain available. Targets use the shared type
parser and budget. No conversion, range check or target-type validation occurs.

Array lists `[]`, `[value]` and `[a, b,]` use kind 24 with absent `left` and a
kind-25 element-link tail in `right`. Each link retains its value in `left` and
the previous link in `right`. `array_element_count` and `array_element` expose
source-ordered values without flattening nested arrays. Array spans include both
brackets; element spans exclude separators. Both kinds share the expression-node
and depth budgets. Repetition `[value; count]` instead uses kind 29, retaining the
single value expression in `left` and a written integer/path count in `right`.
No element links, expansion, count evaluation or value duplication are performed.
Qualified count paths are preserved; arithmetic, calls, casts and parenthesized
count expressions remain outside this narrow subset. Element type compatibility,
constant validity and array evaluation are not checked by the candidate.

`entry_at` and `entry_count` expose source-ordered parameters and statements,
bounded to 1024 entries across the entire parse. Declaration `entry_start` and
`entry_count` select a zero-based range. Entry kinds are 1 parameter, 2 let,
3 var and 4 return; each preserves complete, name and optional type byte spans
plus its expression root. Zero name/type spans mean unwritten, not synthesized.
Errors roll back entries and expression nodes together. Duplicate names, type
compatibility, unreachable returns and binding resolution are semantic concerns
and are not checked here.

`Declaration.body` is a one-based root in a separate postorder statement tree;
zero is reserved for module/import declarations. `statement_at` and
`statement_count` expose nodes: 1 block, 2 binding, 3 return, 4 assignment, 5 if,
6 expression statement, 7 list link, 8 while, 9 break, 10 continue and 11 assert.
Assert retains its condition in `expression` and its full keyword-through-semicolon
span. It requires parentheses and a semicolon, including in value blocks; it is
not a call or tail expression. Condition type and runtime assertion behavior are
outside this syntax layer. Its nodes share existing expression/statement budgets.
Kind 12 is a `discard` statement with a required operand in `expression` and a
full keyword-through-semicolon span. It is neither a return entry nor a value
block tail. No must-use, ownership transfer or destruction behavior is inferred.
While preserves its condition in `expression` and its body in `first`; break and
continue have no children. Loop-context legality remains a semantic concern.
Binding/return `entry` is a one-based
reference into the existing entry table. Assignment `target` and `expression`
refer to distinct expression roots; `assignment_operator` returns its raw spelling.
An if node's `expression` is the condition, `first` the then block, and `second`
the optional else block or else-if node. A block's `first` is a reverse-linked
statement-list tail and `second` its direct child count; list links use `first`
for the child and `second` for the previous link. `block_statement_count` and
`block_statement` return direct children in source order, without flattening nested
scopes. The Go oracle's expression-statement wrapper around block/if is omitted;
the underlying construct, order and byte spans are preserved.

The statement tree has a shared 4096-node limit (including list links) and a
32-frame block/conditional depth limit. Failures roll back statements, expressions
and entries together. Assignment-target legality, shadowing, condition types and
branch return obligations remain semantic checks, not parser acceptance criteria.

Declarations preserve source order, complete spans and name/alias spans. Problems
retain exact source-byte spans and private category codes (1 module, 2 import,
3 function, 4 struct or unsupported source item, 6 resource). A failed declaration never becomes an
ordinary declaration node. Recovery rescans its bounded token range with brace
depth before resuming at a following import/function; nested functions cannot
escape a malformed enclosing body. Original trivia and rejected source remain
available through the retained Stream rather than being reconstructed from nodes.

Problems additionally retain `expected` and the actual source `token` index.
`problem_actual` borrows its exact raw spelling (empty at EOF); `expected_text`
renders the private expectation code: 0 supported syntax, 1 identifier, 2 type,
3 expression, 4 `(`, 5 `)`, 6 `{`, 7 `}`, 8 `;`, 9 `=`, 10 `as`, 11 `]`,
12 `module`, 13 `>`, 14 `.`, 15 attribute argument, 16 array length,
17 `:`, 18 `case`. Code 0 deliberately remains general for unsupported constructs;
these are diagnostic hints, not inserted tokens, repairs or semantic acceptance.
Expectations reset before each source item so recovery cannot leak an old hint.

Allocation failure returns 3; item or 32-problem exhaustion returns 4; lexical
diagnostics return 5. These failures consume the input and publish no Parsed value.
Callers that need lexical diagnostics must inspect the Stream before passing it
to the parser. A successful Result can contain syntax problems: clients must check
`problem_count`, and must not treat this partial syntax tree as a compiler
acceptance decision. `declaration_at`, `problem_at` and `source_view` are read-only.
The scanner's source/work bounds plus monotone parsing and one bounded recovery
rescan limit work; expression recursion is bounded separately as above.

Native tests compare function/import spans, expression/call structure, ordered
parameters and bindings, nested blocks, conditional branches and assignment
operands with normalized Go ASTs. They also exercise missing headers/delimiters,
declaration recovery, import order, nested malformed bodies and byte, depth, node,
entry, diagnostic and work boundaries. Large repetitive inputs are constructed
at runtime without reducing their tested sizes. Focused Mustflow lanes separate
parser checks, Stream/self-lexing, corpus/budget checks and heap accounting;
hosted Go tests still discover all tests normally.

## Own-source syntax coverage

Fixed-array type nodes preserve an element root and an unevaluated integer or
qualified-path length root. Private kinds 8/9/10 represent array/integer length/path
length; all consume the existing 4096-node type budget. Empty brackets remain
slices. Length arithmetic, evaluation and size validity are not parser decisions.

All fifteen current source files, including the integrity checker and token helpers, parse without
syntax problems in both Windows profiles. The [literal/item report](parser-literal-items-2026-09-12.md)
records the latest bounded fixture results; the [control syntax report](parser-control-items-2026-09-12.md)
retains the preceding milestone. The [preceding evidence report](parser-evidence-2026-09-11.md)
records the initial declaration comparison, integrity checks and fixture baseline.
The [earlier coverage report](parser-coverage-closure-2026-09-11.md)
records the preceding thirteen-file milestone.
The corpus test explicitly requires complete syntax for these
named files and fails on diagnostics, budget exhaustion or file removal. New
files remain observations until independently admitted to this regression gate.
For admitted files, the native declaration list is also compared with the Go CST:
kind, source order, complete byte span and declared name/alias span must match.
This is a declaration-level comparison, not full expression-tree equivalence.
The private `integrity::valid` checker additionally validates expression/type/
statement kinds, required and optional arena references, backward same-arena
links, token/source bounds, block link counts and declaration ranges. It runs on
native corpus results, including incomplete results. It does not prove semantic
validity, cross-arena graph acyclicity, unique ownership or full source coverage.
Synthetic-node probes exercise missing, out-of-range, self-referential and
wrong-kind links without mutating the published parse result.
The fixture observation lane reads the checked-in parser manifest unchanged,
with at most 32 cases and 32768 source bytes per case. It separates supported
syntax, unsupported valid syntax, matching rejection and missing file/module
context. Unexpected false acceptance or budget exhaustion fails the lane rather
than becoming a supported result. Both profiles must return identical observations.
The [statement/resource report](parser-coverage-statements-2026-09-11.md) records
the earlier eleven-file result; older baselines are retained as historical evidence.
Full-file exhaustion and separate first-blocker prefix probes are distinguished.
This must not be confused with the lexer's complete source-token comparison.

Postfix propagation uses private expression kind 30 with one operand and the
written question-mark token. Chains, grouping and prefix/infix precedence are
compared with Go AST shapes. This preserves syntax only, not error propagation
execution semantics.

Private declaration kind 7 preserves a written `const` declaration: `result_type`
is its declared type root and `expression` is its initializer. Visibility,
attributes and declaration/name spans retain the existing declaration contract.
Initializers are parsed, not evaluated.

Private statement kind 13 preserves `defer expression;`, including its operand,
source span and position in the enclosing block. Execution timing, call validity
and cleanup order remain semantic/lowering responsibilities.

Private statement kind 14 preserves a classic `for`: optional condition in
`expression`, body in `first` and a kind-15 header in `second`. Header children
are optional initialization (binding/assignment) and post assignment. Zero means
unwritten, not a synthetic true expression. The initialization statement span
includes its terminating semicolon; the header span includes both parentheses.
The body is not desugared into a while loop. Arbitrary expression-only header
clauses remain outside this grammar-bounded slice, even where Go syntax recovery
accepts them for later semantic rejection.

Floating tokens (kind 6) are preserved as literal expressions with their original
token, byte span and lexer-normalized spelling. The parser performs no numeric
conversion or floating-point evaluation.

C-string tokens (kind 9) are also literal expressions. Their cooked view retains
the lexer-produced terminal NUL in its hexadecimal representation; embedded NUL
and invalid escapes remain lexical errors. No pointer or FFI behavior is added.

Private declaration kind 8 preserves module-level `static_assert(expression);`.
Its name spans and result type are zero because no name or type was written;
`expression` is the required condition root. Attributes and public syntax retain
the declaration contract. No constant evaluation occurs, and `static_assert`
remains an ordinary identifier outside the module-item dispatch position.

## Cost baseline

`scanner::storage` reports source, separately owned cooked and diagnostic-token
bytes plus record counts. It does not report allocator capacity or peak heap/RSS.
The opt-in `fairyc_go_measure_lexer_stream` lane records five native driver
roundtrips after one warmup for each scenario/profile; ordinary tests skip it.
See [the measured baseline](cost-baseline-2026-09-10.md) for raw samples, exact
build identities, included overhead and the limits on interpretation.

The separate [SDK heap observation](heap-baseline-2026-09-10.md) counts actual
allocator requests, peak simultaneously live requested bytes, retained capacity
and balanced destruction, including three injected failures. Its instrumented
builds are not latency benchmarks or RSS measurements.

The [sparse cooked-storage comparison](sparse-storage-2026-09-10.md) preserves the
old baseline and records a 41.06% reduction in peak requested heap bytes for the
same token-heavy input, with a small allocation-count increase. The private token
layout changed; the Stream consumer API and logical payload metrics did not.
