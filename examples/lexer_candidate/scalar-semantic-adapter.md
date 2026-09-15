# First private scalar semantic adapter

This is a test-only integration of the native Fairy function writer with the Go
semantic consumer. It is not a parser replacement, public decoder, source-language
restriction or executable Fairy bootstrap stage.

## Closed input contract

- Exactly `module main;` followed by one ordinary function, with whitespace allowed
  between the module prefix and function and after the function.
- One `i32` result; zero to sixteen `i32` parameters; simple ASCII names.
- Exactly one return statement containing decimal integers, simple names, groups,
  unary `+`/`-`, and binary `+`/`-`/`*`. Decimal zero is `0`; leading zeroes,
  separators and radix prefixes are excluded. Signs remain separate unary nodes.
  Decimal spelling already equals its cooked value; no general literal
  decoder is implemented. Unknown names, duplicate parameters and out-of-range
  integers are admitted to the adapter so semantic diagnostics can be compared.
- At most 4096 source bytes, 262144 wire bytes, 36000 numeric fields, 256 records
  and depth below 64. Exact source bytes in the wire must match the supplied input.
- Attributes, modifiers, owners, generics, other types, calls, control flow,
  declarations/bindings, division/remainder, comparisons/logical operators and
  extra items are excluded. These remain language/parser
  features, not omissions to silently fill or fall back to the Go parser.

The private reader verifies the revision/source binding, numeric fields, child
counts, span bounds and containment, record roles and complete consumption. It
constructs function/parameter/type/block/return/expression fields, source line
starts and the integer literal summary in `ast.Module`. Module identity is fixed
only after validating the exact closed module prefix; it is not inferred from a
function record. Documentation attachment and generalized module parsing remain
outside this subset.

## Ownership and oracle separation

`internal/cli/fairy_scalar_adapter_test.go` reads native writer output and builds
the AST directly. It does not call `syntax.Parse`, `frontend.AnalyzeSyntax`, lexer
entrypoints or `ast.Lower`. Source slices supply only the spelled values identified
by validated records. The native producer is the test's own compiled candidate;
this is not an authenticity verifier or an API for attacker-supplied arbitrary
trees. Malformed wire probes are bounded robustness evidence, not general decoder
certification.

Expression conversion recursively retains group/prefix/binary records (202-204),
source envelopes and ordered operands. Prefix and binary Text comes from the
one-byte operator payload, restricted by node role; other records require no
operator payload. The adapter does not parse precedence from source, flatten
groups, fold arithmetic or attach a sign to a literal. It preserves integer
literal summaries in traversal order. Thus the existing semantic analyzer owns
direct `-2147483648` handling and out-of-range diagnostics. These AST expression
nodes carry full expression spans, not a new standalone operator-token span field.

`fairy_scalar_semantic_test.go` separately invokes the Go frontend as an oracle.
Each AST is independently passed to `semantic.Analyze` through a `semantic.File`
with path `main.fy`. The test compares the complete semantic Result, including
typed HIR, resolutions, type facts, diagnostic codes/spans and unsupported results;
it also checks whether each oracle case is expected to succeed or diagnose.
No reference AST or Go-generated tree is passed into the adapter.

## Evidence and limits

The existing native function test reuses each low/optimized MSVC artifact for:

- 21 semantic differential cases: constants, parameter use, CRLF/comments,
  source/parameter boundaries, precedence, left-associative subtraction, groups,
  unary signs, i32 minimum, unknown names, duplicates and integer range errors;
- 25 parser-valid but adapter-unsupported shapes, requiring nil AST plus error;
- 16 malformed private wire cases, including truncation, extra data, source/revision
  mismatch, unknown tag, invalid span/count/operator field, wrong expression arity,
  size overflow and independently injected excess expression depth/record count;
- two candidate-output rejection cases: one more unary level or one more record
  beyond the accepted depth-63 / 256-record arithmetic witnesses, with empty output.

The 256-record witness combines sixteen parameters with a balanced grouped
74-leaf sum; the depth witness uses sixty unary nodes below function/block/return.
Both reader and producer retain the original limits. Arithmetic semantic equality
does not prove runtime arithmetic results or overflow trapping; no input function
is executed by this test.

The existing 39 function positives/43 rejections and 112 body positives/57
rejections remain unchanged. Native execution here runs the candidate parser and
writer; it does not execute the input scalar functions. No default CLI/frontend,
semantic implementation, public ABI, SDK, package schema or release identity is
changed. Full compiler/release suites and hosted CI are separate evidence.
