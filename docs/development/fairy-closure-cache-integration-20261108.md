# Closure source recipes, project envelopes and unit disk cache

This follows [the first parallel integration](fairy-closure-parallel-integration-20261108.md)
for ADR0551. It advances the remaining copied-value closure delivery boundary;
it is not completion of the whole Fairy C roadmap.

Two workers used disjoint source-recipe and unit key/store write sets. The
parent owned project envelope/rebind/rebase/Complete integration, source-bound
unit restore integration, Windows execution scheduling and exact-path commits.
Workers never staged or committed the parent's changes. Product input was
copied into each test's private workspace before native emission/build.

## Completed slices

| Commit | Scope and evidence |
| --- | --- |
| `cb9c5885` | Current-source closure call/recipe admission and independent reachability work accounting. [Receipt](fairy-closure-source-recipes-20261108.md): 16 original closure controls per native producer profile, malformed/source mismatch and exact work assertions. |
| `ca62a1d9` | Explicit FYPRJ002 envelope and current-source project Complete restore, including module rebasing. [Receipt](fairy-closure-project-artifact-20261108.md): seven original multi-module controls, 28 actual generated C executions, stale-source and detached-valid splice refusals. |
| `7369f03d` | FYCACHE2/FYPUNIT2 unit key/store, raw disk recognition and current-source unit restoration without replacement lowering. [Receipt](fairy-closure-unit-cache-20261108.md): 24 fresh cache processes, 36 generated consumer executions and 13 independent restoration controls in each producer profile. |

All three slices have Windows execution evidence. Their receipts retain failed
attempts separately; authored controls are never counted as passes by themselves.

## Integrated checks

The main candidate `check` and `check --locked` passed with 2 instances,
775 source files and 9,765 MIR functions. The ordinary checker regenerated the
source lock and the locked checker verified it. The lock introduces no feature
or format field. The independent 1,266-byte old Program artifact golden passed.
Final `go vet -mod=readonly ./internal/cli`, scoped `gofmt -l` and
`git diff --check` passed. There was no full repository Go test run.

| Native scope | Final Windows result |
| --- | --- |
| Source recipes | 214.72s; low and optimized producer assertions pass |
| Project envelope/Complete/C | 99.60s; low producer, 28 single/partitioned generated consumer executions across both profiles |
| Fresh-process unit disk cache | 462.11s; both producers, 24 fresh replays and 36 generated consumer executions |
| Independent source-bound unit restore | 186.37s; 13 original controls and refusal/work-origin assertions in both profiles |

Main logs are under ignored `.cache/`: `closure-project-artifact-native-20261108.log`,
`closure-unit-cache-final-20261108.log`,
`closure-unit-recipe-restore-native-20261108.log`,
`closure-next-candidate-check-20261108.log`,
`closure-next-candidate-locked-20261108.log`,
`closure-next-old-golden-20261108.log` and
`closure-next-final-vet-20261108.log`. The source-recipe worker's final execution
was session 37431, terminal exit zero, as recorded in its receipt. The earlier
failed recipe/cache logs remain separate. All task-owned sessions and workers
were closed after their observed terminal results.

No timeout or warning policy was raised. Initial native assertion failures are
retained in the slice receipts. The source-recipe work-boundary failure exposed
a real mismatch between the 1MiB reachability ledger and the 1GiB artifact
history; its fix preserves both limits and charges the independent cost once
on success and failure. No assertion was removed to accommodate the failure.

## Remaining delivery boundary

The subsequent [project/mixed-cache and semantic-interpreter closeout](fairy-handoff-20261108.md)
supersedes items 1 and 2 below with Windows low/optimized execution evidence.
The list retains this commit's historical boundary; full CLI/self-host,
distribution, hosted CI and batched Linux are still separate.

1. Route closure unit keys/load/save through mixed project compilation and
   publish cold units only after the linked Complete and interfaces succeed.
   The unit disk fixture alone does not establish project publication atomicity.
2. Whole-project persistent-cache version/key routing, current-source restore,
   corruption repair, source invalidation and fresh-process acceptance.
3. Stable Fairy CLI closure publishers/commands, whole candidate native/self-host
   and the supported-profile baseline. Private restored-owner C acceptance is
   not a stable CLI or optimized monolithic compiler build pass.
4. Batched Linux execution, hosted CI and distribution/self-host acceptance.
   The existing 15-minute Windows package job needs runtime/routing review before
   an authorized push; no tests, limits or workflows were altered to hide that.

No public SDK/ABI, DB, dependency, normative source-of-truth, repository hygiene
or CI/runner configuration changed. Development receipts document execution.
No push, release, deployment, hosted dispatch or Linux run occurred.
