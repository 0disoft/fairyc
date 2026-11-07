# Copied closure parallel integration, 2026-11-08

This continues ADR0551's bounded `fn i32(i32)` value family. Three workers used
disjoint project-link, C, and component/Program/source-admission write sets.
Completed workers also reviewed one another's changes without editing them.
The parent owned native scheduling, integration and exact-path commits on main.

## Local commits

| Commit | Slice |
| --- | --- |
| `4bb341e6` | Body IR/CFG proof and initialized leaf locals |
| `ac5ab289` | Closure C representation, helpers, reachability and native consumers |
| `97d6d4b2` | Source-qualified helpers and value calls across project units |
| `067559f3` | Original-source Complete through explicit project closure C emission |
| `4734f14e` | FYCLEAN2/FYBODY01 component codecs and fresh full reconstruction |
| `2e31db98` | FYPROG02 framing, metadata and whole Program proof |
| `2bb2b39a` | Current-source unit recipe comparison and fresh bound owner |
| `d8a81c62` | Overflow-safe helper/snapshot token comparison |

The final integration commit records this receipt and the normally regenerated
candidate source lock. No new feature/version field is added to that lock.

## Executed Windows acceptance

| Test/scope | Observed pass |
| --- | --- |
| `TestCandidateClosureUnitBodySource/Native` | 11 originals, low/optimized body/CFG and applicable forgery checks |
| `TestCandidateClosureUnitCNative` | 17 originals x two profiles x single/partitioned C: 68 compiled/executed consumers |
| `TestCandidateClosureProjectLinkSource/Native` | 23 originals per native profile, including real automatic module batching and eight language refusals |
| `TestCandidateClosureProjectCControls/Source/Native` | Seven Go interpreter controls; original-source Complete; 28 actual generated C consumers |
| `TestCandidateClosureUnitArtifactSource/Native` | 13 closure originals plus scalar, exact bytes/roundtrip/forgery/work checks; both profiles |
| `TestCandidateClosureProgramArtifactGolden/Source/Native` | Independent 1,266-byte FYPROG00 oracle; 11 closure controls; FYPROG02 roundtrip/forgery/work assertions in both profiles |
| `TestCandidateClosureUnitRestoreSource/Native` | 13 originals, scalar refusal, detached-valid source/helper/metadata mismatches and stale-input refusal in both profiles |
| Final component token-boundary rerun | Last safe sum, first overflowing sum and maximum usize for helper IDs and snapshot source IDs; ordinary controls still pass in both profiles |
| Main candidate `check`, `check --locked` | 2 instances, 775 source files, 9,745 MIR functions; lock updated then verified |
| `go vet -mod=readonly ./internal/cli` | Exit 0 on final Go harnesses |
| Scoped formatting and `git diff --check` | Pass |

No expected C or MIR is injected into the authored compiler. Generated consumers
check the independent 42/84 results, 81/82/83 scalar traps and output constraints.
Structural fixture execution and input-program execution remain distinct.
The final token proof change preserves valid pairs and format layouts. The
focused final component rerun executes all ordinary controls and new malformed
boundaries; unrelated ABI/native fixtures were not repeated.

Large native compiler fixtures use the existing partitioned host builder and
unchanged one-minute command/16MiB stack limits. Source checks do not count as
execution. Native low/optimized cases run sequentially, not competing producers.
Main-source checks run independently of the frozen native input.

### Stable native input and retained logs

A temporary detached worktree at base `4bb341e6` received only frozen slices.
This avoids a producer reading another worker's incomplete API edits. After
acceptance, all 36 modified/untracked derivative paths were verified byte-equal
to the main checkout and that temporary worktree was removed. Other pre-existing
worktrees were left intact. Logs remain ignored local evidence under `.cache/`:

- `closure-unit-body-final-20261108.log`
- `closure-unit-c-native-values-20261108.log`
- `closure-project-link-final-20261108.log`
- `closure-artifact-project-native-20261108.log`
- `closure-unit-artifact-restore-parts-20261108.log`
- `closure-artifact-overflow-final-20261108.log`
- `closure-integration-final-check-20261108.log`
- `closure-integration-final-locked-20261108.log`
- `closure-integration-final-vet-20261108.log`

These logs are execution evidence, not language or source authority. Narrow
reproduction commands and limits are in the individual slice receipts.

## Failures retained rather than reclassified

Initial generated-C compilation exposed unused labels, unreachable completion
traps, implicit narrowed helper results and unused place temporaries. The
emitter fixes passed strict MSVC consumers; warnings were not suppressed.

The first project batching fixture incorrectly exceeded the catalog's actual
128-declaration module cap. The corrected input uses legal parameter entries
to exceed the 1024-entry unit boundary; all 23 cases remain present and pass.

Both artifact compiler fixtures executed successfully under the original CLI
`run`. Their original monolithic `run --release --locked` builds exceeded the
default 60-second compiler bound. Those failures remain in the original batch
log; partitioned semantic acceptance does not turn them into monolithic passes.
The final harnesses use the existing bounded host partition builder.

## Remaining delivery boundary

This is not completion of the entire copied-closure delivery goal. Remaining:

1. Source-bound project envelopes and the complete closure call/source recipe
   family. The unit restorer deliberately rebuilds and compares canonical bytes;
   it makes no lowering-skip or warm-cache speed claim.
2. Unit/project persistent-cache routing, version/key contracts, cold/warm,
   corruption, changed-source and fresh-process restore acceptance.
3. Stable Fairy CLI publishers/commands, whole candidate/native/self-host and
   supported-profile baseline acceptance. Explicit private project C emission
   is verified; stable CLI closure routing is a separate boundary.
4. Batched Linux validation and hosted CI. No Linux, remote runner, push,
   release or deployment was executed here. The full repository Go suite was
   not run.

The existing Windows core job has a 15-minute package test limit. This host's
new native probes cumulatively exceed that budget before older tests; hosted
hardware may differ. CI routing/runtime needs review before a push. No workflow,
runner count, timeout or skip policy was changed to hide that constraint.

Normative/source-of-truth documents, public language/API/SDK ABI, DB,
dependencies and repository-hygiene configuration are unchanged. The edited
development files are execution receipts. No model performance or hosted
same-commit claim follows from this local acceptance.
