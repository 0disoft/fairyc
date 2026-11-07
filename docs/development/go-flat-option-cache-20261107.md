# Go flat Option workspace cache execution, 2026-11-07

Source base: `main` at `2424fcd5`, including Go transport `4e9d5ad8` and Fairy
source/pipeline integration `2424fcd5`. This worker added only
`internal/workspacecompiler/composite_option_cache_test.go` and this record.
No production cache change, policy/schema/version/limit change, commit or push.
Contract: [ADR 0547](../adr/0547-flat-option-generic-transport.md).
Earlier source-to-C proof: [Go flat Option execution](go-flat-option-transport-20261107.md).
Validated test-file SHA256:
`0511a293004db7fb1cbf3129cd7a0a72eb7e51469a9f56f1ac96bd1b37e7daa6`.

## Final command and result

```powershell
$env:GOMAXPROCS='1'
go test -mod=readonly -p=1 ./internal/workspacecompiler -run '^TestCompositeOptionCache' -count=1 -v
```

**PASS, 3.604s package time.** Test times:

The coordinator separately checks the recorded test-file SHA-256 and runs
`go vet -mod=readonly ./internal/workspacecompiler` successfully before commit.

- `TestCompositeOptionCacheMemoryAndDisk`: 0.49s, including memory 0.18s and
  disk 0.31s.
- `TestCompositeOptionCacheNativeRestore`: 2.80s, including memory native proof
  and a real fresh OS child process using persistent artifacts.
- `TestCompositeOptionCacheCorruptArtifactRepair`: 0.21s.
- `TestCompositeOptionCacheChild` passed inside the child process (1.22s).
  Its top-level parent invocation skips intentionally because the child-only
  environment is absent; that skip is not a skipped disk/native proof.

No `t.Parallel` is used. Native consumers build serially with at most one CL
process from this worker. The existing native helper's deadlines and the current
test context are reused; no new timeout or cache limit is introduced.

## Cache fixture and current-source authority

The fixture uses the existing two-package workspace writer and
`genericPersistentConfiguration`. The public dependency owns five scalar generic
helpers: pack, copy/reassign/return relay, unpack, empty and tag selection. The
root specializes all seven extended scalars: i8, i16, isize, u16, u64, f32, f64.

The structural fixture transports None values through public Option helpers for
all seven scalars. It checks both original and copied tags and executes restored
MIR in the interpreter, returning 42. This is None execution, not a claim about
interpreter construction or arithmetic on all seven scalar families.

Every `Check` reloads the workspace from current source. Both cold and warm
results verify MIR, emit both existing C profiles, preserve every `Option<S>`
signature, and join current public source declarations/specializations to actual
MIR functions. Cold and warm provenance must agree. Memory warm reuse and the
new-session persistent reader replace `session.lower` with a fatal callback:
complete hits cannot silently perform ordinary lowering.

| Operation, in both memory and disk tests | Observed cache report |
| --- | --- |
| Cold two-package source | hits 0, misses 2, stores 2, lowered true |
| Unchanged current source, complete hit | hits 2, misses/rejections/stores 0, lowered false |
| Root source comment edit | hits 1, misses 1, stores 2, lowered true |
| Public relay helper body edit | hits 0, misses 2, stores 2, lowered true; specialization identities change |
| Root payload choice i8 to i16 | hits 1, misses 1, stores 2, lowered true |
| Warm source after payload edit | complete two-package hit; lowering forbidden |
| Invalid current helper body | unverified, not eligible, hits/stores 0, no partial generic declarations |

The payload edit is also checked against the current initializer source span:
the concrete callee selected in restored MIR returns `Option<i16>` instead of
the original `Option<i8>`. This prevents a report-only invalidation assertion
from overlooking a stale selected type. The memory report marks reassembly
fallback on the partially changed root/payload graph; persistent reports do not.
Both correctly lower and publish current source, without claiming a new
optimization or defect.

During test authoring, a test-only assertion incorrectly looked for an inert
Option local in resource metadata, then expected an explicit type on its Assign
operand. Those assertions were corrected to inspect the source-linked concrete
call result: inert Option locals are not resources, and the call's selected
callee supplies its result type. No production behavior was changed to satisfy
that assertion. The final command above passes the corrected checks.

## Actual restored payload execution

A separate two-package source fixture uses the same public Option helpers and
independently authored repr(C) seed/observer records. Its eleven values cover
i8/i16/isize minima, u16/u64 maxima, f32 1.25/+0/-0, and f64 -3.5/+0/-0.
Each value crosses pack, relay copy/reassignment/return, and unpack from both
original and copied Option values. Both Some tags are asserted before unpack,
so its fallback cannot mask an incorrectly constructed None. Empty/None tags
are checked separately for every scalar specialization.

C compares active scalar bytes with `memcmp` against independently seeded
expected values. Aggregate padding is not compared. Each executed product must
perform exactly 22 active-payload observations, return 42 and emit no output.
This admits no new scalar literal/arithmetic or Option-by-value FFI behavior.

| Restored MIR source | MSVC 14.51.36231 C11 consumer | Image bytes | Actual result |
| --- | --- | --- | --- |
| Memory complete hit, lowering forbidden | low | 127,488 | exit 42; empty stdout/stderr; 22 byte matches |
| Memory complete hit, lowering forbidden | optimized | 112,640 | exit 42; empty stdout/stderr; 22 byte matches |
| New-process disk complete hit, lowering forbidden | low | 127,488 | exit 42; empty stdout/stderr; 22 byte matches |
| New-process disk complete hit, lowering forbidden | optimized | 112,640 | exit 42; empty stdout/stderr; 22 byte matches |

For disk, the writer closes, a new session checks complete restoration, and that
reader closes before the OS child starts. The child runs the same exact test
executable in a fresh address space, derives its exact build identity using the
existing helper, reloads source, requires two persistent hits with no lowering,
then builds and executes the low/optimized consumers above. Its completion
marker is `composite-option-new-process-disk-42`. This is actual cross-process
restoration, not a warm in-process object retained by the writer.

## Corrupt stable artifact repair

An isolated structural fixture first writes exactly two disk records and closes
its writer. The test verifies its typed disk-envelope mirror matches the actual
canonical bytes, replaces one inner stable-artifact payload with corrupt bytes,
then publishes the damaged generation under its correctly recomputed SHA256
filename. The outer record remains canonical with a valid checksum. This tests
the inner artifact admission rather than only truncated JSON/file corruption.

A new session observes **one valid hit and one rejected artifact**, lowers from
current source, stores both repaired packages and executes None MIR returning
42. The rejected generation is removed. After that session closes, a fresh
reader requires two complete hits with a fatal lowerer and executes restored
None MIR again. Observed repair report:

```text
Outcome=fallback Storage=disk Packages=2 Hits=1 Misses=0 Rejected=1
Stored=2 StoreSkipped=0 StoreFailed=0 Lowered=true ReassemblyFallback=false
```

No concrete production cache defect was found in these cases.

## Remaining scope

This proof covers Go workspace GenericSession memory/persistent MIR caches and
current-source restoration. Parent owns the separate Fairy full CLI/cache work,
shared queue/docs, exact integration evidence and commits. These tests add no
Fairy or CLI acceptance claims. Native Linux, C17 for this cache fixture, broad
native/self-build/full-suite checks and hosted CI were not run because this
slice is limited to the Go workspace cache and bounded Windows C11 execution.
