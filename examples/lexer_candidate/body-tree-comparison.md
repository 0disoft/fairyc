# First canonical function-body comparison

This is a private, disposable test transport between the Fairy candidate and the
Go reference AST. It is not a public AST format, parser replacement, semantic
adapter or decoder. Its first proof walks the complete body of one function with
nested bindings, blocks, if/else-if, while, assignments, calls and returns,
including explicitly written binding types, casts, specialized generic calls,
field access, indexing, slicing, arrays, record construction, references,
propagation, unsafe blocks, switch pattern decomposition, classic for loops and
assert/defer/discard/burn statements and region statements.

A separate function-tree mode wraps that body with a basic ordinary function
header: ordered attributes, function/name spans, owner, `pub`/`unsafe` modifiers,
generic parameter declarations, result type, and parameter/name spans with types
in source order. Attribute arguments compare path/integer/string syntax only;
cooked strings and attribute meaning remain excluded.
Documentation trivia remains bound raw source, not a documentation AST export.

## Encoding

The two independently implemented writers emit space-terminated unsigned decimal
integers. The prefix is `1 source_byte_count source_byte...`; `1` identifies this
test format revision. Original bytes are included once, without normalization,
so every following span is bound to the exact input, including trivia and CRLF.
Type support adds previously unused tags without changing existing records. There
is no persisted artifact, older-reader compatibility window or migration promise.

The remaining stream is a preorder tree. Each record has the following fields:

`kind start_byte end_byte child_count name_start_byte name_end_byte operator_size operator_byte...`

Spans are original half-open byte ranges. Name spans are present on bindings,
field access, record fields, named pattern fields, region names, functions and parameters;
other records use zero. Prefix, binary and assignment operators carry their actual
operator bytes, independently of the source-span envelope; other nodes use length
zero. Child counts and preorder determine all structural edges;
no arena indices, list-link nodes, pointers or host enum values are exported.

| Tags | Meaning and child order |
| --- | --- |
| 90 | Function: ordered attributes, optional pub, optional unsafe, optional owner, optional generic declaration list, result type, ordered parameters, body block; function-name span |
| 91 | Parameter: written type; parameter-name span |
| 92 / 93 / 94 | Present pub / unsafe / owner leaf: exact keyword or owner span, no children and zero name span |
| 95 | Generic declaration list: angle-list span, parameter declarations in source order |
| 96 | Generic parameter declaration: identifier span, no children and zero name span |
| 400 | Attribute: exact attribute and name-path spans, arguments in source order |
| 401 / 402 / 403 | Attribute path / integer / string argument: exact raw span, no children and zero name span |
| 100 | Block: statements in source order, followed by an optional tail expression |
| 101 / 102 | Inferred let / var binding: initializer |
| 103 | Return: optional operand |
| 104 | Assignment: target, value |
| 105 | Expression statement: operand |
| 106 | While: condition, body block |
| 107 / 108 | Break / continue: no children |
| 109 / 110 | Explicitly typed let / var binding: written type, initializer |
| 111 | For loop: initializer, condition, post, body block; always four children |
| 112 / 113 / 114 | Absent initializer / condition / post: no children, zero span |
| 115 / 116 / 117 / 118 | Assert / defer / discard / burn statement: exactly one operand |
| 119 | Region: backing expression, body block; exact region-name span is present |
| 200 / 201 | Literal / path: no children; spelling comes from the bound source span |
| 202 / 203 | Group / prefix expression: operand |
| 204 | Binary expression: left, right |
| 205 | Call: callee, then arguments in source order |
| 206 | Block expression: body block |
| 207 | If expression: condition, then block, optional else block or else-if expression |
| 208 | Cast: operand, written target type |
| 209 | Specialization: target path, then type arguments in source order |
| 210 | Field access: target; name span identifies the selected field |
| 211 | Index: target, index expression |
| 212 | Slice: target, start bound, end bound |
| 213 / 214 | Omitted start / end bound: no children; distinct from a literal zero |
| 215 | Array list (including empty): elements in source order |
| 216 | Record construction: target path, type arguments, fields in source order |
| 217 | Record field: value; exact field-name span is present |
| 218 | Repeat array: value, written count expression; no expansion or folding |
| 219 / 220 | Immutable / mutable reference expression: operand |
| 221 | Propagation: operand |
| 222 | Unsafe expression: body block |
| 223 | Switch: subject, then case arms in source order |
| 224 | Case arm: pattern, body block |
| 225 / 226 / 227 | Unit / positional / named-field pattern: variant path, then bindings or fields |
| 228 | Named pattern field: binding; exact field-name span is present |
| 229 | Pattern binding (including `_`): no children; spelling from its own span |
| 301 | Named type path: no children |
| 302 / 303 | Reference / mutable reference type: element |
| 304 / 305 | Slice / grouped type: element |
| 306 | Generic type: base path, then type arguments in source order |
| 308 | Fixed array type: element, written length |
| 309 / 310 | Integer / path array length: no children; no folding or resolution |
| 311 / 312 | Raw pointer / mutable raw pointer type: element |
| 315 | C function type: result, then parameters in source order |

Go's block-expression wrapper is omitted specifically for if branches, while
bodies, unsafe bodies, case bodies and region bodies, where the common format has an explicit block child. The candidate's
branch-pair helper is likewise omitted. A standalone block expression retains tag
206. Go's operandless break/continue self-expression placeholders are not separate
children; the jump statement itself remains visible. Statement wrappers, grouping,
order and every source-written body expression remain visible.
Literal cooking and semantic analysis are deliberately not encoded or claimed.
Type grouping, mutability, generic nesting, array-length spelling and C function
parameter order remain visible. The Go writer uses AST type structure and obtains
the exact generic base-path span from its originating CST, never by reparsing type
text. The Fairy writer omits its reverse-list helper nodes and restores source
order. Those helpers are not exported records; C parameter-link scans are capped
at 255 parameters, and the shared record budget still charges every emitted type.
Specialized calls retain both the call (205) and specialization (209) nodes: the
callee subtree contains the target and written types, followed by the call's value
arguments. Casts retain their operand before the written type; chained casts are
not flattened. Neither operation resolves names, converts values or checks types.
Slice helper pairs are omitted from the common tree, but all three child roles
remain explicit. An omitted start uses the Go AST's `..` token span; an omitted
end uses its zero-width span at the closing `]`. These are syntax-origin markers,
not fabricated expressions or missing-token recovery. The candidate derives those
same origins from its bounds and slice envelopes. Field names use the final token's
exact byte range, excluding intervening trivia. Access chains are not flattened;
the comparison does not prove runtime indexing safety or assignment validity.
Record fields retain their source order, including duplicate names; no semantic
duplicate-field validation is claimed. Go field-name origins come from its CST;
the candidate reads its existing token origin through a read-only helper appended
only to the temporary test copy of `syntax.fy`. The shipped parser API and private
`Parsed.source` field remain unchanged. Generic record targets omit the candidate's
specialization wrapper and place its type arguments directly before the fields,
matching the Go AST. Dot-prefixed constructor paths retain their complete span
under tag 201. Array list links and record-field links are never exported as edges.
Go's prefix operator values `&` and `&mut` for written `ref` and `ref mut` map
to tags 219 and 220 instead of the ordinary prefix tag. Their operator payload is
empty: the tag preserves mutability and the source span preserves original words
and trivia. Propagation has its own tag, retaining its position in call/access
chains. Unsafe remains distinct from a normal block even when empty or nested.
All three forms charge the same record/depth budget as their operands and blocks.
No borrow validity, error propagation semantics or unsafe-operation permission is
checked or granted by this syntax-only comparison.
Switch subjects, case order and bodies remain explicit; reverse case-link nodes
are omitted. Patterns are not record constructions: the variant path and each
binding have separate spans, and named fields retain their written order, including
duplicates. Go's pattern AST omits the empty named-field delimiter distinction,
so its owning CST closing token distinguishes `.N {}` (227) from `.N` (225).
Pattern field-name origins likewise come from CST-owned identifier tokens rather
than reparsing source text; binding spans retain the exact terminal spelling.
The candidate uses existing pattern tags and token origins. No exhaustiveness,
duplicate-case, duplicate-binding, type, scope or execution validity is claimed.
For loops retain four positional roles even when header clauses are absent. Tags
112/113/114 represent absence, not source-written expressions; their `(0,0)` span
does not invent a source location. An absent condition is never replaced by `true`.
Written clauses keep their structure, names and operators. The initial statement's
span includes its semicolon, matching the candidate's statement parser; the Go
writer obtains that exact envelope from the owning CST initializer clause and
copies the statement value without mutating the AST. The post span excludes `)`.
Header punctuation remains in the bound original source, but clause/header helper
nodes are not separately exported. Initializers are binding or assignment forms;
posts are assignments, matching the candidate parser's existing restriction. The
comparison does not implement execution order, loop scope or continue routing.
Assert, defer, discard and burn retain separate tags and their complete statement
spans, including written terminators. Each visits exactly one operand through the
same expression walker and shared budget, without desugaring into calls or ordinary
expression statements. Their syntax-only fixtures may intentionally lack semantic
validity: this comparison proves no assertion outcome, delayed-call scheduling,
must-use handling, ownership transfer or resource destruction.
Regions preserve their written name, backing expression and body in source order.
The Go writer reads the name's identifier token from the owning CST region node;
the candidate uses its existing region-name node's span. That name is metadata,
not an expression child or a new allocation node. Repeated names and nested regions
are compared as written without resolving scope, allocating storage or validating
lifetimes, ownership and destruction. Bodies retain ordinary statement/tail rules;
the exporter does not turn a region into a value-producing block.

## Failure and resource boundary

Both writers reject unhandled nodes instead of skipping them, so a future or
otherwise unmapped node cannot produce a successful partial comparison. The
[coverage ledger](body-tree-coverage.md) records the bounded production-family
audit, intentional exclusions and remaining verification gaps.
The body-only input and shared budgets below still apply independently of the
parser's grammar support.

Input must be an error-free module plus exactly one ordinary function, within
32768 source bytes. Header/parameter trees are not the root of this body-only
format. Exports are bounded to 256 records and depth less than 64, shared across
statements, expressions and types without resetting at a type boundary. Existing
parser limits apply first (including type-parser recursion less than 32); this
export extension does not widen them. The candidate
first checks the parser integrity and diagnostics, then preflights the entire
read-only walk with output disabled. Only a successful preflight writes the
header and repeats the same immutable traversal. Unsupported, malformed or
over-budget inputs return nonzero with empty stdout; output I/O failures cannot
be treated as successful artifacts.

The Go writer returns either the complete canonical string or an error and an
empty string. Tests compare native stdout byte-for-byte against that reference,
not a hash or just declaration counts. The transport is neither loaded nor cached.

## Owners and verification

- `internal/cli/fairy_parser_body_tree_test.go`: independent Go AST writer and
  native differential/rejection cases.
- `internal/cli/fairy_parser_body_tree_writer_test.go`: read-only Fairy writer
  compiled inside the temporary test package; no shipped candidate module added.
- `TestWindowsFairyParserBodyTree`: the configured
  `fairyc_go_test_parser_body_tree` lane runs only this bounded comparison.

`fairyc_go_test_parser_body_tree` passes in both native profiles (18.407s).
The bounded matrix contains 112 complete bodies per profile, including
ordinary, typed and generic-call exact 256-record boundaries, nested type families,
chained casts, generic/value argument order, ordinary `<` comparison, trivia/CRLF
and depth 63 reached through binding, cast and specialization type trees. Fifty-seven
rejection cases cover malformed bindings, casts, calls and returns,
an extra declaration or function, 257/259/261 records and depth 64 across
those same three type-entry roles. Access cases additionally compare field/index/
call chains, assignment targets, all four slice-bound presence combinations,
explicit zero versus omission and comments around delimiters. Slice records also
exercise the exact 256-record boundary and reject 261 records; field and slice
chains pass at depth 63 and reject depth 64. Malformed field names, empty indices
and an unclosed slice fail without output. Array and record cases cover empty,
nested, trailing-comma and repeat forms, generic and dot-prefixed constructors,
non-alphabetical and duplicate fields, and comments before field names. A wide
array and repeated records each pass at exactly 256 records; 257/261 records,
malformed array separators and missing record values fail. Reference mutability,
propagation/access composition, empty and nested unsafe blocks, comments and CRLF
also compare exactly. A combined unsafe/reference/propagation witness passes at
256 records and rejects 257; reference chains, propagation chains and nested unsafe
bodies pass at depth 63 and reject depth 64. Missing reference operands, prefix
`?` and unsafe without a block fail. Switch cases cover all three pattern forms,
empty named versus unit patterns, wildcard and duplicate named fields, trivia/CRLF,
empty switch, construction-to-decomposition and nested case bodies. A 63-arm
witness passes at 256 records; adding one body node rejects 257. Nested switch
bodies pass at depth 63 and reject 64. Empty positional bindings, missing named
bindings and a missing case colon are rejected. For-loop cases cover all eight
header-presence combinations, inferred/typed declarations, assignment initializers,
compound posts, comments before the initial semicolon, nested for/while bodies and
break/continue. Repeated empty loops pass at 256 records and reject 257; nested
loop bodies pass at depth 63 and reject 64. Missing separators, declaration posts,
malformed conditions and a missing loop body fail. The four operand statements
compare with comments/CRLF, generic calls, propagation, field access and nested
loop/switch/unsafe bodies. Their mixed witness passes at 256 records and rejects
257; each operand role passes at depth 63 and rejects 64, and each missing operand
is rejected. The first new fixture used the reserved word `cleanup` as a callee;
it was corrected to an ordinary identifier without changing grammar. Regions cover
empty and nested bodies, repeated names, comments/CRLF, call/index/field backing
expressions and for/switch composition. Repeated regions pass at 256 records and
reject 257; nested region bodies pass at depth 63 and reject 64. Missing names,
`in`, backing expressions and bodies are rejected. The coverage audit adds every
operator alternative and literal class, mixed-precedence and header/trailing-comma
witnesses, plus import and oversized-source rejection. Rejected native
exports must have empty output and nonzero status. A name-only `let i32 =1;`
is valid inferred-binding syntax, not a malformed typed-binding test. Deep groups
hit the existing parser recursion limit before the exporter depth limit, so slice
suffixes isolate that exporter boundary. Formatting and the scoped
Mustflow contract are checked separately; no full compiler/release suite is needed
for test-only writers. No stable compiler, grammar, ABI, SDK, package format or
default CLI path changes. Function mode has 37 positive and 39 rejection
witnesses per profile, including empty/trailing-comma parameter lists, nested and
C-function types, non-alphabetical and duplicate names, trivia/CRLF and malformed
headers. Header and body share the same 256-record/depth-below-64 budget. Combined
header/body cases pass at 256 records and reject 257; result, parameter and body
paths pass at depth 63 and reject 64. Only parameter entries (kind 1) in the
declaration's entry range are emitted as parameters; body bindings/returns are
visited through the body. Both preflight and emission traverse the entire function.
The original 112-body/57-rejection mode and its boundary witnesses remain unchanged.
Neither mode is a public API or a semantic adapter.

Function modifier/owner leaves are omitted when absent (no placeholder); their
fixed role order precedes the result type, rather than following lexical order.
They consume the same record budget as parameters and body. All eight presence
combinations are checked, including identical owner/result/function spellings,
comments containing decoy keywords, CRLF and non-ASCII trivia. Decorated functions
pass at 256 records/depth 63 and reject at 257/depth 64. Duplicate or reordered
modifiers and missing/multiply-qualified owner names fail without output.
Modifier spans come from the candidate declaration fields and the Go CST tokens
before the declaration's first `fn`. Go AST lowering likewise restricts modifier
flags to this prefix, skipping attributes. The comparison checks that the lowered
flags agree with the independently read origins. Separate result-type witnesses
prevent confusing that keyword or an unsafe body with a declaration modifier.
An AST regression reproduced the former keyword-before-name scan incorrectly
setting Unsafe for a C function result type. Separate semantic tests preserve the
existing unsupported C function value types while requiring E5101 for an unguarded
foreign call and no such diagnostic inside explicit unsafe boundaries. This does
not admit C function values to semantic analysis or native execution.

Generic declaration lists are omitted when absent; an empty written list is
invalid, not an absent-list spelling. The Go writer checks AST name count and order
against the owning CST list and reads its list/name spans. The candidate uses
declaration generic spans and source-order generic accessors, omitting arena links.
Non-alphabetical and duplicate names, identical declaration/use spellings,
trailing commas, comments/CRLF, nested types, generic calls and owner/modifier
composition are compared. No name resolution, type inference, constraints or
execution support is added. A 252-name empty function passes at 256 shared records;
253 names reject at 257. A mixed generic/owner/modifier/body case tests the same
boundary; a generic function's result path passes at depth 63 and rejects at 64.
Empty lists, missing separators/closers, reference declarations, constraints and
defaults are rejected without output.

Attributes are direct function children before modifiers. The Go writer checks
AST attribute/argument order against owning CST nodes, reads the name-path span
and classifies arguments as path/integer/string; it does not export Cooked.
The candidate reads the declaration's attribute range and each attribute's argument
range, retaining order, repeated attributes and repeated argument names. Modifier
origin lookup skips the attribute range. `@tag` and `@tag()` both have zero
argument children; their syntax differs in the bound raw source and outer span,
without a fabricated empty argument node. Unknown names are syntax witnesses,
not newly admitted semantic attributes.
Coverage includes all three argument kinds, qualified paths with comments,
numeric spellings, escaped strings, no arguments, empty parentheses, trailing
commas, repeated attributes, CRLF/non-ASCII trivia and generic/modifier/body
composition. 253 empty attributes or 252 arguments pass at 256 shared records;
one more rejects at 257. Mixed attribute/header/body and depth 63/64 boundaries
are also checked. Missing names/closers, malformed separators, booleans, floats,
byte/C strings and call arguments reject without partial output. This does not
add string decoding, attribute semantics, default compiler integration or a public
AST format. An initial test-driver tag arithmetic type error was corrected using
a usize accumulator; no parser change was required.
