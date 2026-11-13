# Flat signed-error Result transport — Go slice, 2026-11-14

ADR 0555 extends the seven existing success transport scalars to root
`Result<S,i32>`. Generic signatures admit the same resolved error type; the C
backend uses its existing signed field layout. No syntax, operations, scalar
argument vocabulary, artifact tags or Result foreign ABI were added.

`go test -mod=readonly ./internal/compiler -run '^TestCompositeResult' -count=1
-timeout=5m` passes in 11.5s. New controls cover all seven success scalars,
resolved T/E, immutable authored AST, copies and negative error propagation.
Independent C observers cover extrema and floating signed zero; source and
codec-restored MIR run under low/optimized native profiles with expected values
and exactly-once counters. Wider i64 errors and other previously closed wrappers
remain negative controls. Focused C-backend candidate-boundary tests also pass.

Candidate source/artifact/native validation is still running in the next slice.
The first attempt caught a wrong legacy scalar tag (i32 is 1, not 2); that
candidate edit was corrected before committing its implementation. Its separate
graph test now seeds bool rather than i32 to require materializing all three
new records and verify complete rollback at the 4096-type boundary.
Whole CLI/cache acceptance for this new family is not claimed by Go evidence.
