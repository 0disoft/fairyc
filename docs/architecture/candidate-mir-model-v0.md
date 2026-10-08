# Candidate MIR model observations

These private observations accompany `examples/lexer_candidate/src/mir_model.fy`.

Enum operations: 21 constructs a tag and declaration-order payload operands;
22 tests a tag. Operator is the dense member ordinal, excluding payload fields.
23 checks the active tag and copies one scalar payload field; operator is its
Field ID. A mismatch traps as unreachable (7) before accessing union storage.
Private mutable-local scalar MIR. IDs are one-based; zero is absent.
Ranges and edge/signature indexes are zero-based. No HIR tree is retained.
Scalar type tags match hir_model: 1 i32, 2 i64, 3 u8, 4 u32, 5 usize,
6 bool, 7 void. Operators match hir_model's exhaustive operator_code.
Operation kinds: 1 constant, 2 copy/assignment, 3 unary, 4 binary, 5 check,
6 construct (declaration-order operands), 7 field read, 8 functional update.
9 array construct (boolean means repeat once), 10 indexed read, 11 array
update (one value edge), 12 checked index, 13 array length. Checked indexes
freeze usize operands and trap bounds before loads or RHS/store effects.
18 shared named u8-array view, 19 checked byte slice (one end-bound edge).
14 static literal descriptor, 15 view length/empty (boolean selects empty),
16 checked view index (left frozen descriptor, right raw usize), 17 byte read.
A check defines no local. It must end its block immediately before a matching
checked terminator. Actual arithmetic occurs only in the success block.
24 transfers one owned root (donor becomes Dead); 25 discharges an empty
cleanup. Nonempty cleanup uses a kind-5 call with type=exact resource type;
ordinary calls have type=0 and cannot call a private drop target.
Terminators: 1 return, 2 goto, 3 bool branch, 4 checked switch, 5 call,
6 trap, 7 unreachable. Calls have one normal successor, no unwind edge.
Checked switches name check Operation ID plus frozen local operands.
Trap mask: 1 integer-overflow, 2 division-by-zero, 4 signed-division-overflow,
8 invalid-shift, 16 assertion-failed. Trap blocks have no successor/cleanup.
Construction caps: 1048576 source bytes, 512 functions/targets, 16384 locals,
16384 blocks, 32768 operations, 65536 edges/signatures, 688128 payload bytes.
1 invalid input, 2 budget, 3 allocation, 4 malformed/unsupported graph.
This is a checked construction boundary, not the future complete MIR verifier.
It rejects unfinished blocks, malformed links and ownership/range drift.
