# First canonical function-body comparison

This is a private, disposable test transport between the Fairy candidate and the
Go reference AST. It is not a public AST format, parser replacement, semantic
adapter or decoder. Its first proof walks the complete body of one function with
nested bindings, blocks, if/else-if, while, assignments, calls and returns.

## Encoding

The two independently implemented writers emit space-terminated unsigned decimal
integers. The prefix is `1 source_byte_count source_byte...`; `1` identifies this
test format revision. Original bytes are included once, without normalization,
so every following span is bound to the exact input, including trivia and CRLF.

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
| 200 / 201 | Literal / path: no children; spelling comes from the bound source span |
| 202 / 203 | Group / prefix expression: operand |
| 204 | Binary expression: left, right |
| 205 | Call: callee, then arguments in source order |
| 206 | Block expression: body block |
| 207 | If expression: condition, then block, optional else block or else-if expression |

Go's block-expression wrapper is omitted specifically for if branches and while
bodies, where the common format has an explicit block child. The candidate's
branch-pair helper is likewise omitted. A standalone block expression retains tag
206. Go's operandless break/continue self-expression placeholders are not separate
children; the jump statement itself remains visible. Statement wrappers, grouping,
order and every source-written body expression remain visible.
Literal cooking and semantic analysis are deliberately not encoded or claimed.

## Failure and resource boundary

Both writers reject unhandled nodes instead of skipping them. Typed bindings,
specialized calls, arrays and the other not-yet-mapped export forms therefore do
not produce a successful partial comparison. This restriction belongs only to
this first exporter, not to the parser's completed grammar support.

Input must be an error-free module plus exactly one ordinary function, within
32768 source bytes. Header/parameter trees are not the root of this body-only
format. Exports are bounded to 256 records and depth less than 64. The candidate
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

`fairyc_go_test_parser_body_tree` passes in both native profiles (9.963s): four
complete bodies, including the exact 256-record boundary, compare byte-for-byte.
Five rejection cases per profile cover typed bindings, arrays, malformed syntax,
257 records and an extra declaration, all with empty output and nonzero status.
Initial test-package setup, writer arithmetic and Go jump-placeholder mapping
errors were corrected before this successful run. Formatting and the scoped
Mustflow contract are checked separately; no full compiler/release suite is needed
for test-only writers. No stable compiler, grammar, ABI, SDK, package format or
default CLI path changes. The next exporter extension must map additional node
families explicitly before those bodies can participate in this comparison.
