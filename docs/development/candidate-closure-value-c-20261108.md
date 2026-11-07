# Candidate copied closure C values (2026-11-08)

Scope: ADR 0551's private Fairy C backend implementation in the shared isolated
`closure-runtime-20261108` checkout, based on `97e3a617`. No source-of-truth
contract, Go backend, syntax/type/HIR/MIR owner, artifact/cache codec, lockfile,
main checkout, commit, push, runner configuration or compiler flags changed by
this worker. Other workers own the new MIR API and source pipeline.

## Implementation

- `c_text.fy` spells direct type 13 as `struct fy_closure_i32`.
- New `c_closure_values.fy` emits the private invoke pointer and four inline i32
  capture slots. Literal adapters call the actual safe pure lifted target;
  construction explicitly zeros all unused slots. Targets proven not to return
  retain their void/noreturn real signature behind the i32 invoke adapter.
- `c_scalar.fy` handles op 40 before the pre-existing high-kind operation lane.
- `c_emit.fy` initializes closure locals as aggregates, emits the private type
  before nominal fields and adapters after helper prototypes, and handles
  terminator 8 by snapshotting callable then argument exactly once.
- `c_flow.fy` explicitly retains terminator 8's normal continuation. The current
  candidate C path emits **all** supplied MIR functions, so construction-only
  helper targets are retained without a separate target pruning pass. No
  `c_prune.fy` or `c_reach.fy` exists in this checkout. Module-unit C reachability
  uses a different IR and is outside this worker's direct-MIR change.
- Backend admission independently rejects closure-bearing foreign, pointer,
  reference, array, tagged and resource transports, repr(C) records, layout/raw
  operations and malformed closure metadata. The common MIR verifier remains
  responsible for operand initialization, body proof, checked edges and exact
  construction/invocation forms before any C output is written.
- Programs without closure values produce no new declarations or adapters;
  existing scalar/nominal output branches are unchanged. Native addresses exist
  only in generated C initializers; MIR continues to store target IDs and locals.

## Focused acceptance harness

`examples/lexer_candidate/tests/closure_value_c.fy` runs original source bytes
through the authored scanner, preparation, name resolution, type checking, HIR,
MIR, verifier and C emitter. It reports case name, phase, error code, byte range
and original source on stderr before failing. Stdout contains length-framed C
members only. Preparation's public failure API has no precise span; that phase
reports the complete original source interval.

`internal/cli/candidate_closure_value_c_test.go` builds one bounded producer per
normal low/optimized MSVC profile, then builds each emitted C member under both
consumer profiles with existing flags and timeouts. It checks deterministic C
bytes across producers, native exit/trap category and empty stdout/stderr.

Each producer runs these 14 original-source cases; each resulting C member runs
under both normal consumer profiles:

| Case | Native exit |
| --- | --- |
| scalar-unchanged | 42 |
| dead-typed-literal-after-return | 42 |
| signature-only-no-literal | 42 |
| ordinary-generic-beside-closure | 42 |
| returned-four-captures-struct-snapshot | 42 |
| different-targets-one-parameter | 42 |
| unused-trapping-body-copy | 42 |
| unused-local-traps-on-invocation | 82 (division by zero) |
| argument-trap-precedes-body-overflow | 82 (division by zero) |
| unused-body-overflow | 81 (integer overflow) |
| runtime-signed-divide-overflow | 83 (signed division overflow) |
| runtime-divide-success | 42 |
| signed-boundary-success | 42 |
| argument-once-body-locals-repeat | 42 |

The returned factory covers four copied captures, mutation of its original
capture, repeated invocation, ordinary parameter/result transport, direct Holder
copies and extracting the closure field before invocation. Different literals
share one typed callable parameter. Dead post-return construction leaves no
orphan metadata or invocation. Twenty-eight forged MIR mutations fail assembly
or render with the output prefix intact, per producer.

The 14 original-source refusals, per producer, are generic closure transport,
nested Box/Holder carriers, repr(C) closure fields, foreign closure-bearing
struct parameters, foreign pointers to closure-bearing structs, direct and
Holder-wrapped Option, Result, fixed arrays, direct foreign callable parameters,
capturing a closure, mutable callable binding, equality and casts. Nested
carriers remain closed in the current Go MIR (V1032); no exact cross-engine
diagnostic/status parity is claimed.

Signature-only parameter/result transport without a literal is a positive case
with empty target metadata in the explicit new MIR family. An unrelated i32
generic call beside a typed factory checks private-helper sidecar preservation
through actual generic expansion; it does not admit generic closure transport.

The C trap observer records the emitted candidate category through exit code
`80 + category`. It does not replace a check or change compiler warning policy.
No MSVC workaround has been added speculatively.

## Validation status

- Passed: `go test ./internal/cli -run '^TestCandidateClosureValueCSource$' -count=1`.
- Passed: `go test ./internal/cli -run '^TestCandidateClosureValueCCheck$' -count=1 -v`
  (174 sources, 1,940 MIR functions).
- Passed: scoped `git diff --check` for the four existing C consumers.
- Passed: temporary `FAIRYC_RUN_PARITY_INTEGRATION=1`,
  `go test ./internal/cli -run '^TestCandidateClosureValueC$' -count=1 -v -timeout 240s`:
  67.02 seconds, 45-part low and optimized producers, 56 native C executions,
  28 forged MIR mutations and 14 original-source refusals **per producer**.
  Every expected exit matched, execution streams were empty and all generated C
  members were byte-identical between producer profiles. No additional MSVC
  guard workaround, warning relaxation or timeout increase was needed.
- Removed the temporary environment opt-in after successful execution. The
  Windows AMD64 feature test now runs normally without that environment value;
  the platform guard and existing low/optimized compiler settings remain intact.
- Final parent acceptance from the current ungated test passed:
  `go test ./internal/cli -run '^TestCandidateClosureValueC$' -count=1 -v -timeout 240s`.
  `.cache-closure-value-c-attempt2.log` records test duration **64.44 seconds**
  and package duration **64.562 seconds**. Both Fairy producer profiles passed
  all 14 source cases through both C consumer profiles (56 native executions),
  plus 28 forged MIR mutations and 14 source refusals per producer. Original
  source C output was byte-identical between producer profiles. Nested-carrier
  and wrapper refusals were unchanged. The final cleanup only records this
  result; it changes no implementation, test case or compiler setting and runs
  no additional compilation or test.
- Scalar-only emitted-C SHA-256:
  `3579d6952b1aebcd743a3dc74bcdda95994c009fad677046ed7b982c66d0b0b4`.
- Earlier integration blocker (resolved by parent): signature-only source MIR
  code 4, span 48..54, missing type 13 in `mir_state::local`. No backend gate was
  weakened to bypass it.
- Not run: full compiler bootstrap, Linux, artifact/cache restoration, remote
  CI, publication. Parent owns integration and commit.

Native acceptance above covers original-source direct-MIR C execution only.
Source pipeline failures belong to the corresponding parent/worker owner;
this harness deliberately preserves their original failure and diagnostics.
