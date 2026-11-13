# Flat signed-error Result transport — Fairy slice, 2026-11-14

The candidate admits the ADR 0555 family in both executable type resolution and
the separate original-source generic-signature admission pass. Explicit i32 and
resolved E=i32 both use the unchanged legacy scalar tag 1; wider errors stay
closed. No artifact schema, type ID vocabulary or specialization bound changed.

Current-source `TestCandidateCompositeResultSignedErrorPipeline` passed in
284.1s. Its copied 104-part compiler closure produced actual C from both low and
optimized native producers after encoding/decoding/restoring body/program/project
artifacts against the source. All four producer/consumer profile combinations
returned 42 with empty output. Independent C observations cover 13 scalar cases
and 26 exactly-once operands, including extrema, signed zero and -77 errors.

`TestCandidateCompositeResultGraph` passes (18.7s), including the exact 4096-type
boundary and rollback of success scalar, signed error and Result records. Its
fixture now constructs its known-unique prefix linearly instead of quadratically
interning 4096 fixture rows; the actual specialization/interner is still tested.
The resolved-E refusal now explicitly selects i64 rather than the newly admitted
i32 slot.

`TestCandidateCompositeResultSourceDiagnostics` passes in 137.0s: 98 signature
cases plus 21 handling/evaluation cases under both profiles. The expanded driver
hit the existing one-minute monolithic optimized C command limit, so the test
now uses established C partitions and the same per-command deadline. No product
limit was increased. Overall focused replay: 440.1s, log
`.cache/signed-result-fairy-20261114.log`. Go reference controls also pass.

Whole authored CLI/persistent-cache acceptance is intentionally batched with the
other item-3 families. This receipt does not claim that common gate complete.
The example lock was refreshed through workspaceloader/lockfile APIs; that
metadata-only operation is not counted as another semantic check.
