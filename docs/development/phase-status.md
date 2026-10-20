# Phase status

- Phase 3 evidence snapshot: 2026-10-21 at `6c0bea17`; local measurements at clean `b2181e66`
- Earlier Frontend inspection: 2026-10-21 at `24dbbae016c45f50c168a1ff6840aa9bfa6636c1`
- Scope: [roadmap](../product/01-roadmap.md) Phases 0–4, current Stage 0
- Evidence: [Frontend run 39471634965](https://github.com/0disoft/fairyc/actions/runs/39471634965), all nine jobs passed

This page summarizes gate evidence. The roadmap, language contracts, and owning fixtures retain
authority. The roadmap defines no Phase 5, 6, or 7.

## Gate summary

| Phase | Status | Evidence and remaining work |
| --- | --- | --- |
| 0 — contracts | Fixture coverage passes; full exit not established by this review | Mandatory parser, semantic, resource, Region, Result, MIR, verifier, and package runners pass in the inspected Go CI lane. This review did not audit every written contract for contradictions, which is also part of the exit gate. |
| 1 — interpreted semantics | Exit fixtures pass for the admitted scope | Positive/negative, active-payload/drop-trace, and verifier mutation runners pass in that lane. The [verifier runner](../../internal/mirverifyfixture/runner_test.go) checks 11 seeds and 23 isolated mutations. Later language capabilities remain separate admissions. |
| 2 — hosted executable | Partial | [Linux native CI](https://github.com/0disoft/fairyc/actions/runs/39471634965/job/118803299115) observes layout, ABI, runtime, and generated-source probes. The [target spec](../../internal/target/spec.go) still lists four missing maturity entries, detailed below; `conformant` remains false. |
| 3 — modules, packages, cache | Satisfied for current Stage 0 verified-MIR cache scope | Stable local-workspace commands, public/private invalidation, opt-in persistent cache, and v4 incompatible safe misses are verified. The [local snapshot at `b2181e66`](../benchmarks/package-cache/README.md) supplies 65 validated observations. Inline bodies are outside the admitted contract; [future admission reopens this gate](#phase-3-scope-decision-2026-10-21). |
| 4 — Atlas export | Satisfied for current Stage 0 | The [exit evidence](../atlas/phase4-exit-evidence.md) covers the one-way import boundary, stale/incompatible-map rejection, explicit unknown edges, and text non-authority. Broader Atlas capabilities remain separate gates. |

## Linux receipt

The native job emits `fairyc.native-c-report/v36` on glibc 2.39 with Clang 18.1.3 and GCC 13.3.0:

| Observation | Result |
| --- | ---: |
| C emission cases | 52 |
| Interpreter/C outcome and drop-trace comparisons | 208 |
| Sanitizer executions | 104 |
| Layout pass cases / native probe executions | 11 / 4 |
| ABI pass cases / bidirectional Clang–GCC crossings | 14 / 56 |

These receipts close the old hosted layout/ABI observation gaps. Full runtime and standard-library
coverage, toolchain baseline, and toolchain provisioning remain missing in the target spec as
`complete-runtime`, `complete-standard-library`, `toolchain-baseline`, and `toolchain-provisioning`.
The target's `emittable`, `conformant`, and `supported` maturity flags remain false.

## Cache evidence

Current keys use [whole-package cache v4](../package/whole-package-cache-v4.md). The source and
fixture tests distinguish private implementation changes from public-interface changes and reject
incompatible cache inputs. Together with stable local-workspace command support and the local
measurements below, this satisfies the revised bounded Stage 0 gate.

Source checks: [independent-session reuse and compiler-build identity misses](../../internal/workspacecompiler/session_test.go)
(`TestPersistentSessionReusesStableMIRAcrossIndependentSessions`), mixed-artifact fallback and healing
(`TestSessionInvalidatesSourceAndFallsBackFromMixedArtifactContexts`, same file), and
[corruption/conflicting-generation safe misses](../../internal/packagecache/disk_test.go)
(`TestDiskStoreRejectsCorruptionAndConflictingGenerationsAsMiss`). These are source pointers, not new
test receipts; the earlier `24dbbae016c45f50c168a1ff6840aa9bfa6636c1` Frontend run covered the Go suite.

Frontend's Windows smoke explicitly skips measurements. The separate
[Measurements workflow](../../.github/workflows/measurements.yml) last
[failed](https://github.com/0disoft/fairyc/actions/runs/39164148711) at `822fd9ee` during starter-guide
validation, before the final persistent-cache process/growth measurements. Its earlier
[success](https://github.com/0disoft/fairyc/actions/runs/38178268885) used `b892204a`.
Neither establishes fresh measurements for the inspected commit or a current cache regression.
Reference-machine speed comparisons and default-cache policy are later promotion decisions.

The later [local cache-only run](../benchmarks/package-cache/README.md) at clean `b2181e66`
records 65 validated observations on a declared Windows machine. It supplies raw cold/warm,
edit-invalidation, and retained-byte evidence without rerunning the full smoke or claiming hosted
measurement success. These observations cover verified MIR caching, not native executable builds.

## Phase 3 scope decision (2026-10-21)

The old literal inline-body exit clause is intentionally revised to match the deferred design.
[ADR 0139](../adr/0139-public-interface-hash-v1.md) (superseded) states that Stage 0 has no
public inline-body contract and defers cross-package inline bodies to a separate versioned hash and
invalidation class. Its Accepted successor, [ADR 0173](../adr/0173-public-interface-coordinate-hash-v2.md),
preserves the semantic facts and excludes inline bodies; the version chain through public-interface
v5 retains that boundary. This is scope alignment, not evidence that inline tests passed.

Before admitting an inline function contract or new cross-package inline-body reuse, reopen Phase 3
and require that separate versioned hash/class plus body-only dependent-cache invalidation on
cold/warm and persistent paths. Keep private-implementation stability and incompatible safe misses
checked. No language/runtime feature or cache algorithm changes here; Phase 2 remains partial.

## Next work

1. Resolve Phase 2's four target-evidence entries before conformance promotion.
2. Reopen Phase 3 before any inline contract or cross-package inline-body reuse admission, as specified above.
