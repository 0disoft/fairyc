# Completion plan

- Started: 2026-10-23, from clean `31ce0891`
- Priority: a small compiler, fast builds, and bounded retained build files
- Execution: small local commits; publication and other owner decisions remain separate

This is the current work queue, not a new language revision or a claim that later candidates are
accepted. The roadmap's Phases 0–4 retain their existing meanings. Implemented functionality and
valid execution records are reused instead of reopening completed work.

## Ordered work

| Order | Work and completion condition | Status |
| --- | --- | --- |
| 1 | Establish a current native-build time/footprint baseline with the existing runner, and keep one finite list of unfinished work. | Complete: six cases and 42 verified builds at clean `31ce0891`. |
| 2 | Finish the selected C interoperability slices: exact C-scalar array members, then explicitly based C enum array members. Assess record-element arrays and pointer extensions against actual users and the declared target scope. Preserve target-specific layout and separate-object value crossings. | C-scalar and C enum arrays committed with Windows proof. Scope assessment below retains the other deferrals; native Linux is pending. |
| 3 | Close missing implementation and execution in the declared Windows/Linux runtime, SDK and ABI profiles. Reconcile actual generated-object requirements with the Windows maturity matrix. Promote no target on incomplete evidence. | Windows full correctness smoke and four mixed-stack executions pass in the [current record](../ops/evidence/candidate-region-workspace-gaps-20261023.json). Hosted execution, remaining unwind obligations, Linux execution and clean-machine evidence remain open. |
| 4 | Improve measured no-op/edit build bottlenecks, SDK reuse, cache retention and failed-build cleanup. Repeated edits must remain bounded; a cache must justify its retained bytes with saved work. | Local Windows improvements implemented: smaller preview, bounded release object reuse and empty-directory collection. Measured costs below; Linux retains its existing path. |
| 5 | Prepare a Windows preview ZIP, extracted starter path, explicit runtime dependencies and separate editor tools. Validate clean-environment compiler/program use where an environment is available. Public release and repository visibility remain owner decisions. | All twelve local packaging/starter steps pass at clean `c8f5964e`. A separate hashed portability ZIP passes seven toolchain-free execution steps locally; its dependency manifest and standalone verifier are ready for transfer. Clean-machine proof and publication remain open. LSP stays separate. |
| 6 | Renew latest-source candidate/reference comparison and consecutive self-build equality. Finish Region diagnostic parity, then implement the Linux host path. Measure before selecting a new default compiler; keep the reference bootstrap available. | Full CLI builds and fixed-point equality pass for snapshot `91762c18`; the [CLI follow-up](../ops/evidence/candidate-cli-parity-20261024.json) closes its recorded Windows comparisons. ADR 0487 now changes the runtime selector/asset inputs: 249 runtime byte comparisons pass, but full CLI/fixed-point renewal is pending for those new inputs. Linux host implementation/execution remain open. |
| 7 | Add a small opt-in public-interface/effects comparison if the completed compiler needs it. Large-source notices belong in lightweight tooling. | Pending |

## Scope and evidence

- Phase 1, verified-MIR cache Phase 3, and the admitted Atlas Phase 4 scope already have exit
  evidence. Phase 0's bounded audit passes without a universal document-consistency claim.
- Phase 2 is partial. Integer/floating array value transport, nested cleanup observations and the
  constant-trap MSVC fix are already implemented. The current target scope is the checklist for
  further ABI work, not old deferred prose or closed issue labels.
- The representative runtime/SDK minimum is covered. Completeness labels do not justify adding
  arbitrary tests or unrelated SDK APIs. A missing requirement must name the actual behavior.
- The Windows authored native companion exists. Full latest-source comparison/fixed-point
  renewal and Linux candidate execution remain separate from its focused proof.
- The first preview does not wait for complete self-hosting, a new backend, or universal platform
  support. Windows distribution still needs its own executable evidence.

## Cost baseline

Use `fairyc-bench` for tokenizer, many-module and file-search workloads, development and release,
five edits including a public edit, and one initial reference sequence. This is a local regression
reference, not a statistical language ranking. A suspected timing regression needs a focused
repeat before a design decision.

Record compiler size, cold/no-op/private/public-edit time, executable and total retained bytes,
cache growth and post-clean bytes. The existing runner does not observe transient peak disk
usage; that remains an explicit measurement gap. Keep only compact receipts and summaries in Git,
not experiment workspaces, binaries, caches or another set of localized report images.

The [2026-10-23 baseline](../benchmarks/completion-baseline-20261023.json) records six passing
cases and 42 verified builds. The compiler is 17,831,936 bytes. Retained native-build totals are
133,229-296,041 bytes, with zero growth across five edits in each case. Cleaning retains
11,225-105,765 shared-cache bytes. No-op builds still take roughly 1.6-8.5 seconds; investigate
that repeated work in step 4, with focused repeats because release timings vary substantially.

## Scope assessment and cost follow-up

The 2026-10-23 caller assessment inspected the SDK's alloc/bytes/text/fs/io/process/optional-vec
sources, the Windows file/output/target and owned-text examples, and the candidate's Windows
host/process/rooted-file bridges. Their foreign pointers use the existing scalar, opaque, record
and `void **` paths; no record-element C array requirement was found in that bounded set.
Keep those extensions deferred until a caller needs them. This does not erase the broader C ABI
contract or make Phase 2 complete. The representative Linux runtime minimum is already covered;
the separate missing Linux filesystem extensions and Windows maturity/distribution rows remain
listed in their owning documents.

The focused file-search probe at `3679d2a3` measured about 68 ms for checking and 998 ms for a
warm development native build. Process waiting dominated the profile; this single run is not a
speedup claim. Do not add another frontend cache on that evidence. Packaging omits Go compiler
debugging data, reducing its executable by 26.6%; see the [release procedure](../ops/release.md).

The [follow-up observations](../benchmarks/completion-followup-20261023.json) at clean `8ba3fab7`
cover tokenizer and file-search in both native profiles. Loading/checking takes 75–76 ms and C
emission/support assembly 25–30 ms. With runtime object hits, native builds still take 745–941 ms
in development and 3.22–3.51 seconds in release. The next optimization should address generated-unit
compilation; increasing frontend cache retention does not address this measured cost. These are
single observations per workload/profile, not statistical speed claims.

The same record captures actual generated/runtime object relocations, default libraries and final
PE imports/unwind. See the [Windows matrix](../abi/windows-target-maturity-v0.md#bounded-production-observations-2026-10-23).
The local preview ZIP is 4,619,368 bytes and its compiler 13,094,400 bytes. ZIP hashes, extraction,
starter generation, locked checking, semantic/native tests and both run profiles pass. A separate
stripped-compiler smoke at `3679d2a3` also passes with an empty PATH on this development host.
Neither observation substitutes for transfer to a clean Windows machine. The artifact remains
under `.cache/starter-smoke-wJnMau/`; no release, push or visibility change was performed.

## Windows build cost result

[ADR 0486](../adr/0486-bounded-windows-generated-unit-reuse.md) caches only the two most recent
Windows release generated-unit objects, with a 4 MiB encoded-byte ceiling. Development retains
runtime-only reuse: the exploratory measurements did not justify its extra time and storage.
Fresh preprocessing and linking preserve header/library changes. Cache corruption, incompatible
objects, persistent link failures and outage are covered by native execution. Eight generated-source
edits keep the two-record limit. Empty key-directory collection now also removes evicted directories
and retries the writer/collector race without deleting live temporaries or unrelated files.

The [three-repeat file-search comparison](../benchmarks/completion-unit-cache-20261023.json)
checks 24 identical-output builds across clean `8ba3fab7` and `d6aa974b` compilers:

| Release observation | Before | After |
| --- | ---: | ---: |
| Cold build, median | 4.29 s | 4.20 s |
| No-op build, median | 3.60 s | 0.93 s |
| Edited build, median | 3.64 s | 3.91 s |
| Retained total after two edits | 228,653 B | 291,895 B |
| Shared cache after project clean | 105,773 B | 169,015 B |

No-op time falls about 74%; edits cost about 0.28 s more because they must prepare and publish a
new object. Total shared-cache retention grows by 63,242 bytes in this workload, and exactly two
unit-key directories remain after collection. The stripped compiler is 13,111,808 bytes. This is a local
synthetic comparison, not a universal build-speed guarantee or a transient peak-disk measurement.

The current local preview contains the cache changes at clean `4c5967c9`. All twelve packaging and
starter steps pass, including extracted native tests and runs in both profiles. The ZIP is
4,622,649 bytes, SHA-256 `405d51b5d48d24d0bf0b76603a60f61b62f6720b8d29b0e1cb586a67051f0696`;
its receipt is `.cache/starter-smoke-TvUBvY/receipt.json`. This supersedes the earlier preview for
local use, without changing the outstanding clean-machine and publication gates.

The fixed-point verifier fix in `e6a16993` independently authenticates the authored companion and
runtime pack, then copies the verified files into the second-generation directory. Only historical
Go-companion receipts retain the builder-equality check. The earlier runtime pin refresh in
`cf6c4206` is preserved in the [self-hosting history](../architecture/self-hosting-readiness-v0.md).

The [Windows long-path evidence](../ops/evidence/candidate-windows-long-path-20261023.json) passes
234 exact runtime C byte/admission comparisons through Go-native and development/release candidate
test binaries. It also passes 24 trap comparisons using the development command compiler, with
12 cases each in development and release program profiles. The oversized source-key case keeps category-only
output; ordinary dependency and replaced-source cases retain their locations.

The [current record](../ops/evidence/candidate-region-workspace-gaps-20261023.json) at clean
`1dbca9ad` includes both 512-source full CLI builds, exact consecutive C-packet/optimized-PE
equality, and the Windows correctness smoke. Command tests and test listing pass 208 and 178
exact comparisons respectively across both profiles. Native-test/example comparisons were
incomplete at that snapshot; the later records below supersede them. The [Region workspace
audit](phase2-remaining-work.md#region-workspace-audit-2026-10-23) records the gaps subsequently
addressed by the Region input and diagnostic changes below. Linux execution,
clean-machine portability and the remaining Windows maturity requirements stay open.

Region input transport is now implemented without extra runtime allocation. Callees borrow the
caller's handle and never begin or reset it. Candidate E3102/type and E3101/allocation diagnostics
now match the reference. The six real-SDK cases pass in both compiler profiles, including E3104
without the extra E3103 introduced by the earlier Go storage check. The [current
record](../ops/evidence/candidate-region-diagnostics-20261024.json) identifies each compiler and
comparison. The older whole-CLI observations above still belong to `1dbca9ad`. Current checks
match 173 rows per profile, with the skipped Region graph covered separately by the real-SDK
adapter. Semantic-test and test-list comparisons match 208 and 178 rows. Command metadata and
the version-only case use the candidate's authenticated build oracle; behavior uses Go `b0cd35b1`.
The two compilers therefore retain their own source identities without forcing a rebuild of
unchanged Fairy sources. Consecutive C packets and optimized PE images are byte-identical;
native-test and example comparisons have since completed: 30/30 native-test cases and
78/78 examples match in each profile. The compiler self-source example is excluded from that
example count and covered by the separate fixed-point receipt.

The subsequent native CLI sequence found that adding ordinary C transport scalars had accidentally
widened Go's generic signatures. `cbe0bff1` restores the existing generic boundary without rejecting
ordinary C transport or unused planner type arguments. All 22 original-signature cases and the two
Fairy native report tests pass. The full development/release native CLI sequence also passes against
that clean Go build, including exact diagnostics, preserved output on failure, program streams,
trap locations, exit bits and temporary-directory cleanup. The [follow-up
record](../ops/evidence/candidate-cli-parity-20261024.json) retains the earlier comparisons' own
reference revision; they were not silently relabeled or repeated. No candidate source changed.

The Go preview at `b0cd35b1` is 4,628,074 bytes zipped, with a 13,126,144-byte compiler.
All twelve packaging/starter steps pass in `.cache/starter-smoke-5jmjhK/receipt.json`.
The authored release compiler is 6,407,680 bytes, excluding its separate companion/runtime pack.
Keep Go as the default: the two positive Region check observations took 4.44–4.67 seconds with
the authored release compiler versus 0.13–0.15 seconds with Go. These were single observations
during other verification, not a benchmark. A smaller executable alone does not justify promotion.

The [Windows portability record](../ops/evidence/windows-portability-local-20261024.json) refreshes
the Go preview at clean `c8f5964e`: 4,632,523 bytes zipped and 13,134,336 bytes for the compiler,
with all twelve starter steps passing. A separate 4,761,405-byte transfer ZIP contains the same
compiler, two prebuilt native profiles, a file/DLL manifest, a PowerShell verifier and licenses.
Seven executions pass after extraction with empty child PATH; byte-buffer/file round trips match
and scratch files are removed. The verifier also rejects corrupted images, incorrect native
behavior and existing receipt paths. PowerShell 5.1 and 7 are exercised, with their compiler
identities retained separately. Static imports are only `kernel32.dll`; runtime-loaded DLL closure
is not established. This host has Visual Studio and the Windows SDK, and no Windows Sandbox
executable is available, so clean-machine execution remains open. No host service was installed.

## Feature proposals

The first [portable file-to-Buffer slice](../adr/0487-portable-file-buffer-reads.md) adds a shared
reserve/read/commit implementation and Linux adapter while retaining the Windows symbol. Both
MSVC profiles pass the native allocation/read/failure observer; the Linux probe cross-compiles
and links at low/optimized settings but has not executed here. The authored runtime pack is
150,925 bytes across 22 assets, four bytes smaller than the preceding pack. Its two authored
tests pass 249 exact output/error comparisons through Go-native and both candidate C profiles.
SDK `0.0.0-stage0.87` now links the public operations to the portable symbol. Both-target
emission, Windows low/optimized binary/multi-window reads, and the existing allocation-failure
probes pass. The candidate SDK source pin, embedded literals, foreign rule and C result record
are updated. The [SDK transition record](../ops/evidence/candidate-sdk-bootstrap-refresh-20261024.json)
provides a refreshed private seed, 249 runtime comparisons in three execution paths and both
SDK bundle tests in development/release candidate C. Full CLI rebuilding and fixed-point
renewal follow; earlier full-CLI records remain evidence for their original snapshots.
Linux native execution remains open.

Keep existing effects. Consider only the narrow public-interface comparison and source-size
notices after the compiler/user path is ready. Data trust and state protocols first use ordinary
types, resources and APIs in real applications. Change Fence, Task Tree and SMT-based invariants
are not current implementation tasks.

Broader generics/interfaces/closures, QBE or LLVM, additional targets, remote package management,
Atlas impact/MCP/structural edits, and chain-specific execution/persistence remain later candidates.
They need a demonstrated user or measured benefit before becoming work in this queue.

## Completed slices

- Exact C-scalar array fields: [ADR 0484](../adr/0484-repr-c-scalar-array-transport.md).
  All sixteen identities retain target layout and internal value transport. Local compiler,
  MIR/interface and Linux/Windows emission checks pass; MSVC C11/C17 low/optimized native
  execution passes for all fifteen portable providers. Native Linux execution remains pending.
- Explicit C enum array fields: [ADR 0485](../adr/0485-repr-c-enum-array-transport.md).
  All thirteen bases preserve nominal identity and target-owned representation through whole-array
  copies. Source/MIR/interface and both target-emission checks pass; four MSVC combinations each
  execute 36 enum-array cases, including undeclared foreign ordinals. Native Linux is pending.
  The caller assessment above retains record-element arrays and further pointer forms as deferrals.

Local Linux execution is currently unavailable: `wsl --list --quiet` exits with
`Wsl/CallMsi/Install/REGDB_E_CLASSNOTREG`, and no Docker command is installed. This is a host
limitation, not a successful Linux run. Continue target-independent implementation and Windows
execution; retain Linux-native checks for an available runner without installing host services.

## Working rules

- Implement, check the affected behavior, and commit each coherent slice.
- Reuse unchanged evidence. Batch hosted execution only after an authorized push; do not create
  new workflows or repeatedly dispatch large measurement matrices.
- Preserve both runtime behavior and size/time costs. A shorter source file alone is not a
  performance improvement, and deleting an obligation is not completing it.
- If external validation or a release decision is unavailable, record the exact missing evidence
  and continue independent local work. Never relabel an unexecuted check as passed.

## References

- [Phase status](phase-status.md)
- [Remaining C ABI scope](phase2-target-scope.md#remaining-source-boundaries-after-f1f3)
- [Phase 2 assessment](phase2-remaining-work.md)
- [Windows maturity](../abi/windows-target-maturity-v0.md)
- [Self-hosting](../architecture/self-hosting-readiness-v0.md)
- [Build footprint measurements](../benchmarks/README.md)
- [Release](../ops/release.md)
