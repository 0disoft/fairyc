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
| 2 | Finish the selected C interoperability slices: exact C-scalar array members, then explicitly based C enum array members. Assess record-element arrays and pointer extensions against actual users and the declared target scope. Preserve target-specific layout and separate-object value crossings. | Pending |
| 3 | Close missing implementation and execution in the declared Windows/Linux runtime, SDK and ABI profiles. Reconcile actual generated-object requirements with the Windows maturity matrix. Promote no target on incomplete evidence. | Pending |
| 4 | Improve measured no-op/edit build bottlenecks, SDK reuse, cache retention and failed-build cleanup. Repeated edits must remain bounded; a cache must justify its retained bytes with saved work. | Pending |
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

## Feature proposals

Keep existing effects. Consider only the narrow public-interface comparison and source-size
notices after the compiler/user path is ready. Data trust and state protocols first use ordinary
types, resources and APIs in real applications. Change Fence, Task Tree and SMT-based invariants
are not current implementation tasks.

Broader generics/interfaces/closures, QBE or LLVM, additional targets, remote package management,
Atlas impact/MCP/structural edits, and chain-specific execution/persistence remain later candidates.
They need a demonstrated user or measured benefit before becoming work in this queue.

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
