# Stage 0 body-tree coverage ledger

## Audit boundary and result

This bounded audit compares [Stage 0 grammar](../../spec/grammar/stage0.ebnf),
the Go surface AST, Fairy candidate arenas and the private
[canonical body writer](body-tree-comparison.md). The pre-audit baseline is
`1868fcf0854ca6d051347d08c5f187c059cf4c05`.
This is a production-family inventory, not a proof over all possible programs.

- **Connected:** every normative statement, expression, pattern and written-type
  family reachable from the selected ordinary function body has a writer route.
- **Actual missing route:** none found within that boundary during this audit.
  No parser implementation or transport-tag change was needed.
- **Verification gap addressed:** the earlier corpus did not explicitly cover all
  operator alternatives and literal classes. `bodyTreeGrammarAuditInputs` adds
  51 witnesses: 18 binary operators, 7 prefixes, 11 assignments, 11 literals,
  two mixed-precedence expressions, a trailing-comma/contextual-path combination
  and a decorated header.
- **Intentional exclusions and residual verification gaps:** below. Connected
  does not mean semantically valid, executable or ready to replace the frontend.

## Grammar to tree mapping

`S`, `E` and `T` denote candidate statement, expression and type arenas.
The final column uses only private transport tags. Every row is **connected**;
existing and new witnesses run together in `TestWindowsFairyParserBodyTree`.

| Grammar family | Go surface AST | Fairy candidate | Common records |
| --- | --- | --- | --- |
| `function_body`, `statement_block`, `block`, `tail_expression` | Block, statements, optional tail | S1, optional expression | 100; standalone block expression 206 |
| Inferred/typed `binding_stmt`, let/var | Binding, optional Type | S2, entry 2/3 | 101/102 or 109/110 |
| `assignment_stmt`, all 11 assignment operators | Assignment target/operator/value | S4 | 104, exact operator bytes |
| `expression_stmt`, `return_stmt` | ExprStmt, Return | S6/S3 | 105/103 |
| assert/defer/discard/burn statements | Distinct kinds and operand | S11/S13/S12/S16 | 115/116/117/118 |
| while/break/continue statements | While, jump statements | S8/S9/S10 | 106/107/108 |
| `for_stmt`, init/post clauses | ForLoop with four roles | S14, header S15 | 111, absence 112/113/114 |
| `region_stmt` | Region, CST name origin, expression/body | S17, name E17 | 119 |
| `if_statement`, `if_expression` | If, branch blocks | E13, pair E14 | 207 |
| `block_statement`, `unsafe_expression` | Block or Unsafe | E12/E31 | 206/222, body 100 |
| Logical through multiplicative expression layers; all 18 binary operators | Binary tree | E5, precedence table | 204 |
| `cast_expression` | Cast, written Type | E23 | 208 |
| All 7 prefix alternatives | Prefix; refs represented as & or &mut | E4/E21/E22 | 203/219/220 |
| Path and bare contextual variant | Path | E2/E28 | 201 |
| `grouped_expression` | Group | E3 | 202 |
| Call, argument list, explicit type call | Call, Specialize, type arguments | E6/E20, links E7 | 205/209, written types |
| Index and four slice-bound presence forms | Index, Slice, OmitBound | E9/E26, pair E27 | 211/212/213/214 |
| Field suffix, propagation | FieldExpr, Propagate | E8/E30 | 210/221 |
| Empty/list/repeat array | Array form and children | E24/E29, links E25 | 215/218 |
| Record or contextual payload construction, record fields | Record, types, Field values | E10/E11, optional E20 target | 216/217 |
| Switch and cases | Switch, arm Pattern | E15/E18, links E19 | 223/224 |
| Unit/positional/named patterns, pattern fields | Pattern, paths, fields; CST distinguishes empty named form | E16/E32, binding E17, field E33 | 225/226/227/228/229 |
| Boolean/null/integer/float/text/byte/C-string literals | Literal | E1 | 200, raw span only |
| Named/generic types, type argument/list productions | Named TypeReference, arguments | T1/T6, links T7 | 301/306 |
| Reference/raw-pointer types and mutability | Reference/RawPointer | T2/T3/T11/T12 | 302/303/311/312 |
| Postfix slice/array and grouped types | Slice/Array/Grouped | T4/T8/T5 | 304/308/305 |
| C function type | ABI, result, parameter types | T15, links T16 | 315 |
| Const expression in array length/repeat | Written literal/path, no folding | T9/T10 or E1/E2 | 309/310 or 200/201 |

Precedence and list productions need not each have a record: the ordered tree
carries their structure. Helper-node removal and source-origin normalization are
documented in the encoding owner; this ledger does not authorize new normalization.

## Intentional exclusions

| Surface | Current behavior |
| --- | --- |
| Source/module/imports/multiple items | Selection requires module plus one ordinary function. Imports and extra declarations/functions are rejected. Module bytes are bound, but module structure is not exported. |
| Function name/owner, result, parameters, generic parameters, pub/unsafe/attributes | Header syntax may be accepted, but is not structurally compared. The decorated-header witness proves acceptance only; raw bytes are not a header-tree comparison. |
| Const/static_assert/state/struct/resource/cleanup/enum/foreign declarations and fields | Not selected as roots. Separate parser tests are not promoted into body-export evidence. |
| Lexical productions, identifier classes, trivia, cooked literal values | Raw bytes and spans participate. Token kinds, cooked values and a separate lossless token/CST stream are not exported. Literal witnesses do not prove decoding semantics or exhaustive Unicode handling. |
| Error/missing nodes and recovery trees | Error-free parsing and candidate integrity are preconditions. No successful partial output; recovery fidelity is not this format's contract. |
| Legacy/non-normative Go kinds, such as KindMatch | Their presence in an AST enum does not add them to current grammar. They are not missing current language features. |
| Expression-only for init/post accepted by a permissive Go parse helper | Normative grammar and candidate restrict these to binding/assignment or assignment. Other forms remain rejected by the writer. |
| Resolution, types, effects, ownership, lowering, execution | Excluded. Duplicate names and semantically invalid programs can be valid syntax witnesses. |
| Oversized/deep bodies; persistence/public AST API | 32768 source bytes, 256 shared records and depth below 64 remain in force; parser limits apply first. No persistence or compatibility promotion. |

## Evidence and remaining verification gaps

`fairyc_go_test_parser_body_tree` passes with **112 positive bodies and 57 rejection
cases per Windows MSVC profile**, low and optimized (18.407s). Import and oversized
source rejection supplement the existing declaration/function and record/depth
boundaries. Every positive compares native stdout byte-for-byte with the independent
Go writer. Every negative requires nonzero native status, empty stdout/stderr/trap
output and an empty Go result with error.

Owners inspected: `spec/grammar/stage0.ebnf`, Go `internal/syntax/` expression and
statement parsers, `internal/ast/` lowering and types; candidate `bodies.fy`,
`body_state.fy`, `parse_tokens.fy`, `types.fy`, `syntax.fy`, `integrity.fy`; both body
writers. The new `internal/cli/fairy_parser_body_tree_coverage_test.go` witnesses
are explicitly appended by the existing native test, not a disconnected test.

Remaining **verification gaps**, not newly discovered implementation defects:

- This is a manual production-family ledger, not an automatic EBNF coverage proof.
  Future grammar/AST changes require updating and rerunning it.
- All operator combinations, lexical spellings, malformed recovery paths and
  cross-family combinations are not exhaustively enumerated.
- The writers share source bytes and deliberate normalization choices. Agreement
  is not an independent proof of language semantics.
- No claim is made about non-Windows native execution, full compiler/release
  health, hosted CI, public frontend replacement or release readiness.

The next bounded integration target is **function-header structural comparison**,
not another speculative body grammar addition. Semantic integration remains a
separate design and verification step after its required tree boundary is complete.
