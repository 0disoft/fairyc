# Immutable byte-view comparison assessment

Status: Assessment completed; contract implemented in SDK stage0.78 under ADR 0422.
Date: 2026-10-02 (Asia/Seoul).
Inspected compiler repository revision: `4eb1748f23f6ad6ce938c24c54f394184e886be7`.
Bundled SDK at assessment: `0.0.0-stage0.77`; language `/20`; runtime `/0`.

Decision follow-up: [ADR 0422](../adr/0422-sdk-immutable-byte-view-comparison.md)
accepts and records implementation of the exact contract below. This assessment
retains the original inspected evidence and proposal rationale; the ADR owns
the contract and execution evidence. The export is available in SDK stage0.78,
but was absent from the inspected stage0.77 bundle.

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
At assessment time the maintained lexer was one concrete future adopter. External round4 projects
remain preserved evidence, not migration targets or new execution results.

A small local loop remains a valid alternative. The benefit is removing repeated
ordering semantics used in sorting and validation, without adding language
syntax or a general sorting API. This is not evidence of a compiler defect,
speedup, constant-time behavior or a need for a native comparison intrinsic.

## Proposed signature and exact ordering

```text
pub fn i32 compare(ref u8[] left, ref u8[] right);
```

The proposal below was accepted and implemented under ADR 0422 in SDK stage0.78.
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
Its separate implementation unit activates SDK stage0.78 and records verification
and a distinct MIR byte-literal limitation in the ADR. Historical source identities
below remain the original assessment evidence, not current implementation hashes.

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

## Round4 adoption review, 2026-10-03

This read-only review uses compiler checkout
`242ebd3c6834d21a757566e4b1c0bb4478c3a827` and bundled SDK stage0.78.
The nine round4 `src/` directories were searched for byte-order helpers and
their callers. Build backups and `repro/` files are excluded from adoption
candidates. The original assessment identities above remain historical evidence.

Four projects have byte-order loops compatible with the SDK contract. The log
analyzer is an additional consumer absent from the original assessment table.

| Project and source location | Candidate replacement | Policy to preserve |
| --- | --- | --- |
| 01 ledger `src/ledger.fy:30`, caller `:120` | Compare the two account subviews with `bytes::compare(...) < 0` | Group equal accounts before sorting; keep strict-less insertion sorting, account validation, checked i64 totals and complete-input-before-output behavior |
| 02 log analyzer `src/analyzer.fy:17`, caller `:72` | Compare the two path-pool subviews with `bytes::compare(...) < 0` | Group identical paths; keep numeric status ordering, strict-less path insertion sorting, streaming boundaries, error lines and checked i64 aggregation |
| 05 resource pack `src/codec.fy:8`, callers `src/create.fy:48` and `src/readpack.fy:38` | Keep the local comparator signature and delegate its body to `bytes::compare(a, b)` | Keep strict-less sorting, duplicate-name rejection and separate zero/positive reader diagnostics; preserve name validation, offsets, payload bytes and publication policy |
| 09 KV `src/codec.fy:8`, callers `src/snapshot.fy:46` and `:65` | Keep the local comparator signature and delegate its body to `bytes::compare(a, b)` | Keep zero/positive corruption diagnostics and `<= 0` insertion/update placement; preserve generation, checksum, selected-key replacement and write/sync/close/replace_file protocol |

For the ledger and log analyzer, form shared subviews from the existing start
and length arrays; retain the surrounding range and lifetime assumptions.
Equal views must return false from `less`. Do not add the lexer's index tie-break
to these helpers: their equal-name grouping and strict-less insertion policy
already belong to their applications. For pack and KV, exact -1/0/1 results
match the local helper; changing only that body leaves call-site policy explicit.

Projects 03, 04, 06 and 07 have byte equality helpers but no byte-order helper
in their product sources. Project 08 sorts signed i64 interval endpoints; byte
ordering would change its semantics. Equality-only callers, including command,
magic and placeholder matching, remain a separate `bytes::equal` adoption
question. No parsing, UTF-8 validation, checksum or numeric loop is a comparator
migration target. Existing `codec::equal` wrappers can continue testing the local
comparator result for zero; routing all equality through comparison is unnecessary.

The maintained lexer has already adopted `bytes::compare` in commit `242ebd3c`,
retaining preceding field priorities and `ai < bi`. That unit passed full
diagnostic-output comparison and binary/prefix/duplicate cases in development
and release, plus self-lexing of 18 source files and four boundary inputs.
These are lexer results, not round4 migration or revalidation results.

The freshly inspected log source SHA256 is
`b62fdfbcff1b4f6c43724e61a58172970631f6944bad078270ff6ddff09111a4`.
The seven round4 source hashes listed in the original table still match.
The round4 compiler EXE SHA256 still matches
`7d58dd9c8860f01723f55f2f289a609d0941fa3cd6b06699954ca46336eeb917`;
its retained receipt/version record identifies revision `c34718bd`, unmodified,
SDK stage0.73, language `/20` and runtime `/0`. The matching revision's SDK
source contains neither `compare` nor `equal`; importing the new helper requires
a separately selected SDK stage0.78 toolchain. Do not rewrite the pin or receipts.

Review-time SHA256 checks cover 79 files: product sources, all nine reports and
manifests, and the four pinned-toolchain files. They are unchanged across this
review. No round4 check/build/native execution, benchmark, fault experiment or
approval of project completion was performed. Any future application experiment
should use a separate copy and compare complete outputs and, for pack/KV,
serialized file bytes; it must retain the original error and preservation cases.

## Isolated ledger adoption evidence, 2026-10-03

[`verify-ledger-compare-adoption.ps1`](../../tools/verify-ledger-compare-adoption.ps1)
copies only the ledger manifest, four product sources and selected fixtures into
a unique compiler-checkout `.cache/ledger-compare-adoption/` directory. It keeps
one unchanged source variant and changes only `less` in the other variant:

```text
let usize a = starts[left]; let usize b = starts[right];
return bytes::compare(data[a..a + lengths[left]], data[b..b + lengths[right]]) < 0;
```

The edited source uses LF; the unedited files retain their original bytes.
Equality, grouping, insertion sorting, validation, checked i64 arithmetic,
file reads and output publication are unchanged. Both variants use the same
newly built SDK stage0.78 compiler; this isolates helper adoption from a compiler
upgrade. The old stage0.73 compiler is hashed but is not executed or replaced.

Final run `2b8950a036a746b59de93bf02d842969` passed two checks, four native builds
and **68 exact-output executions**: 17 cases in each development/release image
of both variants. Sixteen selected historical cases use retained independent
expectations and verify fixture hashes against the original receipt. They cover
empty input, Korean/space/prefix account ordering, CRLF/final-line handling,
i64 maximum and negative balance, sum/parse overflow, late invalid fields,
invalid UTF-8, maximum name length, 128/129 accounts, help and CLI/file errors.
The new fixture uses ten distinct names including `10`/`2`, case and space
differences, proper prefixes, Korean, and distinct composed/decomposed `é`.
Repeated rows aggregate values beyond i32; BigInteger totals and UTF-8 hex
ordinal sorting independently determine the complete expected output.

Every exit/stdout/stderr matches its expectation, establishing matching results
between variants for these cases. Native readers explicitly select strict UTF-8.
Copied inputs remain byte-identical; 22 protected original source, metadata,
fixture and compiler files retain their SHA256. An initial trial accidentally
split the decomposed name into two names in the PowerShell fixture; parenthesized
construction and an inventory assertion correct it before the final run above.

The final directory retains `receipt.json`, `raw-executions.jsonl`, both source
copies, new lockfiles and all four EXEs. The receipt records harness/compiler/EXE
SHA256, source/SDK digests, per-case expectations and results, input before/after
hashes and protected-file hashes. Compiler revision is `27bc9d20`, modified=false,
SHA256 `0922d0ca699abcf747cdb5a2604151e9a9353dc0d306cca4ac5996b94ff49a99`.
Baseline source digest matches the historical ledger digest; candidate source
digest is `fysrc1-sha256-2a8a7e700bbefa3bca2980a1c2b47f65d9d6468646d97f2e9d62b2a2b1db9059`.

Reproduce locally after building that separate compiler:

```powershell
bun tools/run-go.ts build -mod=readonly -trimpath -o .cache/ledger-compare-toolchain/fairyc.exe ./cmd/fairyc
./tools/verify-ledger-compare-adoption.ps1
```

This opt-in harness requires the preserved external round4 ledger fixtures and
SDK stage0.78. It is not a CI test, full 92-case rerun, historical-compiler rerun,
fault-injection test or performance measurement. No SDK/API/version change or
round4 migration was made. Subsequent log and pack experiments are recorded
below; KV still has only the static review above.

## Isolated log analyzer adoption evidence, 2026-10-03

[`verify-log-compare-adoption.ps1`](../../tools/verify-log-compare-adoption.ps1)
copies the log manifest, five product sources and selected fixtures into a unique
`.cache/log-compare-adoption/` directory. The legacy variant is copied unchanged;
the SDK variant replaces only the path-order loop in `analyzer.fy::less`:

```text
return bytes::compare(pool[starts[left]..starts[left] + lengths[left]], pool[starts[right]..starts[right] + lengths[right]]) < 0;
```

The edited file uses LF. The path pool, equality-based lookup/grouping, strict-less
insertion sorting, numeric status ordering, streaming reader, i64 overflow checks,
validation/error-line policy and complete-response-before-output behavior remain
unchanged. Both variants reuse the exact stage0.78 compiler from the ledger
experiment above. Compiler/SDK implementation sources have not changed since its
revision `27bc9d20`; no compiler rebuild or historical stage0.73 execution is needed.

Run `fda5f6e7f8d44122938d893b870e0416` passed two checks, four native builds and
**104 exact-output executions**: 26 cases per development/release image for each
variant. Twenty-five selected historical cases reuse independent expectations
and match the original input hashes. They cover empty input, Korean/space/prefix
paths, original URL/Unicode spellings, BOM/mixed newlines/final line, i64 maximum,
overflow and invalid UTF-8, late status/UTF-8 errors, 128/129 paths and reuse after
128 paths, all 500 numeric statuses, help and CLI/file errors.

The boundary fixtures cover row lengths 4095/4096/4097, a UTF-8 character starting
at byte offsets 4094/4095/4096, CR/LF at 4095/4096, and a row spanning multiple
4096-byte read-request boundaries. Successful complete output matches in both
variants and profiles. This is not forced short-read injection or proof that the
OS returned exactly 4096 bytes on each read. Error cases retain complete stderr,
1-based error-line reporting, exit status and empty stdout.

The new fixture has ten distinct paths, including numeric-looking suffixes,
case/space/prefix differences, Korean, and composed/decomposed Unicode spellings.
Each path occurs under three unordered status codes. BigInteger totals, numeric
status sorting and UTF-8 hex ordinal path sorting determine the expected response
independently. Each path has three requests and 4294967303 bytes; each Unicode
spelling remains a separate entry. Total is 30 requests and 42949673030 bytes.

All per-case exit/stdout/stderr matches establish matching behavior between
variants within this subset. Strict UTF-8 native readers, input before/after
hashes and 32 protected-original hashes are checked. The final directory retains
`receipt.json`, `raw-executions.jsonl`, source copies, new lockfiles and four EXEs;
harness/compiler/EXE SHA256 and source/SDK digests are recorded and verified.
The legacy source digest matches the historical log digest. Candidate source
digest is `fysrc1-sha256-ef6328548673aeb57d5be2c277ea6208c4876ca184d60076450708178b99a8c8`.

With the separate stage0.78 compiler available at the ledger toolchain path,
run `./tools/verify-log-compare-adoption.ps1` from the compiler checkout. The
script requires the preserved external log fixtures and permits copied input
paths only within their named `samples/` entries. This is an opt-in local
adoption experiment, not a CI test, full 186-case rerun, fault experiment,
performance measurement or migration of the preserved round4 project.

## Isolated resource pack adoption evidence, 2026-10-03

[`verify-pack-compare-adoption.ps1`](../../tools/verify-pack-compare-adoption.ps1)
copies the pack manifest, seven product sources and two hash-checked independent
oracle helpers into a unique `.cache/pack-compare-adoption/` directory. The legacy
sources are copied unchanged. The SDK variant replaces only `codec.fy::compare`
with the same-signature delegation:

```text
pub fn i32 compare(ref u8[] a, ref u8[] b) { return bytes::compare(a, b); }
```

The edited file uses LF. The equality wrapper, insertion sort, duplicate-versus-
unsorted diagnostics, name validation, binary encoding/range checks, lookup,
complete-validation-before-output and exclusive file creation remain unchanged.
Both variants use the same stage0.78 compiler identified in the ledger experiment;
compiler/SDK implementation sources have not changed since revision `27bc9d20`.
The stage0.73 compiler is hashed without execution or replacement.

Run `59910b3e53ea41e494f4b4a667255393` passed two checks, four native builds and
**128 executions**: 32 cases per development/release image for each variant.
Twenty-six cases reuse the reviewed original oracle helpers. They cover empty
and single-entry packs, multi-entry sorting, case/prefix/space/Korean names,
logical names resembling paths, reversed input producing the same pack,
handwritten fixed-hex list/extraction, duplicate and unsorted indexes, late
invalid UTF-8 during list/extract, duplicate creation, missing extraction names,
existing output, source/TSV aliases and hardlinks.

Six new cases use ten names with numeric-looking strings, proper prefixes,
case/space differences, Korean, and distinct composed/decomposed Unicode
spellings. Payloads include NUL/0xFF/0x80 and a distinct final byte per name.
Creation, listing, three exact-name extractions and reversed-input creation
match independent UTF-8 hex ordinal sorting, BinaryWriter little-endian encoding
and BigInteger length/offset arithmetic. Every generated pack and extracted file
matches the complete oracle bytes, not only its length or a successful exit.

All exit/stdout/stderr expectations, protected input bytes and per-case file
trees pass, with no unexpected residual files. Readers explicitly select strict
UTF-8. Twenty-one protected original source, metadata, oracle, EXE and compiler
files retain their SHA256. An initial pre-build attempt stopped at the helper
identity gate because the harness transcribed one extra character in a hash;
the corrected guard precedes the successful run above.

The run retains `receipt.json`, the identity raw log, both workspaces' source,
lockfiles, oracle helpers, per-case raw logs and four EXEs. The receipt records
harness/helper/compiler/EXE SHA256, source/SDK digests, complete per-case
expectations/results, oracle output hashes and protected-file before/after hashes.
Legacy source digest matches the historical pack digest; SDK candidate digest is
`fysrc1-sha256-a611851b278e785ab10c31eb38a949ce70662c4274c55085aab6960e4d377ec1`.

With the separate stage0.78 compiler available, run
`./tools/verify-pack-compare-adoption.ps1` from the compiler checkout. It requires
the preserved external pack and its reviewed oracle helpers. This opt-in local
experiment is not a CI test, full 218-case rerun, historical-compiler rerun,
forced write/close fault experiment, performance measurement or migration of
the preserved round4 project. No SDK/API/version or publication-policy change
is introduced; KV adoption remains a separate follow-up.
