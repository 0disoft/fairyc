# Go grouped static workspace cache proof — 2026-11-07

Scope: ADR0549 cache proof in isolated detached checkout
`C:/Users/cherr/Documents/workspace/zerodi-wd1/projects/hobby/opensource/fairyc/.cache/grouped-static-20261107`.
Baseline HEAD: `c5b6c9b430037de0b57cf1d0bc53a3fd520aa7f1`.

This worker owns only this receipt and
`internal/workspacecompiler/static_interface_group_cache_test.go`.
The coordinator owns integration and commits; Hume owns Go production changes.
No main-root changes, staging, commits, dependency installation, remote work,
limit increases, Linux execution, or runner changes were performed by this worker.
The coordinator's `internal/cli/candidate_grouped_static_admission_test.go`
and other writers' files were left untouched.

## Executed checks

Go readiness was observed before execution: Hume reported successful grouped
and legacy Windows C11/C17 low/optimized execution. All commands below ran in
the isolated checkout, with `GOMAXPROCS=1`.

```powershell
$env:GOMAXPROCS='1'
go test -mod=readonly ./internal/workspacecompiler -run '^(TestGroupedStaticCache|TestStaticInterfaceCacheInvalidation)' -count=1 -parallel=1 -p=1 -v
go vet -mod=readonly -p=1 ./internal/workspacecompiler
```

Final targeted run: **PASS**, package time **5.351s**, exit 0.
Vet: **PASS**, exit 0, no output. Existing standalone
`TestStaticInterfaceCacheInvalidation/memory` and `/disk` passed unchanged.
The direct parent invocation skips the subprocess-only child; the parent
actually launches and passes that child in a new OS process.
Final file hygiene: `gofmt -l` is empty; `git diff --no-index --check` emits no
whitespace diagnostics for either owned file. Each no-index comparison exits 1
because a new file differs from `/dev/null`; this is not a test/vet exit code.

## Coverage and observed results

| Check | Expected and observed result |
| --- | --- |
| Cold memory and disk | Two package misses, two stored packages, actual lowering |
| Warm memory and disk | Two hits, zero misses/rejections/stores, no lowering |
| Disk new session | Two hits, no lowering; lower callback replaced with fatal guard |
| Disk new OS process | Child uses same real on-disk cache, two hits, fatal lower guard; returns 42 and prints completion marker |
| Group sizes 2 through 16 | Each size has complete bundles for all 13 scalars; cold/warm memory passes; 26 selected add+algorithm specializations; restored Windows C emission succeeds |
| Used declaration/provider edit | Both packages invalidated; declaration rename returns 42, provider + to - returns -2 |
| Unused sibling declaration/provider edit | Both packages invalidated; selected algorithm identity changes while result remains 42 |
| Unused scalar sibling provider edit | Equal-width i64 multiply edit invalidates selected i32 add consumer; result remains 42 |
| Unrelated private body edit | One hit/one miss, dependent algorithm identity and exported interfaces unchanged |
| Partial bundles | E4501 for both requested i32 and unused i64 bundles; no cached executable exposed |
| Duplicate tuples | E4501 for used add/i32 and unused multiply/i64 tuples; no cached executable exposed |
| Wrong signatures | E4501 for unused multiply/i32 and unused multiply/i64; no cached executable exposed |
| Restoring valid source after rejected edits | Original two-package warm cache remains reusable with fatal lower guard |
| Valid-checksum inner metadata forgery | One hit/one rejection, source lowering repairs both stored packages; fresh reader restores both without lowering |
| Restored native program | Fresh disk reader, separate generated/observer objects, MSVC C11 low and optimized: ordered once-only argument evaluation, exit 42, empty stdout/stderr |

The primary consumer's algorithm calls only `add<T>`. Provenance asserts exactly
one `add<i32>` and one `algorithm<i32>`, with no multiply or i64 specialization.
Each edit vector is exactly equal-width; source declaration identities and spans
are compared independently of the intentionally changed signature. Thus unused
sibling invalidation is not explained by moving the selected AST spans.

All 13 supported scalars are listed explicitly in the test: bool, i8, i16, i32,
i64, isize, u8, u16, u32, u64, usize, f32, f64. The size matrix supplies
`13 * size` providers. Each scalar has a typed probe that requests the generic
algorithm; all probes are checked, lowered, cached, restored and C-emitted.
The interpreter calls the bool/i64 probes and the i32 algorithm; it does not
execute the other ten typed probes. This receipt does not claim native execution
of all 13 scalar probes (Hume owns the separate scalar transport proof).

Expectations are literal cache/result vectors. The test does not reproduce
production's group digest, template closure or specialization algorithm.
The inner forgery preserves the outer interface/key, all MIR bytes, and valid
SHA-256 outer generation filename; only the canonical inner interface digest is
changed while retaining its actual prefix. The rejected generation is removed.
There is no claim of authenticating an adversary-rewritten executable body.

## Exact identities

Hashes below are SHA-256 of actual bytes. Production inputs are recorded for
review only; this worker did not modify them.

| Input | SHA-256 |
| --- | --- |
| Owned final test file | `b4932fdbb40b5dc98dcdfdfe2f89d9f2a89410924850bcbbf8dc68c1a957d47d` |
| Go production static_interface.go | `e84637802d0c28ab7527e7921f823aafd1e82f2b1feb045c74ec618e93232c8d` |
| Go production static_interface_group.go | `f36d8ee73b2eab82f81fd6c96314a2d0c22f79bb45404189491fe93659609f74` |
| Primary library fixture | `8fadb1878f0fc4d87c815cd71dec3b626d5eff40509619b9159d244eb9c2dff3` |
| Primary consumer fixture | `e98e3b81d0d07606ffa3880d8f98ddefbd7caea74d8b450e8321bb55fdf0f9ec` |
| Ordered-argument native consumer fixture | `66644c4a1bc6aa1757092caed0f9fbb46d32c7fbe845f383448f42ea6a7daa4f` |
| Restored generated C | `2d32493267b5f90e3100170bb22bfb2ebf7d447c2e994e8e6faf6a77061ef441` |
| Native runtime and observer C | `fa7bde1cd5fa8cb1a280d122a3be2b5958de3a69c290b5ce6cc101aa18348c1b` |

Final test executable identity:
`fairyc.compiler-executable-sha256/v1:d5850b794e38735195eb96d9a81a1bc3c28c5b675b3b1eac28af8b39ffffc977`.
Go: `go1.27.1 windows/amd64`; local MSVC: `14.51.36231`.

Final native executable hashes (actual executions, no reproducibility claim):

- C11 low: `d37fd38760e379410071f3b6b2963893470912821fee711879fed0842d50296c`.
- C11 optimized: `0e31ba5a7c51305ffbe16c0d290de39c5724c4e6fbbbdd12c801e859f301983c`.

Final metadata-forgery run:
original inner interface
`fygif0-sha256-44b614ae5b8ae66c0042bb175b9304f6af22cbac13090dee429e932528ac37c2`;
forged outer-record SHA-256
`232090347ecea6bfc86619c89d2d2d2273a700393a5f3529e50b01f74d9d6695`.
The original interface includes temporary workspace identity and is not a fixed
expectation vector.

Primary algorithm specialization before edits:
`g0_a360c125d4898d9ff6b2d32a03095cffb3c12b272d8d0cd0a0bd807e18ade410`.

| Equal-width edit | Edited source SHA-256 | Algorithm specialization after edit |
| --- | --- | --- |
| Used declaration | `ad92a0b344ea43cd06803e1ddae6face357e3f944aabc6e28121e91a4fd6e8c8` | `g0_bcf107620d932f52208e3f0ffea2136dfeff5bc2253b2043a593ac355cb81b7d` |
| Unused declaration | `bd16c5d372dfe13ea1b9e3ebddd5b19774270f4aacdea92b16ad04d5637c650d` | `g0_0553f4682c99d9f43599585320317bf1fd1cf3630c1acf1e5f74f80c7c51f798` |
| Used provider | `24609dd433807ab98449b76086848ba0285eec6888c89606ac19b31c33670263` | `g0_8694a1cb7497306cfcbc9cf93db9d0670e6b4868af3471145c579eae41f3532e` |
| Unused sibling provider | `fef8b04f18f6c5d66f6c7fdaef2d73bda395426ec8d2955c09208f0ac6e0bd0b` | `g0_a32b8b496d5d3af923d41dafb7f527bb9c9b599c6b3eb44d8a295a27e8b9c3c7` |
| Unused i64 sibling provider | `40050ea09c5f8c365c2f1098ee61e1f84cf7a480eccf2a9677284632a5ec09b2` | `g0_94e9a314e7a659c24202db073506247963d6f43ed867ff0456409c25d90c8a37` |
| Private body | `bdef9b6e0ef54a34f5daefd7a787a3f59772f1afe6872b68bdf20fc7cb38bd18` | Unchanged from baseline |

## Retained unsuccessful attempts

These were fixture/seam assumptions, not reproduced grouped-production defects.
No production change or limit increase was used to make them pass.

First complete targeted attempt: exit 1, package time 4.973s.
Exact test executable:
`fairyc.compiler-executable-sha256/v1:bd10b7e192476b3f1fbb5fa4ec43e3d6b1ea1bb78d8db89cf23f592b71d37c8f`.
Primary and native fixture source hashes were the same as the final run.
The 13-scalar consumer then used unsupported equality expressions; its source
hash was `914f9cf562a94f69df7985c84cc39194681b8acaed35dbd9c22bc692e987b4c7`.
All sizes 2..16 failed as ineligible. The forgery test separately failed its own
64-byte raw-hex assumption because real interface text has a prefix.

```text
--- FAIL: TestGroupedStaticCacheGroupSizesAndCompleteScalarTuples (1.29s)
    --- FAIL: TestGroupedStaticCacheGroupSizesAndCompleteScalarTuples/2 (0.07s)
    --- FAIL: TestGroupedStaticCacheGroupSizesAndCompleteScalarTuples/3 (0.07s)
    --- FAIL: TestGroupedStaticCacheGroupSizesAndCompleteScalarTuples/4 (0.07s)
    --- FAIL: TestGroupedStaticCacheGroupSizesAndCompleteScalarTuples/5 (0.08s)
    --- FAIL: TestGroupedStaticCacheGroupSizesAndCompleteScalarTuples/6 (0.08s)
    --- FAIL: TestGroupedStaticCacheGroupSizesAndCompleteScalarTuples/7 (0.09s)
    --- FAIL: TestGroupedStaticCacheGroupSizesAndCompleteScalarTuples/8 (0.08s)
    --- FAIL: TestGroupedStaticCacheGroupSizesAndCompleteScalarTuples/9 (0.10s)
    --- FAIL: TestGroupedStaticCacheGroupSizesAndCompleteScalarTuples/10 (0.08s)
    --- FAIL: TestGroupedStaticCacheGroupSizesAndCompleteScalarTuples/11 (0.11s)
    --- FAIL: TestGroupedStaticCacheGroupSizesAndCompleteScalarTuples/12 (0.09s)
    --- FAIL: TestGroupedStaticCacheGroupSizesAndCompleteScalarTuples/13 (0.08s)
    --- FAIL: TestGroupedStaticCacheGroupSizesAndCompleteScalarTuples/14 (0.09s)
    --- FAIL: TestGroupedStaticCacheGroupSizesAndCompleteScalarTuples/15 (0.09s)
    --- FAIL: TestGroupedStaticCacheGroupSizesAndCompleteScalarTuples/16 (0.11s)
    static_interface_group_cache_test.go:363: inner envelope seam no longer matches canonical storage <nil>
--- FAIL: TestGroupedStaticCacheInnerMetadataForgeryRepair (0.10s)
FAIL
FAIL	github.com/0disoft/fairyc/internal/workspacecompiler	4.973s
FAIL
```

A focused retry corrected the interface prefix and passed the complete inner
metadata rejection/repair test (package time 0.415s). Its combined `-run`
expression did **not** select the scalar subtest; this is not counted as scalar
validation. Exact executable identity:
`fairyc.compiler-executable-sha256/v1:3bc674189e832e3014d1b45e2aa211fb931b812e829b3995a4a23130c156b913`.

The explicit scalar retry used:

```powershell
go test -mod=readonly ./internal/workspacecompiler -run '^TestGroupedStaticCacheGroupSizesAndCompleteScalarTuples$/^2$' -count=1 -parallel=1 -p=1 -v
```

It retained the same equality fixture and exited 1, exposing the unsupported
features that the first attempt's diagnostic-only message omitted:

```text
=== RUN   TestGroupedStaticCacheGroupSizesAndCompleteScalarTuples
=== RUN   TestGroupedStaticCacheGroupSizesAndCompleteScalarTuples/2
    static_interface_group_cache_test.go:236: source library-sha256=244c15f35dca0f1064421721c00b3bd30d2defbbb62f5969de44432d51b2c762 main-sha256=914f9cf562a94f69df7985c84cc39194681b8acaed35dbd9c22bc692e987b4c7 test-executable=fairyc.compiler-executable-sha256/v1:3bc674189e832e3014d1b45e2aa211fb931b812e829b3995a4a23130c156b913
    static_interface_group_cache_test.go:238: cache: err=<nil> report={Outcome:not-eligible Storage:memory Packages:0 Hits:0 Misses:0 Rejected:0 Stored:0 StoreSkipped:0 StoreFailed:0 Lowered:false ReassemblyFallback:false} diagnostics=[] unsupported=[{Feature:equality comparison on i8 Span:{Package:1 Source:main.fy StartByte:143 EndByte:179}} {Feature:equality comparison on i16 Span:{Package:1 Source:main.fy StartByte:217 EndByte:254}} {Feature:equality comparison on isize Span:{Package:1 Source:main.fy StartByte:446 EndByte:485}} {Feature:equality comparison on u16 Span:{Package:1 Source:main.fy StartByte:595 EndByte:632}} {Feature:equality comparison on u64 Span:{Package:1 Source:main.fy StartByte:745 EndByte:782}} {Feature:equality comparison on f32 Span:{Package:1 Source:main.fy StartByte:901 EndByte:938}} {Feature:equality comparison on f64 Span:{Package:1 Source:main.fy StartByte:976 EndByte:1013}}]
--- FAIL: TestGroupedStaticCacheGroupSizesAndCompleteScalarTuples (0.07s)
    --- FAIL: TestGroupedStaticCacheGroupSizesAndCompleteScalarTuples/2 (0.07s)
FAIL
FAIL	github.com/0disoft/fairyc/internal/workspacecompiler	0.173s
FAIL

```

The final test changes these probes to return typed values instead of comparing
unsupported scalar types. It also reports `Semantic.Unsupported` on future
eligibility failures. The final complete targeted command above then passed.

First-attempt native executions also passed, with these distinct PE hashes:

- Low: `b8f681e03f9091f357ef844db136b64604d5d770e95c06d68f3a6ead25b4e235`.
- Optimized: `28cb5fd52135419d0eb202e7b45a89059e1b39421066c971297a344cd9cc5f71`.

## Boundaries and handoff

This is a focused two-package Go workspace cache proof. Native restored execution
covers Windows C11 low and optimized only. Linux, full CLI/cache suites,
self-build/fixed-point, distribution, hosted CI and the separate Fairy metadata
proof were not run by this worker. The existing admission/cache bounds are
unchanged. The size matrix's complete 13-scalar selection is memory-backed;
fresh-session/process disk proof uses the primary complete i32+i64 group.

No source-of-truth, API/ABI, DB, production code, dependency or runner change is
owned here. Both assigned files remain uncommitted for coordinator review and
integration. No live process/session or blocker remains.
