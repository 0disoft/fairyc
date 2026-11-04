# Hardening review follow-up, 2026-11-04

Windows local evidence is recorded per slice below. Native Linux execution remains
queued for one combined validation; no Linux pass is implied.

## 1. Stable contract-state type depth

Reject more than 256 compound levels before long prefix suffixes reach type
interning. Boundary and 64 KiB prefix-bomb tests run on Windows. Linux: rerun
`go test -mod=readonly ./internal/contractstatemir ./internal/packageartifact`.

## 2. Windows execution ownership

Owned target processes start suspended, enter a kill-on-close Job, then resume.
Normal completion and timeout terminate/wait for descendants before temporary
image removal. Transient image-release/sharing failures receive bounded cleanup
retries; permanent cleanup failures are returned. Windows subprocess tests cover
stdout inheritance, trap receipts and descendants holding the owned image.
Linux: no Windows Job execution claim; rerun the common cdriver suite later.

## 3. Windows project object residency

Candidate publication uses the image-only project builder. Objects are still
validated individually; only paths survive until linking. Existing object-returning
APIs retain their behavior under a 256 MiB aggregate diagnostic-object budget.
MSVC tests compare exact image bytes and execute ordinary/stack-reserved images.
Linux: rerun common candidatebuild/cdriver tests in the combined pass.

## 4. MIR initialization-state snapshots

Single-predecessor chains transfer owned state instead of retaining/copying maps
for every block. Joins and forks retain independent snapshots under a cumulative
1,048,576-entry copy budget. Entry backedges count the implicit caller edge.
Windows MIR tests cover a 2,048-block chain, loop convergence, budget boundaries,
state isolation and the existing ownership/merge mutation cases.
Linux: rerun `go test -mod=readonly ./internal/mir`.

## 5. Generic materialization

The whole expansion checks its cumulative 1,048,576-node output budget before
cloning specializations. Declaration collection no longer makes discarded AST
copies. Windows genericsource tests cover inclusive boundaries, source immutability,
zero work and arithmetic overflow. Linux: rerun genericsource/compiler suites.

## 6. Expanded array operands

One lowering operation shares an inclusive 1,048,576-operand allowance across
functions. Reservation precedes array slice allocation; repeat evaluation remains
single-shot. Windows MIR tests cover exact/cumulative limits, overflow, malformed
shape precedence and cancellation. Linux: rerun MIR/compiler/backend suites.

## 7. Streaming cache publication

Ordinary/generic/storage-transfer persistent workspace miss paths use scoped batches; single Put keeps
its immediate collection. Checkpoints cap pending publications at 256 entries or
16 MiB (or smaller caller bounds); closing/error paths collect remaining records.
No encoded-record collection or mutable cross-process index is introduced.
Windows packagecache/packageartifact tests pass. A 32-record batch asserts exactly
one full collection; boundary, byte/entry retention, failed collection, idempotent
close and borrowed-handle lifetime tests pass. Linux: rerun packagecache,
packageartifact and workspacecompiler suites in the combined pass.

## Validation scope

Windows focused C-driver subprocess/Job/trap tests, MSVC project equivalence,
candidatebuild, contractstatemir, MIR, genericsource, packagecache and packageartifact
checks passed. The expanded `internal/compiler` suite reached its local five-minute
package timeout during native cases, so that attempt is incomplete, not a pass.
The exact active nominal-generics native case and core Check cases are rerun separately.
No full `go test ./...`, Linux execution, hosted CI dispatch, push or release is claimed.

The separated nominal-generics native case and six core Check cases passed on
Windows. The current ordinary workspacecompiler suite and focused eight-package
vet also passed. The 256-block/256-local MIR allocation benchmark compared the
baseline verifier through a Go overlay against the changed verifier (three iterations
per measurement): 4,964,394 to 28,360 B/op, and 2,599 to 283 allocations/op.
Elapsed timings are not used: concurrent unrelated machine builds varied CPU load.

After extending batching to generic/storage-transfer publication, focused generic,
storage-transfer and persistent tests in packageartifact/workspacecompiler passed,
and vet for all affected cache packages passed again.

## 8. Resource observation residency

Only the resource fixture entry point collects exhaustive state observations.
Ordinary ownership checking skips observation-map scans, fact allocation and
sorting. A 256-resource/4,096-expression before/after regression asserts zero
observation allocations instead of 2,097,152 facts. Observed and ordinary analyses
have identical remaining results for valid, use-after-move and branch-join cases.
Windows semantic/resource/semantic-fixture/Region/Result suites and focused vet
pass. Linux: rerun semantic, resourcefixture, compiler and workspacecompiler in
the combined validation.

## 9. Windows project response file

The MSVC link command uses one fixed response-file argument for all object
basenames. Windows linked 4,098 objects from a 68,553-byte response file in a
directory containing spaces and executed the image with exit 23. Two compiler
invocations plus small object copies exercise the admitted link size without
4,096 compiler invocations. Existing low/optimized reproducibility, image-only
equivalence, stack reservation and compile/link failure tests pass; cdriver vet
passes. Linux: rerun common cdriver/candidatebuild tests in the combined pass;
the MSVC native case remains Windows-only.

## 10. Source ZIP metadata admission

A framing walk rejects excess advertised or actual directory entries before
archive/zip metadata allocation. Windows tests cover 8,191/8,192/8,193 entries,
ZIP32/ZIP64, prefixes, maximum comments, malformed ZIP64 and understated counts
and directory sizes. Existing path/identity/budget/validation-order tests and
packagesource vet pass. A 73,728-header baseline accepted by archive/zip allocated
19,381,032 B and 221,234 objects in a single benchmark iteration; admission rejects
the same input with 0 B and 0 allocations. Timings are not used as performance
claims. Linux: rerun packagesource and remote-fetch CLI tests in the combined pass.

## 11. Atlas canonical body reuse

Context Pack v1/v2/v3 and attributed-text v0 encode each logical body once for
both the checksum and final envelope. A shared header/body join checks exact
encoded length (including escaping and header overhead) before final allocation;
the initial body encoding is still O(D) memory. Tests compare whole-struct
canonical bytes, retain the frozen attributed-text checksum, and cover one byte
below/at/above the limit, maximum uint64, escaped/control text and input isolation.
Windows jsonenvelope/contextpack/attributedtext/evidencebundle/architecturecontract
suites, focused Atlas pack/text/evidence CLI tests and vet pass. The synthetic
three-iteration envelope benchmark observed accepted allocation at 4,395,160 to
2,640,272 B/op and rejected allocation at 4,390,864 to 1,320,234 B/op; these are
serialization-only measurements, not whole-CLI or peak-RSS claims.

The full workspacecompiler package and focused ordinary compiler Check cases
also pass after resource observation separation; the remote-package offline CLI
regression passes after ZIP admission. No full compiler/native suite or Linux
execution was repeated. Linux: rerun jsonenvelope, contextpack, attributedtext,
evidencebundle and the same focused CLI tests in the combined validation.

## 12. Nested borrow-flow work

Closed scalar loops retain unchanged borrowed-parameter environments without
duplicate probes. Non-neutral loops share a bounded per-function work allowance
across liveness, summary inference and observable analysis; incomplete proofs
report unsupported analysis and stop that function's remaining flow stages.
Windows tests cover 20 nested parameter-borrow loops under 512 work units,
inclusive/shared limits, non-neutral rejection, cancellation and existing borrow
diagnostic equivalence. Semantic and all semantic/resource/Region/Result fixture
suites pass. Linux: rerun semantic, compiler and workspacecompiler in the combined
validation; no Linux execution is claimed.

## 13. Stable descriptor string counts

String-list admission divides remaining bytes by the eight-byte minimum item
prefix before allocating slots. Windows boundaries cover empty/exact/over-limit
counts, short prefixes and uint64 overflow. A 1 MiB remainder claiming 1,048,576
strings is rejected before any item read instead of reserving 16 MiB of slots
on amd64. Existing stable state/body/cache round trips retain their encoding.
Linux: rerun contractstatemir and packageartifact in the combined validation.

## 14. Linux normal-completion process ownership

The Go artifact runner observes the leader with waitid(WEXITED|WNOWAIT), kills
the original group while its ID remains reserved, then lets exec.Cmd.Wait reap
the leader and drain output. Ordinary nonzero/trap status is preserved; cleanup
errors cannot be mistaken for an ordinary exit. ECHILD never signals a reused ID.
Windows completion-order/error tests pass, as do Linux-amd64 test cross-compilation
and Windows/Linux-target cdriver vet. Native GCC/Clang cases are authored for
inherited/closed pipes, exit 23 and trap receipt 3; execution remains queued with
the existing timeout and binary-input/trap tests. This is not a Linux runtime pass.

## 15. Source generic dependency collection

Function calls, aggregate type references and cleanup calls share one local
exact dependency index per declaration. First-use order is retained; duplicates
do not use the existing 32-entry allowance, and the 33rd unique edge reports the
existing catalog-budget code at the declaration before template serialization.
Windows genericsource/genericplan suites pass, including 32/33/4,000 dependencies
for functions and aggregates, 10,000 duplicate additions, exact tuple identity,
source immutability and failure provenance. Linux: rerun genericsource,
genericplan, compiler and workspacecompiler in the combined validation.

## 16. Source-edit snapshot directory spelling

One fingerprint attempt reuses an exact-case name index for shared parent
directories. A new attempt gets a fresh index; segment Lstat, symlink rejection,
regular-file/size bounds and opened-file identity checks remain per read. The
loader's 100,000-entry ceiling also bounds index residency and enumeration;
failed reads/closes do not publish partial listings and directory handles close
on success, I/O errors and budget failure. Windows tests cover a 256-member
workspace-shaped catalog (770 to 258 directory enumerations, shared-parent
enumerations 514 to 2), case/path/symlink rejection after cache hits,
99,999/100,000/100,001 entries, cross-directory allowance, failure recovery,
unchanged fingerprints and manifest/lock changes between attempts. Linux: rerun
sourceedit, workspaceloader and CLI TestAtlasEdit* in the combined validation.

Windows sourceedit and Atlas edit CLI suites pass, including native apply/build
execution and both file/directory symlink replacement cases. The ordinary
workspacecompiler suite, genericsource/genericplan suites, focused compiler Check
cases and sourceedit/genericsource/genericplan vet also pass. No full repository
suite, Linux runtime execution, CI dispatch, push or release was performed.

## 17. Result flow work (2026-11-05)

Closed scalar loops preserve pending and handled incoming Result states without
duplicate back-edge visits. Other Result control flow shares a per-function
1,048,576-unit work allowance; incomplete analysis reports unsupported once and
does not reach completed Flow or later MIR. Windows semantic and semantic,
resource, Region and Result fixture suites plus semantic vet pass. Tests compare
diagnostics/facts with forced baseline visits and cover 20 nested loops under
512 units, exact/overflow limits, per-function reset and cancellation. Linux:
rerun semantic, semanticfixture, resourcefixture, regionfixture, resultfixture,
compiler and workspacecompiler in the combined validation.

## 18. Streaming package extraction (2026-11-05)

Ordinary/generic/storage memory and persistent miss paths publish one independently
verified package before extracting the next. Shared symbol metadata and owner
membership survive; a complete slice of cloned workspace contexts does not. For
P packages and C cloned declaration metadata, producer context residency becomes
O(C), replacing O(P*C); this is not a measured whole-process peak-RSS claim.
Existing ExtractAll APIs retain their independent-copy contracts. Windows
packageartifact/workspacecompiler suites and vet pass. Tests compare complete
artifact values/order, source and sibling isolation, and immediate stop on callback
failure/cancellation before the next package is extracted. Persistent scoped batches
still close on early errors. Linux: rerun packageartifact, packagecache,
workspacecompiler and workspace cache/session CLI tests in the combined validation.

## 19. Process cache JSON admission (2026-11-05)

Both ordinary and storage process codecs measure canonical envelope size after
verification and before json.Marshal. The existing closed counter now admits
only the two explicit PackageID-to-digest maps, with allocation-free key counting;
general map schemas remain rejected. Windows tests retain exact encoded bytes,
one-byte-below/at/above admission, nil/empty maps, maximum uint32 keys, escaping,
verification-before-size precedence and bounded canonical decode re-encoding.
A 16 MiB field rejected at 1 KiB allocates under 1 MiB at the marshal boundary;
artifact verification has separate graph-copy costs. Linux: rerun packageartifact,
workspacecompiler and workspace cache/session CLI tests in the combined validation.
