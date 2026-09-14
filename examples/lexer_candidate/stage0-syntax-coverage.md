# Private candidate Stage 0 syntax coverage

Scope: syntax recognition and source-faithful private trees for the existing
`spec/grammar/stage0.ebnf`. This does not admit new language semantics, execute
the parsed corpus, replace the Go frontend, or establish self-hosting readiness.
Existing source, arena, recursion and token-lookahead limits remain applicable.

## Completed-work ledger

| Grammar boundary | Implementation and evidence |
| --- | --- |
| Ordinary `unsafe fn` | Implemented in `src/syntax.fy`; `TestWindowsFairyParserForeign` compares ordinary/unsafe/public/attributed headers against Go. |
| Type-owned function names | Implemented; owner and method have distinct spans, including trivia and generic parameters. Invalid empty/multiply-qualified/foreign owner names are rejected and following declarations recovered. |
| C function types | Implemented in `src/types.fy`; native type-tree differential and recovery/limit tests pass in both Windows profiles (48.538s). |
| `burn` statements | Implemented as statement kind 16; native tests compare full statement/operand spans to Go, reject malformed statements, and check 2047/2048 shared statement-arena limits. |
| `region` statements | Implemented as statement kind 17; Go CST comparison, malformed-input rollback, detached-node checks and 31/32-level depth tests pass in both Windows profiles after module extraction. Control-item suite: 69.725s. |
| Attributed cleanup declarations | Implemented with separate declaration/cleanup attribute ranges; native source-span, rollback, mutation and shared attribute/argument-limit tests pass in both profiles. |
| Unclosed named-pattern recovery | Production-specific missing-close marker at the case colon preserves the following function; empty, trailing-comma and nested-function isolation cases pass in both profiles (value suite 69.274s). |
| Block/if expressions in statement position | Implemented through the shared expression reader; branch tails, explicit semicolons and continuations pass Go-backed native probes. |
| Whole grammar coverage audit | Completed: [production map and final evidence](stage0-grammar-audit.md), 27 cross-family witnesses, required fixtures and SDK corpus. No identified existing Stage 0 syntax omission remains within the unchanged candidate limits. |

## Block expressions in statement position

The grammar audit found that the old standalone `if` reader rejected branch
tails accepted in value position, and that standalone block/if shortcuts stopped
before postfix or binary continuations. Both now use the ordinary expression
reader and a kind-6 expression-statement wrapper, matching the Go surface tree.
This also preserves an optional written semicolon. A function, while, for or
region body still forbids its own tail; inner block/if expressions may have tails.
Place validity and branch type agreement remain semantic, not parsing, checks.

The block-tree oracle retains source-order, child-count and span assertions using
the shared expression representation. A dedicated statement-position probe covers
branch tails, else-if, semicolons, calls, fields, indexing, arithmetic, assignment,
UTF-8/CRLF spans and malformed-declaration rollback.

The additional expression nodes initially exhausted the self-corpus budget for
`bodies.fy`. A shared node factory alone was insufficient; moving the leaf pattern
reader to `body_state.fy` restored complete parsing of all eighteen sources in
both profiles (8.711s) without increasing any budget. The factory preserves all
six node fields and the moved reader retains the production-specific recovery
marker. After the move, value regressions pass in both profiles (84.521s), as does
the full candidate oracle including block/entry trees and resource-limit checks
(197.530s). These are private syntax checks, not execution of the parsed inputs.

## Named-pattern recovery

Private expected-token code 25 means a missing named-pattern `}` at the case
colon and renders as `}`. It is produced only by the named-pattern reader.
Declaration recovery discounts exactly one unmatched opener at that recorded
token; input bytes are neither rewritten nor accepted as a valid program.
The malformed declaration and its nodes are discarded, while a following
top-level declaration can survive. The marker is not used for arbitrary missing
braces and does not promote a nested function to module scope.

The prior regression expected the following function to disappear. It now
requires that function to survive, with additional empty/trailing-comma cases
and a subsequent malformed nested-function declaration testing marker isolation.
Duplicate token-movement helpers in `syntax.fy` now use `body_state` implementations;
the source and arena limits remain unchanged.

## Function-header representation

`burn` owns one expression and requires a semicolon, including inside a value
block. It is not an expression or an inferred binding. Parsed ownership and
resource-consumption semantics remain unchanged. Mutated burn nodes with a
missing operand or unexpected statement child are rejected by the verifier.

Ordinary functions, including `unsafe fn`, retain declaration kind 3. Raw foreign
declarations remain kind 9 and require `unsafe extern C fn` plus a semicolon.
`unsafe_start`/`unsafe_end` retain the keyword's half-open byte span; both are zero
for safe declarations. `owner_start`/`owner_end` retain the optional single owner
identifier; `name_start`/`name_end` refer to the method identifier, not the complete
qualified spelling. Both owner fields are zero for module-local functions.
The integrity verifier checks these ranges and their allowed declaration kinds.

Header fixtures contain UTF-8 comments, CRLF, visibility, attributes, generic
parameters, ordinary bodies and malformed headers. Recognition does not resolve
the owner or check receiver, visibility, effects, or unsafe-call semantics.

## Cleanup attribute representation

The declaration's `attribute_start`/`attribute_count` still own only its outer
attributes. `cleanup_attribute_start`/`cleanup_attribute_count` own the cleanup
prefix in the same attribute arena; zero count requires zero start. A nonempty
cleanup prefix may start at index zero when there are no earlier attributes.
`cleanup_start` begins at the keyword, matching the Go cleanup node, while the
prefix attributes retain their own byte spans. Failed declarations roll back
both attribute and argument vectors along with fields, types and body nodes.

`cleanup_syntax.fy` is explicitly required in probe copying and the self-corpus
inventory. It receives current arena lengths from the vector owner, so no new
mutable-to-shared reborrow support is required from the Go compiler. Declaration
and cleanup prefixes share the unchanged 256-attribute and 1024-argument limits.
All eighteen required candidate files pass the self-corpus gate in both native
profiles (8.820s), including the unchanged 32768-byte per-source cap.
Attributes before resource fields remain outside the normative grammar, even
though Go's resource parser accepts a broader attribute placement.

## Region statement representation

Kind 17 stores the arena expression in `expression`, a kind-17 name expression
in `target`, and the statement-only body in `first`. `second`, `entry` and
`operator` are zero. The name has its own original token and half-open byte span.
The verifier requires all three children and rejects absent or wrong-kind links.

The arena's outer expression and its prefix/binary continuations stop before
`{`; grouped expressions, call arguments and indices retain ordinary record
construction. This follows Go's `allowRecord` distinction without changing
ordinary expression parsing. Token-only statement classifiers live in the
existing `parse_tokens` module to retain the unchanged per-source byte limit.
The region increment initially exhausted a parser budget while parsing
`bodies.fy` itself. Its unchanged cursor/node/statement records, token movement,
node insertion and statement factory were moved to `body_state.fy`, with that seventeenth source explicitly included
in probe copying and the mandatory self-corpus inventory. This does not raise
the byte, expression, statement or type limits or allow partial corpus success.
The complete seventeen-file self-corpus gate passes in both profiles (9.011s);
`bodies.fy` is 31460 bytes and `body_state.fy` is 2759 bytes at this increment.
No region allocation, borrow analysis, lifetime semantics or execution is added.

## C function type representation

Kind 15 stores the required result type in `left` and an optional kind-16
parameter-list tail in `right`. Kind-16 links store a value type and the previous
parameter link. All children precede parents in the shared 4096-node type arena;
zero represents the empty parameter list. The full function type and every
parameter retain original half-open byte spans. `unsafe extern C fn` is required;
ABI selection, callback safety and foreign-call execution remain outside syntax.

Tests compare each recursively nested node against the Go AST. They also cover
31/32 nesting levels and 2047/2048 parameters at the existing arena boundary.
An older attributed-generic-resource rejection was stale: it is now a positive
oracle-backed case, with a malformed generic header preserving negative coverage.

The Go syntax frontend is not an unconditional acceptance oracle: it accepts
`A<>` and literal attribute arguments such as `true` and byte strings, while the
normative grammar requires nonempty type arguments and limits attribute arguments
to paths, integers and text strings. These remain explicitly tested normative
rejections, not missing candidate features. Neither the grammar nor Go behavior
is changed by the candidate work.

Malformed function-type recovery remembers the original failure position and
skips embedded `unsafe ... fn` headers rather than reclassifying their `fn` as
a new module-level declaration. This is separate from the unclosed named-pattern
recovery described above.

The first enlarged type probe exceeded its 180-second lane budget. Splitting
checks into borrowed per-position helpers and separate tree/recovery probes
retains every assertion while avoiding the oversized generated function.
