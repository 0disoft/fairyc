# Stage 0 candidate grammar audit

This audit covers the existing normative `spec/grammar/stage0.ebnf`, from
`source` through the external lexer terminal aliases. It is a syntax-recognition
and source-span audit of the private candidate, not a compiler, semantic, runtime,
release or whole-self-hosting promotion. The grammar was not changed.

## Requirement map

Every production belongs to one of the contiguous grammar sections below.
Witness numbers refer to the named cases in
`internal/cli/fairy_parser_grammar_coverage_test.go`; that helper is run by
`TestWindowsFairyParserFixtureCoverage/stage0_grammar_productions`.

| Normative production section | Implementation owner under `src/` | Executed witness cases |
| --- | --- | --- |
| `source`, `module_decl`, `import_decl`, `item`, `item_body` (grammar lines 23-37) | `syntax.fy`, `file_syntax.fy` | 00-07; every item alternative, imports before items, full source |
| `attribute` through `attribute_argument` (39-42) | `attributes.fy`, `cleanup_syntax.fy` | 01, 04; absent/empty/nonempty arguments, paths, integer/text arguments, trailing comma |
| `const_decl`, `static_assert_decl`, `state_decl`, `state_field_decl` (44-53) | `syntax.fy`, `records.fy` | 01-02; constants, static assertions, cell/map fields |
| `struct_decl`, `field_decl`, `resource_decl`, `cleanup_decl` (57-61) | `syntax.fy`, `records.fy`, `cleanup_syntax.fy` | 03-04; public/private fields, generic aggregates, empty resources and attributed cleanup |
| `enum_decl`, `enum_variant`, `enum_payload`, `enum_field_decl` (63-66) | `records.fy` | 05, 24; no payload, empty payload, named payload fields |
| `fairy_function_decl` through `foreign_parameter` (68-85) | `syntax.fy`, `bodies.fy`, `types.fy` | 06-07; safe/unsafe, owner names, generic/trailing-comma parameters, empty and nonempty foreign functions |
| `type` through `type_arguments` (90-107) | `types.fy` | 03, 08-09; reference/pointer mutability, qualified/generic/grouped types, slice/array suffixes, nested C function types |
| `block`, `statement_block`, `function_body`, `statement` (109-128) | `bodies.fy`, `body_state.fy` | 10-15, 23-24; statement-only bodies, inner value blocks and all statement alternatives |
| `binding_stmt` through `tail_expression` (130-161) | `bodies.fy`, `parse_tokens.fy` | 10-15, 23; inferred/typed let/var, all assignment operators, optional return operands, assert/defer/discard/burn, if/block/unsafe, while/for, jumps and region |
| `expression` through `multiplicative_expression` (163-177) | `bodies.fy`, `parse_tokens.fy` | 16-17; all precedence levels and comparison spellings |
| `cast_expression`, `prefix_expression`, `prefix_operator` (179-181) | `bodies.fy`, `types.fy`, `parse_tokens.fy` | 18; chained casts, every unary spelling; pointer and C-function casts also in `TestWindowsFairyParserCasts` |
| `postfix_expression` through `field_suffix` (183-191) | `bodies.fy`, `types.fy` | 19-20; ordinary/explicit-type calls, trailing arguments, chained field/index/slice/question-mark suffixes |
| `primary_expression` through `grouped_expression` (193-210) | `bodies.fy` | 21-24, 25-26; qualified paths, leading-dot variants, grouping and every primary alternative |
| `array_expression` through `record_field` (212-219) | `bodies.fy` | 21-22; empty/list/repetition arrays, named/integer counts, contextual/qualified/generic records, empty/trailing fields |
| `if_expression` through `unsafe_expression` (221-238) | `bodies.fy`, `body_state.fy` | 13, 23-24; else-if and absent else, shallow payloadless/positional/named/empty patterns, case value blocks, unsafe blocks |
| `literal` through `path` (240-259) | `scanner.fy`, `numbers.fy`, `strings.fy`, `bodies.fy`, `types.fy` | 00, 08, 21-22, 25-26; every literal kind, const-expression alternative and path composition |
| `DEC_INTEGER` through `CONTEXT_C` (265-294) | `scanner.fy`, `numbers.fy`, `types.fy`, `syntax.fy` | 00, 06-09, 25; decimal/hex/binary separators, both float forms and exponent signs/case, identifier and C contextual spelling |
| `STRING_TOKEN` through `EOF` (296-324) | `scanner.fy`, `strings.fy`, `utf8.fy` | 26 and full-source checks; text/byte/C strings, every simple escape, byte/Unicode escapes, raw UTF-8, quote/backslash and source-end terminals |

The map is not a claim that one successful example proves all cross-products or
malformed inputs. The existing focused parser tests additionally own expression
and type tree shapes, ordered children, spans, rollback, detached-node mutations,
comparison-chain rejection, lookahead, depth and shared-arena boundary checks.
The grammar witness adds a cross-family recognition gate: each source must parse
without diagnostics, pass integrity verification, retain its original byte
length, and match every Go declaration's kind, order, full span and name span.
Native checks run in both Windows low and optimized profiles.

## Audit corrections

- The standalone block/if shortcuts were not complete expression readers. They
  now share the normal expression path, including branch tails, continuations and
  written semicolons. See `stage0-syntax-coverage.md` for the implementation and
  unchanged-budget self-corpus evidence.
- The cast test still rejected the now-supported `x as ptr T`. It now accepts
  pointer, mutable-pointer and C-function-type casts with a Go syntax oracle;
  `x as ptr` retains missing-pointee rejection and rollback coverage.
- The older variant test still rejected `.R {.x=1}`, despite contextual record
  construction already being supported and tested by the record suite. It now
  accepts that expression and rejects `.R {.x=}` instead. No malformed-input
  assertion was relaxed and no parser feature was removed to satisfy a test.

## Verification and boundary

The audit is complete for the existing Stage 0 syntax boundary. No identified
grammar production remains unsupported within the unchanged candidate limits.
The detailed feature ledger remains the implementation history; this page owns
the final grammar map. Final checks on the implementation at `055ee677`:

| Configured intent | Verified result |
| --- | --- |
| `fairyc_go_test_parser_evidence` | 29.952s; all 27 grammar witnesses, all 22 required fixtures (12 accepted, 10 rejected), inventory controls and private integrity tests; both profiles |
| `fairyc_go_test_parser_operands` | 57.328s after the stale cast expectation was corrected; arrays, casts, generic calls and references, including recovery and boundary checks; both profiles |
| `fairyc_go_test_parser_closure` | 39.712s; contextual variants, cleanup bodies and repeated arrays, including the corrected contextual-record expectation; both profiles |
| `fairyc_go_test_parser_sdk_gate` | 10.931s; all seven required SDK files completely parsed with exact Go declaration records; missing/partial/exhausted inventory controls also pass |
| `fairyc_go_test_parser_self_corpus` | Reused unchanged implementation evidence: all eighteen sources, 8.711s, both profiles |
| `fairyc_go_test_parser_candidate` | Reused unchanged implementation evidence: full block/entry/expression oracle and budget controls, 197.530s, both profiles |
| `fairyc_go_test_parser_values` | Reused unchanged implementation evidence: block/if values, enum declaration-construction-decomposition and named-pattern recovery, 84.521s, both profiles |

The first operand run failed because its legacy negative expected a valid pointer
cast to be rejected; the updated run above passed. The final audit changes tests
and private documentation only. Full stable compiler, semantic/runtime and release
suites were not rerun: those surfaces are unchanged and are not promoted by this
syntax task. There is no delegated standalone docs-validation intent; local links
and the exact documentation diff were reviewed directly. No remote CI, push or
publication claim is made.

The original 32768-byte source cap, shared arenas, recursion and 256-significant-
token generic lookahead remain part of the candidate contract. Exceeding a budget
still fails rather than reporting partial success. Semantic rejection is not a
syntax omission. Generic enum headers, enum discriminants, empty type arguments,
boolean/byte-string attribute arguments and attributes on resource fields are not
admitted by this normative grammar; broader Go acceptance does not expand it.
No stable compiler/language/runtime/SDK version or public API is changed.
