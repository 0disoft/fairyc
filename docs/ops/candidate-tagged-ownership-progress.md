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
| 8 | Result MIR/C execution for both alternatives | Complete: candidate-result-native-20261007.json; 36 ordered pairs in both arms, 36 MIR mutations, 116 native executions and 6 sample/coexistence executions |
| 9 | Postfix ? position and return-error compatibility checks | Complete: candidate-result-propagation-types-20261007.json; 190 CLI observations, 6 native API tests, exact question spans and error identity, 3 Go reference checks |
| 10 | Postfix ? success extraction and early error return execution | Complete: candidate-result-propagation-native-20261007.json; 8 forged HIR controls, 14 literal-guard facts, 6 native API tests, 937 CLI observations, 130 native executions and 2 sample executions; optimized always-trapping caller emission fixed |
| 11 | Result must-use handling obligations | Complete: candidate-result-handling-20261007.json plus candidate-result-handling-alignment-20261007.json; 449 CLI observations, 58 native runs, 14 native API tests and 237 prior positive checks. Go's ? error-edge omission fixed; related Go tests and all 24 Result fixtures pass; all eight reference cases now agree in both candidate profiles. Original bootstrap EXEs preserved. |
| 12 | Nested Option/Result combinations needed by compiler sources | Complete: candidate-nested-tagged-20261007.json; 52 source fixtures, 417 CLI observations, 78 generated native executions, 16 native API tests, 17 aligned Go reference cases and 6 existing-sample native regressions. Fixed MSVC C1054 on deep implicit aggregate initialization and Go's omitted Result capture obligations on exited break/continue edges. Scalar bootstrap EXEs preserved. |
| 13 | Struct/diagnostic payloads in Option and Result | Complete: candidate-struct-payloads-20261007.json; 27 fixtures, 217 CLI observations, 30 generated native executions, 16 native API tests, and 2 sample executions. The existing nested lane passes 417 CLI/78 native observations. Go comparisons include 15 accepted programs, 6 diagnostic rejections and 2 known subset-unsupported rejections; Go's semantic-only acceptance of recursive layouts is recorded separately, while the candidate rejects the cycle. Fourteen-bit nominal leaves preserve declaration identity; thirty-bit composite capacity remains explicit. |
| 14 | Required enum/Option/Result storage in struct fields and arrays | Complete: fields and array types/HIR in the preceding part receipts; candidate-tagged-array-native-20261007.json adds 33 fixtures, 265 CLI observations, 46 generated native executions and 4 final array API tests in dev/release. Construction, repetition, copy, indexing, stores, calls/returns, field storage and inferred rank-two layouts execute. Verifiers reject wrong elements/index lengths and forged nominal array operations before publication. RHS sibling changes, initializer order/once and bounds-before-RHS are checked. A late nested-tag array after 1024 type nodes previously asserted; bounded 4096-node conversion fixes it and the 350-field reproduction now executes. The 24 pre-fix array/struct/nested API regressions retain their own image/source identities in the receipt. Go tagged-array subset rejections remain separately recorded; resource/view elements and higher ranks remain later work. |
| 15 | Resource type identity and field ownership facts | Complete: candidate-resource-types-20261007.json; 32 fixtures, 256 source CLI observations, 5 native compiler API tests in each profile, 27 Go reference observations, and a focused array regression with 57 CLI observations / 10 native executions. Resource IDs and tagged payload identities remain disjoint; owned HIR retains copy/drop/field ownership facts. Missing/nonempty cleanup, resource arrays and invalid storage reject; all resource MIR/C execution stays closed. Go's cycle acceptance and fixed-array resource-field subset limitation are recorded separately. |
| 16 | Explicit resource factories and construction typed HIR | Complete: candidate-resource-factories-20261007.json; 24 fixtures / 192 CLI observations, 3 native compiler API tests per profile, six forged construction mutations, sample check/HIR in both profiles, 24 Go comparisons and the 32-fixture unit-15 regression. Same-module ordinary factories and exact resource initializers retain declaration identity, nested ownership and written order. All resource MIR/C remains closed; imported visibility and nonempty cleanup remain later units. Go's support for nonempty cleanup and its fixed-array resource-field subset limitation are recorded separately. |
| 17 | Whole-value moves in assignments, calls and returns | Pending |
| 18 | Use-after-move rejection | Pending |
| 19 | Branch-sensitive moved state and subsequent use | Pending |
| 20 | Loop/backedge-sensitive moved state and subsequent use | Pending |

Completion requires actual source behavior and matching verification, rather
than parser coverage or Go-bootstrap-only behavior. Cleanup and broader SDK
integration remain later numbered work; these rows do not claim them complete.
