# Ordered work - 2026-10-31

- Status: Approved follow-up implementation queue
- Updated: 2026-10-31
- Supersedes: current-status claims in the [2026-10-23 ordered work](completion-plan.md#historical-ordered-work-superseded-2026-10-31)
- Change boundary: documentation and contract planning only in this revision
- Authorization: named sequence implementation; no implicit syntax, API or target support

This is the current ordered queue. Item 1 is complete and the list below is the resulting work
record. Item 2 is next / in progress. Items 3 through 22 are queued. Existing bounded evidence is
retained, but a row remains queued until its stated completion condition is met.

`Next / in progress` includes the routine bounded-contract and implementation work for that row.
It is not a request for a fresh approval of the same task.

Approval of this queue authorizes implementation of the named sequence. It does not by itself
admit a grammar form, public API, ABI, backend choice, target, package registry, persistence
product or distribution claim. Where a design is not selected, Engineering writes the bounded
contract in the same work slice: input and output surface, ownership and lifetime rules, limits,
failure and negative cases, compatibility impact, and exact evidence. The implementation proceeds
under that contract without another routine approval. The owner is asked only when adopting a
technology or changing scope would make a decision on their behalf; backend and target choices
remain `UNDECIDED` until an explicit owner decision.

## Queue order

| Order | Work | Status | First bounded contract or next gate | Completion condition |
| --- | --- | --- | --- | --- |
| 1 | Reconcile the current work list | **Complete** | Make this document the current queue, link it from `completion-plan.md`, preserve older records as history, and make later completion evidence supersede an earlier pending statement. | This queue and its status vocabulary are recorded in the two documentation files. |
| 2 | Fairy-authored GNU companion | **Next / in progress** | Define and implement one private companion boundary for an already prepared Linux packet: compiler discovery, argv/stdin/environment handoff, stream capture, diagnostics, deadlines and process cleanup. Failure is explicit; the existing contract has no automatic frontend fallback to Go. Do not admit distribution or replace the default compiler. | The companion executes the named Linux build path from Fairy, preserves bounded cleanup and exact diagnostics, and has Linux-native plus Windows-development evidence. Go remains the default until a separate promotion decision. |
| 3 | Aggregate transport in both directions | Queued | Name the first incomplete aggregate family and prove by-value arguments and results in separate Fairy/C translation units in both directions. Keep each added shape narrow and preserve target-owned layout. | Every shape in the contract crosses C to Fairy and Fairy to C with exact values, traps, ownership and profile checks; unsupported shapes fail explicitly. |
| 4 | COFF relocation coverage | Queued | Enumerate each relocation family emitted by current Fairy and runtime objects. Pair the object-level record with its linked-image result instead of treating a parse as link proof. | Each mandatory family has an exact object record, target resolution and final linked-image assertion in the declared Windows profile. Existing partial probes remain partial evidence. |
| 5 | Weak linking: chains, cycles and replacement | Queued | Define bounded weak-external chain, cycle and replacement cases, including selected and discarded targets and the observable failure mode. | Exact synthetic and live-object cases show deterministic selection, fallback or rejection through the real linker path. |
| 6 | Default libraries and CRT archive | Queued | Record the exact default-library and CRT archive inputs used by each supported Windows build profile, including the expected selection or failure when a member is unavailable. | The inputs, archive search mode, selected member and final symbols are attributable to a same-source receipt; relying on a developer machine's implicit library path is insufficient. |
| 7 | Archives and COMDAT | Queued | Define the production archive, associative and any-COMDAT shapes to cover, including parent/child retention, discard and interaction with library inputs. | Each contracted production shape has exact retention/removal, relocation and executed-image evidence for the declared profile. |
| 8 | Unwind v2, handlers and chained frames | Queued | Fix the version-2 record, handler-payload and chained-tail forms that the emitted target can produce, then name the exact Fairy/foreign frame boundaries to execute. | Version-2, handler and chain cases are inspected and executed through their exact frames, including chained-tail relocations and failure cases. |
| 9 | Same-commit mixed-stack, callback and trap CI | Queued | Retain the live mixed-stack walk, C boundary catch and Fairy trap outcomes as one source-addressed receipt. Keep callback retention and asynchronous delivery outside the contract. | The authorized hosted lane runs the cases on the same commit and records stacks, handler addresses, trampoline results, trap mapping and artifact identity. A push is separately authorized work and is not performed by this documentation update. |
| 10 | CodeView, PDB and DbgHelp retention and lookup | Queued | Bound the object records, PE/PDB association, retained artifact lookup and DbgHelp symbol result required by the proposed debug contract. | Corrupt or mismatched identities fail closed; a retained PE/PDB pair resolves the named functions and lines on a declared profile. Full CodeView syntax is not implied. |
| 11 | UCRT, VCRuntime and dynamic DLL dependencies | Queued | Enumerate static imports and declared runtime-loaded DLL dependencies for development and release outputs, with a bounded manifest and execution check. | Each declared dependency is present, resolvable and exercised without relying on an unrecorded developer PATH; absent or mismatched files fail with the declared error. Clean-machine transfer remains a separate evidence gate. |
| 12 | Interface/effects diff and large-source notice | Queued | Specify an opt-in comparison over public interfaces and declared effects, including exact/bounded/partial/unknown states. Separately define when lightweight tooling emits a large-source notice. | Comparisons are deterministic and preserve unknown evidence; no heuristic rename becomes a fact. Large files produce the bounded notice without changing compilation or silently truncating source. |
| 13 | `@repr(C)` record-element arrays and additional arrays | Queued | Admit one named record-element or additional array family at a time with exact element layout, copy and replacement rules. Keep bare foreign arrays and unspecified element operations closed. | Each contract-named family passes source typing, MIR, interface/cache, both-target emission and separate-object native crossings. Existing scalar, enum, integer and floating array evidence is not reimplemented. |
| 14 | Raw-pointer targets, operations and nesting | Queued | Name the first pointer target or operation to add, such as a new scalar pointee, a specific dereference/arithmetic/cast rule, or one nested pointer shape. Define provenance, null and lifetime boundaries. | The exact admitted operation and target cross the foreign boundary with matching layout and behavior; every other target, operation and nesting remains an explicit rejection. |
| 15 | Ordinary callbacks | Queued | Broaden the existing bounded synchronous callback slice through a separately named ordinary callback contract. Keep retained callbacks, asynchronous delivery, unwinding through Fairy and `longjmp` across Fairy excluded unless a later decision reopens them. | The admitted signature and user-data lifetime are checked end to end, invoked synchronously with exact results, and bounded across C to Fairy and Fairy to C call paths. |
| 16 | Zero-field resources and nested owning tags | Queued | Define storage, construction, move and recursive cleanup for one zero-field resource case and one nested owning-tag case. Keep compiler-owned conditional root flags separate. | Both positive cases execute with exact drop order and active-payload observations; malformed and unsupported nesting fails closed. Direct nested-resource evidence does not stand in for arbitrary tagged nesting. |
| 17 | Cross-package inline and cache | Queued | Reopen the Phase 3 gate in [the current phase decision](phase-status.md#phase-3-scope-decision-2026-10-21), then define the inline contract, visibility, body identity and cache invalidation rule before reuse. | Inline bodies have versioned identity, owner/dependent invalidation, deterministic reassembly and mismatch rejection. Existing package cache v4 behavior remains intact until an explicit successor is accepted. |
| 18 | Broader generics, interfaces and closures | Queued | Select one representative user gap and define its ownership, inference, specialization, code-size and diagnostic bounds. Interfaces and closures need separate rules because they alter dispatch and capture. | The named feature has measured need, accepted semantics, verified MIR and owned artifact identity, plus representative compile-time, code-size and runtime evidence. Existing limited generics are not relabeled as this broader admission. |
| 19 | QBE, LLVM or additional targets | Queued | Treat backend and target choice as **UNDECIDED**. Start with an explicit comparison contract and workload that measures build time, executable size, semantics, ABI coverage and maintenance cost. | One choice is supported by a recorded decision and measured benefit, or the work remains deferred. This row does not select QBE, LLVM or any new target. |
| 20 | Remote packages | Queued | Define source identity, trust, version resolution, network failure, cache lookup, lockfile reproducibility and offline behavior before any registry client. | A bounded client resolves and locks the named package inputs reproducibly and fails closed on identity, network or cache mismatch. Registry policy and deployment remain separate. |
| 21 | Atlas impact, MCP and structural edits | Queued | Gate each capability separately on current `.fymap` identity and incompleteness behavior. MCP needs its own permission and untrusted-data boundary; structural editing needs an atomic source transform contract. | Impact and semantic diff report bounded uncertainty, MCP exposes only admitted facts with explicit permissions, and structural edits round-trip through parse and semantic validation. Heuristic source rewriting is insufficient. |
| 22 | Chain execution and persistence | Queued | Keep the chain target **UNDECIDED**. Define the transaction execution subset, storage port, ownership, reentrancy, conflict and failure-atomicity rules before native or durable integration. | The named execution and persistence adapter passes deterministic local oracle and native evidence with restart/conflict/failure tests. Production chain deployment and consensus remain outside this queue. |

## Completed Linux baseline

The Linux candidate implementation is not a pending item in this queue. The maintainer-supplied
external record [Linux bootstrap acceptance after frontend memory reclamation](../ops/linux-bootstrap-revalidation-62f2a006.md)
validates source `62f2a006436e3ef054955b1681d7041fa87892aa`. It records successful Debian-native
bootstrap builds, CLI/reference comparisons, cache invalidation, tests, traps, binary streams and
generated-C execution. The four default manual images passed 576 CLI calls, 60 Go-reference
comparisons and 12 generated-C executions in total; the repeated GCC optimized image was not
double-counted.

| Completed scope | Status | Evidence | Limit attached to the evidence |
| --- | --- | --- | --- |
| Linux bootstrap and candidate command path | **Complete within this record** | [Evidence and scope](../ops/linux-bootstrap-revalidation-62f2a006.md#evidence-and-scope), [default acceptance](../ops/linux-bootstrap-revalidation-62f2a006.md#unchanged-default-acceptance), [regression results](../ops/linux-bootstrap-revalidation-62f2a006.md#regression-results) | Transcribed external results on Debian 13.3 amd64, Linux 6.18.57, glibc 2.41, Go 1.26.10, GCC 14.2.0 and Clang 18.1.8, with a four-core/4 GiB cgroup. It was not a new local execution or hosted-CI receipt, and the raw-results ZIP was not supplied. |
| Linux `check/build/run/test/new` and cache behavior | **Complete as covered by the record** | [Matching bootstrap pair](../architecture/candidate-linux-build-v1.md#matching-bootstrap-pair), cited regression results | Coverage includes ordinary/prepared checks, cache cold/hit/corruption/invalidation, stale locks, replay, stdin/EOF/timeouts, traps, binary output and argument boundaries. It does not establish every possible host, load or input. |

The record explicitly leaves a full `go test ./...`, the complete fixture/sanitizer/ABI sweep,
Ubuntu execution, native Windows execution, clean-machine distribution, self-host acceptance,
repeated load distribution and compiler throughput comparison outside its evidence. Those are
separate scopes. In particular, future Linux execution must reuse this completed baseline rather
than reopen it merely because local Windows probes skip Linux.

## Excluded from this queue

These actions and evidence remain separate and do not become complete through this queue:

- Push, pull-request creation, hosted workflow dispatch, publication and release.
- Deployment, service activation, repository visibility and supported-distribution claims.
- Clean-machine or fresh-OS transfer, installation and portability proof.

No excluded action was performed while updating this document.

## Status and evidence rules

- Execute the rows in order. A bounded contract may be prepared for the next row while the current
  row is being completed, but implementation evidence does not skip ahead.
- A passing focused probe is evidence for its named slice. It does not complete a broader row,
  promote a target or turn a documented gap into a reproducer.
- A completion statement names the source revision, commands or receipt, profiles, omitted checks
  and material limits. An unexecuted or unavailable check is recorded as skipped with its reason.
- Later completion records supersede earlier pending prose. This document replaces the status
  decisions in the historical ordered-work table on `completion-plan.md`; neither preserves a
  conflicting "pending" claim after a later owner supplies completion evidence.
- Keep code, grammar, target flags, contracts, fixtures, CI and packaging changes in their own
  reviewed slices. This queue authorizes the named implementation sequence; it does not make
  unlisted syntax, API or target support available. Those surfaces change only with their owning
  contract, implementation and fixtures.
