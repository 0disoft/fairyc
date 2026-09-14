# First canonical function-body comparison

This is a private, disposable test transport between the Fairy candidate and the
Go reference AST. It is not a public AST format, parser replacement, semantic
adapter or decoder. Its first proof walks the complete body of one function with
nested bindings, blocks, if/else-if, while, assignments, calls and returns,
including explicitly written binding types, casts and specialized generic calls.

## Encoding

The two independently implemented writers emit space-terminated unsigned decimal
integers. The prefix is `1 source_byte_count source_byte...`; `1` identifies this
test format revision. Original bytes are included once, without normalization,
so every following span is bound to the exact input, including trivia and CRLF.
Type support adds previously unused tags without changing existing records. There
is no persisted artifact, older-reader compatibility window or migration promise.

The remaining stream is a preorder tree. Each record has the following fields:

`kind start_byte end_byte child_count name_start_byte name_end_byte operator_size operator_byte...`

Spans are original half-open byte ranges. Name spans are present only on bindings;
other records use zero. Prefix, binary and assignment operators carry their actual
operator bytes, independently of the source-span envelope; other nodes use length
zero. Child counts and preorder determine all structural edges;
no arena indices, list-link nodes, pointers or host enum values are exported.

| Tags | Meaning and child order |
| --- | --- |
| 100 | Block: statements in source order, followed by an optional tail expression |
| 101 / 102 | Inferred let / var binding: initializer |
| 103 | Return: optional operand |
| 104 | Assignment: target, value |
| 105 | Expression statement: operand |
| 106 | While: condition, body block |
| 107 / 108 | Break / continue: no children |
| 109 / 110 | Explicitly typed let / var binding: written type, initializer |
| 200 / 201 | Literal / path: no children; spelling comes from the bound source span |
| 202 / 203 | Group / prefix expression: operand |
| 204 | Binary expression: left, right |
| 205 | Call: callee, then arguments in source order |
| 206 | Block expression: body block |
| 207 | If expression: condition, then block, optional else block or else-if expression |
| 208 | Cast: operand, written target type |
| 209 | Specialization: target path, then type arguments in source order |
| 301 | Named type path: no children |
| 302 / 303 | Reference / mutable reference type: element |
| 304 / 305 | Slice / grouped type: element |
| 306 | Generic type: base path, then type arguments in source order |
| 308 | Fixed array type: element, written length |
| 309 / 310 | Integer / path array length: no children; no folding or resolution |
| 311 / 312 | Raw pointer / mutable raw pointer type: element |
| 315 | C function type: result, then parameters in source order |

Go's block-expression wrapper is omitted specifically for if branches and while
bodies, where the common format has an explicit block child. The candidate's
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

## Failure and resource boundary

Both writers reject unhandled nodes instead of skipping them. Array expressions,
record construction and the other not-yet-mapped export forms therefore do
not produce a successful partial comparison. This restriction belongs only to
this first exporter, not to the parser's completed grammar support.

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

`fairyc_go_test_parser_body_tree` passes in both native profiles (9.887s).
The bounded matrix contains fourteen complete bodies per profile, including
ordinary, typed and generic-call exact 256-record boundaries, nested type families,
chained casts, generic/value argument order, ordinary `<` comparison, trivia/CRLF
and depth 63 reached through binding, cast and specialization type trees. Twelve
rejection cases cover malformed bindings, casts, calls and returns, unsupported
array expressions, an extra declaration, 257/259/261 records and depth 64 across
those same three type-entry roles. Rejected native
exports must have empty output and nonzero status. A name-only `let i32 =1;`
is valid inferred-binding syntax, not a malformed typed-binding test. Deep groups
hit the existing parser recursion limit before the exporter depth limit, so slice
suffixes isolate that exporter boundary. Formatting and the scoped
Mustflow contract are checked separately; no full compiler/release suite is needed
for test-only writers. No stable compiler, grammar, ABI, SDK, package format or
default CLI path changes. The next exporter extension must map additional node
families explicitly before those bodies can participate in this comparison.
