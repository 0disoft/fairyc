# ADR 0447: Asserted checked-failure reachability in C emission

- Status: implemented; Windows dev/release native verified, Linux execution deferred
- Date: 2026-10-04
- Related: ADRs 0412, 0444, 0445, 0446

## Problem and oracle

The ADR 0446 baseline reads a nested i64 through a shared-return helper,
asserts that it equals the maximum, then increments the same stored field.
Semantic checking and the MIR interpreter accept the defined integer-overflow
trap. MSVC recognizes that the callee cannot return and rejects its generated
caller continuation with C4702/C2220 (CLI exit 6). The newly retained regression
reproduces that failure before this patch, in addition to the prior clean
251d15b0 baseline receipt. Warnings remain enabled and treated as errors.

Both dev/release builds must succeed. Execution must still distinguish an
assertion failure from a subsequent integer-overflow failure, preserve preceding
effects, and omit effects after the trap. A write or unknown call that changes
the asserted value must retain the normal continuation.

## Bounded proof

C emission builds transient facts from verified MIR. On the success edge of an
assertion whose condition is scalar equality with an integer literal, it records
that value for the selected scalar or stored field. Assign/initialize copies,
shared reborrows, field borrows, field reads and dereference snapshots preserve
identity only while their referents remain unchanged. A separate summary admits
only an effect-free single-block helper made of reference copies/projections and
one return of a borrowed parameter or its field path. It never trusts an origin
annotation alone or summarizes arbitrary calls, recursion or writes.

Facts flow only along unique-predecessor normal paths. Merges and unwind entries
start without facts; recursive predecessor traversal cuts cycles. Redefinitions,
addressed-local writes, indirect/field writes, unknown operations and unknown
calls discard all facts and identities. Unmodeled values remain unavailable.
The proof selects only checked + / - / * or signed-negation failure guards using
ADR 0444's overflow comparisons; it does not evaluate overflowing arithmetic,
fold safe arithmetic, infer inequalities or optimize division/shifts.

The same per-function failure map drives emitted guards, reachable blocks,
non-returning-callee inference and integer-helper demand. Assertion checks and
failure edges remain in their original order. Existing unwind edges remain
conservatively reachable. Verified MIR is never modified; both C profiles retain
full MIR capability checks before pruning. No warning suppression, noinline
workaround, host flag change or cache bypass is introduced.

## Verification

`internal/compiler/testdata/assert_checked_trap.fy` retains the original source
shape. Compiler tests compare interpreter outcomes and unchanged serialized MIR,
and inspect both Windows and Linux generated C for removed continuations/helpers.
They cover shared-call/direct-field/scalar assertions, earlier assertion failure,
field/scalar writes, alias writes, unknown calls and aggregate snapshots. Backend
boundary tests additionally reject facts across wrong places, false edges,
merges/backedges, unwind, unknown operations and writing reference helpers.

Windows native tests execute the original/shared-field traps, assertion failure,
scalar rebinding, unknown-call mutation and field mutation under both profiles.
The ADR 0444 ten-case native corpus and ADR 0446 success/dynamic-trap cases also
pass: 36 native executions total. A freshly built public compiler checks, builds
and runs the exact baseline and an effects fixture under dev/release; only
`before` appears before the integer-overflow trap. Backend tests and related
compiler tests pass; Go vet passes. Raw outcomes, compiler/SDK/source digests and
image hashes are recorded in `docs/ops/evidence/assert-checked-c-20261004.json`.

## Compatibility and limits

SDK stage0.84, language /20, runtime /0, grammar, MIR/wire schemas, public APIs,
ABI and target maturity remain unchanged. Compiler revision and executable
identity distinguish implementation/cache generations. Linux C emission is
inspected; actual GCC/Clang execution remains deferred to the owner's batch.
The existing Linux strict-consumer test skips on this Windows host. This bounded
proof does not cover arbitrary assertion expressions, alias equivalence, merged
facts or host deductions beyond the modeled paths. No whole-repository suite,
benchmark, dependency, installation, CI or runner change is made.
