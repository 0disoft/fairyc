# Candidate HIR model observations

These private numeric observations accompany `examples/lexer_candidate/src/hir_model.fy`.

Private owned HIR. No syntax/type input table is retained. All entity IDs are
one-based; zero is absent. Query table indexes and edge offsets are zero-based.
Module indexes remain zero-based and declaration IDs retain their source index
plus one (including module/import declarations). Binding IDs preserve resolved
symbol IDs, but their owner/target/type facts below are independently owned.
The fixed SDK/compiler admits enum switching, not enum equality or enum casts.
These exhaustive private-format tags also permit exact scalar comparisons.
Binding owner is the enclosing Function Binding ID; scope is its lexical scope
ID. Function/External use module_index+declaration as authoritative call target.
Callable result is type; signature_start/count index the Type signature arena.
An empty body has no callable/receiver. Every resource owns exactly one
classified drop definition; a nonempty body links one private typed function.
left/right/alternative are Expression IDs. If uses left=condition,
block=consequent Block ID, alternative=else Expression ID (zero if absent),
right=0. Binary uses left/right; Unary uses left.
block is Block ID (Block expression or If consequent). Call binding is target;
edge_start/count index ordered argument Expression IDs. Reference binding is
authoritative; callee syntax and grouping nodes are eliminated. Integer stores
canonical decimal ASCII in payload[start..end]; signed-minimum has no child.
Construct binding is a Struct or Resource Binding ID; its edges retain Initializer
Expression IDs in written order. Initializer/Field binding is a one-based
nominal Field ID. Initializer left is the value; Field left is the owner.
expression and target are Expression IDs; binding is defined local Binding ID.
body is Block ID; initial/post are Statement IDs for explicit For clauses.
Omitted for condition is expression=0 (true), initial/post=0 mean absent.
Assignment operator is Assign or its arithmetic/bitwise operation; RHS is
evaluated once. Defer captures an unevaluated void Call expression for scope
exit. Break/Continue target the nearest enclosing While/For Statement ID.
Ordered statement IDs in edge arena; tail is optional Expression ID.
owner is Function Binding ID; scope is the resolver lexical scope ID.
Hard caps: source 1048576 bytes, 512 functions/declarations, 1152 bindings,
16384 expressions, 8192 statements, 8192 blocks, 32768 edges, 32768 signature
entries, 344064 canonical payload bytes (16384*21), recursion depth 64.
Errors: 1 unsuccessful input, 2 budget, 3 allocation, 4 unsupported/malformed
semantic graph. Failure returns .err only, never a partially usable Program.
Keep enum shape/capture checks outside the large general assembly verifier.
Earlier expressions have already passed their local shape checks.
Resource fields are module-private. This checked factory is the lowering
handoff, after all local SDK containers have been populated. No container
mutation through a resource field is required by either construction or query.
Walk owned lexical structure rather than trusting the source-side permission
flag. Unsafe never grants effect authority or alters borrow/cleanup policy.
