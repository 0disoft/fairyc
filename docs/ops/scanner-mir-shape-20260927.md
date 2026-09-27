# Scanner MIR shape comparison

Date: 2026-09-27. The [receipt](evidence/scanner-mir-shape-20260927.json)
projects verified executable MIR for the two real scanner functions changed
by the three private vector loops. The indexed and candidate scanner source
hashes match the existing generated-C attribution. Both package analyses
produce 21 SDK specializations.

| Function and source form | Blocks | Operations | Call sites | Drop operations | Trap paths |
| --- | ---: | ---: | ---: | ---: | ---: |
| `scanner::report_work`, indexed | 74 | 100 | 18 | 6 | 9 |
| `scanner::report_work`, vector | 75 | 101 | 18 | 6 | 9 |
| `scanner::storage`, indexed | 24 | 33 | 9 | 0 | 4 |
| `scanner::storage`, vector | 26 | 39 | 9 | 0 | 4 |

For `report_work`, assignments change from 33 to 35, unary operations from
7 to 6, and `goto` terminators from 24 to 25. For `storage`, assignments
change from 12 to 20, field borrows from 7 to 5, and `goto` terminators from
4 to 6. The distinct callee counts remain 11 and 7 respectively. The test
also requires equal cleanup-path and cleanup-action kind counts: the reporter
has six drops on its return cleanup path in both forms, while storage has no
cleanup actions. These changes are consistent with the candidate's hidden
owner, bound and index, but the projection alone does not assign each added
operation to a particular source construct.

The projection omits source spans, entity IDs, local IDs, operand data and
full CFG edges. Generic callee names are source-hash-dependent and are not
paired by semantic identity here. Equal call-site and cleanup counts therefore
do not prove equivalent helper behavior, dataflow or execution semantics.
The separate low/optimized native tests establish the covered scanner outputs
and cleanup behavior; the generated-C and linked-size receipts remain separate.
This comparison closes the missing structural accounting for these functions,
not the complete normalized-MIR equivalence, build-time or 8% runtime gates.
