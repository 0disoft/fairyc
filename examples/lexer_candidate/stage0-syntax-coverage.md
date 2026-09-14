# Private candidate Stage 0 syntax coverage

Scope: syntax recognition and source-faithful private trees for the existing
`spec/grammar/stage0.ebnf`. This does not admit new language semantics, execute
the parsed corpus, replace the Go frontend, or establish self-hosting readiness.
Existing source, arena, recursion and token-lookahead limits remain applicable.

## Remaining-work ledger

| Grammar boundary | Implementation and evidence |
| --- | --- |
| Ordinary `unsafe fn` | Implemented in `src/syntax.fy`; `TestWindowsFairyParserForeign` compares ordinary/unsafe/public/attributed headers against Go. |
| Type-owned function names | Implemented; owner and method have distinct spans, including trivia and generic parameters. Invalid empty/multiply-qualified/foreign owner names are rejected and following declarations recovered. |
| C function types | Implemented in `src/types.fy`; native type-tree differential and recovery/limit tests pass in both Windows profiles (48.538s). |
| `burn` statements | Pending implementation and differential tests. |
| `region` statements | Pending implementation and differential tests. |
| Attributed cleanup declarations | Pending implementation and differential tests. |
| Unclosed named-pattern recovery | Pending recovery improvement; the earlier test records consumption of the following function. |
| Whole grammar coverage audit | Pending production-by-production audit and selected corpus gates after the above changes. |

## Function-header representation

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
a new module-level declaration. This is separate from the still-open unclosed
named-pattern recovery item.

The first enlarged type probe exceeded its 180-second lane budget. Splitting
checks into borrowed per-position helpers and separate tree/recovery probes
retains every assertion while avoiding the oversized generated function.
