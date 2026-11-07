# Go flat Option transport execution, 2026-11-07

Source base: `main` at `0bb199e7`, with the scoped uncommitted Go edits below.
Owner: [ADR 0547](../adr/0547-flat-option-generic-transport.md).
This is Go execution evidence; the coordinator owns Fairy, shared docs, queue,
lockfile and commits. No push or commit was performed by this worker.

## Implemented files

- `internal/compiler/generic_admission.go`: exact flat `Option<S>` exception
  after structural substitution; other extended wrapper signatures remain closed.
- `internal/compiler/generic_admission_test.go`: seven scalars times ten rejected
  wrapper shapes, with valid typed forwarding calls, two deduplicated written
  unsupported spans and resolve-first missing-name checks.
- `internal/compiler/generic_admission_budget_test.go`: existing exhaustion
  tests now use still-unsupported Result instead of newly supported Option.
- `internal/compiler/composite_option_transport_test.go`: private, explicit-file
  and package analysis; direct written and binder-substituted Option; source AST
  immutability; MIR verification; existing C profiles; negative composite generic
  arguments; ordinary nested/Result emitter refusal; real Windows payloads.
- `internal/cbackend/tagged_types.go`: flat-Option-only candidate and target scalar
  payload layout. Shared Result/aggregate payload predicates and direct FFI are
  unchanged. Surrounding tagged roots cannot inherit the new leaf permission.
- `internal/cbackend/option_transport_test.go`: all-seven candidate/layout,
  nested/Result/borrowed payload refusals, no shared layout widening, and no new
  scalar or Option argument/result FFI.

No production helper outside those two existing Go owners was necessary.
No dependency, API/DB, runner, limit, timeout, target or wire-schema change.

## Validation

Every command sets `$env:GOMAXPROCS='1'` and uses `-p=1`. Native consumer builds
are serial: this worker starts at most one CL process at a time, within the
shared maximum of two. Existing native helpers/timeouts are reused.

Initial focused admission/candidate run passed after fixing test-only default
package IDs and a layout result field name: compiler 0.180s, C backend 0.043s.
A first native run passed all four Windows consumers in 2.314s. The final native
fixture additionally asserts Some tags before extraction and passes every copied
value through an ordinary written `Option<S>` function, so a fallback value
cannot conceal an accidentally constructed None. Final results are recorded
below; earlier timings do not attest the changed final fixture.

Final focused command:

```powershell
$env:GOMAXPROCS='1'
go test -mod=readonly -p=1 ./internal/compiler ./internal/cbackend -run '^(TestCompositeOption.*|TestGenericSignature.*|TestRestrictGenericSignatures.*|TestTaggedCandidates.*)$' -count=1 -v
```

The native fixture contains eleven independently seeded scalar cases: i8/i16/
isize signed minima; u16/u64 maxima; f32 1.25, +0, -0; f64 -3.5, +0, -0.
Each value is wrapped, copied/assigned/returned, passed through an ordinary
written Option function, then extracted from both original and copy. C compares
only the active scalar bytes with `memcmp`, including zero sign. Both Some tags,
the None branch and exactly 22 observer calls are asserted per product execution.
Existing repr(C) seed/observer records carry scalars across FFI; Option itself
does not cross FFI.

Final focused command: **PASS**, compiler package **4.809s**, C backend package
**0.059s**. This includes the seventy generic wrapper/resolve-first cases,
twenty-eight ordinary nested/Result emitter cases, all candidate/FFI boundary
checks, the preexisting compact/deep tagged candidate controls and the native
execution below. Source ASTs stayed byte-identical under JSON serialization
after all three compilation paths. Report exhaustion tests passed with the
existing limit and terminal marker.

The final Go fixture produced 47 MIR functions and 74,046 bytes of MSVC C.
Windows MSVC 14.51.36231 results:

| C consumer | Build time | Image bytes | Actual execution |
| --- | --- | --- | --- |
| C11 low | 1.1035457s | 128,512 | exit 42, empty stdout/stderr |
| C11 optimized | 0.4853486s | 112,128 | exit 42, empty stdout/stderr |
| C17 low | 0.3729828s | 128,512 | exit 42, empty stdout/stderr |
| C17 optimized | 0.5860216s | 112,128 | exit 42, empty stdout/stderr |

Native test time: **3.84s**. All four executions verified the eleven Some
payloads in both original and copy, signed/unsigned boundaries, both floating
zero signs, Some tags, None branches and the exact observer count.

The source-negative matrix uses actual type-correct forwarding calls. A separate
ordinary-source emitter matrix first verifies source/MIR and then requires both
C profiles to reject nested Option and Result roots. Its first Result attempt
correctly failed existing Result handling (E4101); the final fixture handles the
Result with exhaustive switch before testing C refusal. No production Result
handling was changed to make that test pass.

## Integration boundary

Before coordinator integration, the existing out-of-scope
`internal/cbackend/tagged_rejection_test.go::TestEmitStillRejectsUnsupportedFloatingTaggedPayload`
expected `Option<f64>` C rejection. That assertion contradicted the new
contract. This worker did not edit that file. Before migration its isolated
command was actually run and **failed as expected** (0.075s package):

```powershell
$env:GOMAXPROCS='1'
go test -mod=readonly -p=1 ./internal/cbackend -run '^TestEmitStillRejectsUnsupportedFloatingTaggedPayload$' -count=1
```

Failure: `unsupported payload rejection changed: <nil>`. The newly admitted
Option emits successfully, contradicting the obsolete rejection assertion.
This is historical evidence of the obsolete assertion, not a failure of the
coordinator's replacement.

The coordinator has since migrated the fixture to `Option<Option<f64>>` and
retained interpreter None/return-42 behavior plus a CapabilityError naming the
exact nested type. This worker inspected that source diff and confirmed the
migration is present. **Test migration is complete.** The coordinator subsequently
reported the full `internal/cbackend` package **PASS, 77.462s**, after migration.
This worker did not repeat that package run or broaden testing.

Fairy graph native tests were reported passed in both profiles by the coordinator
(14.819s), including exact 4096-node capacity and rollback. The coordinator's
final integration update separately reports:

- Fairy restored Option pipeline **PASS, 209.852s**, covering four Windows
  producer/consumer combinations and finite/positive-zero/negative-zero active
  payload bit comparisons.
- Source check **PASS, 23.816s** and locked check **PASS, 25.624s**: 755 files,
  9,543 MIR functions, with all 1,226 Fairy files matching.

These are coordinator-reported integration results, not executions repeated by
this worker. The parent owns exact integration evidence, links and commit, plus
the remaining diagnostic regression. The restored-pipeline pass establishes its
named Fairy execution slice; it does not close that separate regression or the
whole remaining compiler queue.

Option-specific fresh-process persistent cache/whole-CLI execution is pending;
the existing cache schemas were not changed or newly attested. Linux execution
stays in the authorized batch. No broad build, full suite, hosted CI, clean-machine
distribution test or final Atlas/self-build acceptance was run by this worker.
