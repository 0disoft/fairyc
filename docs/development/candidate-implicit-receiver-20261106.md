# Fairy implicit owner receiver borrows, 2026-11-06

Part of `implementation-queue-20261105.md`, not completion of that objective.
The existing method contract owns this source form. No new public syntax or ABI
is introduced. Native Linux execution remains in the maintainer's combined batch.

## Implementation

Named local struct/resource owners can call ordinary methods whose first `this`
parameter is `ref Owner` or `ref mut Owner`. The checked first argument retains
the original owner expression/type and the expected borrow type separately.
Ordinary arguments and explicitly qualified static calls gain no conversion.
Immutable exclusive receivers, consumed owners and live conflicting loans remain
errors. Returned borrows retain the original owner and its lifetime.

`module_unit_receiver_borrow` owns source selection and place/type admission.
Move checking reads instead of consumes the owner. Origin checking uses the same
loan checks and source coordinates as written borrows. Detached-body and project
verification independently recheck the method/call/first-argument/place relation.
C emission passes the address of the retained owner; the interpreter resolves
the same place. Generic callers and resource cleanup bodies use these paths too.
Grouped, projected and temporary receiver expansion is not claimed here.

The private body-IR cache profile advances from `/1` to
`fairyc.candidate-owned-body-ir/2`, separating the changed argument admission from
old cache inputs. The independent complete-key Go oracle uses that exact profile.
Public interfaces and stable cache wire formats do not change.

The Go prerequisite `ecab5f4f` registers the synthetic receiver-borrow AST type.
Without it, a method returning its receiver falsely reports an unknown origin.
The fix preserves origin/escape/conflict checks rather than weakening them.

## Independent observations and acceptance

Atlas comparisons use original Go source facts for moves, borrows, source spans,
mutable/shared kinds and escape flags. Ordinary functions, generic callers and
concrete generic cleanup owners pass through current-source map production.
Logical Atlas intentionally omits nongeneric private drop glue; the checked-body
collector's separate comparison derives those owners from Go declarations and
their observations from raw Go semantic borrow facts. It does not invent empty
observations or change production completeness for an unobserved family.

Persistent replay uses the existing unit and whole-project cache ports in fresh
processes. Cold/warm hit policies, stored record counts, unchanged cache/source
bytes, exact cold/restored C packets and execution of restored C are asserted.
Cold execution already covers both C consumer profiles; restored consumers use
the low profile while compiler hosts run low and optimized.

Windows low/optimized receipts:

| Check | Result and scope | Seconds | `.cache/` log |
| --- | --- | ---: | --- |
| Borrow observations | Passed cold/restored Go comparisons; enclosing command later fails in the separate sites harness | 99.77 | `implicit-receiver-atlas-final-20261106.log` |
| Borrow sites | Passed 33 source families and unchanged frame/work/count controls; enclosing command later exposes the separate move bug | 69.82 | `implicit-receiver-atlas-sites-moves-20261106.log` |
| Private cleanup borrow | Passed the added nongeneric cleanup case against nonempty raw Go semantic facts | 71.920 | `implicit-receiver-private-cleanup-20261106.log` |
| Move sites | Passed 36 admitted sources and twenty controls per profile, including borrowed and consuming receivers | 582.683 | `implicit-receiver-moves-final-20261106.log` |
| Complete cache keys | Passed 15 independently generated ordinary/generic package pairs per profile | 182.049 | `implicit-receiver-keys-versions-20261106.log` |
| Versioned source production | Passed six new families, 36 native processes / 60 independent current maps, including incompatible-schema refusal | 269.347 | same log |
| Unit and whole-project persistence | Passed six families, 48 fresh candidate processes and 24 restored native C executions | 543.62 | `implicit-receiver-acceptance-final-20261106.log` |
| Interpreter and emitted C | Passed six families, twelve interpreter/emitter processes and 24 native C executions | 103.16 | same log |
| Complete admission | Passed both profiles, including exact negative codes | 62.70 | same log |
| Actual authored CLI | Passed six new `pack-sources/borrows-implicit-*` families under both profiles: 42 commands | 421.502 | `implicit-receiver-full-cli-20261106.log` |

The follow-up private-cleanup replay additionally requires at least one cleanup
borrow in the independently checked Go oracle before either native comparison.
Both Windows profiles pass in 67.461s (`implicit-receiver-private-cleanup-positive-20261106.log`).
This prevents an accidentally empty oracle from satisfying the positive case;
it changes no compiler source or prior execution inputs.

The combined persistence/execution/Complete admission command passes in 709.654s
with no selected skips. Its Complete cases include ordinary/static mismatches,
immutable exclusive receivers, consumed owners, returned local escapes and
retained shared/mutable conflicts, with independent Go admission controls.

Authored check/locked verify 686 files and 8,557 MIR functions. Semantic/MIR/
snapshot/map/pack regressions, scoped vet and standalone profile generation pass.
The borrow roots above retain their bounded passed evidence; their failed
enclosing commands are not reclassified as successes. The later Atlas move fix
does not change their borrow/origin owners.

The actual CLI header is 2,097,042 bytes; runtime 153,115 bytes; 26 body units,
largest 2,096,369 bytes. Existing 2 MiB partitions, warnings, deadlines and stack
policy remain unchanged. Retained CLI source/profile/toolchain and image SHA-256:

| Profile | Source/profile/toolchain digest | Image SHA-256 |
| --- | --- | --- |
| low | `f9f4e998bc8e6cad67dc50faa4925561cb86d576e9e143fd15c67e4f62a958d5` | `66124fd8e205c5d0abc5b04f92b31ff9fd08d1b4ae31304160bb610e36c46d8a` |
| optimized | `679be0eccb651a31a833f6db761758bd3fee39225249c1abc95880f234de279a` | `30fb025867454ca89b49324e26fb83ac6afed6746e9cc51a003fab8f7ecf236e` |

Selected reproduction commands, run from the repository root:

```powershell
go test -mod=readonly -count=1 -timeout 12m -failfast -v ./internal/cli -run '^TestCandidateAtlasMoveSites$'
go test -mod=readonly -count=1 -timeout 15m -failfast -v ./internal/cli -run '^TestCandidate(OwnerQualifiedComplete|ImplicitReceiverPersistence|ImplicitReceiverExecution)$'
go test -mod=readonly -count=1 -timeout 12m -failfast -v ./internal/cli -run '^TestCandidateAtlasSourceVersion$/(low|optimized)/^borrows-implicit-'
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
go test -mod=readonly -count=1 -timeout 10m -failfast -v ./internal/cli -run '^TestCandidateCompleteCacheKeysV5$'
$env:FAIRYC_ATLAS_CLI_ARTIFACTS=Join-Path (Get-Location).Path '.cache/implicit-receiver-cli-20261106'
go test -mod=readonly -count=1 -timeout 15m -failfast -v ./internal/cli -run '^TestCandidateAtlasCLI$/(low|optimized)/^pack-sources$/^borrows-implicit-'
go vet -mod=readonly ./internal/cli ./internal/semantic
```

Two earlier collector commands failed because the test compared private
nongeneric cleanup frames against logical maps that omit those frames. Their
borrow-observation roots passed, but the enclosing commands are failures. The
corrected checked-body oracle retains those frames and reads Go semantic facts.
The next move replay exposed a real false move at the borrowed owner span.
The collector now uses the detached receiver proof before classifying a checked
argument as consuming; its final standalone replay passes all original consuming
and new borrowed cases. The first persistent replay aborted on the reused
fixture's assumption of one root function. The adaptation selects the written
`main` by its original source declaration, retaining all cache assertions.
Initial formatting commands used the example working directory for repository-
relative paths and failed; formatting was checked from the repository root.

No full repository suite, consecutive full self-build equality, clean-machine
distribution, current hosted CI or native Linux pass is claimed. No public
CLI/SDK ABI, dependency, database, runner, warning level, work limit, timeout or
stack policy changes. No push, dispatch, release or deployment.
