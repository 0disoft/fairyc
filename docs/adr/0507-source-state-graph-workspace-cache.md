# ADR 0507: Current-source state graph workspace artifact reuse

- Status: Accepted for the explicit Go workspace state-graph session
- Date: 2026-11-07
- Extends ADRs 0503-0506 and the existing verified package cache contract.

`NewExperimentalStateGraphSession` selects the existing workspace artifact
coordinator with the explicit current-source graph analyzer. One source file
without imports is admitted. Ordinary workspace/session constructors keep
their existing admission. The profile `experimental-source-state-graph/1`
participates in every memory/disk compiler-build cache key; a call-only MIR
projection cannot be reused under ordinary source semantics.

Every request rebuilds current source layout, helpers, typed graph operations,
effects and ownership before cache lookup. Existing package source, interface,
state-body, coordinate, target and exact compiler-build identities continue
to own invalidation. Even a comment-only source edit misses while retaining
the same semantic state-body hash. Helper body edits change that hash. Invalid
or canceled source produces no executable reuse or cache publication.

Memory hits reuse ordinary MIR and lower current state MIR. Disk hits rebind
both ordinary and state artifacts through the existing stable compilation
bundle. Neither stores transaction outcomes or native executable images.
A verified complete hit skips the coordinator's ordinary MIR lowering entry;
the later native handoff still independently verifies the compiled result.

Cache reassembly uses a fresh function-symbol namespace. The state profile
clones the reconstructed program and binds exact function names and internal
calls back to current HIR symbols before native handoff. Disk state roots bind
through the same map; memory state roots already use current HIR symbols.
Inactive void-return destinations and absent helper lists regain the fresh
source projection's representation. No executable instruction is discarded,
native equality/verifier check is relaxed, or cached object is mutated.
Unknown/duplicate owners, foreign contexts and non-projection terminators
refuse. A rejected rebind falls back to fresh lowering; disk repair discards
only the observed rejected generation before immutable publication.

Windows evidence covers all four authored source families, memory/disk
cold/warm, four actual fresh Go processes, exact restored C/body/program
identity, source/helper invalidation, source rejection, cancellation and
rebind repair. Sixteen reconstructed C images across both profiles perform
32 real native commit/reopen attempts against independent Go state-MIR and
local snapshots, with exactly one snapshot and one CAS per attempt.
Commands, results and failed-attempt accounting are in
[the cache integration record](../architecture/source-state-graph-cache-v1.md).

This closes the explicit Go workspace artifact/cache slice only. Fairy
Complete/source authority and its cache path, ordinary CLI selection, native
Linux and all final acceptance gates remain required. Existing stable cache
schemas, language/CLI/SDK/runtime ABI, DB/provider, dependencies, limits and
CI/runner policies are unchanged.
