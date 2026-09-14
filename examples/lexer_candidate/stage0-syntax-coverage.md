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
| C function types | Pending implementation and differential tests. |
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
