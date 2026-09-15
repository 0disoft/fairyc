# First private scalar semantic adapter

This is a test-only integration of the native Fairy function writer with the Go
semantic consumer. It is not a parser replacement, public decoder, source-language
restriction or executable Fairy bootstrap stage.

## Closed input contract

- Exactly `module main;` followed by one ordinary function, with whitespace allowed
  between the module prefix and function and after the function.
- One `i32` result; zero to sixteen `i32` parameters; simple ASCII names.
- Exactly one return statement with one plain decimal integer or simple name.
  Decimal zero is `0`; leading zeroes, separators, radix prefixes and signs are
  excluded. Decimal spelling already equals its cooked value; no general literal
  decoder is implemented. Unknown names, duplicate parameters and out-of-range
  integers are admitted to the adapter so semantic diagnostics can be compared.
- At most 4096 source bytes, 262144 wire bytes, 36000 numeric fields, 256 records
  and depth below 64. Exact source bytes in the wire must match the supplied input.
- Attributes, modifiers, owners, generics, other types, calls, control flow,
  declarations/bindings and extra items are excluded. These remain language/parser
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

`fairy_scalar_semantic_test.go` separately invokes the Go frontend as an oracle.
Each AST is independently passed to `semantic.Analyze` through a `semantic.File`
with path `main.fy`. The test compares the complete semantic Result, including
typed HIR, resolutions, type facts, diagnostic codes/spans and unsupported results;
it also checks whether each oracle case is expected to succeed or diagnose.
No reference AST or Go-generated tree is passed into the adapter.

## Evidence and limits

The existing native function test reuses each low/optimized MSVC artifact for:

- 9 semantic differential cases: constants, parameter use, CRLF, source/parameter
  boundaries, unknown names, duplicate parameters and an out-of-range integer;
- 20 parser-valid but adapter-unsupported shapes, requiring nil AST plus error;
- 11 malformed private wire cases, including truncation, extra data, source/revision
  mismatch, unknown tag, invalid span/count/operator field and size overflow.

The existing 39 function positives/43 rejections and 112 body positives/57
rejections remain unchanged. Native execution here runs the candidate parser and
writer; it does not execute the input scalar functions. No default CLI/frontend,
semantic implementation, public ABI, SDK, package schema or release identity is
changed. Full compiler/release suites and hosted CI are separate evidence.
