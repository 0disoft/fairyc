# Fairy compiler candidate: requested units 1..20

This ledger preserves the owner's full sequential objective. Units port existing
language semantics into `examples/lexer_candidate`; source acceptance by the Go
bootstrap is separate from acceptance and execution by the Fairy candidate.
Each completed unit receives a local commit. Linux execution and performance
work remain deferred to the owner's consolidated batch.

| Unit | Required result | State / evidence |
| --- | --- | --- |
| 1 | Option types/owned HIR prerequisite and MIR some/none construction | Complete: candidate-option-mir-20261007.json; hir_options native test |
| 2 | Option MIR owner/type/tag/payload/initialization mutation rejection | Complete: candidate-option-verifier-20261007.json; 5 layout and 15 operation mutations |
| 3 | Option C construction/copy/call/return/switch native execution | Complete: candidate-option-native-20261007.json; 26 normal native runs and 2 injected inactive-payload traps |
| 4 | Option if/loop/early-return value and control-flow preservation | Complete: candidate-option-flow-20261007.json; joins, subject-once, early exits and payload traps |
| 5 | Result success/error type identity and type checking | Complete: candidate-result-types-20261007.json; 36 ordered scalar pairs, argument/return/assignment rejection |
| 6 | Result construction and exhaustive contextual patterns | Complete: candidate-result-patterns-20261007.json; contextual constructors, both payload types and terminal arms |
| 7 | Owned Result HIR, payloads, bindings and source locations | Complete: candidate-result-hir-20261007.json; owned four-field layouts, captures/spans and seven forged layouts |
| 8 | Result MIR/C execution for both alternatives | Pending |
| 9 | Postfix ? position and return-error compatibility checks | Pending |
| 10 | Postfix ? success extraction and early error return execution | Pending |
| 11 | Result must-use handling obligations | Pending |
| 12 | Nested Option/Result combinations needed by compiler sources | Pending |
| 13 | Struct/diagnostic payloads in Option and Result | Pending |
| 14 | Required enum/Option/Result storage in struct fields and arrays | Pending |
| 15 | Resource type identity and field ownership facts | Pending |
| 16 | Explicit resource factories and construction typed HIR | Pending |
| 17 | Whole-value moves in assignments, calls and returns | Pending |
| 18 | Use-after-move rejection | Pending |
| 19 | Branch-sensitive moved state and subsequent use | Pending |
| 20 | Loop/backedge-sensitive moved state and subsequent use | Pending |

Completion requires actual source behavior and matching verification, rather
than parser coverage or Go-bootstrap-only behavior. Cleanup and broader SDK
integration remain later numbered work; these rows do not claim them complete.
