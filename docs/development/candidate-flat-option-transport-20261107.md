# Fairy flat Option transport, 2026-11-07

Owner: [ADR 0547](../adr/0547-flat-option-generic-transport.md). This implements
the bounded Fairy half of the Go/Fairy extension; the
[Go record](go-flat-option-transport-20261107.md) owns its separate admission,
target layout, C11/C17 and native results. Linux remains batched.

## Implementation

`module_types::walk_type` receives a private leaf permission only when a root
Option has exactly one direct named payload. Recursive traversal uses `walk`,
which resets the permission. Descriptor handles, already-interned substitutions
and written names converge on the same canonical scalar/Option nodes. Nested
Option, Result, references, pointers, arrays and views do not inherit the new
permission. Aggregate-enabled resolution keeps its previous independent policy.

`generic_signature_diagnostics` applies the same complete-shape exception while
retaining concrete-argument validation, source matching, work charging, original
spans, sorting and duplicate compaction. Unsupported signatures still report
their written source form. No type tag, graph bound or artifact/cache schema is
changed. Existing cleanup/IR verification, public type rendering, source-owned
artifact restoration and tagged C emission carry the new Option values.

## Windows execution

All commands set `GOMAXPROCS=1` and `FAIRYC_RUN_PARITY_INTEGRATION=1`.
Tests use existing compiler/command bounds and warning flags.

```powershell
go test -mod=readonly -count=1 -failfast -timeout 6m -v ./internal/cli -run '^TestCandidateGenericTransportAdmission$/^generic_scalar_arguments$'
go test -mod=readonly -count=1 -failfast -timeout 12m -v ./internal/cli -run '^TestCandidateCompositeOptionTransportPipeline$'
go test -mod=readonly -count=1 -failfast -timeout 12m -v ./internal/cli -run '^TestCandidateGenericTransport(Admission|Pipeline)$' -skip '^TestCandidateGenericTransportAdmission$/^generic_scalar_arguments$'
go test -mod=readonly -count=1 -failfast -timeout 4m -v ./internal/cli -run '^TestCandidateGenericTransportAdmission$/^generic_signature_diagnostics$'
```

The graph selection passes in **14.819s** under low/optimized. Seven payloads
test descriptor, interned and written canonicalization. An exact 4,096-node
result succeeds; starting with 4,095 records leaves room for only the new scalar,
then wrapper admission fails with code 10. Every original record is checked
after rollback, not just the count. Existing wrong-descriptor/stale-source and
other-wrapper refusals run in the same selection.

The new restored-artifact pipeline passes in **209.852s**. Its independent Go
workspace check first admits the exact source, with only the workspace module
name adapted to its physical path. Actual Fairy compiler producers at low and
optimized each build current-source Complete, encode/decode the artifact,
restore it against the original sources and emit C. Four producer/consumer
combinations compile/link/execute successfully, returning 42 with empty output.

The source passes Some through a generic Option parameter/result, copies and
assigns it, constructs None and branches on its tag, then extracts the active
Some payload. The independent C observer checks i8/i16 minima, an isize below
INT32_MIN, u16/u64 maxima and finite f32/f64 values. Additional `memcmp` checks
preserve positive/negative zero and finite payload bytes. Thirteen scalar
observations run per product (52 total, including repeated finite values).
No inactive None payload or aggregate padding is observed; no new FFI lane is
used. Private typed probes keep the actual scalar C signatures.

The existing direct-scalar pipeline passes in **198.25s**, and concrete-type
controls pass in **22.19s**. The combined command then fails on the old fixture
case-count assertion: the Go oracle now has 24 cases rather than 22. That
failure is retained in `.cache/flat-option-diagnostics-regression-20261107.log`.
Only the fixture's expected count is updated before the focused diagnostic
replay; the production budgets are unchanged. The focused final replay passes
both profiles in **29.134s**, comparing all 24 independent Go cases and original
diagnostic spans/features, including the new positive flat Option and negative
nested Option/Result sources. Log:
`.cache/flat-option-diagnostics-final-20261107.log`.

Fresh current Go bootstrap SHA-256 is
`00be89383b0049a349b0bf8559ea2c81c86e8af85feddf349069001ce53982fa`.
Final isolated `check` / `check --locked` pass in **14.968s / 12.817s** with
755 source files / 9,543 MIR functions and empty stderr. All 1,226 authored
Fairy files and the lock match the checked copy. Receipt:
`.cache/flat-option-source-20261107-all-check.json`; final log:
`.cache/flat-option-source-check-final-20261107.log`. The earlier source check
belongs to the prior fixture count and is not substituted for this final copy.
Scoped compiler/C-backend/CLI vet, gofmt and diff checks pass. The complete
C-backend package passes in 77.462s after its obsolete flat-Option rejection
fixture is changed to a nested Option rejection, without changing the emitter
to preserve an obsolete refusal.

Earlier new-pipeline attempts fail on a workspace module/path mismatch and on
the copied direct fixture's unsupported Go extended-scalar defer call. The
oracle adapts only its physical module name, and the new Option fixture omits
defer (which is outside this decision). Both failed logs remain separate;
neither becomes a successful execution claim. Logs:
`.cache/flat-option-graph-20261107.log`,
`.cache/flat-option-native-pipeline-20261107.log` and the two `attempt` logs.

## Remaining gates

Option-specific actual full CLI/fresh-process persistent-cache execution,
refreshed self-build/fixed-point, native Linux, full Go suite, hosted CI and
clean-machine distribution remain pending. The running closure-local and Atlas
checks are independent families. No source grammar, public CLI flag, SDK/runtime
ABI, DB/provider, dependency, runner, repository policy or product timeout change.
The accepted ADR and candidate transport architecture are the contract updates;
no push, dispatch, release, deployment or publication occurs.
