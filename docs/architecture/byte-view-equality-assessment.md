# Byte-view equality assessment after round4

Status: Historical pre-decision assessment; accepted contract linked below.
Date: 2026-10-02 (Asia/Seoul).
Inspected compiler repository revision: `3761e2e76d9017044811f1593669684b8919593f`.
Current bundled SDK: `0.0.0-stage0.76`; language `/20`; runtime `/0`.

This document preserves the pre-decision assessment. The subsequent contract is
accepted by [ADR 0421](../adr/0421-sdk-immutable-byte-view-equality.md), but is not
implemented. SDK stage0.76 still has no `bytes::equal` export. The candidate
reasoning and inspected-source identities below remain historical evidence;
ADR 0421 now owns the accepted semantics and implementation/adoption gates.

## Selected next step

Advance one ordinary immutable byte-view equality function to a separate ADR
candidate. Its proposed home is `std::bytes` and its proposed shape is:

```text
pub fn bool equal(ref u8[] left, ref u8[] right);
```

This spelling is a proposal, not available SDK code. It would replace repeated
length-and-byte loops without introducing language syntax, an error enum, an
allocation, I/O, a foreign comparison symbol or a generic collection API.
The current `std::bytes` source has no public equality function.

The primary round4 `src` trees contain equality helpers in all nine projects;
seven implement the length-and-byte loop directly, while the pack and KV codec
helpers use `compare(left, right) == 0`. Build snapshots and rejected `repro`
files are not additional consumers. In the maintained repository,
`examples/task_manager_cli/src/format.fy` has the same direct loop and actual
call sites for CLI verbs, help and the FYTASK1 header. That is one concrete
prospective application adopter, not an already migrated consumer. Its README
still records the historical stage0.72 verification; this assessment does not
re-execute that evidence against stage0.76.

Keeping a local seven-line loop remains a valid alternative. The reason to
consider an SDK helper is the recurrence of exactly the same low-level semantics
across unrelated commands, identifiers and binary headers, together with a
maintained application that can demonstrate adoption. This is a modest reduction
in repeated code, not evidence of a compiler defect, speedup or security benefit.
Do not add an API solely to reach a function count or roadmap target.

## Compared options

| Option | Decision | Evidence and boundary |
| --- | --- | --- |
| Immutable byte-view equality only | Advance to ADR candidate | Exact recurring operation; one maintained application can replace its loop while retaining CLI and storage behavior. |
| Keep all equality loops local | Valid fallback | Lowest public API cost; preserves a small repeated implementation. Reconsider this if adoption fails to show useful simplification. |
| Lexicographic byte comparison at the same time | Defer to a separate proposal | Round4 pack/KV and maintained lexer diagnostics have real ordering loops, but equality does not require a tri-state return or an ordering contract. The lexer's preceding fields and final stable-index tie-break remain caller policy. |
| Full-range usize decimal parser | Defer | Existing signed parsing and checked conversions cover the inspected maintained numeric quickstarts. Round4 split/pack bounded-number loops also enforce application limits; they do not by themselves require values above i64 MAX. |
| Dedicated digits-only parser | Keep deferred | The [composition assessment](digits-only-i64-parsing-assessment.md) and [executed example](../../examples/digits_only_quickstart/README.md) already show the leading-minus guard plus signed SDK parser. |
| Generic binary codec/endian family | Defer | Pack/KV field widths and limits and the maintained codec's capacity/high-bit preflight differ. Shared byte representability is already implemented; a generic codec needs its own complete input/output contract. |
| Universal row reader, UTF-8 diagnostic or field validator | Defer | Streaming, EOF/CRLF rules, field limits, error ordering and line diagnostics are application-owned. Repeated utility names are not enough to merge those contracts. |
| Another Buffer append or file-publication wrapper | Do not duplicate existing capabilities | Public byte append and file read/replace operations already exist. Their application-level transactions remain separate. |
| Modify the external round4 workloads | Outside this task | Preserve pinned experiments, sources, toolchains and receipts; demonstrate future adoption in the maintained repository. |

The existing [checked integer](../adr/0418-sdk-checked-integer-conversions.md),
[checked byte](../adr/0419-sdk-checked-byte-conversion.md) and
[signed decimal parsing](../adr/0420-sdk-checked-i64-decimal-parsing.md) contracts
remain unchanged. Neither a full-range unsigned parser nor a binary codec is
implied by those APIs.

## Inspected source identity

SHA256 identifies the inspected file bytes, including checkout line endings.
External paths are relative to
`C:\Users\cherr\Downloads\fairy_c_test_projects\round4`;
repository paths are relative to the compiler checkout. These hashes are current
source-inspection evidence, not new executions of those applications.

| Scope and source | Evidence | SHA256 |
| --- | --- | --- |
| External `01-ledger-reconciler/src/ledger.fy` | Length-and-byte equality; account field lookup; separate ordering loop | `e9b5afccb0b0da265f04b13e6070490f0d8103c02ff3f340b5f3707895f95488` |
| External `02-access-log-analyzer/src/format.fy` | Length-and-byte equality; path grouping uses the helper | `385503eeadf2e4b08058ae3647488fd2dfa253873b0d5c8c507cd2e2b2b75475` |
| External `03-binary-patch-cli/src/patches.fy` | Length-and-byte equality; CLI help matching | `2e0b25805d7129f2a82e5f7650aeb88cd3ae22b93e6e195a42fd027a20de519d` |
| External `04-file-split-join/src/numbers.fy` | Length-and-byte equality; bounded numeric parsing and noncryptographic checksum are separate | `e7ce0e90faa97db0c37d6d1b1795ea41a3233b3cd086fb491e775e804205ad95` |
| External `05-resource-pack-cli/src/codec.fy` | Equality through lexicographic comparison; byte-field codec has format-specific limits | `afb09ec8c09d47f896079d30f60ade4129e9f3a72d6dfa8c0e5092841660a5c8` |
| External `06-template-renderer/src/util.fy` | Length-and-byte equality; variable-name lookup and duplicate checks | `fb30d1684705f80863210d6b2e863f31dbfe4f44c33f742d4fe62d2754631d7c` |
| External `07-grid-pathfinder/src/util.fy` | Length-and-byte equality; CLI mode matching | `e12a836505b0140b20f04e82e1cf844d3fe1bbce1d2b773529cfa06f04b3aee0` |
| External `08-interval-merger/src/util.fy` | Length-and-byte equality; CLI mode matching; UTF-8 diagnostics are separate | `aa39a060a7a7dad35108bd38b379b5c777d2cd3baa42cc54a61aa7ebedfc14a2` |
| External `09-persistent-kv-store/src/util.fy` | Length-and-byte equality; its codec also derives equality from ordering | `e12a836505b0140b20f04e82e1cf844d3fe1bbce1d2b773529cfa06f04b3aee0` |
| Repository `internal/sdkstdlib/sources/std/bytes.fy` | Existing Buffer/view/append operations; no public equality function | `9467e6b60c6bc943689b0b387ab0e4e9dad2dce89d636f6495e1dae20d98d17c` |
| Repository `examples/task_manager_cli/src/format.fy` | Identical direct loop and exact header comparison | `b9e542110f5a83bad2af74ea8862bccb9654820fbb1499d93f8aa5f1c9cbcbb3` |
| Repository `examples/task_manager_cli/src/main.fy` | Actual equality call sites for CLI verbs and help | `33a7c742e4c4758f611e140fede0b359293c0881dccf345d3e34b09d3cf2aab0` |
| Repository `examples/lexer_candidate/src/diagnostics.fy` | Separate byte ordering inside a larger stable diagnostic sort | `3235e9307f2ad31c0fa74bafd3bd03ccda526053458db45b9c760fff3a3c2839` |
| Repository `examples/checked_byte_codec_quickstart/src/codec.fy` | Existing checked byte conversion plus caller-owned endian/width policy | `565e28f50422ad4af2c5d331e2bcd1dc8126896d76dab6ad6d68adca08365fa7` |

Identical hashes for grid and KV utility files indicate copied bytes, not two
independent implementations. The table identifies nine project uses, not nine
different algorithms or nine new validation runs. Secondary call sites were
inspected for reachability; their files are not implementation evidence for the
candidate. The task-manager and lexer examples provide different comparison
needs and must not be counted as interchangeable adopters.

## Proposed equality semantics and exclusions

For valid immutable views, return true exactly when both logical lengths match
and every corresponding `u8` is equal. Two empty views are equal. A prefix with a
different length is unequal. Comparison uses all binary byte values, including
NUL, 128..255 and malformed UTF-8; no text validation or normalization is implied.
The backing object's identity is irrelevant. Separate owners, the same owner,
identical subviews and overlapping shared subviews are valid read-only inputs.

A simple source implementation can reject unequal lengths, scan corresponding
bytes left to right and return false at the first difference. It reads only the
logical views, retains no borrow, writes nothing and allocates nothing. Worst-case
work is linear in the equal-length input size; there is no library input cap.
Valid view bounds suffice for its indexing. Invalid slicing can still trap at the
caller before the helper is invoked; this is not a fallible slice-construction API.

Early return makes this proposal unsuitable as a constant-time secret comparison.
It supplies neither authentication nor cryptographic verification. Keep any
timing-resistant comparison separately named and separately evidenced. Equality
also establishes no UTF-8, case folding, Unicode normalization, locale ordering,
file identity, input canonicality, checksum validity or authorization fact.

The proposed return is an ordinary `bool`, not a new Result or sentinel encoding.
Do not add implicit slice equality operators, text overloads, mutable sibling
borrows, generic element comparisons, a sorting API, a native memcmp import or
compiler intrinsic in this proposal. Caller limits, diagnostics and control flow
stay explicit. Existing borrow rules continue to protect mutation/movement while
shared input views are live.

## Acceptance and implementation boundary

The next unit should write a separate ADR with exact equality semantics and the
cost/ownership exclusions above. Follow the
[candidate/accepted/implemented boundary](../adr/0218-modern-language-concept-admission-boundary.md)
and [workload admission gates](../adr/0328-production-workload-capability-admission-gates.md).
An ordinary library solution is sufficient; no language feature is justified.
This assessment does not accept an ADR, reserve an SDK version or promise delivery.

If accepted later, implementation needs SDK source/interface/version and cache
identity activation plus focused source, Result-independent bool use, ownership,
MIR/interpreter and generated/native execution evidence. Use an independent byte
comparison oracle and cover empty/nonempty pairs, unequal lengths, prefixes,
first/interior/last mismatches, all byte values, NUL/invalid UTF-8, distinct owners,
same-owner identical and overlapping subviews, offset sentinels and unchanged
owners. Retain ordinary borrow rejection for mutation or movement during live
shared views. No performance benchmark or constant-time claim is needed for this
simple early-exit contract.

Adoption is a separate bounded unit: replace task-manager equality calls without
changing command names, header format, validation ordering, path/storage limits,
publication/cleanup behavior or error messages. Check and exercise that example
with the current toolchain then, rather than reusing its historical stage0.72
receipt as current proof. Ordering in lexer diagnostics remains unchanged unless
a separately accepted comparison contract is implemented and validated.

This task validates local document links, whitespace, 14 inspected source hashes,
SDK API absence and exact commit scope. It runs no new Fairy check/build/native
test, external round4 suite or performance experiment. Product sources, SDK
version, compiler behavior, roadmap and all external files remain unchanged.
