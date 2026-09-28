# Scanner MIR delta by vector loop

Date: 2026-09-29. Child source baseline: `14906413`. The
[raw receipt](evidence/scanner-mir-loop-delta-20260929.json) analyzes the
checked-in indexed scanner, each of its three private Vec-loop substitutions
separately, and the combined substitution. Every analysis reaches verified
MIR with 21 SDK specializations. Each row below compares one substitution to
the same indexed baseline.

| Substitution | Function | Reachable blocks | Reachable operations | Operation-kind delta | Successor-edge delta |
| --- | --- | ---: | ---: | --- | --- |
| Token | `report_work` | +1 | +1 | `assign` +2, `unary` -1 | `goto` +1 |
| Cooked buffer | `storage` | +1 | +3 | `assign` +4, `field-borrow` -1 | `goto` +1 |
| Diagnostic | `storage` | +1 | +3 | `assign` +4, `field-borrow` -1 | `goto` +1 |
| All three | Both functions | +3 | +7 | Sum of the three rows | `goto` +3 |

The other function is unchanged for each single-loop variant. The test
requires the combined block, operation, operation-kind, operand-shape and
successor-kind deltas to equal the sum of the three isolated deltas.

Operand shapes resolve source-hash-dependent SDK generic names to their exact
declaration and type bindings. After that normalization, the token loop has
two more assignments from local operands and one fewer shared Vec borrow.
Each storage loop has three more assignments from local operands, one more
`usize` zero assignment and one fewer borrow of its respective vector field.
This is consistent with the compiler's owner, bound and index preparation,
but the receipt does not map each numeric MIR local to a specific hidden
binding. The original unnormalized diagnostic was retained only in ignored
`.cache/` and is not used as evidence.

All added blocks and operations are reachable, and the only added successor
edges are unconditional `goto` edges. Operand-shape equality still omits
numeric local identities, source positions, successor targets and actual
definition/use chains. It is not full CFG or dataflow equivalence, nor a
native instruction count. Compiler-time parity and the independent-C 8%
runtime gate remain open; the Linux C-only null control still fails.
