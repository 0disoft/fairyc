# Copied closure value MIR sidecar — 2026-11-08

Authority: [ADR 0551](../adr/0551-copied-closure-values.md).
Worktree: `C:/Users/cherr/Documents/workspace/zerodi-wd1/projects/hobby/opensource/fairyc/.cache/closure-values-20261108`.
Worker-owned new files only:

- `internal/mir/closure_values_test.go`
- `docs/development/go-closure-value-mir-20261108.md`

Production MIR, semantic preparation, interpreter/backend and Fairy files are
parent/other-worker owned. This worker made no production, FY, lock, ABI, format,
runner, dependency or limit changes, and no commit or remote action.

## Executed checks

```powershell
$env:GOMAXPROCS='1'
go test -mod=readonly ./internal/mir -run '^TestClosureValueMIR' -count=1 -parallel=1 -p=1 -v
gofmt -l internal/mir/closure_values_test.go
```

Final narrow test **PASS**, package **0.079s**, six top-level tests, **96 leaf
cases** (100 named runs including grouping parents). Read-only gofmt check
produced no output. Whitespace checks of the two new files produced no errors.
No native work, Result matrix, broad suite, stress test or additional vet run
was performed; this task requests the new MIR tests only.

Retained logs, relative to the worktree:

- `.cache/closure-mir-attempt1.log`: FAIL, package0.053s; two accepted forgeries.
- `.cache/closure-mir-attempt2.log`: FAIL, package0.071s; the same two accepted
  forgeries after adding initialized-type/local-type-spoof probes.
- `.cache/closure-mir-attempt3.log`: **PASS**, package0.079s; parent target-body
  correction and global transport gate present; all added transport probes run.

Session91518 and session20865 completed exit1; session93972 completed exit0.
No owned process/session remains running.

Final Go test executable identity:
`fairyc.compiler-executable-sha256/v1:298aa9013c6c8b6f04a22429da29084c81789db843521c7d44250511016035b8`.

## Source-to-MIR evidence

The external `mir_test` package reuses `lowerIgnoredCleanupSource` from
`ignored_cleanup_test.go`: actual frontend syntax analysis, semantic flow,
MIR lowering and verification. No production lowering algorithm is duplicated.
The independent handwritten MIR controls are verified before mutation.

- Two distinct returned `fn i32(i32)` factories retain real, pure lifted
  targets and explicit closure-create instructions. `apply` keeps its closure
  parameter; ordinary Box construction, copy and field extraction remain MIR.
  The fixture has two targets, two constructors, three value-call sites and one
  closure field read. Construction remains a target-symbol edge even though
  the factory never invokes the closure.
- Source capture arities0..4 each retain one constructor and one target with
  the matching capture/value parameter count. Handwritten controls independently
  accept all five arities; mismatches at each arity and five captures reject.
- Two copied captures are frozen by separate preceding assignments in written
  order. The subsequent creator-local write does not alias either snapshot.
  Callable materialization precedes exactly one ordinary argument call; its
  continuation contains exactly one value invocation using those exact locals.
- The unused checked `1/0` body local retains trap control flow in the lifted
  helper. No trap is moved into creation/copy/factory execution. This is a MIR
  structure assertion; no interpreter/native execution is claimed here.

Factory source SHA-256:
`0d484e382812505748d363797d4e748e17df5315b50e2e332c195bbf1df4d5e1`.

## Forged MIR evidence

Forty contract cases cover missing/unmarked targets, stray capture metadata,
signature/result/effects/body restrictions (including dead blocks), parameter
and local immutability, duplicate parameters, reachable/dead cycles, wrong and
empty returns, capture kind/type/annotation/dirty-bit/uninitialized forgeries,
construction shape, callable type/initialization/ABI/callee/unwind/successor
shape and argument/result restrictions. A typed callable defined only in dead
code rejects as uninitialized at the reachable invocation. Local annotations
cannot turn a scalar into a callable or a bool into an i32 capture.

After the parent added `verifyClosureValueTransport`, 43 additional negative
cases require **V1032 with the copied-closure-transport diagnostic prefix**,
rather than accepting an unrelated verifier error as sufficient evidence:

- Ordinary direct closure fields and monomorphic closure/Box parameter-return
  paths have valid controls. repr(C), nested struct, wrapped field, enum and
  resource storage reject, including ordinary carrier structs used as payloads.
- Option, Result success/error, array, shared/mutable/raw pointers and wrapped
  carrier structs reject in parameters and results. Closure and carrier FFI
  parameters/results reject.
- Unary/switch consumption, scalar-cast production, closure owner operations
  and index type spoofing reject.
- Exact ordinary-call arguments distinguish scalar, closure leaf and Box;
  ordinary carrier return mismatches reject.
- Duplicate constructor/assignment, initialized parameter overwrite, duplicate
  struct destination and ordinary call-result overwrite reject.

## Failures reported and parent correction

Attempts1/2 reproduced two production holes without editing production:

1. `TestClosureValueMIRForgedContracts/target-return-bool`: a marked i32 target
   returned an otherwise valid bool literal (`OperandI32`, value1, TypeBool).
2. `TestClosureValueMIRForgedContracts/target-return-empty`: the same non-void
   target returned an empty operand.

Both originally passed `mir.Verify`, so the tests failed with
`forged closure MIR was accepted`. The parent subsequently required exact i32
body operands/return values. Attempt3 rejects both through V1021 and passes
the unchanged rejection expectations. Failed logs are retained, not rewritten.

## Input fingerprints at final verification

These are observed working-tree bytes; the parent owns production integration
and the combined-main source lock refresh. The worktree base HEAD at inspection
was `b4ac33904430281b81f366db21154570496f5165` with other writers' changes present.

| Relative path | SHA-256 |
| --- | --- |
| `internal/mir/closure_values_test.go` | `ab83df20ab4c5eab5b1a11b9799ff64740bc60e62c6debc555c0b87b940bf3b5` |
| `internal/mir/closure_values.go` (parent) | `b7a99f8976c9ef9a8bebba135dd50e346119624ae6d69951159071b5d2b9e8e6` |
| `internal/mir/closure_value_transport.go` (parent) | `e6367604de71d9ad518b30529607ee4ad5bac4ed65527608e09dc53f57ad07ef` |
| `internal/mir/lower_closure_values.go` (parent) | `b47eb9a340bfa89829ef6b380df5d19c7cb7f071463365da67586841942fcb13` |
| `internal/mir/verify_function.go` (parent) | `31400128b2ffb3ec20edadf59ec731626be0981c7b4f4acfac216d46b96a3afc` |

The source contract ADR is unchanged. This narrow pass does not claim native,
codec/cache restoration, whole CLI, Fairy executable closure support or public
ABI admission. Those gates remain with their assigned owners.
