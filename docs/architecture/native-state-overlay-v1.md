# Native state physical-key overlay v1

Status: compiler-private execution-core prerequisite for step 3 of
[native persistence integration](native-state-persistence-v1.md).
This implements the multi-key C overlay and seven canonical codec validators,
not broader MIR emission, wire framing, state CLI or durable native admission.
The original single-cell emitter/runtime and its CAS path remain unchanged.

## Sequential delivery

1. **Implemented:** actual allocation-free C multi-key overlay, exact write
   policy, seven canonical codec validators and independent Windows native/Go
   comparisons. Header and runtime are separately available to the compiler.
2. Generate checked multi-field/cell/map operations and exact physical addresses
   from the existing verified state MIR, then bind snapshots, complete sorted
   overlays and read traces in a versioned transport to the existing one-attempt
   CAS executor. Validate every literal key's original encoded-byte limit before
   hashing. Do not equate a 32-byte physical address with its map-key byte cost.
3. Expand the same runtime path to data-dependent keys/values, graph exits,
   helpers and exact-address borrow generations, with independent Go/native
   observations before each route is admitted. Complete source/artifact/cache/
   CLI integration and all final gates from the full implementation queue.

## Memory and lifetime

The caller owns an immutable sorted snapshot, an entry workspace of `MaxWrites`
elements and two distinct writable byte arenas. Each arena has capacity at least
the positive `MaxWriteBytes`. An entry retains its full 32-byte physical key,
value bytes and length, and a separate deletion flag. Empty present values and
deletions are therefore different even when they cost the same 33 bytes.

The kernel allocates nothing. Each mutation copies only the current packed
overlay values into the other arena and swaps roles. Replacing a value does not
accumulate prior copies or consume an operation-count-sized history buffer. A
current read view can be the next write's input without being overwritten while
copied. Key input is retained before entry movement, including self-key aliasing.
Views expire on a successful mutation or close; this is not the MIR borrow API.
All caller workspaces must be distinct live allocations of the declared sizes.

Commit seals the proposed sorted overlay for the enclosing transport; it never
publishes a store, increments a persistent version or grants CAS authority.
Abort clears both caller arenas and written metadata, discards write statistics
and closes the attempt. Reads, writes, commit and abort reject after close.
Invalid begin arguments and malformed/duplicate/unordered snapshots leave the
transaction object unchanged. A valid begin requires a fresh caller-owned frame;
it is not an implicit retry or reset policy.

## Cost and failure order

Lookups use binary search: reads cost O(log W + log S) for W overlay entries and
S snapshot entries. Sorted insertion plus arena packing costs O(W + V), where V
is the current overlay value bytes. Workspace is O(W + B) with two B-byte arenas,
rather than `W * MaxValueBytes` fixed slots. This bounded copying tradeoff is
explicit; it does not establish production throughput or gas pricing.

A mutation checks open state, per-value bytes, distinct write count and complete
overlay byte cost before touching entries or arenas. Cost is exactly 32 key
bytes, one operation byte and value bytes. Replacement subtracts the old entry
cost first; deletion costs 33 bytes. Checked subtraction/addition refuses
overflow, including a declared SIZE_MAX-sized input before it is read. Failure
leaves the previous overlay and current views intact. The enclosing transaction
runner must abort the whole attempt when its semantics require rollback.

`fycs_codec_valid_v1` implements the existing bool/u32/u64/bytes/UTF-8/bytes20/
bytes32 canonical formats. UTF-8 excludes invalid leads, continuations,
truncation, overlong encodings, surrogates and values above U+10FFFF; it accepts
ordinary NUL, noncharacters and valid maximum scalars just like Go. Physical
keys do not expose their field/codec, so the compiler-owned typed operation and
input decoder must explicitly call the appropriate validator. The overlay does
not infer schema or trust from arbitrary hashes.

## Windows evidence

`TestNativeOverlayRuntime` compiles the actual checked-in header/kernel and
binary probe with strict Windows C11 low/optimized. Each profile compares
4,778 codec inputs against `contractstate.ValidateCanonicalEncoding`, including
0/1/4/8/20/32 boundaries, 65,536-byte values, all one-byte boolean/UTF-8 inputs,
lead/continuation combinations and valid/truncated Unicode boundaries.

Thirteen parameterized overlay cases per profile compare every read, operation
error, current write count/byte cost and complete sorted proposed overlay with
the independent Go transaction kernel. They include multiple fields, binary-key
map entries, absent/empty/present reads, replacement/deletion, competing value/
count/byte failures and 80 growing/shrinking replacements in one slot. Self-view
writes preserve current bytes; snapshots remain byte-for-byte immutable. A
separate compiled probe checks invalid begin inputs, closed attempts, self-key
aliasing, size overflow and full abort-buffer clearing.

The first mixed probe disagreed at the first final map key: its independent
test derivation had mistakenly used a four-byte key-length prefix. The existing
canonical contract uses an eight-byte prefix. The probe was corrected; runtime,
oracle, warning policy and key derivation contract were not relaxed.

```powershell
go test -mod=readonly -count=1 -timeout 3m -failfast -v ./internal/contractstatenative -run '^TestNativeOverlayRuntime$'
go test -mod=readonly -count=1 ./internal/cstdlib -run '^TestContractState'
go vet -mod=readonly ./internal/cstdlib ./internal/contractstatenative
git diff --check
```

Final native replay passes in 1.727s, with 9,556 codec results and 26 overlay
cases across the two profiles. The existing single-cell execution/CAS regression
passes in 17.793s; its runtime/emitter/executor sources are unchanged. The original
runtime-owner structural test passes in 0.040s; scoped vet, formatting, UTF-8/LF
and diff checks pass. These are kernel conformance observations, not
compiler-source transactions or durable multi-key execution. Native Linux and
all later steps remain open. No source grammar, public CLI/SDK/runtime ABI,
provider/DB, dependency, default timeout, workflow/runner or repository-policy
change is made; no push, dispatch, release or target promotion is implied.
