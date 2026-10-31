# Ordered work - 2026-10-31

- Status: Approved follow-up implementation queue
- Updated: 2026-11-01
- Supersedes: current-status claims in the [2026-10-23 ordered work](completion-plan.md#historical-ordered-work-superseded-2026-10-31)
- Change boundary: ordered planning and bounded implementation evidence
- Authorization: named sequence implementation; no implicit syntax, API or target support

This is the current ordered queue. Item 1 is complete and the list below is the resulting work
record. Item 2 awaits native Linux evidence. Item 3 has Windows evidence with Linux deferred;
item 4 has Windows evidence for the named production-object corpus; item 5 has bounded Windows
weak-chain and CRT selection evidence; item 6 has bounded Windows default-library and CRT
failure evidence; items 7 through 22 are queued. Existing bounded evidence is
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
| 3 | Aggregate transport in both directions | **Windows verified / Linux deferred** | Code `d558eb1f` checks 116 portable record shapes in separate generated-Fairy/C objects under MSVC C11/C17 low/optimized, plus two C-to-Fairy assertion-trap executions. The reverse entry is a fixture-only C bridge, not a public export or aggregate callback. | The bounded Windows matrix and six unsupported-shape checks pass. The 122-shape POSIX matrix emits locally; native cross-compiler execution is in the batched queue below. Full public-export and target conformance claims remain closed. |
| 4 | COFF relocation coverage | **Windows verified for named corpus** | Code `07fd28a9` inventories actual `unit.obj`/`runtime.obj` from five SDK workspaces under ordinary C11 low/optimized flags, with map/library-log capture. It checks every retained REL32 and ADDR32NB field against its owned source/target positions and final PE bytes. | Ten native executions and twenty objects account for 3,133 fields: 2,404 linked-value checks and 729 explicitly discarded fields. Object addend, final-field, map-owner and discard-evidence mutations fail. This is the named corpus, not every possible Fairy program or CRT archive member; unobserved relocation kinds retain their earlier separate evidence. |
| 5 | Weak linking: chains, cycles and replacement | **Windows verified for named cases** | Code `cbd3b26a` inventories actual generated/runtime objects, distinguishes class-105 records from linker directives, and binds explicit-object chains, strong replacements and strong cuts of cycles to map owners and final PE fields. Actual CRT startup and SDK data alternatives are exercised separately. | Forty controlled selections, four SDK data cases, two CRT startup cases and ten corpus images pass; four unresolved cycles fail in both the inspection planner and real LINK with LNK2019/LNK1120. This does not add Fairy weak syntax, reverify all archive search modes or establish general COMDAT removal. |
| 6 | Default libraries and CRT archive | **Windows verified for named cases** | Code `53d7ed5f` records actual /MT object defaults, absolute searched archive paths and hashes, loaded CRT members, owned final symbols and PE entry at low/optimized. It verifies the five production workspaces and a C control relinked with absolute inputs and no default search. | Sixteen native executions and twenty expected link failures cover missing archives, suppressed defaults, removed demanded members and absent-but-unused OLDNAMES. Ordinary selection uses recorded toolchain directories; explicit relinks have empty LIB/LIBPATH. Broader archive/COMDAT policy, other CRT modes, clean-machine transfer and target promotion remain separate. |
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

## Item 2 progress: authored GNU companion implementation

Commit `11b781f3` adds authored toolchain policy and ELF64 admission. The reported focused
tests pass both-target emission and Windows MSVC low/optimized. ELF inspection is bounded to
64 MiB with a 4 MiB name scan; extended section numbering remains unsupported.

Commit `53840962` adds `linux_runtime.fy` and `linux_runtime_contract.fy` with
`TestLinuxGNUCompanionRuntime` as the Go oracle. The current comparison covers 105 plans from
the authoritative Go standard-library and host sources, with both-target emission and Windows
low/optimized passing. Unknown standard-library, vector, Linux-host, and Windows-dependency
markers are rejected.

Commit `13243c0d` adds `linux_runtime_assets.fy` and a generator that pins 25 source-owned raw
C files in a 138,102-byte runtime pack. The loader verifies the exact source count, size and
SHA-256 pin. The reported Linux runtime checks compare 48 complete C outputs in low/optimized
profiles. The GNU build workflow and POSIX-only C bridge now cover toolchain admission,
separate compilation and linking, bounded stage I/O, ELF admission, descendant-aware process
capture and replacement publication. Windows MSVC low/optimized policy tests exercise real
Fairy compile/link/publish code; the C bridge only runs there through POSIX substitutes, so no
Linux-native execution is claimed.

The `linux_gnu_companion.fy` entry matches the private CLI flags and
loads the fixed sibling `fairyc-linux-runtime-assets.pack` through the Linux rooted reader.
`tools/prepare-candidate-linux-cli.py --companion` selects it in an isolated main. The matching
companion and pack require explicit opt-in installation; the original Go backend remains the
default and has no automatic replacement. Option checks pass under MSVC low/optimized;
complete entry emission passes for both targets with closed Linux support and no Windows
foreign symbols. The companion preparation mode only selects the entry; frontend/cache
platform rewrites stay with the ordinary CLI preparation mode.

Fresh original-workspace `check`/`check --locked` pass with 574 sources and 7,232 MIR
functions; the isolated companion workspace passes with 574 sources and 7,230 functions.
The focused companion suite and GNU adapter boundary checks pass, as do runtime-pack pin
verification, Python syntax checks, scoped Go vet and diff checks. Native adapter and complete
companion tests are authored but skipped on Windows; WSL reports
`Wsl/CallMsi/Install/REGDB_E_CLASSNOTREG`. Earlier Debian evidence covers the Go backend. Native
Linux GNU launch, fixed-point reproduction and distribution remain unverified. Item 2 remains
**Next / in progress** and the target maturity state is unchanged. This progress introduces no
new owner approval gate and does not trigger push, CI, release or deployment.

## Item 3 progress: aggregate call transport

Code `d558eb1f` introduces the shared bidirectional probe and fixes accidental backend admission
of Fairy `bool` as a repr(C) field; exact `c::bool` remains supported. Windows MSVC 14.51.36231
passes 116 record shapes with three field/element-varied C value sets in C11/C17 low/optimized.
Each execution retains two object receipts and checks final PE/unwind structure, status 42 and
empty output. A fixture-only bridge enters actual generated Fairy reconstruction bodies from C;
the marker and independent C observer check their execution and every result leaf. Two additional
low/optimized executions prove assertion-trap delivery on the private channel inside that path.
Six unsupported aggregate forms fail explicitly. Both target emission and focused backend tests,
scoped Go vet and diff checks pass. The [receipt](../ops/windows-aggregate-crossings-20261101.md)
states the exact boundary: no public exports, aggregate callback signatures, retained pointers,
ownership transfer, foreign exception unwinding, hosted CI or target promotion.

## Item 4 progress: generated and runtime COFF relocations

Code `07fd28a9` adds `InspectGeneratedRelocations` and the native
`TestGeneratedRelocationCoverage`. The corpus is `std_bytes_quickstart`,
`std_alloc_quickstart`, `std_text_from_buffer_quickstart`, `std_vec_quickstart` and
`process_args_quickstart`. Each is checked/lowered/emitted from Fairy source; native support is
selected from its actual foreign-symbol set. No handwritten object replaces generated output.
MSVC 14.51.36231 passes low/optimized builds, final-value assertions and exact native outcomes.

REL32 accounts for 1,510 object fields (1,322 linked, 188 discarded); ADDR32NB accounts for 1,623
(1,082 linked, 541 discarded). Source section identity, object addend, target resolution and final
image bytes are recorded for every retained field. Pdata is matched by unique begin/end pairs;
code/data section bases use owned map symbols. COMDAT omission requires the existing exact
selection/discard proof. Unknown kinds and unresolved ownership fail rather than disappearing
from the counts. The [receipt](../ops/windows-coff-relocations-20261101.md) records commands,
source/object/image hashing, negative controls and the bounded scope. Full PE/COFF unit checks,
scoped Go vet and diff checks pass. This Windows-only slice does not add a Linux job or change
the deferred Linux queue.

## Item 5 progress: weak chains and actual alternatives

Code `cbd3b26a` and the [Windows receipt](../ops/windows-weak-aliases-20261101.md) cover
class-105 ALIAS records and C linker directives in low/optimized profiles, with forward/reversed
object order. Fallback, root/middle strong replacement and strong cuts of a two-node cycle bind
both call sites to exact selected owners and final displacement bytes. An unresolved cycle
fails in the bounded inspection planner and independently in LINK, with two unresolved symbols.
The inspection helper does not add a new production driver policy.

The production corpus has no class-105 weak records. The process-argument runtime offers one
SDK data alternative, but it has no root references and neither name is in its final map.
Separate real-header C probes prove fallback value 0 and strong provider value 7 at both
profiles. Actual `libcmt.lib(initializers.obj)` metadata and startup map/library logs establish
strong UCRT/VCRuntime initialization selection in two more executions. The named suites pass
56 native executions and four expected LINK failures, plus PE/COFF unit checks, scoped Go vet
and diff checks. Archive search/absence, broader COMDAT removal, hosted CI and target promotion
remain separate. Linux work stays in the unchanged deferred queue below.

## Item 6 progress: default libraries and CRT archive failures

Code `53d7ed5f` and the [Windows receipt](../ops/windows-default-libraries-20261101.md) bind
ordinary `/MT` low/optimized object defaults to exact searched archive paths/hashes and loaded
CRT owners. Startup must be `LIBCMT:exe_main.obj` at the final PE entry; UCRT/VCRuntime
initialization must have their expected strong archive owners and executable final addresses.
All five generated/runtime workspaces pass again with source/object/image and library receipts.

The C control also links and runs from the same objects with default search disabled, absolute
library inputs and empty `LIB`/`LIBPATH`. Missing explicit CRT archives fail with LNK1181;
suppressing required defaults or removing their demanded actual members fails with LNK1120.
An isolated directory missing unused OLDNAMES still fails default lookup with LNK1104, while
suppressing that unused default succeeds. Sixteen positive executions and twenty expected
link failures pass, alongside PE/COFF unit checks, scoped Go vet and diff checks. Installed
archives remain unchanged. Broader archive/COMDAT forms are next; Linux remains deferred below.

## Batched native Linux validation, 2026-11-01

The maintainer requested deferred, combined Linux validation because this laptop has no usable
WSL environment. Continue bounded Windows work without repeating WSL discovery or provisioning.
Unexecuted Linux cases remain unexecuted and do not block the next Windows work item.

| Pending slice | Native command from the repository root | Current evidence |
| --- | --- | --- |
| Fairy-authored GNU companion entry, runtime pack and publication | `go test -mod=readonly ./internal/cli -run '^TestLinuxGNUCompanionNative$' -count=1 -v` | Windows-development and entry emission checks pass; no native execution for this companion. |
| GNU POSIX adapter | `go test -mod=readonly ./internal/candidatehost -run '^TestLinuxGNUAdapterNative$' -count=1 -v` | Windows POSIX substitutes pass; native test remains unexecuted. |
| Aggregate calls in both directions | `go test -mod=readonly ./internal/foreignstructprobe -run '^TestBidirectionalAggregateLinux$' -count=1 -v` | POSIX 122-record emission passes on Windows. The authored native test links Clang/GCC in both compiler directions and low/optimized profiles; it has not run. |

Use one later source snapshot and record compiler versions, exact source commit, commands, exit
codes and omissions for these slices together. This queue neither launches Linux work now nor
relabels earlier Go-bootstrap evidence as proof of the new Fairy companion or aggregate matrix.

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
| Linux `check/build/run/test/new` and cache behavior | **Complete as covered by the record** | [Historical Go bootstrap pair](../architecture/candidate-linux-build-v1.md#historical-go-bootstrap-pair), cited regression results | Coverage includes ordinary/prepared checks, cache cold/hit/corruption/invalidation, stale locks, replay, stdin/EOF/timeouts, traps, binary output and argument boundaries. It does not establish every possible host, load or input. |

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

- Execute the rows in order, with the explicit batched-Linux exception above. A bounded contract may be prepared for the next row while the current
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
