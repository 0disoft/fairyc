# Native state persistence integration v1

Status: experimental compiler-private implementation in sequential slices.
This implements the existing execution queue; it does not replace its full
native-state requirement with single-cell acceptance. Linux execution stays in
the maintainer's combined batch. No push, hosted dispatch or release is authorized.

## Required implementation sequence

1. Seal exact native input/output transport for the existing verified
   straight-line `cell<u64>` emitter. Preserve the old emission route and runtime
   ABI. Execute generated transport C under Windows low/optimized against the
   independent Go transaction oracle, including intermediate-limit rollback.
2. Build an opaque executable from that prepared owner. Execute the actual C
   once against one persistence snapshot, fully validate its response and submit
   at most one complete overlay through the existing layout/version
   compare-and-publish authority. Preserve rollback, causes, conflicts, no-op
   version semantics and caller-visible one-attempt execution. Add real local
   store/reopen, competing writer, failed publication and process-error evidence.
3. Expand native lowering/transport to the existing admitted multi-field, map,
   canonical-codec, graph, helper and read-borrow state MIR, with independent
   comparisons and actual native execution at each step. Retain explicit refusal
   before backend/storage I/O until each route is implemented.
4. Connect source/developer invocation and current artifact/cache authority;
   complete the full Windows suite, authored Linux batch, same-source compiler
   comparisons, self-build, distribution and authorized hosted acceptance.

Steps 1 and 2 are implemented for the original single `cell<u64>` Windows MSVC
family. Source/developer native invocation, broader state MIR and steps 3/4
remain open. The initial transport commit alone established only step 1.

Step 3 now has an independently exercised multi-key C overlay and all seven
canonical codec validators, plus actual verified literal multi-field/cell/map
MIR-to-C emission. [Overlay v1](native-state-overlay-v1.md) records the memory
policy, Windows evidence and remaining transport/CAS/graph integration. Those
prerequisites alone do not admit durable multi-key native state execution.

## Sealed transport owner

`internal/contractstatenative/persistent_transport.go` mints `PersistentArtifact`
only after the existing handoff/body/MIR/target gate. Its C source, field key,
read-operation order, permitted exit and identities stay private. A separate
domain/target-bound SHA-256 of the generated entry C distinguishes transactions
that share one package-body hash. Source is
available as an immutable Go string for compilation; input bytes are copied.
The initial route admits the same single `cell<u64>` and Windows target as
ADR 0382. The existing 1,024-operation verifier bound also bounds this transport.
Zero owners, unsupported targets and corrupt programs cannot create requests.

Each native attempt runs in a fresh process, so transport globals are local to
that attempt. It reuses `stage0_contract_state.c` for overlay reads, writes,
delete, commit and rollback. The original `Emit` route adds no hooks or framing;
only the explicit persistent route records all reads and checks per-operation
write policy. No interpreter or Go MIR execution supplies native result bytes.

## Private binary framing

All integers are unsigned big-endian. There is no native struct serialization,
padding dependency, implicit integer promotion, JSON conversion or line framing.
Windows inherited stdin/stdout are explicitly binary; NUL, CR/LF, 0x1a and 0xff
are ordinary bytes. The process consumes exactly one request and EOF, emits one
complete response, flushes it and exits. Malformed input exits 4 with no response.
Successful computation still exits 0 for reject/trap/limit response statuses;
the host must inspect the complete response before any storage operation.

The fixed 137-byte request is:

| Offset | Bytes | Meaning |
| --- | --- | --- |
| 0 | 8 | `FYSPIN01` |
| 8 | 32 | Exact prepared layout hash |
| 40 | 32 | Exact prepared package-body hash |
| 72 | 32 | Exact prepared native program hash |
| 104 | 8 | Original snapshot version, including UINT64_MAX |
| 112 | 8 | Positive maximum overlay write bytes |
| 120 | 8 | Positive maximum value bytes |
| 128 | 1 | Presence, exactly 0 or 1 |
| 129 | 8 | Canonical u64 bytes, all zero when absent |

The input factory also checks the snapshot layout, zero/one entries, exact cell
physical key, canonical value and all four positive execution limits. The
single-key route cannot exceed one distinct overlay write; it retains the
existing positive MaxWrites and MaxMapKeyBytes requirements.

The response has a 135-byte header followed by 13 bytes per observed read:

| Offset | Bytes | Meaning |
| --- | --- | --- |
| 0 | 8 | `FYSPOU01` |
| 8 | 104 | Exact request layout, body, native program and snapshot version |
| 112 | 1 | Commit 0, reject 1, trap 2, write-byte limit 3, value limit 4, runtime failure 5 |
| 113 | 1 | Committed dirty overlay, exactly 0 or 1 |
| 114 | 1 | Result presence, exactly 0 or 1 |
| 115 | 8 | Result u64 or zero absent padding |
| 123 | 4 | Number of observed reads |
| 127 | 8 | Zero cost, matching the existing legacy straight-line Go execution |
| 135+13N | 4+1+8 | Original operation index, presence, canonical read value/zero padding |

At most 1,024 reads produce a 13,447-byte response. Decode checks size and count
before allocation, exact identities/version, enums, flags, zero padding, cost,
read order and prefix, and full read coverage for completed exits. A rollback,
failure or clean commit must retain the original snapshot bytes. A successful
dirty commit supplies a proposed state, not evidence of durable publication.
The decoder copies values and reads; caller-owned buffers cannot mutate them.

## Intermediate write limits and rollback

The transaction kernel charges each current overlay entry as its 32-byte physical
key, one operation byte and value bytes. A u64 write therefore costs 41 bytes;
delete costs 33 bytes. Replacement updates the same entry rather than accumulating
all previous writes. Each native write checks the value limit before write-byte
accounting; delete checks only its 33-byte overlay. A failed intermediate write
cannot be hidden by a smaller final delete. The runtime discards its overlay on
failure, and the response retains the untouched input store. Read-only execution
can observe an existing eight-byte value even with a smaller write-value limit.

The echoed version is never incremented by native code. Version comparison and
advancement belong to the existing persistence port. CAS integration, receipt
validation and handling post-publication errors are required in step 2.

## Validation

`persistent_transport_test.go` owns independent wire construction, immutable
inputs/results, wrong hash/key/length/limits, zero owners, response corruption,
read order/count/padding, trailing/truncated bytes and 0/1,024/1,025 operations.
The first oversized fixture incorrectly requested 4,096 operations; the existing
verifier rejected it at 1,024. The fixture now tests the actual unchanged limit.

`persistent_transport_windows_test.go` compiles eleven transaction families under
both host profiles. Six use real Fairy source and `PrepareCompiled`; five add
verified HIR controls for read-only/empty, intermediate write/delete and reads
around mutations. Two initial-presence states and six boundary policies yield
240 native/Go comparisons; four maximum-read native executions cover all 1,024
records and the exact response ceiling. Twenty additional native requests check malformed
input. Original binary bytes and UINT64_MAX version must survive the protocol.
Trap reads use a separate Go reject-exit observer over the same operations;
actual trap rollback/category are compared independently. Native values/reads
are never synthesized by the Go oracle.

An independent two-owner/same-package regression initially reproduced acceptance
of another transaction's result because both owners shared layout/package-body
hashes and compatible read shapes. The per-entry native program hash now binds
both native request validation and Go response admission. This is identity
binding for a compiler-owned process, not cryptographic sender authentication.
The new frame is an unpublished v1 prototype; no released cache/SDK wire changes.

```powershell
go test -mod=readonly -count=1 -timeout 4m -failfast -v ./internal/contractstatenative -run '^TestPersistentTransport'
go test -mod=readonly -count=1 -timeout 4m ./internal/contractstate ./internal/contractstatemir
go test -mod=readonly -count=1 -timeout 3m -v ./internal/cstdlib -run 'ContractState'
go vet -mod=readonly ./internal/contractstatenative
git diff --check
```

The final prepared-owner replay passes in 74.629s without selected skips,
including 244 native/Go observations and twenty malformed native requests.
The initial ten-family prototype (52.751s) and its separate maximum-read probe
(16.572s) preceded the owner-binding correction; they are not added to the final
264-run count. The unchanged original native handoff suite passes in 6.865s.
Full transaction/persistence and state-MIR regressions pass in 0.369s/0.103s;
the unchanged runtime-owner structural check passes in 0.061s. Scoped vet,
formatting and diff checks pass. At that initial transport commit, native Linux,
actual durable C execution, wider state MIR and all final queue gates remained
pending. Public CLI, language,
SDK/runtime ABI, DB, dependencies, timeouts, runner configuration and repository
policy are unchanged.

## One-attempt native execution and CAS publication

`BuildPersistent` compiles only a sealed `PersistentArtifact` with the existing
contract-state runtime and returns an opaque `PersistentExecutable`. Compilation
is separate from execution and touches no state store. Caller-provided executable,
source or result bytes cannot construct a valid production execution owner.

`ExecutePersistent` checks the owner and positive execution limits before storage
I/O. It takes exactly one complete snapshot, runs the actual generated C once in
an owned process and validates the complete response against that snapshot and
prepared native program. No Go interpretation supplies the native result. A
successful commit submits at most one complete overlay to the original layout/
version compare-and-publish port. Read-only and empty commits still compare that
version with an empty overlay; their receipt preserves the original version and
reports zero writes/bytes. Dirty writes reject version exhaustion before CAS.

Reject, trap, intermediate limit, process failure and malformed response never
call publication. Trap retains the named Go trap category after native rollback.
Conflict/layout/backend errors retain their causes and never trigger reexecution
or publication retry. Cancellation is checked before snapshot, after snapshot,
after execution and before CAS. The context-aware Windows driver retains the
existing owned job, bounded pipes, timer cleanup and original execution error.
It can terminate a running child when its caller cancels or reaches a deadline.
The synchronous persistence port itself retains its existing cancellation contract.

After a successful CAS acknowledgement, cancellation does not change known
success into failure. An acknowledged receipt with inconsistent layout/version/
write count returns `PublicationReceiptError` with that receipt: state may already
be published and must be inspected rather than blindly retried. Local publication
retains ADR 0500's process-restart scope; power-loss durability is not established.

The new Windows tests compare commit/reject/trap, deletion, read-only and empty
execution against the independent Go runner with present/absent initial values
and both native profiles. Each store is closed and reopened, then both execution
paths run again: 64 paired native/Go results. Four additional real C attempts
share two stale snapshots; scheduled publication against the real OS-locked local
store yields one success and one explicit conflict per profile, with no retry.
Four real C executions exercise injected publication error and actual local value
capacity failure; two corrupt-image attempts exercise launch failure. Failed
publication, reject/trap and clean commits preserve durable bytes as applicable.
Controlled-port tests additionally cover malformed responses, admission/error
ordering, cancellation boundaries, acknowledgement inconsistency and no-op
versions 0, UINT64_MAX-1 and UINT64_MAX.

```powershell
go test -mod=readonly -count=1 -timeout 4m -failfast -v ./internal/contractstatenative
go test -mod=readonly -count=1 -timeout 3m -failfast -v ./internal/cdriver -run '^(TestWindowsTargetContextCancellation|TestWindowsExecutionBoundsInheritedOutput|TestWindowsExecutionDrainPreservesCompletedOutput|TestExecuteWindowsHostWithTimeoutBounds|TestExecuteWindowsMSVCTargetWithInputRejectsNilInput|TestWindowsTrapControlResultErrorPrecedence)$'
go test -mod=readonly -count=1 -timeout 4m ./internal/contractstate ./internal/contractstatemir
go vet -mod=readonly ./internal/cdriver ./internal/contractstatenative
```

The complete native-state package passes in 76.981s without selected skips,
including the existing 244 transport/Go comparisons and twenty malformed native
requests. The standalone new execution replay passes in 18.318s before the added
no-op UINT64_MAX boundary control; that control is included in the final complete
package run. Context/pipe/trap-precedence regression passes in 12.383s, transaction
and state-MIR regressions in 0.340s/0.071s, and the unchanged runtime structural
owner in 0.039s. Scoped vet, formatting and diff checks pass. Initial context-test
compilation rejected a code-free runtime translation unit at the existing COFF
inspection gate; the fixture now contains a real function, with no inspection or
warning relaxation.

Native Linux, multi-field/maps/codecs/graphs/helpers/borrows, source/native CLI,
artifact/cache integration and every final queue gate remain pending. No public
CLI/language/SDK/runtime ABI, dependency, DB/provider, default timeout, workflow/
runner or repository-policy change accompanies this private execution path.
