# First private scalar semantic adapter

This is a test-only integration of the native Fairy function writer with the Go
semantic consumer. It is not a parser replacement, public decoder, source-language
restriction or executable Fairy bootstrap stage.

## Closed input contract

- Exactly `module main;` followed by one ordinary function, with whitespace allowed
  between the module prefix and function and after the function.
- One `i32` or `bool` result; zero to sixteen `i32` / `bool` parameters;
  simple ASCII names.
- Zero or more sequential explicitly typed `let` / `var` declarations (`i32`
  or `bool`) and simple-name
  assignments (`name = expression;`), followed by exactly one final return
  statement. Initializers, assignment values and the return
  expression contain decimal integers, `true` / `false`, simple names, groups,
  unary `+`/`-`, binary `+`/`-`/`*`, and comparisons `== != < <= > >=`.
  Value-producing `if (condition) { value } else { value }` expressions are
  admitted recursively. Both branches must be explicit blocks with exactly one
  tail expression and no statements. A direct `else if` chain is excluded;
  a nested `if` as a branch's tail expression is admitted.
  Decimal zero is `0`; leading zeroes,
  separators and radix prefixes are excluded. Signs remain separate unary nodes.
  Decimal spelling already equals its cooked value; no general literal
  decoder is implemented. Unknown names, duplicate parameters and out-of-range
  integers and type-incompatible operands/values are admitted to the adapter so
  semantic diagnostics can be compared. It does not infer operand types.
- At most 4096 source bytes, 262144 wire bytes, 36000 numeric fields, 256 records
  and depth below 64. Exact source bytes in the wire must match the supplied input.
- Attributes, modifiers, owners, generics, other types, calls, loops,
  statement-form conditionals, branch-local declarations/assignments/returns,
  inferred bindings, compound or non-name assignments, standalone block expressions,
  division/remainder, logical operators (including unary `!`) and
  extra items are excluded. These remain language/parser
  features, not omissions to silently fill or fall back to the Go parser.

The private reader verifies the revision/source binding, numeric fields, child
counts, span bounds and containment, record roles and complete consumption. It
constructs function/parameter/type/block/binding/assignment/return/expression fields, source line
starts and the binding and integer/boolean literal summaries in `ast.Module`. Module identity is fixed
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
one- or two-byte operator payload, restricted by node role. Binary operators may
include the six comparisons; prefix operators remain only `+` and `-`.
Assignment records require
exactly `=`; other records require no operator payload. The adapter does not parse precedence from source, flatten
groups, fold arithmetic or attach a sign to a literal. It preserves integer and
boolean literal summaries in traversal order (`true`/`false` retain their cooked
spelling and `boolean` kind). Thus the existing semantic analyzer owns
direct `-2147483648` handling and out-of-range diagnostics. These AST expression
nodes carry full expression spans, not a new standalone operator-token span field.

Typed binding records (109/110) retain mutability, the declared type, name, initializer,
statement span and source order. Type/name/initializer spans must be ordered;
statement envelopes cannot overlap. Earlier returns and a missing final return
are rejected. Resolution and typing remain entirely owned by the Go analyzer,
including parameter shadowing, same-scope duplicates and use before declaration.
Assignment records (104) retain the operator, simple-name target and value in
source order; overlapping operands are rejected. Immutable or unresolved targets
are admitted so the semantic consumer, not the adapter, owns their diagnostics.

Conditional records (207) require three ordered children: a condition expression
and two block records (100). Each branch becomes an AST block expression with a
single Tail; the transport has no separate tail record. Only expression nodes
may occupy that role. Branch and condition spans are preserved without folding
constant conditions or dropping an unselected branch. The Go analyzer owns
condition typing, branch type compatibility and final return compatibility.

`fairy_scalar_semantic_test.go` separately invokes the Go frontend as an oracle.
Each AST is independently passed to `semantic.Analyze` through a `semantic.File`
with path `main.fy`. The test compares the complete semantic Result, including
typed HIR, resolutions, type facts, diagnostic codes/spans and unsupported results;
it also checks whether each oracle case is expected to succeed or diagnose.
No reference AST or Go-generated tree is passed into the adapter.
Function AST fields and module binding/literal summaries are also compared to
the independently lowered oracle, including names, spans and initializer order.

## Evidence and limits

The existing native function test reuses each low/optimized MSVC artifact for:

- 76 semantic differential cases: constants, parameter use, CRLF/comments,
  source/parameter boundaries, precedence, left-associative subtraction, groups,
  unary signs, i32 minimum, unknown names, duplicates and integer range errors,
  sequential locals, parameter shadowing, self/forward references, reassignment,
  immutable/unresolved targets, assignment range errors, depth boundaries,
  boolean signatures/bindings/literals, all six integer comparisons, arithmetic
  precedence inside comparisons, mismatched return/binding/assignment/operand types,
  value conditionals in returns/initializers/assignments, nested branches, invalid
  conditions, branch/return type mismatches and unresolved names in both conditions
  and unselected branches;
- 46 parser-valid but adapter-unsupported shapes, requiring nil AST plus error;
- 39 malformed private wire cases, including truncation, extra data, source/revision
  mismatch, unknown tag, invalid span/count/operator field, wrong expression arity,
  size overflow, independently injected excess expression depth/record count and
  inferred binding tags, child roles, name spans, statement overlap and invalid
  assignment operators, target roles and operand overlap, plus invalid comparison
  length, byte range, operator combination and node role, and invalid conditional
  arity, condition/branch roles, tail count, branch span order and name metadata;
- four candidate-output rejection cases: excess unary or conditional nesting,
  or one more record beyond the accepted arithmetic/local witnesses, with empty output.

The 256-record witness combines sixteen parameters with a balanced grouped
74-leaf sum; the depth witness uses sixty unary nodes below function/block/return.
Both reader and producer retain the original limits. Arithmetic semantic equality
also covers a depth-63 initializer and 83 locals plus a return at 256 records.
An 84-local body reaches 257 records and is rejected. Semantic equality
also covers thirty nested value conditionals with a deepest tail at depth 63;
one more conditional is rejected by the producer. This comparison
does not prove runtime arithmetic results or overflow trapping; no input function
is executed by this test.

The existing 39 function positives/43 rejections and 112 body positives/57
rejections remain unchanged. Native execution here runs the candidate parser and
writer; it does not execute the input scalar functions. No default CLI/frontend,
semantic implementation, public ABI, SDK, package schema or release identity is
changed. Full compiler/release suites and hosted CI are separate evidence.
