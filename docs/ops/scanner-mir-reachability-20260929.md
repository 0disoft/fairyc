# Scanner MIR reachability comparison

Date: 2026-09-29. Child source baseline: `c2cd68c9`. The
[raw receipt](evidence/scanner-mir-reachability-20260929.json) extends the
verified-MIR shape projection for the current explicit-index and private
three-loop vector scanner. Both variants were analyzed and lowered from the
same current checkout. The scanner source hashes in this receipt differ from
the 2026-09-27 shape receipt, so the paired rows below are the comparison.

| Scanner function | Indexed reachable blocks / operations | Vector reachable blocks / operations | Added reachable `goto` edges | `call` edges | `switch` edges |
| --- | ---: | ---: | ---: | ---: | ---: |
| `report_work` | 74 / 100 | 75 / 101 | 1 | 18 / 18 | 42 / 42 |
| `storage` | 24 / 33 | 26 / 39 | 2 | 9 / 9 | 12 / 12 |

Every reported block and operation is reachable from its function entry in
both forms. The candidate has three additional reachable blocks and seven
additional reachable operations across these functions. All three added
successor edges are unconditional `goto` edges; the counts of `call` and
`switch` successor edges are unchanged. The traversal follows each verified
terminator's successor list, counts each block once, and fails if a reachable
successor is missing.

These are MIR graph counts, not native instruction counts. The projection does
not compare successor targets, operands, local values or dataflow, and it does
not prove full normalized-MIR equivalence. The existing native output and
ownership tests and generated-C attribution answer different questions.
Compiler-time parity and the independent-C 8% runtime gate remain open. The
Linux C-only null control still fails, so this deterministic receipt does not
change the performance verdict or admit the source form as stable.
