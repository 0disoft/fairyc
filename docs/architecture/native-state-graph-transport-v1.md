# Native state graph transport admission v1

Status: private Go request/path-witness/response admission, actual C framing and
opaque Windows build/run implemented. One-attempt graph persistence is implemented
in [graph persistence v1](native-state-graph-persistence-v1.md), completing step 3 of
[the complete graph plan](native-state-graph-v1.md). This is not an alternate
completion target. Native Linux remains batched; no push or hosted dispatch.

The Fairy counterpart has Complete-bound audited entry/framed-C preparation
and immutable request encoding under ADRs 0530–0532. Its complete C/hash/request
bytes match Go and execute on Windows. ADR 0533 adds the sealed Fairy response,
CFG/call-return and snapshot/overlay admission pipeline, with independent
Windows native evidence in `../development/candidate-state-graph-response-20261107.md`.
Native orchestration and persistent CAS remain separate mandatory steps.

## Authority and input

`PrepareGraphTransport` selects the verified graph emitter, seals copied site
descriptors and keeps the existing literal transport separate. The program hash
uses a graph-transport-specific domain over the exact audited graph hash,
native work/literal bounds and C framing implementation. The graph hash already
binds layout, package body, target, C and the sorted trap table.

Input uses the existing 152-byte big-endian snapshot framing with separate
`FYGRIN01` magic and the graph transport program hash. Four positive limits
retain all 64 bits. Snapshots retain the 65,536-entry and 16MiB encoded-payload
ceilings, sorted unique physical keys, and canonical known cell values. Values
larger than the write-value policy may still be observed. Request bytes are
copied before return; callers receive copies rather than mutable owner storage.

The whole request SHA-256 binds its limits and complete snapshot, including
otherwise identical layout/body/program/version headers. A different snapshot
at the same version cannot consume the old response. These hashes bind private
compiled-process evidence; they do not authenticate an arbitrary executable.

## Response

Numbers are big-endian; the fixed header is 186 bytes:

| Offset | Bytes | Meaning |
| --- | --- | --- |
| 0 | 8 | `FYGROUT1` |
| 8 | 104 | Exact request layout/body/program hashes and original version |
| 112 | 32 | SHA-256 of the complete request |
| 144 | 1 | Existing overlay/graph status, 0 through 11 |
| 145 | 1 | Completed exit: commit 0, reject 1, trap 2; errors retain 0 |
| 146 | 4 | Exact sealed trap-table ID, zero outside a completed trap |
| 150 | 8 | Complete execution cost |
| 158 | 8 | Complete successful operation counter |
| 166 | 4 | Successful path-event count |
| 170 | 4 | Complete sorted final-write count |
| 174 | 8 | Complete final write-byte cost |
| 182 | 4 | Complete event/write payload size |

An event has an eight-byte post-charge operation counter, four-byte static site,
four-byte original encoded map-key length, four-byte value length, one-byte
presence flag, then exact key and value bytes. Cells carry no encoded map key.
Reads carry the observed bytes; absent ordinary reads carry no bytes, while
absent borrow observations preserve the Go oracle's canonical zero encoding.
Writes carry their actual bytes and presence 1; deletes have no value and flag 0.
Calls, borrow view/copy and terminators carry no key/value/flag payload.

Ordinary operation sites retain the graph emitter's one-based static numbering.
Terminators occupy a separate high-bit namespace in body/block emission order.
Each legacy body has a synthetic zero-cost exit event. Graph operations cost 1,
legacy storage operations cost 0, calls cost 1, and graph terminators cost 1;
operations advance the separate counter, terminators do not. These are the
existing verified submodel's cost rules, not a new gas schedule.

Final writes follow events and use the existing 37-byte physical-key/deletion/
length framing plus value bytes. No padding, native struct layout or trailing
data is accepted. The complete response ceiling remains 64MiB.

## Admission

The decoder walks only the sealed CFG/call-return transitions, including either
admitted branch successor, explicit helper return continuations and terminal
commit/reject/trap sites. It recomputes cost and operation indices from received
sites. A successful response must end at its declared exit with exact counters
and trap ID. A failed observer may lose exactly the next already-charged event;
an error otherwise retains only its fully checked prefix. No failed response can
claim a completed exit or carry final writes/write-byte cost.

Map events preserve canonical original key bytes and their byte policy before
deriving physical addresses. Literal operands must match their declared bytes;
dynamic operands retain the private compiled-owner assumption. Admission does
not interpret SSA values, recompute branch conditions or run MIR to manufacture
an output. Reads must match the request snapshot or the preceding received
overlay. Returned read/write bytes are copied from the admitted response.

Every received mutation checks canonical value, original key, distinct-write,
value-byte and current overlay-byte policies. Replacements update the per-key
cost; a later delete cannot hide an earlier oversized write. Completed commit
writes must exactly equal the event-derived final overlay, sorted and unique,
with exact byte cost. Reject/trap/error grant no publication authority. Storage
CAS is deliberately not reachable from this admission-only API.

## Windows validation and remaining work

Authored frames use explicit offsets, event/site IDs and independent address
hashing rather than the production planner/decoder or a C record writer. Tests
cover direct/helper commit/reject/trap, exact helper cost/index order, alternate
graph branches, dynamic map operands, same-version snapshot replacement, read
consistency, replacement/delete accounting, intermediate policy failure, full
64-bit input limits, request/result ownership, all eleven error statuses,
thirty corruption controls and every truncated prefix of a valid response.
Valid 64MiB-minus-one and exact-64MiB responses are accepted; one extra byte is
refused before evidence allocation. A read larger than the write-value limit
is exercised without weakening that limit.

```powershell
go test -mod=readonly -count=1 -timeout 3m ./internal/contractstatenative -run '^(TestGraphTransport|TestOverlayTransport|TestNativeStateGraph(SlotBoundary|HashBindsTrap))'
go vet -mod=readonly ./internal/contractstatenative
git diff --check
```

The final selected Go replay passes in 0.408s and scoped vet passes. The first
new branch fixture incorrectly referenced a predecessor-local SSA value;
the unchanged verifier rejected it. Explicit typed edge arguments correct that
fixture; no verifier rule or native admission is relaxed. These tests run on
Windows but do not execute C framing. Actual C framing/allocation/EOF cleanup,
opaque build/run, graph CAS/reopen/conflict/cancellation, source/artifact/cache/
CLI and all final Linux/compiler/self-build/distribution/hosted gates remain
required at that checkpoint. Public CLI/API, SDK/runtime ABI, DB/provider, dependencies, default
limits, CI/runners and repository policy are unchanged.

## Actual C framing and built owner, 2026-11-07

The explicit audited emitter preserves the original `EmitGraph` read-observer
ABI and adds a separately selected operation/terminator witness callback.
Successful operations charge their original counters before observation; calls
record before entering the helper, and graph terminators record before taking
their selected edge/exit. Storage events retain original map operands and the
actual immutable read/write bytes. The full emitted source participates in the
transport identity; ordinary graph probes keep their separate entry.

`GraphTransport.Source`, `BuildGraph` and `RunGraph` provide actual binary stdin/
stdout framing and an opaque image/decoder owner. The host accepts no command
arguments, performs exact identity/count/length/order/canonical-cell/EOF checks,
hashes header and payload through the private allocation-free SHA-256 runtime,
and emits no response for malformed input or allocation failure. Every allocated
snapshot, workspace, epoch/value arena and trace follows one cleanup path.
Nil contexts reject, cancellation causes remain inspectable, and the existing
one-minute process and explicit 64MiB capture ceilings are retained.

Native scratch uses a conservative sum of all static blocks and transitive
acyclic helper invocations, including mutually exclusive paths. The bound
saturates at the transport ceiling and combines with the observed snapshot and
literal sizes. It never substitutes the write-value policy for the maximum
readable size. Overlay count/bytes and value/trace arenas remain bounded;
exhausting a transport cap below the caller's policy is a framing/resource
failure rather than a fabricated transaction-policy failure. No threshold or
warning level is relaxed.

Windows low/optimized execute the original nineteen v0-v4 fixtures and fourteen
seven-codec dynamic graphs through the real framing and decoder: 66 native
semantic comparisons against the existing independent Go transaction oracle.
Twelve additional direct/helper commit/reject/trap executions match separately
authored response bytes exactly. Sixteen injected allocation failures and the
successful cleanup controls observe zero outstanding allocations at exit.
Malformed header/identity/policy/count/EOF/snapshot/codec/duplicate controls
cannot publish bytes. Seeded dynamic-map snapshots preserve exact full-width
limits and current request hashes. Binary reads cover present-empty and
65,535/65,536/65,537 bytes under a one-byte write-value policy.

Actual C responses at 64MiB-minus-one and exactly 64MiB pass in both profiles.
A one-byte larger complete witness fails with the checked read prefix and no
completed exit, writes or byte cost; it is not truncated into success. The new
two-span SHA-256 entry also matches independent Go digests at SHA padding/block
boundaries, every selected split and three output-alias modes, preserving output
on null/length-overflow rejection. The existing runtime's 4,698 key comparisons
and value/borrow observations pass after adding this private digest symbol.

```powershell
go test -mod=readonly -count=1 -timeout 5m -failfast -v ./internal/contractstatenative -run '^(TestGraphFraming|TestGraphTransport|TestOverlayTransport)'
go test -mod=readonly -count=1 -timeout 3m -failfast -v ./internal/contractstatenative -run '^(TestNativeStateGraph(Digest|Runtime)|TestGraphFramingSnapshot)$'
go vet -mod=readonly ./internal/contractstatenative ./internal/cstdlib
git diff --check
```

One-attempt graph CAS/reopen/conflict/cancellation, source/developer/artifact/
cache/CLI and the complete current-source/self-build/distribution/hosted/Linux
acceptance remain required. This adds only a compiler-private native digest and
transport; public CLI/API, SDK/runtime ABI, DB/provider, dependencies, default
budgets/timeouts, CI/runners and repository policy stay unchanged. No push,
dispatch, release or deployment.

The final framing/wire/literal-transport replay passes in 72.864s with no
selected skips (`.cache/native-graph-framing-final-20261107.log`). Digest,
snapshot-cancellation and runtime controls pass separately in 6.940s; a final
unchanged-runtime/slot/hash replay passes in 2.108s. Eight original read-observer
ABI executions (linear, transitive helper, nested cell borrow and unrelated-map
borrow under both profiles) pass in 4.584s after the audited emitter was added.
Scoped vet, formatting and diff checks pass. These results cover the selected
native routes, not the full repository or outstanding persistence/final gates.
