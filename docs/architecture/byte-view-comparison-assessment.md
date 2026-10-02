# Immutable byte-view comparison assessment

Status: Assessment completed; contract accepted in ADR 0422; implementation pending.
Date: 2026-10-02 (Asia/Seoul).
Inspected compiler repository revision: `4eb1748f23f6ad6ce938c24c54f394184e886be7`.
Bundled SDK at assessment: `0.0.0-stage0.77`; language `/20`; runtime `/0`.

Decision follow-up: [ADR 0422](../adr/0422-sdk-immutable-byte-view-comparison.md)
accepts the exact contract below. This assessment retains the original inspected
evidence and proposal rationale; the ADR owns the accepted contract. The export
is not yet implemented or available in the inspected SDK.

## Recommendation and observed need

The assessment recommended one ordinary immutable byte-view comparison helper;
ADR 0422 now accepts it. SDK stage0.77 provides [equality](../adr/0421-sdk-immutable-byte-view-equality.md),
but `std::bytes` has no public `compare` export. Equality alone cannot express
ordering or distinguish duplicate keys from out-of-order keys.

| Inspected consumer | Comparison use | Policy that remains with the caller |
| --- | --- | --- |
| Round4 ledger `ledger.fy:30` and `:120` | Unsigned byte order followed by shorter-prefix order for account sorting | Account validation, grouping, aggregate output and stable insertion order |
| Round4 pack `codec.fy:8`, `create.fy:48`, `readpack.fy:38` | Tri-state name comparison for sorting, duplicate detection and index order validation | Name/UTF-8 restrictions, duplicate diagnostics, offsets and file publication |
| Round4 KV `codec.fy:8`, `snapshot.fy:46` and `:65` | Tri-state key comparison for duplicate/order checks and insertion position | Key validation, overwrite choice, generation, checksum and save protocol |
| Maintained lexer `diagnostics.fy:40` | Byte comparison of diagnostic token views after preceding fields tie | Start, end, code, exit-repair priority and final stable-index tie-break |

Pack and KV contain identical comparison function bodies. They are two consumers,
not two independent comparison algorithms. The ledger and lexer expose the same
byte/prefix ordering through boolean `less` functions. Only the byte-ordering
portion is a prospective shared operation; their larger sorting policies are not.
The maintained lexer is one concrete future adopter. External round4 projects
remain preserved evidence, not migration targets or new execution results.

A small local loop remains a valid alternative. The benefit is removing repeated
ordering semantics used in sorting and validation, without adding language
syntax or a general sorting API. This is not evidence of a compiler defect,
speedup, constant-time behavior or a need for a native comparison intrinsic.

## Proposed signature and exact ordering

```text
pub fn i32 compare(ref u8[] left, ref u8[] right);
```

The proposal below was accepted by ADR 0422; acceptance does not imply availability.
Return exactly `-1`, `0` or `1`, not an arbitrary difference whose sign must be
interpreted. Scan corresponding bytes from index zero as unsigned `u8` values
in the range 0..255. The first unequal pair determines the order. If all bytes
in the common prefix match, the shorter logical view sorts first. Equal lengths
and equal bytes return zero, including two empty views.

| Left bytes (hex) | Right bytes (hex) | Proposed result | Reason |
| --- | --- | ---: | --- |
| empty | empty | 0 | Equal empty views |
| empty | `00` | -1 | Empty is a proper prefix |
| `00` | empty | 1 | Longer after matching prefix |
| `61` | `61 00` | -1 | Proper prefix; NUL is still data |
| `61 FF` | `62 00` | -1 | First difference wins, regardless of the later byte |
| `80` | `7F` | 1 | Unsigned order across the high-bit boundary |
| `00 FF` | `00 FF` | 0 | NUL and malformed UTF-8 are ordinary bytes |
| `31 30` | `32` | -1 | ASCII "10" sorts before "2"; no numeric interpretation |

For valid views the proposal defines a total order over finite byte sequences:
`compare(a,b) == 0` exactly when `bytes::equal(a,b)` is true; swapping arguments
negates the result; and the less-than relation is transitive. Distinct owners
with the same viewed bytes compare equal. Owner identity, allocation capacity,
bytes outside the views and implicit string terminators never participate.

There is no UTF-8 validation, Unicode normalization, case folding, locale,
natural-number ordering or text overload. If callers need valid UTF-8 keys, they
must continue validating those keys separately. UTF-8 byte order is not a
human-language collation promise. Invalid slicing can trap before comparison;
this helper would not validate arbitrary pointers or construct fallible views.

## Return representation and implementation limits

| Alternative | Assessment |
| --- | --- |
| Exact `i32` -1/0/1 | Preferred: matches concrete tri-state consumers and supports ordinary `< 0`, `== 0`, `> 0` branches without a new public type |
| Boolean `less` only | Insufficient for a single-call duplicate-versus-order check; leaves equality or reverse comparison to callers |
| New `Ordering` enum | Valid design alternative, but adds a public nominal type and exhaustive matching where existing consumers use integers; no demonstrated benefit for this narrow helper |
| Subtract bytes or lengths | Reject: unsigned subtraction can underflow, length differences can overflow or require excluded conversions, and magnitude is not the desired contract |

The prospective implementation is ordinary safe Fairy source: a guarded loop
over the common prefix, explicit `<`/`>` byte branches, then explicit length
branches. Return constant `i32` values only. Index while both logical lengths
allow the access; use existing checked `usize` increment. No allocation, I/O,
error enum, Result, retained borrow, mutation, unsafe scope, raw pointer,
foreign `memcmp`, compiler intrinsic or public byte-view C ABI is justified.

Worst-case work is linear in the common-prefix length, with constant auxiliary
storage and no library input-length cap. Application input/work limits remain
explicit. Early exit is permitted; no timing or cryptographic guarantee follows.
The existing equality helper stays available and need not scan a prefix when
unequal lengths already establish inequality. Do not rewrite equality merely
to route it through comparison.

## Ownership and compatibility boundary

Existing immutable view rules govern distinct owners, identical same-owner views
and overlapping shared subviews. The scalar result contains no borrow. Calls
would use existing [shared reborrowing](../adr/0414-call-shared-reborrows.md),
argument-evaluation protection and last-use release. Mutation/movement during
live shared loans must retain their diagnostics. This does not admit multiple
mutable sibling views; [ADR 0415](../adr/0415-exclusive-byte-subviews.md) still applies.

No sorting algorithm, stable-sort API, collection comparator, callback protocol,
generic overload, Unicode service, numeric parser, binary codec, runtime ABI,
grammar or target-maturity change is proposed. No SDK version is reserved here.
The existing SDK, equality contract and all application behavior remain unchanged.

## Separate acceptance, implementation and adoption gates

Follow [ADR 0218](../adr/0218-modern-language-concept-admission-boundary.md) and the
[workload admission gates](../adr/0328-production-workload-capability-admission-gates.md).
ADR 0422 accepts this exact contract and selects the three-value `i32` return.
Its implementation gate remains pending; the documentation-only acceptance
does not implement the export or change the SDK version.

Required implementation evidence includes:

- An independent unsigned-byte oracle such as Go `bytes.Compare`, checked against
  the exact three-value result and current `bytes::equal`, not a second Fairy loop.
- Empty and proper-prefix pairs in both directions; first/interior/last mismatch;
  bytes 0..255; NUL, BOM, invalid UTF-8 and different Unicode spellings; long common
  prefixes; antisymmetry, equality consistency and transitivity over bounded data.
- Distinct owners, identical/overlapping shared subviews, nonzero offsets,
  zero/end empty views, surrounding sentinels and unchanged complete owners;
  ordinary return/branch use and existing reborrow/mutation/move rejection.
- SDK discovery and source/interface/lock/cache identity activation. Older bundles
  must not resolve the export accidentally. Version selection belongs to that unit.
- Source check, verified-MIR codec round trip, interpreter, C generation and
  Windows native low/optimized execution, retaining exact identities and UTF-8
  argv/cwd/stdout/stderr/exit evidence and EXE SHA256. C generation is not Linux
  native execution. No benchmark or constant-time claim is required.

Application adoption is a later independent unit. In the maintained lexer,
replace only the token-byte loop, retaining start/end/code/exit-repair ordering
and `ai < bi` when the compared token bytes tie. Compare complete diagnostic
output and deterministic tie cases with current-toolchain evidence. Do not
change external round4 sources, compiler pins or receipts to demonstrate adoption.

## Inspected source identity and validation scope

SHA256 identifies exact inspected file bytes, including checkout line endings.
Round4 paths below are relative to
`C:\Users\cherr\Downloads\fairy_c_test_projects\round4`; other paths are relative
to the compiler checkout. Call-site inspection establishes reachable uses,
not executed correctness or performance.

| Source | SHA256 |
| --- | --- |
| SDK `internal/sdkstdlib/sources/std/bytes.fy` | `2fdb25f2bc59de2bf9c9af150fe4cb81a9bd30fc2a18601616d8feb451df30f8` |
| SDK identity `internal/sdkstdlib/stdlib.go` | `7754635748cb47abeb1a10bd3bd883876fb46ee06c89cc2fec49ffaf8cff499a` |
| Maintained `examples/lexer_candidate/src/diagnostics.fy` | `3235e9307f2ad31c0fa74bafd3bd03ccda526053458db45b9c760fff3a3c2839` |
| Round4 `01-ledger-reconciler/src/ledger.fy` | `e9b5afccb0b0da265f04b13e6070490f0d8103c02ff3f340b5f3707895f95488` |
| Round4 `05-resource-pack-cli/src/codec.fy` | `afb09ec8c09d47f896079d30f60ade4129e9f3a72d6dfa8c0e5092841660a5c8` |
| Round4 `05-resource-pack-cli/src/create.fy` | `0faffc8ee848fbda7eaa0e05c780ca813f1c8e437a37f0e15cb00e65bb517171` |
| Round4 `05-resource-pack-cli/src/readpack.fy` | `c6c6f022cbaafae8da6097a6941ed8d580a7f83803124c420ecf15cee943b648` |
| Round4 `09-persistent-kv-store/src/codec.fy` | `0cd44acb5c24b04ee8eb0d211c0d631912137ae4b2c038b6f7ca9ea75f9270d7` |
| Round4 `09-persistent-kv-store/src/snapshot.fy` | `78a429920f20997c238daab4917ceca2672ed65cb5e66474664a2024c4ba33d7` |

The original assessment validated local document links, whitespace, nine source hashes,
comparison API absence, candidate boundaries and the eight proposed hex examples
against a normalized independent byte-order oracle. That oracle checks document
examples only; it does not execute Fairy or establish an SDK implementation.
No new Fairy check/build/native test, external suite or performance experiment
was run. No product source, SDK version, compiler behavior or historical file changed.
