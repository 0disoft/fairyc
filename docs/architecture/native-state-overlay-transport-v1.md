# Native literal-state overlay transport v1

Status: sealed request/response admission, actual C framing, opaque Windows
executable and one-attempt literal multi-key CAS implemented. This is a delivery step of
[overlay v1](native-state-overlay-v1.md), not durable multi-key native acceptance.

## Authority and bounds

`PrepareOverlayTransport` calls `EmitOverlay` and retains its private owner,
exact layout/body/native-entry hashes, codec-bound operation-ordered reads and
complete sorted final literal-write plan. Layout-derived cell/map physical
addresses share the emitter's canonical helper; no test or Go execution result
can mint a production response owner. The original single-cell framing remains
unchanged and has different magic bytes.

`Input` validates the owner, layout and positive execution limits before
snapshot serialization. A complete immutable snapshot has at most 65,536 sorted
unique entries and 16MiB of encoded entry payload. Known literal addresses must
hold their exact canonical field value; other physical map addresses cannot be
inverted into a field, so their bytes are retained without guessing a schema.
The persistence port remains responsible for the trusted layout's full state.
Snapshot values can exceed the current write-value limit: read-only observation
must not acquire an unrelated write restriction. Both request and result bytes
are copied; caller mutation cannot change prepared admission or owned results.

The response ceiling is 64MiB, matching the explicit Windows binary-capture
ceiling. It is an I/O/resource bound, not a promise that every possible trace
fits or a replacement language transaction limit. Oversized evidence must never
be truncated into a valid result or published. Actual C framing must bound
allocation, lengths and EOF before storage authority is reachable; cleanup,
native limit failures and exact maximum response cases need their own evidence.

## Input frame

All numbers are unsigned big-endian. Limits are all 64-bit so positive Go `int`
limits and UINT64_MAX write bytes are preserved without clamping or truncation.
The fixed header is 152 bytes:

| Offset | Bytes | Meaning |
| --- | --- | --- |
| 0 | 8 | `FYOVIN01` |
| 8 | 32 | Exact layout hash |
| 40 | 32 | Exact package-body hash |
| 72 | 32 | Exact prepared native-entry hash |
| 104 | 8 | Original snapshot version |
| 112 | 8 | Positive maximum distinct writes |
| 120 | 8 | Positive maximum overlay write bytes |
| 128 | 8 | Positive maximum written value bytes |
| 136 | 8 | Positive maximum original encoded map-key bytes |
| 144 | 4 | Snapshot entry count |
| 148 | 4 | Complete entry payload bytes |

Each snapshot record is its 32-byte physical key, a four-byte value length and
exact value bytes. Empty values remain present records. Count/order/size limits
are checked before allocating the request payload; the total request is bounded
by 152 bytes plus 16MiB. Version is echoed unchanged, including UINT64_MAX;
only the existing CAS authority may advance it.

## Output frame

The fixed header is 142 bytes:

| Offset | Bytes | Meaning |
| --- | --- | --- |
| 0 | 8 | `FYOVOUT1` |
| 8 | 104 | Exact request layout/body/entry hashes and original version |
| 112 | 1 | C status: 0 success, 1 invalid, 2 closed, 3 value limit, 4 count limit, 5 byte limit, 6 codec, 7 encoded map-key limit |
| 113 | 1 | Completed exit: commit 0, reject 1, trap 2; failures retain 0 |
| 114 | 4 | Observed read count |
| 118 | 4 | Proposed write count |
| 122 | 8 | Complete proposed write-byte cost |
| 130 | 8 | Zero legacy cost; graph metering is not admitted |
| 138 | 4 | Complete read/write payload bytes |

A read is a four-byte original operation index, one-byte codec ID, one-byte
presence, four-byte value length and exact bytes. IDs 1..7 are bool/u32/u64/
bytes/UTF-8/bytes20/bytes32. Absence has no value bytes. A present empty byte
value is distinguished by its presence flag. A write is a 32-byte key, one-byte
deletion flag, four-byte value length and exact bytes; deletion has no value.
Reads precede the sorted unique writes. No padding, trailing bytes, native struct
layout, text conversion or implicit numeric coercion is allowed.

## Admission before publication

`Decode` checks the full response ceiling, header, exact request identities and
version, declared payload length, count bounds and minimum record sizes before
record allocation. Every completed exit has exactly the owner's read plan;
errors may retain only an operation-ordered prefix. Codec/index/presence and
canonical values are checked on each read. The zero legacy cost cannot fabricate
metering evidence.

Only a completed commit carries writes. Its count, ordered keys, deletion flags
and literal value bytes must exactly match the sealed final-write plan. The
decoder compares immutable declared constants but does not interpret MIR to
produce results: the admitted writes are copies of the actual received bytes.
The sum of 33 bytes plus each value length must match the reported cost and the
request's write count/byte/value policies. Reject/trap/error carry no writes or
write-byte cost. An error cannot claim a completed reject/trap exit.

Trap category comes from the exact selected entry, never arbitrary wire text.
Unknown statuses/codecs, reordered or duplicated observations/writes, changed
constant results, contradictory exits, short/trailing data, impossible lengths
and foreign-owner input/results fail closed. Native sender ownership remains a
private compiled-process assumption, not cryptographic authentication.

## Verification and remaining work

Independent test frames are assembled from the Go transaction oracle with
explicit offsets/codec IDs, not the production request or decoder. Tests cover
three completed exits, forty-five boundary/corruption controls including seven
write-free failure statuses, 0/one/exact 65,536/65,537 entries, exact/one-over
16MiB snapshot bytes, UINT64_MAX versions/write policy, full-width `int` limits,
canonical bool failures and buffer ownership. The wire constructor initially
failed Go compilation by slicing a returned hash array; it now assigns the
array before slicing, without changing the wire or production behavior.

```powershell
go test -mod=readonly -count=1 -timeout 3m -failfast -v ./internal/contractstatenative -run '^TestOverlayTransport'
go test -mod=readonly -count=1 -timeout 3m -failfast -v ./internal/contractstatenative -run '^(TestOverlayTransport|TestOverlayMIR|TestNativeOverlayRuntime|TestPersistentExecution)'
go vet -mod=readonly ./internal/contractstatenative ./internal/cdriver
```

The final 64-bit-limit wire replay passes in 0.094s. The preceding combined
native-MIR/overlay/single-cell-CAS replay passes in 27.181s; its native path and
shared address-helper inputs are unchanged by the final Go wire-width update.
The subsequent C framing implementation and evidence are recorded below. Real
multi-key local publication/reopen/conflicts, graph/helper/borrow and
source/cache/native CLI remain open. Linux execution, the complete final suite, distribution and hosted
acceptance remain deferred. No public language/CLI/SDK/runtime ABI, DB/provider,
dependency, default limit, workflow/runner or repository-policy change is made.

## Actual C framing and built execution

`OverlayTransport.Source` emits the original verified operation C plus a bound
binary entry. `BuildOverlay` returns an opaque Windows image; `RunOverlay`
executes that image once with binary stdin, a one-minute deadline and explicit
64MiB combined capture, then admits only a zero-exit, empty-stderr, trap-free
complete response through the existing decoder. It grants no storage authority.

The native entry checks request magic/identity, full-width policies, snapshot
counts/lengths, sorted unique keys, known canonical codecs and exact EOF.
Payload, snapshot, writes, two work arenas and trace storage have bounded owned
lifetimes and one cleanup path. The transaction is initialized before that
cleanup path, preserving strict MSVC warning checks on early allocation errors.
The sealed literal plan bounds workspace by the sum of 33 plus the largest
literal per written key, capped at 64MiB. Read trace storage is bounded by the
actual read count and largest observed/literal value, capped at the response
ceiling. A response that cannot fit fails without any writes. Partial stream
output or process failure cannot satisfy host admission and cannot be published.

Windows low/optimized evidence includes all four exit families and seven
policies (56 native/Go comparisons), sixteen malformed requests per profile,
all six allocation failures and successful cleanup, and a 65,537-byte binary
read with capture limits 65,688/65,689/65,690. A separate six-read fixture proves
responses of exactly 64MiB-1 and 64MiB; one additional byte returns a bounded
failure prefix with no writes. Every returned binary observation is checked.

```powershell
go test -mod=readonly -count=1 -timeout 4m -failfast -v ./internal/contractstatenative -run '^(TestOverlayFraming|TestOverlayTransport|TestOverlayMIR|TestNativeOverlayRuntime|TestPersistentExecution)'
go test -mod=readonly -count=1 -timeout 4m -failfast -v ./internal/contractstatenative -run '^TestOverlayFramingResponseCeiling$'
go vet -mod=readonly ./internal/contractstatenative ./internal/cdriver
git diff --check
```

The combined replay passes in 71.168s before the ceiling test is added; that
test alone passes in 2.899s with unchanged implementation inputs. No selected
skips. Scoped vet, formatting and diff checks pass. The first strict build
reported possibly uninitialized transaction cleanup; explicit initialization
fixes it without disabling warnings. This closes native framing only. Real
multi-key CAS/reopen, graph/helper/borrow, source/cache/commands and every final
acceptance gate remain required; native Linux remains batched.

## One-attempt multi-key persistence

`ExecuteOverlayPersistent` accepts only an opaque built owner. It checks context,
preparation and positive policies before storage access, takes one complete
snapshot, executes the actual C image once, validates the complete response and
submits at most one sorted overlay through the existing layout/version CAS.
It never evaluates Go MIR to obtain native output and never retries a conflict.
The process decoder is shared with `RunOverlay`, including exit/stderr/trap
channel refusal and exact response admission.

Only a completed commit can reach CAS. Reject returns its read trace; trap
rethrows the original typed category. All seven runtime failure statuses,
malformed responses, process/limit/storage errors and pre-CAS cancellation publish
nothing. Native value/count/byte/map-key failures retain the original Go error
identities. A clean commit still compares its version with an empty overlay;
it preserves the version, including UINT64_MAX. Nonempty writes at UINT64_MAX
reject before CAS. Intermediate deletes and writes retain their operation
policies even when the final values are absent or unchanged.

Success validates the acknowledged layout, next version and write count. A
cancel arriving during successful publication does not convert it into failure.
An inconsistent acknowledged receipt uses the existing `PublicationReceiptError`
and returns the commit outcome/receipt for inspection without a retry. Storage
and process failures preserve causes and simultaneous cancellation. Local storage
retains its existing restart-visible guarantee, without a power-loss claim.

Windows low/optimized exercise commit/reject/trap/empty against actual local
stores, absent and fully populated initial states, close/reopen and repeated
execution (32 independent native/Go comparisons). Initial state also contains an
unmentioned second map key: every commit/rollback/reopen must retain it. Separate
real-native tests prove one winner/one conflict from two concurrent stale writers
per profile, four rollback policies, backend/capacity/invalid-image failures,
snapshot cancellation and successful-but-canceled publication surviving reopen.
A 65,540-byte binary read-only commit succeeds with a one-byte write limit,
preserves exact durable bytes/version and reopens unchanged. Controlled ports
cover complete process/protocol failures, cancellation phases, acknowledged
receipt inconsistencies, all failure statuses and empty CAS at version boundaries.

```powershell
go test -mod=readonly -count=1 -timeout 4m -failfast -v ./internal/contractstatenative -run '^TestOverlayPersistence'
go test -mod=readonly -count=1 -timeout 4m -failfast -v ./internal/contractstatenative -run '^(TestOverlayFraming|TestOverlayPersistenceNativeLocalStore)'
go vet -mod=readonly ./internal/contractstatenative ./internal/cdriver ./internal/contractstate ./internal/contractstatemir
git diff --check
```

The first complete persistence replay passes in 14.559s without selected skips.
After adding the unrelated-map initial-state control, its changed local-store
root and shared framing regression pass together in 34.308s. Other persistence
roots retain unchanged inputs. State and state-MIR regressions pass in
0.272s/0.069s; scoped vet, formatting and diff checks pass. Dynamic
state values/keys, graphs/helpers/borrows, source/developer commands, artifact/
cache authority, native Linux and all full/self-build/distribution/hosted gates
remain open. No stable CLI/language/SDK ABI, DB format/provider, dependency,
default limit, workflow/runner or repository-policy change.
