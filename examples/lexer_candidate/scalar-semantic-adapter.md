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
  assignments (`name = expression;`), statement-form conditionals or `while` loops,
  followed by exactly one final return
  statement. Initializers, assignment values and the return
  expression contain decimal integers, `true` / `false`, simple names, groups,
  unary `+`/`-`, binary `+`/`-`/`*`, and comparisons `== != < <= > >=`.
  Value-producing `if (condition) { value } else { value }` expressions are
  admitted recursively. Both branches must be explicit blocks containing zero or
  more typed `let`/`var` declarations, simple-name assignments, statement conditionals
  or `while` loops followed by exactly
  one tail expression. A direct `else if` chain is excluded;
  a nested `if` as a branch's tail expression is admitted.
  Statement-form `if (condition) { statements }` permits an optional `else`
  block. These branches may be empty and contain declarations, assignments or
  nested statement conditionals or `while` loops, but no tail expression. Both forms retain their
  distinct block rules; a value-form branch may include statement conditionals
  before its required tail.
  At the end of a branch, a nested statement `if` needs a trailing semicolon;
  without it the existing parser classifies the nested `if` as a tail expression,
  which is excluded from statement-form branches.
  A `while (condition) { statements }` body may be empty and uses the same
  statement-only rules, including nested loops. The condition may use the admitted
  scalar expressions. No loop execution or termination is inferred by the adapter.
  Decimal zero is `0`; leading zeroes,
  separators and radix prefixes are excluded. Signs remain separate unary nodes.
  Decimal spelling already equals its cooked value; no general literal
  decoder is implemented. Unknown names, duplicate parameters and out-of-range
  integers and type-incompatible operands/values are admitted to the adapter so
  semantic diagnostics can be compared. It does not infer operand types.
- At most 4096 source bytes, 262144 wire bytes, 36000 numeric fields, 256 records
  and depth below 64. Exact source bytes in the wire must match the supplied input.
- Attributes, modifiers, owners, generics, other types, calls, `for`, `break`, `continue`,
  branch-local early returns,
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
and two block records (100). Each branch becomes an AST block expression with
ordered Statements and a single Tail; the transport has no separate tail record.
Only expression nodes may occupy the final role. The same bounded declaration/
assignment converter serves the function body and branches; early returns are
not accepted inside branches. Statement and tail spans cannot overlap.
Branch and condition spans are preserved without folding
constant conditions or dropping an unselected branch. The Go analyzer owns
condition typing, branch type compatibility and final return compatibility.
Lexical scopes are not recreated by the adapter: the Go consumer resolves outer
names, shadowing, same-scope duplicates and invalid sibling/escaped local uses.
The module binding summary reserves a declaration's position before converting
its initializer so nested branch declarations retain preorder, matching Go lowering.
Statement conditionals use an expression-statement record (105) wrapping an if
record (207) with a condition and one or two explicit blocks (100). They build
AST KindExprStmt/KindIf nodes and blocks with Statements but no Tail. The adapter
does not synthesize an absent else or discard non-void branch values. It rejects
other expression statements and direct else-if chains. Existing scope and type
analysis owns effects of outer-variable assignment and invalid condition types.
While records (106) require exactly a condition and one block (100), in source
order. They become KindWhile statements with a KindWhile expression containing
the condition and block expression, matching Go lowering. The statement-only
body conversion is shared with statement conditionals, with the distinct wire
depth preserved. Loop-local scopes, outer assignments and condition diagnostics
remain owned by the Go consumer.

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

- 115 semantic differential cases: constants, parameter use, CRLF/comments,
  source/parameter boundaries, precedence, left-associative subtraction, groups,
  unary signs, i32 minimum, unknown names, duplicates and integer range errors,
  sequential locals, parameter shadowing, self/forward references, reassignment,
  immutable/unresolved targets, assignment range errors, depth boundaries,
  boolean signatures/bindings/literals, all six integer comparisons, arithmetic
  precedence inside comparisons, mismatched return/binding/assignment/operand types,
  value conditionals in returns/initializers/assignments, nested branches, invalid
  conditions, branch/return type mismatches and unresolved names in both conditions
  and unselected branches, branch declarations/assignments, shadowing, sibling
  isolation, escaped locals, duplicate/forward names, immutable targets, nested
  initializer summary order, branch initializer depth, statement conditionals
  with optional/empty else blocks, nested statements, following computations,
  scope isolation, statement/value conditional composition, accumulation-shaped
  loops, empty/nested loops, loop/conditional composition, loop-local scope,
  invalid loop conditions and immutable loop targets;
- 58 parser-valid but adapter-unsupported shapes, requiring nil AST plus error;
- 50 malformed private wire cases, including truncation, extra data, source/revision
  mismatch, unknown tag, invalid span/count/operator field, wrong expression arity,
  size overflow, independently injected excess expression depth/record count and
  inferred binding tags, child roles, name spans, statement overlap and invalid
  assignment operators, target roles and operand overlap, plus invalid comparison
  length, byte range, operator combination and node role, and invalid conditional
  arity, condition/branch roles, tail count, branch span order and name metadata,
  plus overlapping branch statements/tails, an early-return record and invalid
  statement-conditional wrapper, if arity and branch roles, and invalid loop
  arity, condition/body roles and overlapping condition/body spans;
- seven candidate-output rejection cases: excess unary, conditional or loop nesting,
  or one more record beyond the accepted arithmetic/local witnesses, with empty output.

The 256-record witness combines sixteen parameters with a balanced grouped
74-leaf sum; the depth witness uses sixty unary nodes below function/block/return.
Both reader and producer retain the original limits. Arithmetic semantic equality
also covers a depth-63 initializer and 83 locals plus a return at 256 records.
An 84-local body reaches 257 records and is rejected. Semantic equality
also covers thirty nested value conditionals with a deepest tail at depth 63;
one more conditional is rejected by the producer. A branch-local initializer with
57 unary prefixes reaches depth 63; 58 prefixes are rejected. Twenty nested
statement conditionals with an innermost declaration reach initializer depth 63;
one additional statement conditional is rejected. A loop-local initializer with
58 unary prefixes reaches depth 63; wrapping that loop in another loop exceeds
the limit and is rejected. Deep loop-fixed-point stress is not part of this
adapter suite; ordinary nested loops still undergo full semantic comparison.
This comparison
does not prove runtime arithmetic results or overflow trapping; no input function
is executed by this test.

The existing 39 function positives/43 rejections and 112 body positives/57
rejections remain unchanged. Native execution here runs the candidate parser and
writer; it does not execute the input scalar functions. No default CLI/frontend,
semantic implementation, public ABI, SDK, package schema or release identity is
changed. Full compiler/release suites and hosted CI are separate evidence.
