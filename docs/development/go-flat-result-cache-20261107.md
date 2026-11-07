# Go flat Result cache restoration, 2026-11-07

Status: **PASS**. The final focused package run completed in **10.305s**, including
four actual Windows MSVC C11 executions and a fresh test-executable process.
Execution completed on 2026-11-08 KST. This is a Go workspace-cache proof for
[ADR 0550](../adr/0550-flat-result-success-generic-transport.md), not full authored
Fairy CLI or current Fairy source/artifact acceptance.

Only these two new files belong to this sidecar:

- `internal/workspacecompiler/composite_result_cache_test.go`
- `docs/development/go-flat-result-cache-20261107.md`

All work ran inside the existing detached checkout `.cache/flat-result-20261107`,
baseline `bf719aa872375ac22706c20cb232db42efb28700`, with the Go owner's uncommitted
flat Result implementation present. The coordinator had already integrated that
Go implementation into main as `b4ac3390`. No main-root files, production code,
existing tests, limits, ADR, Fairy source/tests, queue or lockfile were written
by this sidecar. Other workers' changes in the checkout remain intact. No
staging, commit, push or deployment was performed.

## Cache architecture and current-source checks

Reuse `composite_option_cache_test.go`'s ordinary memory/disk configuration,
`genericPersistentConfiguration`'s exact current executable build identity,
`newGenericPersistentFixture`, workspace fixture loader, and exact canonical
disk-envelope helper. No cache/artifact/planning bound, capability, schema,
transport policy or production default is changed.

The provider declares six public generic helpers: pack, relay, error binding,
propagate, unpack and failure. A seventh ordinary function supplies the u32
error code. Relay copies and reassigns Results while explicitly handling every
pending evaluation. The consumer transports all seven success leaf types:
`i8`, `i16`, `isize`, `u16`, `u64`, `f32`, `f64`. Generic error binding specializes
`Result<T,E>` with E=u32 only.

`TestCompositeResultCacheMemoryAndDisk` proves:

- Cold memory/disk checks miss two packages, invoke actual lowering and store
  two artifacts. In-session hits and fresh disk-session hits restore both
  packages with no misses, rejection, store or lowering.
- Every claimed complete hit replaces the actual `GenericSession.lower` callback
  with an unconditional test failure. `Lowered:false` is an additional report
  assertion, not the sole evidence of bypassing lowering.
- Fresh source is loaded for every check. Six current public generic declaration
  owners and all generated MIR specializations are joined and verified. The
  error-only fixture has 28 specializations; cold/hit provenance must match.
  Both ordinary and Windows C emitters accept each restored MIR.
- A root source comment, provider relay body change, i8-to-i16 payload-choice
  edit and ordinary error-code change 77-to-78 each invalidate affected artifacts
  and lower/store fresh results. The provider generic edit misses both packages
  and changes specialization closure. The last edit changes execution 42 to 43.
- The payload-choice initializer's current source span joins to the concrete
  restored MIR callee returning `Result<i16,u32>`, including on a forbidden-lower
  warm hit; checking fresh HIR alone would be insufficient.
- A copied-but-unhandled Result is refused with **E4101**. Changing the enclosing
  error contract to i32 around a u32 `?` is refused with **E4103**. Neither invalid
  current source may return verified/executable MIR, provenance, cache hits or
  stores. Restoring the exact good source hits without lowering again.

These assertions preserve must-handle and the u32 error-family boundary. They do
not introduce extended scalar literals, arithmetic, wrapper nesting, composite
generic arguments or a new Result foreign ABI.

## Inner metadata rejection and healing

`TestCompositeResultCacheInnerMetadataForgeryRepair` starts with two actual disk
records. Before mutation, both the outer record and inner stable-artifact seam
must round-trip to their exact canonical bytes. Only the inner generic-interface
digest is replaced. Current source, inner MIR, key and outer public-interface
metadata stay intact. The forged outer record is rehashed and published under
its correct new SHA-256 filename, replacing the original generation.

The reader observes exactly **one rejected artifact and one hit**, invokes fresh
lowering, stores two repaired packages and removes the forged generation.
Execution returns 42. A new disk reader then restores both healed packages with
actual lowering forbidden. This exercises inner metadata admission, rather than
merely rejecting a malformed JSON envelope or a wrong filename checksum.

Final observed original interface:
`fygif0-sha256-f3957aa970b1c0c5af515c7ee0c9d85917abcd3d584fcb89cbca8d2113c9d089`.
Final forged outer SHA-256:
`db07b9125b7633865f544ca4486799d401f6e8bf458bcf8aca8f39c5a4f2d016`.
The outer digest is build-identity-specific and is an execution receipt, not a
new frozen protocol vector.

## Actual restored C and independent observer

`TestCompositeResultCacheNativeRestore` checks a memory hit and a fresh disk
session with actual lowering forbidden. Disk execution runs in a new process
of the exact same test executable; no parent memory artifacts or compiler result
are passed to it. That child must independently hit both packages with its
lowerer replaced before emitting/executing restored C. The native source has 35
current generic specializations.

Both the memory-restored result and fresh-process disk-restored result execute
under **low and optimized MSVC 14.51.36231 / C11**, four images total. All return
42 with empty stdout/stderr. There is no expensive authored compiler-host build.

The C observer authors its own expected scalar values and counters. Eleven
success vectors include signed minima, unsigned maxima, nonzero f32/f64 values,
positive zero and negative zero. Only scalar/repr(C) bridge values cross FFI;
the Result itself remains internal. Each image requires:

- **33 bytewise active-success matches** across original, copied/reassigned and
  propagated values. memcmp preserves signed-zero distinctions.
- Success-tag assertions on originals and copies. Successful propagation must
  select `.ok`; its `.err` arm asserts false instead of masking a bad tag with
  a fallback equal to the expected success value.
- **22 counted operand evaluations**, one successful and one failing path for
  each vector. Every path passes through generic propagation and the caller's
  `?`; duplicate operand evaluation changes the independent C counter.
- **22 seeds**, eleven initial inputs and eleven successful counted calls. The
  failed propagation path returns before producing a success payload.
- **11 u32 errors**, including 0, 77 and UINT32_MAX, extracted only from `.err`.
  The `.ok` arm asserts false even for expected error 0, so a wrong success tag
  cannot accidentally satisfy the error observer. No observer reads an inactive
  success/error member or assumes a public tagged layout.

## Executed checks and retained attempts

From the isolated checkout, with `GOMAXPROCS=1`:

```powershell
$env:GOMAXPROCS='1'
$env:FAIRYC_RESULT_CACHE_RECEIPTS=Join-Path (Get-Location).Path '.cache/result-cache-receipts/final'
go test -mod=readonly ./internal/workspacecompiler -run '^TestCompositeResultCache(MemoryAndDisk|NativeRestore|InnerMetadataForgeryRepair)$' -count=1 -v -timeout 2m
```

Final results:

| Group | Duration | Result |
| --- | ---: | --- |
| Memory + disk, edits and refusal | 1.98s | PASS |
| Memory + fresh-process disk restored native | 7.78s | PASS |
| Inner metadata forgery/healing | 0.36s | PASS |
| Complete package invocation | 10.305s | PASS |

The fresh child passed in 2.00s, including both native profiles. These are focused
tests, not a whole repository suite. The child-only test is invoked by the disk
parent, rather than executed independently without its required fixture paths.

Scoped final `GOMAXPROCS=1 go vet -mod=readonly ./internal/workspacecompiler`
passed with exit 0. The new Go file is gofmt-clean. Both assigned files are
nonempty UTF-8/LF with final newlines and below 40KB.

Retained under this checkout's `.cache/result-cache-receipts/`:

- `go-attempt1.log`: genuine initial harness failure, E4101 after a provider edit
  returned `copied` without handling the newly assigned `replaced` value. The
  metadata-forgery group already passed. Fixed only the edit fixture to handle
  the replaced Result before returning its copy; no production rule was relaxed.
- `go-attempt2.log`: corrected memory/disk invalidation/refusal pass, 1.704s.
- `native-attempt1.log`: first four native images passed, 4.467s. Retained those
  sources/images; this was followed by stronger explicit success/error tag
  assertions and the final combined run above.
- `final.log`: authoritative final pass, 10.305s. `final/workspace-*` retain the
  generated source workspaces; `final/native-*` retain the exact emitted C,
  observer/runtime support and low/optimized executable bytes.

No failed attempt was overwritten or counted as a pass.

## Exact source and native receipts

| Source/input | SHA-256 |
| --- | --- |
| Go generic admission implementation | `80b1ad61c1c806c2ebdb319832cc8e5c58a85d77243b3988aa216b803c816dd7` |
| Go tagged backend implementation | `0baa981376c2adb46d1860c545ccd447aa7bd567f603e97e665f381256c53171` |
| Reused Option cache test/harness | `0511a293004db7fb1cbf3129cd7a0a72eb7e51469a9f56f1ac96bd1b37e7daa6` |
| New Result cache test | `61f3406b17801111bd79b5f4bf86fd55b3cb11c84a99cc84faa2bd5895191f67` |
| Provider source | `2e1e711046489b9983aac0f8c6bafe7c5fd169c05b6dba6fe1db198831cd7ffc` |
| Error-only main | `5bf50c426e026d87807faee86c72409ec5f897ebe7a8ee814dcc5cff37a71559` |
| Final native main | `15f49f6d92e100444df665859180b5ca9724cf26c5702d17734f649e5ac0dbf8` |
| Independent C observer | `517789fa5b4d25932f3d7f84ac031db6bcd0c62c195b57b4900b5f75b60315e0` |

| Restored artifact | Generated C SHA-256 |
| --- | --- |
| Memory | `0122e1ec8818830410ea9e3aef24c0b2fa5aadfdcabc4bc0d4bdaf50e2572537` |
| Fresh-process disk | `47a74a99ca052596a7154e68a28c49596f3f07d5434a5ee0c136953de320fc28` |

| Image | SHA-256 | Bytes |
| --- | --- | ---: |
| Memory / low | `6f4310bc13f40ff4a0ee0748b192e7d311a48130e516c6a6335ab08b441192f6` | 154,112 |
| Memory / optimized | `ef56564faca494569ad93bb8e1038bbfb12ac8abcf43d803bee0dc4db939f93c` | 120,832 |
| Disk child / low | `56f9797374664accc8e856c8e0f292c1c44c5ba896d17a27625de0f64a2961cd` | 154,112 |
| Disk child / optimized | `8ce3f6f31b620672d075cfaae08129089c0b30c92471f7815ff1654110e3a791` | 120,832 |

Memory and disk use different workspace package identities, so their emitted C
and image hashes need not match. Both executions use the same independent C
expected vectors, rather than deriving expectations from the other execution.
Logs retain the source/provider edit hashes and specialization provenance hashes.

## Remaining gates and handoff

Native Linux stays batched. Fairy source/artifact tests, full authored CLI/cache,
complete current-source checks and hosted CI remain with their owners. No API,
ABI, database, dependency, runner or repository hygiene behavior is changed.
No source-of-truth document is modified by this sidecar. The two new files are
prepared for coordinator review/integration; this sidecar makes no commit.
