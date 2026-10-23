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
| 3 | Close missing implementation and execution in the declared Windows/Linux runtime, SDK and ABI profiles. Reconcile actual generated-object requirements with the Windows maturity matrix. Promote no target on incomplete evidence. | Pending |
| 4 | Improve measured no-op/edit build bottlenecks, SDK reuse, cache retention and failed-build cleanup. Repeated edits must remain bounded; a cache must justify its retained bytes with saved work. | In progress: preview compiler size reduced; native compile/link cost is the next performance focus. |
| 5 | Prepare a Windows preview ZIP, extracted starter path, explicit runtime dependencies and separate editor tools. Validate clean-environment compiler/program use where an environment is available. Public release and repository visibility remain owner decisions. | Pending |
| 6 | Renew latest-source candidate/reference comparison and consecutive self-build equality, then implement the Linux host path. Measure before selecting a new default compiler; keep the reference bootstrap available. | Pending |
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

## Feature proposals

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
  Next: assess record-element array and raw-pointer scope against the target contract and users.

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
