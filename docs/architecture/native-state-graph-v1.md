# Native state graph values and execution v1

Status: private allocation-free value/key/borrow runtime and verified operation
C emission implemented. Go graph request/path-witness/response admission is now
implemented in [transport v1](native-state-graph-transport-v1.md); actual C framing
and graph CAS remain required. This is a
step of [native persistence](native-state-persistence-v1.md), not a replacement
for the complete implementation queue. Windows is verified below; native Linux
remains batched. No push, hosted dispatch, release or deployment is authorized.

## Required sequence

1. Provide stable typed values, canonical dynamic map-key derivation and checked
   read-borrow lifetimes over the actual multi-key C overlay. Compare its native
   behavior with independent Go storage/codec/hash expectations.
2. Lower the already verified acyclic state graphs, typed edge arguments, exact
   operation/cost accounting, helper calls and borrow operations into actual C.
   Require branch/join/rollback/trap/borrow values and traces against Go MIR.
3. Bind graph output to its complete program/request identity, admit dynamic
   observations/writes and connect the same one-attempt storage CAS. Preserve
   original local-store/reopen/conflict/cancellation guarantees.
4. Complete source/developer invocation, artifact/cache authority and the full
   current-source/compiler/self-build/distribution/authorized-CI acceptance.

Steps 1 and 2 are implemented below for the verified native admission. The existing literal overlay native/CAS route
retains its separate sealed admission and does not silently accept graphs.

## Caller-owned value and mutation storage

`ContractStateGraphDeclarations` and `ContractStateGraphRuntimeSource` explicitly
select `stage0_contract_state_graph.h`, its value/borrow implementation and the
private map-key implementation, together with the existing overlay kernel.
Nothing adds this runtime to an ordinary program or extends stable source/SDK.

One graph context belongs to one open transaction and is initialized once.
Its value arena and sorted mutation-generation entries are caller-owned,
bounded and disjoint from snapshot/overlay/context storage. All mutations must
pass through that graph owner. No malloc, file, socket, clock or global mutable
state is hidden in the kernel. The future built transport owns allocation and
cleanup; this runtime alone grants no publication authority.

An SSA read stores an immutable copy instead of a pointer into an overlay arena
that will be repacked on a later write. Slot/edge assignments may share those
immutable bytes. Missing reads produce exact canonical zero values for all seven
codecs, with absence kept separately. Present empty values remain present.
Value-space exhaustion rejects before advancing the arena or changing output.
Read values can exceed the transaction's write-value limit; typed writes still
enforce it. Only a successful write advances its physical key's mutation epoch.
Value/count/byte/codec failures leave epochs and prior values unchanged.

Epoch lookup is binary search over distinct written keys; insertion remains
bounded linear movement. Values use space proportional to retained copied
bytes. The caller's arena bounds it; no unbounded collection or allocation is
introduced. Step accounting separately guards operation/cost overflow and never
partially increments one counter when the other cannot advance. The generated
graph emitter still needs to supply the actual canonical steps.

## Dynamic map addresses

`fycs_graph_map_key_v1` validates the key codec and positive byte policy, then
streams SHA-256 over the exact existing domain, namespace prefix, unsigned
eight-byte length and canonical encoded key. This preserves the Go physical
address, without an input-sized concatenation allocation. Hashing uses the same
schedule/constants as the project-owned Fairy SHA modules and constant-sized
scratch. All input is consumed before writing output, including prefix/key
aliases. Invalid codecs, encoded lengths above policy, SIZE_MAX/bit-length
overflow and null required owners fail before output publication. This key hash
is deterministic addressing, not publisher authentication or a cryptographic
authorization boundary.

## Read borrows

Borrowed bytes are stable copies with one context, exact physical key and
observed mutation generation. Unrelated writes cannot corrupt them. Exact-key
replacement/deletion invalidates old handles, even if the replacement bytes are
equal. A missing borrow has no live handle. Cross-context and closed-owner
access reject. Nested ranges retain the original owner/epoch and validate by
subtraction before pointer movement, including SIZE_MAX and exact-end cases.

Only a checked borrow copy creates a value that survives a later mutation.
A UTF-8/fixed-width byte view can be noncanonical for its original codec; copying
the view preserves that codec and bytes, matching Go. The later typed key/write
is the canonicality boundary. Zero-length views at the end and present empty
borrows retain their distinct lifetime behavior. Source MIR admission remains
unchanged, including its block-local borrow and nonempty-view restrictions.

## Windows evidence

`TestNativeStateGraphRuntime` compiles the actual selected runtime with strict
Windows target flags under MSVC low/optimized. Each profile compares 2,349 key
cases against independent Go canonical validation and standard-library SHA-256:
all seven codecs, binary/UTF-8 failures, SHA block/padding boundaries, 0..130,
255/256/257, 4095/4096/4097, 65535/65536 and 1MiB key bytes, one-short/exact/
one-over policies and three output-alias modes.

The native value/borrow observation stream matches a separate Go transaction:
saved values across repacking, unrelated-key mutation, nested view/copy,
exact-key invalidation, deletion, typed absent zeros and copied-value reuse.
Additional C assertions cover bounded arenas, failure atomicity, cross-context
handles, empty/present/missing borrows, noncanonical UTF-8 views, overflow and
closed-owner behavior. These are actual kernel calls, not a generated graph.

```powershell
go test -mod=readonly -count=1 -timeout 4m -failfast -v ./internal/contractstatenative -run '^(TestNativeStateGraphRuntime|TestNativeOverlayRuntime)$'
go vet -mod=readonly ./internal/contractstatenative ./internal/cstdlib
git diff --check
```

The final combined runtime replay passes in 3.150s without selected skips,
including the existing 9,556 codec decisions and 26 overlay/Go cases. The first
graph probe exceeded the existing ordinary 64KiB capture default; the harness now
uses the already available explicit bound of expected bytes plus one. No product
limit or warning is relaxed. Full cstdlib tests were launched earlier, but their
process handle was unavailable on continuation and completion is unobserved;
they are not counted as a pass. The focused native replay is the runtime evidence.

Subsequent C emission/execution evidence is recorded below. Graph framing/CAS,
source/cache/CLI, Linux and all final gates remain open. No public language/CLI/
SDK ABI, DB/provider, dependencies, default limits or CI/runner policy changes.

## Verified state MIR to C

`EmitGraph` re-verifies its prepared exact-body owner before lowering. It emits
the current acyclic branch/jump graph, explicit typed join arguments, dynamic
cell/map read/write/delete operands, block-confined borrow/view/copy operations
and existing static void helpers through the actual graph runtime. Legacy and
graph bodies share the helper overlay, original operation indices and exact
cost rules. The ordinary HIR helper-call admission stays unchanged: graph/helper
combinations that its existing owner closes are not newly admitted by C code.

Each body owns local SSA/borrow slots. Verified global uniqueness and absence
of backedges let a predecessor assign fresh target parameters directly without
overwriting an input. Immutable value bytes stay in the caller's arena.
Helper-scoped arena bytes are released at every return/failure; helpers return
no values, and overlay writes own their copies. Successful root exits seal the
overlay once; reject/trap/runtime failure clear it. Trap codes bind a canonical
private category table. The per-entry hash includes that table as well as C,
layout, body and target, distinguishing identical C with different trap labels.

The native owner limits total root/helper value and borrow slots to 4,096
before source generation. Exact 4,096 succeeds and 4,097 refuses, without
changing the general state verifier or literal admission. This explicit native
work bound keeps caller-created graphs from producing unbounded C locals.
Zero owners, unsupported targets and corrupt MIR reject before build or I/O.

`contractstatefixture.LoadHIRCases` exposes fresh authored HIR and limits from
the existing validated manifests, not oracle output or preselected native
answers. Native tests re-enter `Prepare`; the original fixed-result manifest
runner remains an independent prerequisite. Both Windows consumer profiles run
nineteen original v0-v4 cases (38 compiled C executions): linear cell/maps,
branch/join, transitive helper commit/reject, nested and unrelated-map borrows,
and native limit rollback.

Fourteen additional graphs cover all seven codecs with present/missing reads,
both found branches, two-argument joins, dynamic map keys and values, deletion,
SSA bytes surviving an unrelated arena repack, canonical zero fallback and
commit/reject/trap. Their 28 C executions compare complete successful read
traces, operation indices, typed binary values, cost, sorted writes/deletions and
byte cost against Go. Error cases require matching status and no proposed
writes. A trap-to-reject Go control makes the otherwise panicking trap path's
observations/cost independently visible, without supplying native result bytes.

```powershell
go test -mod=readonly -count=1 -timeout 5m -failfast -v ./internal/contractstatenative -run '^TestNativeStateGraph(Emission|Admission)$'
go test -mod=readonly -count=1 -timeout 5m -failfast -v ./internal/contractstatenative -run '^TestNativeStateGraphEmission$/v4-dynamic'
go test -mod=readonly -count=1 -timeout 2m -v ./internal/contractstatenative -run '^TestNativeStateGraph(SlotBoundary|HashBindsTrap)$'
go test -mod=readonly -count=1 -timeout 2m ./internal/contractstatefixture ./internal/contractstatemir
go vet -mod=readonly ./internal/contractstatenative ./internal/contractstatefixture ./internal/cstdlib
git diff --check
```

The original native matrix/admission passes in 24.370s. After adding the seven-
codec cases, their selected matrix passes in 17.021s on unchanged product
inputs. New slot/hash controls pass in 0.057s; fixture/state-MIR regressions
pass in 0.057s/0.074s. Scoped vet, formatting and diff checks pass; no selected
skips. This is actual verified-operation C execution through a test acceptance
entry, not production graph framing or durable graph execution. Graph binary
transport/admission, real one-attempt graph CAS/reopen, source/developer/cache/
CLI integration, native Linux and every final acceptance gate remain required.
