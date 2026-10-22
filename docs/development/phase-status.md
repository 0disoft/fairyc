# Phase status

- Phase 3 evidence snapshot: 2026-10-21 at `6c0bea17`; local measurements at clean `b2181e66`
- Earlier Frontend inspection: 2026-10-21 at `24dbbae016c45f50c168a1ff6840aa9bfa6636c1`
- Focused Linux receipt: `7f7362b87c231c30ea89f3907396957f0d599032` (v3, 28 cases),
  [Linux Go job 119918006715](https://github.com/0disoft/fairyc/actions/runs/39805612550/job/119918006715)
- Previous focused Linux receipt: `5856f62109877c53c6388a4a03e0ac56c101a7a6` (v2, 24 cases), historical
- Previous SDK receipt: `ea0957e6d45dc63bf248108cdb5f47b6a0f3002d`, historical evidence
- Scope: [roadmap](../product/01-roadmap.md) Phases 0–4, current Stage 0
- Evidence: [Frontend run 39798530273](https://github.com/0disoft/fairyc/actions/runs/39798530273), all nine jobs passed; native
  [job 119899431035](https://github.com/0disoft/fairyc/actions/runs/39798530273/job/119899431035) and Linux Go [job 119892384769](https://github.com/0disoft/fairyc/actions/runs/39798530273/job/119892384769)
  passed. Earlier [run 39782215105](https://github.com/0disoft/fairyc/actions/runs/39782215105),
  [run 39768680761](https://github.com/0disoft/fairyc/actions/runs/39768680761), and
  [run 39581207722](https://github.com/0disoft/fairyc/actions/runs/39581207722) remain historical evidence

This page summarizes gate evidence. The roadmap, language contracts, and owning fixtures retain
authority. The roadmap defines no Phase 5, 6, or 7.

## Gate summary

| Phase | Status | Evidence and remaining work |
| --- | --- | --- |
| 0 — contracts | Fixture coverage passes; full exit not established by this review | Mandatory parser, semantic, resource, Region, Result, MIR, verifier, and package runners pass in the inspected Go CI lane. This review did not audit every written contract for contradictions, which is also part of the exit gate. |
| 1 — interpreted semantics | Exit fixtures pass for the admitted scope | Positive/negative, active-payload/drop-trace, and verifier mutation runners pass in that lane. The [verifier runner](../../internal/mirverifyfixture/runner_test.go) checks 11 seeds and 23 isolated mutations. Later language capabilities remain separate admissions. |
| 2 — hosted executable | Partial | [Linux native job](https://github.com/0disoft/fairyc/actions/runs/39798530273/job/119899431035) passes provisioning, layout, ABI, runtime, Region byte, and generated-source probes. The separate [Linux Go job](https://github.com/0disoft/fairyc/actions/runs/39805612550/job/119918006715) passed all 28 focused Linux cases without skips, including four trap-receipt cases and 12 equality, comparison, and checked-decimal cases. Only `complete-runtime` and `complete-standard-library` remain missing for Linux in the [target spec](../../internal/target/spec.go); all three maturity flags remain false. |
| 3 — modules, packages, cache | Satisfied for current Stage 0 verified-MIR cache scope | Stable local-workspace commands, public/private invalidation, opt-in persistent cache, and v4 incompatible safe misses are verified. The [local snapshot at `b2181e66`](../benchmarks/package-cache/README.md) supplies 65 validated observations. Inline bodies are outside the admitted contract; [future admission reopens this gate](#phase-3-scope-decision-2026-10-21). |
| 4 — Atlas export | Satisfied for current Stage 0 | The [exit evidence](../atlas/phase4-exit-evidence.md) covers the one-way import boundary, stale/incompatible-map rejection, explicit unknown edges, and text non-authority. Broader Atlas capabilities remain separate gates. |

## Linux toolchain baseline receipt (v37, historical)

[Linux job 119164496059](https://github.com/0disoft/fairyc/actions/runs/39581207722/job/119164496059)
at `c845848c95395622441b059c25c4a27a4bb6fee7` passed both Provision reference Linux toolchain and
native fixtures on Ubuntu 24.04 amd64. Its `fairyc.native-c-report/v37` records glibc 2.39,
Clang 18.1.3, and GCC 13.3.0:

| Observation | Result |
| --- | ---: |
| C emission cases | 52 |
| Interpreter/C outcome and drop-trace comparisons | 208 |
| Sanitizer executions | 104 |
| Layout pass cases / native probe executions | 11 / 4 |
| ABI pass cases / bidirectional Clang–GCC crossings | 14 / 56 |
| Scripted Linux console failure executions | 4 |

Every runtime and generated-source-call counter is 4. This receipt fulfills the bounded Linux
toolchain baseline/provisioning evidence alongside the existing layout/ABI observations.
Full runtime and standard-library coverage remain missing as `complete-runtime` and
`complete-standard-library`.
The target's `emittable`, `conformant`, and `supported` maturity flags remain false.

The [2026-10-22 Linux coverage audit](linux-runtime-coverage.md) maps the implemented runtime and
SDK surfaces to source owners, fixtures, and the current Frontend evidence at `5856f621`.
It distinguishes missing Linux implementations from incomplete execution evidence and design
exclusions. Windows API parity is not the Phase 2 exit gate.

The [focused Linux receipt](linux-runtime-coverage.md#focused-linux-receipt) passes all 24 File
progress, heap OOM, prepared-publication, equality, comparison, and checked-decimal executions
without skips. The [active-payload
mapping](linux-runtime-coverage.md#active-payload-observation-mapping) adds three direct-cleanup
cases: implicit `Some`/`None` cleanup and explicit discard of `Result<Success, Failure>` for both
alternatives. All 55 cases now pass the expanded hosted Linux lane with 220 process-status/drop
comparisons and 110 sanitizer executions.
`complete-runtime` and `complete-standard-library` remain missing, and `emittable`, `conformant`,
and `supported` stay false.

[ADR 0474](../adr/0474-linux-toolchain-baseline-and-provisioning.md) and the
[Linux toolchain contract](../abi/linux-toolchain-v0.md) now define version floors, preflight,
and installation. The hosted receipt above verifies their implementation for the bounded Ubuntu
24.04 native profile; Windows target maturity flags are unchanged. Phase 2 is still partial.

The same run's [Linux Go lane](https://github.com/0disoft/fairyc/actions/runs/39581207722/job/119159050977)
and [Windows Go lane](https://github.com/0disoft/fairyc/actions/runs/39581207722/job/119159050850)
pass the native low/optimized [trap-location probes](../../internal/cbackend/trap_location_native_test.go):
distinct bounds/assert sites, dynamic shifts, escaped UTF-8 names, and unknown overlong names.
The [runtime contract](../runtime/runtime-stdlib-v0.md#thin-runtime) defines that bounded payload.
Stable `run` and experimental `run-workspace` now display its source path and one-based line/column
from the compiled snapshot, using build-local package IDs to distinguish duplicate file names across
packages and feature instances; unknown locations retain category-only output. This CLI addition
does not promote target maturity.

Static byte-string index/slice failures now feed C-only non-returning callee inference, fixing
the MSVC optimized-build C4702 observed in the cross-package probe. Local low/optimized native
probes check the original `bounds` category and source range, including a wrapper call chain;
the package-aware CLI probe checks both run commands. The later hosted receipt below also covers
the corrected code; target maturity is unchanged.

## Public SDK Linux receipt (v38)

[Linux job 119899431035](https://github.com/0disoft/fairyc/actions/runs/39798530273/job/119899431035)
at `5856f62109877c53c6388a4a03e0ac56c101a7a6` records report
`fairyc.native-c-report/v38` for `x86_64-linux-gnu` on Ubuntu 24.04 with glibc 2.39,
Clang 18.1.3, and GCC 13.3.0. It records 55 C emission cases, 220 interpreter/C and
220 drop-trace executions, 110 sanitizer executions, 56 ABI crossings, and four layout probes.
It also records four public SDK storage executions. SystemHeap, Buffer, Arena, text-view,
owned-text, and public-File counters are each four.
The SDK fixture executes `alloc_zeroed_bytes` and `copy_bytes` with normal, empty, and forced
payload and tracking-node OOM, including recovery, preserved data, and zero live allocations.
These four compiler/profile SDK executions are separate from the 110 sanitizer executions.

All nine Frontend jobs passed. The same run's
[Linux Go job 119892384769](https://github.com/0disoft/fairyc/actions/runs/39798530273/job/119892384769) passed all 24 focused Linux cases without skips under v2, including the 12 new helper cases.
The previous [run 39782215105](https://github.com/0disoft/fairyc/actions/runs/39782215105)
remains historical evidence for its 52-case receipt. The earlier
[Linux job 119796130897](https://github.com/0disoft/fairyc/actions/runs/39768680761/job/119796130897)
at `54cbeb17` also remains historical evidence for its recorded revision, including public Buffer
`empty`, `resize_zeroed`, `truncate`, `view_mut`, and text and `from_buffer` cases. The
[coverage decision](linux-runtime-coverage.md#phase-2-exit-decision-2026-10-22) maps the current cases.
Phase 2 remains partial; neither target maturity nor supported-distribution status changes.

## Phase 2 exit decision (2026-10-22)

Keep Phase 2 partial. The representative minimum is covered by the v38 receipt, but
`complete-runtime` and `complete-standard-library` remain missing and all three target flags stay
false. Conformance requires every declared language, layout, bidirectional ABI, runtime, and
toolchain fixture to pass without skips, as defined by the [C ABI target claims](../abi/c-abi-v0.md#target-status)
and [roadmap exit gate](../product/01-roadmap.md#phase-2-first-hosted-executable-slice).

The [v3 Linux Go receipt](https://github.com/0disoft/fairyc/actions/runs/39805612550/job/119918006715)
records all 28 required compiler/profile cases passing with no missing, skipped, or failed cases.
Linux receipt-write progress contributes four cases, each running six deterministic scenarios
with exact receipt suffix, pointer, trace, and exit checks. Equality, comparison, and decimal
parsing retain their 12 passing cases. The earlier v2/24-case and v1/12-case receipts remain
historical. The 0–191 process-status oracle and manifest deferrals do not observe the whole target. The
[coverage decision](linux-runtime-coverage.md#phase-2-exit-decision-2026-10-22) records the source
owners and remaining evidence. Windows filesystem parity and distribution remain outside this gate.

## Region byte evidence (2026-10-22)

The Region byte contract is committed at `ca44226b`; zeroed allocation lowering is committed at
`e035e79c`. Byte-copy lowering is committed at `0ed33081`. Targeted workspace tests pass for
both operations, including one-time evaluation, zeroed and copied bytes, subviews and independent
writes, using a copy after its source local leaves scope, legacy `place`, and the expected `E3101`, `E2101`, and
`E1006` rejections. See [Region byte tests](../../internal/workspacecompiler/region_bytes_test.go)
and [copy tests](../../internal/workspacecompiler/region_copy_test.go).

`TestSDKStorageProbe` passed Linux C emission on Windows and low/optimized MSVC runtime execution.
It covers payload and tracking-node forced OOM for both operations, empty requests without
allocation, recovery, prior-byte preservation, and zero live allocations. See
[SDK storage source](../../internal/cbackendnative/sdk_storage_source.go) and
[SDK storage test](../../internal/cbackendnative/sdk_storage_test.go).
`TestRegionByteCopyPersistentWorkspace` passed a cross-session disk-cache hit and execution; it
executes all three Region modes before and after disk-cache restoration.

[Frontend run 39798530273](https://github.com/0disoft/fairyc/actions/runs/39798530273) covers both
byte operations in four compiler/profile SDK executions at the current source revision. The fixture
checks normal, empty, forced payload and tracking-node OOM, recovery, prior-data preservation, and
zero live allocations. The SDK probe is separate from the 110 sanitizer executions. Related
Go/fixture suites and vet pass locally. The local full repository Go suite was not rerun; the hosted
native and focused Linux Go jobs passed. Phase 2 remains partial, and every target maturity gate remains unchanged.

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

1. Add the pure nested-resource cleanup fixture to the existing Linux differential/drop/sanitizer
   matrix. Require return 42 and four ordered drop events under GCC/Clang low/optimized; see the
   [remaining-work audit](phase2-remaining-work.md). Keep Phase 2 partial and both missing markers.
2. Reopen Phase 3 before any inline contract or cross-package inline-body reuse admission, as specified above.
