# Ordered work - 2026-10-31

- Status: Approved follow-up implementation queue
- Updated: 2026-11-02
- Supersedes: current-status claims in the [2026-10-23 ordered work](completion-plan.md#historical-ordered-work-superseded-2026-10-31)
- Change boundary: ordered planning and bounded implementation evidence
- Authorization: named sequence implementation; no implicit syntax, API or target support

This is the current ordered queue. Item 1 is complete and the list below is the resulting work
record. Item 2 awaits native Linux evidence. Item 3 has Windows evidence with Linux deferred;
item 4 has Windows evidence for the named production-object corpus; item 5 has bounded Windows
weak-chain and CRT selection evidence; item 6 has bounded Windows default-library and CRT
failure evidence; item 7 has bounded production archive/COMDAT evidence; item 8 has bounded
unwind/handler-data evidence; item 9 has locally verified CI wiring with hosted execution pending;
item 10 has bounded Windows debug retention/lookup evidence; item 11 has named Windows CRT/DLL
dependency evidence; item 12 has a bounded Windows-verified opt-in review tool;
item 13 has bounded Windows record-array evidence with Linux deferred; item 14 has exact C-scalar
pointer evidence with Linux deferred; item 15 has ordinary synchronous callback evidence with
Linux deferred; item 16 has zero-field and bounded nested owning-tag evidence with Linux deferred;
items 17 and 18 have bounded Windows language evidence with Linux deferred; item 19
has a backend decision and Windows C baseline; items 20 and 21 have bounded Windows
tool evidence with Linux deferred. Item 22 now has a Windows-verified Go source runner
and local persistence prototype; chain selection and generated-C persistence remain open.

The Go/Fairy compiler parity follow-up now has Windows evidence for bounded inline
source admission, typed body identity, current-source cache restoration, v5 keys and
shared-header C execution. See the [inline receipt](../ops/windows-candidate-inline-20261102.md).
The newer generics, interface and closure families remain separate small slices.

`Next / in progress` includes the routine bounded-contract and implementation work for that row.
It is not a request for a fresh approval of the same task.

Approval of this queue authorizes implementation of the named sequence. It does not by itself
admit a grammar form, public API, ABI, backend choice, target, package registry, persistence
product or distribution claim. Where a design is not selected, Engineering writes the bounded
contract in the same work slice: input and output surface, ownership and lifetime rules, limits,
failure and negative cases, compatibility impact, and exact evidence. The implementation proceeds
under that contract without another routine approval. The owner is asked only when adopting a
technology or changing scope would make a decision on their behalf; backend and target choices
remain `UNDECIDED` until an explicit owner decision.

## Queue order

| Order | Work | Status | First bounded contract or next gate | Completion condition |
| --- | --- | --- | --- | --- |
| 1 | Reconcile the current work list | **Complete** | Make this document the current queue, link it from `completion-plan.md`, preserve older records as history, and make later completion evidence supersede an earlier pending statement. | This queue and its status vocabulary are recorded in the two documentation files. |
| 2 | Fairy-authored GNU companion | **Next / in progress** | Define and implement one private companion boundary for an already prepared Linux packet: compiler discovery, argv/stdin/environment handoff, stream capture, diagnostics, deadlines and process cleanup. Failure is explicit; the existing contract has no automatic frontend fallback to Go. Do not admit distribution or replace the default compiler. | The companion executes the named Linux build path from Fairy, preserves bounded cleanup and exact diagnostics, and has Linux-native plus Windows-development evidence. Go remains the default until a separate promotion decision. |
| 3 | Aggregate transport in both directions | **Windows verified / Linux deferred** | Code `d558eb1f` checks 116 portable record shapes in separate generated-Fairy/C objects under MSVC C11/C17 low/optimized, plus two C-to-Fairy assertion-trap executions. The reverse entry is a fixture-only C bridge, not a public export or aggregate callback. | The bounded Windows matrix and six unsupported-shape checks pass. The 122-shape POSIX matrix emits locally; native cross-compiler execution is in the batched queue below. Full public-export and target conformance claims remain closed. |
| 4 | COFF relocation coverage | **Windows verified for named corpus** | Code `07fd28a9` inventories actual `unit.obj`/`runtime.obj` from five SDK workspaces under ordinary C11 low/optimized flags, with map/library-log capture. It checks every retained REL32 and ADDR32NB field against its owned source/target positions and final PE bytes. | Ten native executions and twenty objects account for 3,133 fields: 2,404 linked-value checks and 729 explicitly discarded fields. Object addend, final-field, map-owner and discard-evidence mutations fail. This is the named corpus, not every possible Fairy program or CRT archive member; unobserved relocation kinds retain their earlier separate evidence. |
| 5 | Weak linking: chains, cycles and replacement | **Windows verified for named cases** | Code `cbd3b26a` inventories actual generated/runtime objects, distinguishes class-105 records from linker directives, and binds explicit-object chains, strong replacements and strong cuts of cycles to map owners and final PE fields. Actual CRT startup and SDK data alternatives are exercised separately. | Forty controlled selections, four SDK data cases, two CRT startup cases and ten corpus images pass; four unresolved cycles fail in both the inspection planner and real LINK with LNK2019/LNK1120. This does not add Fairy weak syntax, reverify all archive search modes or establish general COMDAT removal. |
| 6 | Default libraries and CRT archive | **Windows verified for named cases** | Code `53d7ed5f` records actual /MT object defaults, absolute searched archive paths and hashes, loaded CRT members, owned final symbols and PE entry at low/optimized. It verifies the five production workspaces and a C control relinked with absolute inputs and no default search. | Sixteen native executions and twenty expected link failures cover missing archives, suppressed defaults, removed demanded members and absent-but-unused OLDNAMES. Ordinary selection uses recorded toolchain directories; explicit relinks have empty LIB/LIBPATH. Broader archive/COMDAT policy, other CRT modes, clean-machine transfer and target promotion remain separate. |
| 7 | Archives and COMDAT | **Windows verified for named forms** | Code `e9ca8758` binds five selected CRT objects to both archive indexes. Matrix `4f300ea1` checks ordinary, NODUPLICATES, ANY and direct associative unwind forms in actual generated/runtime objects and selected CRT members, plus existing controlled archive/direct competition. | Twenty native executions and two expected LNK2005 failures pass. The production corpus accounts for 226 parents and 462 unwind children; exact archive selection, retained fields, discarded payloads and input-order behavior have bounded receipts. Unobserved policies, arbitrary CRT members, chained/non-unwind associations, runtime unwinding and target promotion remain separate. |
| 8 | Unwind v2, handlers and chained frames | **Windows verified for named states** | Code `4ae0abbe` binds C/GS scope payloads to object relocations, final bytes and live C dispatch; `a025545d` accounts for remaining CRT fragment entries; `01d30ff6` checks fixed RBP/XMM6 chained states and exact tails. Existing v2 and mixed-frame probes are rerun. | Twenty native executions, including eight expected traps, pass; controlled unwind contexts total 204. Scope payload mutations, incorrect saved/frame states and changed metadata fail. Extra CRT fragments have entry-only evidence. Arbitrary handler grammars, frame/opcode families, production chain emission, full conformance and hosted CI remain separate. |
| 9 | Same-commit mixed-stack, callback and trap CI | **CI wired / hosted pending** | The existing Windows native smoke reuses its ten executions to record ordered frames, handler/data/caller addresses, synchronous callback results and private assertion traps. Its checked-out source revision and source/object/image hashes are bound into one validated CI receipt. See the [local verification and CI contract](../ops/windows-mixed-stack-ci-20261101.md). Retained/asynchronous callbacks remain excluded. | Windows local execution, receipt rejection tests and workflow syntax pass. Completion still requires the authorized hosted lane to pass the receipt on that exact commit. No push or workflow dispatch was performed; local verification is not hosted evidence. |
| 10 | CodeView, PDB and DbgHelp retention and lookup | **Windows verified for named debug scope** | Private generated-C debugging retains PE/PDB, actual Fairy/C inputs, independent objects and map under the image SHA-256. A fresh process resolves generated entry `main` and C boundary `fairyc_debug_boundary` to exact function starts and retained C lines in low/optimized. See the [debug contract and evidence](../ops/windows-debug-retention-20261101.md). | Four fresh-process lookups, two generated-Fairy native executions, both profiles' CodeView pairs and corruption/identity/lookup rejection controls pass. Storage `a5e5a7c0` and exact end-label fix `6383494f` support the lookup slice. This is local private retention and named generated-C attribution, not Fairy source variables/types, arbitrary optimized/inline/ICF attribution, a public symbol server or hosted artifact retention. |
| 11 | UCRT, VCRuntime and dynamic DLL dependencies | **Windows verified for named paths** | `00a82403` inventories ordinary PE imports; `6b906918` records actual child LOAD_DLL/UNLOAD_DLL events with an empty PATH; `f34827f3` restricts PSAPI to the system loader path. See the [CRT/DLL contract and receipt](../ops/windows-runtime-dependencies-20261101.md). | Twenty-one named traces cover five SDK workspaces in both profiles, original rooted adapter operations and six missing-export controls, two retained-PDB developer lookups and Go CLI startup. Two further loader mutations fail with DLL-not-found/entrypoint-not-found. Import functions, system paths/hashes and static CRT owners are verified; PSAPI memory queries pass separately. Arbitrary execution paths, other CRT modes and clean-machine transfer remain separate gates. |
| 12 | Interface/effects diff and large-source notice | **Windows verified for bounded tool** | `03195a2e` compares verified root-package ordinary v5 interface facts and effects; `586ecba4` adds a metadata-only source-size scan. `fairyc-review diff/notice` exposes text/JSON with explicit exact/bounded/partial/unknown evidence. See the [contract and Windows validation](../ops/interface-source-review-20261101.md). | Deterministic signature/layout/effects and add/remove facts pass; generic-module or unavailable-input gaps cannot become unchanged/removed declarations. A built Windows executable verifies advisory and 0/1/2 behavior, including read-only workspaces. Strict >40,000-byte notices preserve source and skip declared output/vendor directories and symlinks. No product/compiler/CI admission change; behavior/ABI compatibility and broader generic review remain unknown. |
| 13 | `@repr(C)` record-element arrays and additional arrays | **Windows verified / Linux deferred** | Code `e5b03f3f` and the final probe admit nonempty acyclic `@repr(C)` record fields `Element[N]` for `N=1..4096`. Elements retain existing ABI leaves and admitted scalar, enum, nested-record and record-array fields. Actual C struct arrays and per-element assignment carry values; bare arrays, direct array-of-array elements, new nonplain element operations and nested stores for the new shape stay closed. See the [Windows receipt](../ops/windows-record-array-transport-20261101.md). | The final clean package pass covers 133 portable/139 POSIX emission shapes, 428/448 functions, four forward native profiles with 12 record-array cases each, four reverse profiles at 133 records x 3 sets, and two assertion traps. Seven focused packages have full tests; compiler coverage is filtered and scoped vet passes. Native Linux execution remains in the batched queue. |
| 14 | Raw-pointer targets, operations and nesting | **Windows verified / Linux deferred** | Code `3ea7a217` adds exact `ptr c::T` and `ptr mut c::T` for the sixteen existing C scalars when the target supplies them, plus one mutable `ptr mut ptr mut c::T` output slot. The old `void **` shape remains; other qualifiers, depth-three pointers, record/array/Fairy-scalar nesting and every raw operation outside the listed borrow casts stay closed. See [ADR 0489](../adr/0489-c-scalar-pointer-output-slots.md) and the [Windows receipt](../ops/windows-c-scalar-pointers-20261101.md). | Focused compiler, full C backend, interface/package identity and Windows probe pass. Windows emits 15 portable targets and POSIX 16; four MSVC C11/C17 low/optimized executions cover 15 types x 3 rotations = 45 cases each. Native Linux remains pending; scoped vet passes. |
| 15 | Ordinary callbacks | **Windows verified / Linux deferred** | Code `c45fc974` and [ADR 0490](../adr/0490-ordinary-synchronous-callbacks.md) extend exact signatures to 0-8 scalar/pointer values plus explicit user data, scalar/pointer or void results, and named unsafe targets without effects. Retained/asynchronous/cross-thread delivery, captures, by-value aggregates, callback storage, unwinding and longjmp remain excluded. | The [Windows receipt](../ops/windows-ordinary-callbacks-20261101.md) records MSVC C11/C17 low/optimized, scalar boundaries, mixed floating/stack arguments, typed data, pointer slots and nested Fairy-to-C helper calls; partitioned callback identity also executes at both profiles. Linux emission passes; native crossing is queued below. |
| 16 | Zero-field resources and nested owning tags | **Windows verified / Linux deferred** | Zero-field code `24561cbd` and [ADRs 0491/0492](../adr/0492-nested-owning-tag-cleanup.md) define initialized private-byte storage and up to eight owning Option/Result layers. Complete-root moves and recursive active cleanup retain existing rules. | The [Windows receipt](../ops/windows-resource-forms-20261101.md) records zero-field low/optimized and combined C11/C17 low/optimized passes, 27 exact cleanup events per combined run, interpreter/native trace agreement and process/disk cache restoration. Malformed extraction, partial moves, use-after-move, MaybeLive use and unsupported payload/field/depth shapes reject. Owning resource fields and arbitrary tagged families remain closed; native Linux is queued below. |
| 17 | Cross-package inline and cache | Windows verified for bounded Go and Fairy inline prototypes; Linux deferred | [ADR 0493](../adr/0493-cross-package-inline-bodies-and-cache-v5.md) reopens Phase 3 for public safe nongeneric i32 arithmetic exports, emitted with private `static inline` bodies and shared-header partition visibility. [Cache key v5](../package/whole-package-cache-v5.md) binds owner and dependency inline identities. | [Windows receipt](../ops/windows-cross-package-inline-20261101.md): C11/C17 low/optimized native execution, ordered once-only argument evaluation, split-TU execution, memory/fresh-session disk invalidation and twelve ordinary/generic mismatch-repair cases pass. Exact-generation quarantine preserves concurrent records. The [candidate inline receipt](../ops/windows-candidate-inline-20261102.md), sources `324042ea`, `61ac698f` and `39365ee3`, adds actual bounded source admission, Go-identical body/package identities, missing/extra-marker rejection and shared-header C execution at both profiles. Broader inline forms remain closed; native Linux is queued below. |
| 18 | Broader generics, interfaces and closures | **Windows verified for three bounded prototype families / Linux deferred** | [ADR 0494](../adr/0494-broader-scalar-generic-transport.md) adds seven direct scalar generic families; [ADR 0495](../adr/0495-explicit-static-operation-interfaces.md) binds explicit single-operation scalar interfaces to exact implementations; [ADR 0496](../adr/0496-explicit-copied-local-closures.md) lifts copied i32 captures into local private helpers. | [Windows receipt](../ops/windows-language-expansion-20261101.md): sixteen native C11/C17 low/optimized executions, interpreter equivalence, four captures plus four arguments, ordered once-only arguments, source rejection/immutability, memory/fresh-session disk and provider/body invalidation pass. Representative MIR/C/image sizes and native build times are recorded. Inference, wider nominal arguments, dynamic/multi-method interfaces, first-class/escaping closures and stable SDK/candidate promotion remain outside this bounded admission; native Linux is queued below. |
| 19 | QBE, LLVM or additional targets | **Review and Windows C baseline complete / challengers deferred** | [ADR 0497](../adr/0497-backend-and-target-review.md) retains the C backend, prioritizes external textual LLVM IR only as a future measured challenger, and defers QBE adoption. [Target comparison](backend-target-review-20261101.md) keeps additional target selection UNDECIDED. | [Windows baseline receipt](../ops/windows-backend-study-20261101.md), harness `769d5835`: seven workloads, two profiles and three samples yield 42 passing native executions plus interpreter/C observers and structural PE/unwind checks. LLVM/QBE remain unmeasured. New backend/target adoption requires representative paired benefit and semantic/runtime/ABI/debug evidence; Linux execution remains deferred. |
| 20 | Remote packages | **Pinned HTTPS preparation verified on Windows / Linux deferred** | [ADR 0498](../adr/0498-explicit-pinned-remote-package-sources.md) admits [exact source plans](../package/remote-sources-v1.md), bounded TLS ZIP retrieval, SHA-256 and manifest identity validation, explicit local member publication and verified offline cache. Registry selection remains UNDECIDED. | [Windows receipt](../ops/windows-remote-packages-20261101.md): cold/warm/offline restoration, rejected unsafe archives/cache/conflicts, unchanged graph lock, stale dependency detection and native low/optimized package calls pass. Range solving, registry services, Git, signing, automatic upgrades and Fairy-authored candidate parity remain separate; native Linux is queued below. |
| 21 | Atlas impact, MCP and structural edits | **Bounded Go tools verified on Windows / Linux deferred** | [ADR 0499](../adr/0499-bounded-atlas-impact-mcp-and-body-edits.md) admits [current resolved-call impact, explicitly permitted read-only MCP stdio, and private single-body preview/apply](../atlas/impact-mcp-edit-v1.md). Each retains current identities and bounded/partial uncertainty. | [Windows receipt](../ops/windows-atlas-tools-20261102.md): reverse traversal/cutoffs, stale/unknown data, protocol permissions, body verification, injection/drift/link rejection and two edited native profiles pass. A standalone MCP subprocess negotiates and lists only the read tool. Full semantic diff, inferred rename, multi-file edits, MCP writes and candidate parity remain separate; Linux is queued below. |
| 22 | Chain execution and persistence | **Bounded Go local execution verified on Windows / chain integration open / Linux deferred** | [ADR 0500](../adr/0500-local-contract-state-execution-and-persistence.md) connects verified source state MIR to one caller-visible persistence attempt and a bounded rooted local store. `fairyc-state init/execute/snapshot` exposes the existing experimental source subset. Chain/VM selection stays **UNDECIDED**. | [Windows receipt](../ops/windows-local-state-20261102.md): commit/reject/trap, no automatic retry, restart, independent-process locking, forced exits before/after publication, stale writers, exact layout guards, corruption and symlink rejection pass. Six built-Windows-executable calls persist u64 42 and preserve committed bytes across reject/trap. This is Go MIR execution; generated-C durable integration, power-loss certification and chain runtime/deployment remain separate. |

## Item 2 progress: authored GNU companion implementation

Commit `11b781f3` adds authored toolchain policy and ELF64 admission. The reported focused
tests pass both-target emission and Windows MSVC low/optimized. ELF inspection is bounded to
64 MiB with a 4 MiB name scan; extended section numbering remains unsupported.

Commit `53840962` adds `linux_runtime.fy` and `linux_runtime_contract.fy` with
`TestLinuxGNUCompanionRuntime` as the Go oracle. The current comparison covers 105 plans from
the authoritative Go standard-library and host sources, with both-target emission and Windows
low/optimized passing. Unknown standard-library, vector, Linux-host, and Windows-dependency
markers are rejected.

Commit `13243c0d` adds `linux_runtime_assets.fy` and a generator that pins 25 source-owned raw
C files in a 138,102-byte runtime pack. The loader verifies the exact source count, size and
SHA-256 pin. The reported Linux runtime checks compare 48 complete C outputs in low/optimized
profiles. The GNU build workflow and POSIX-only C bridge now cover toolchain admission,
separate compilation and linking, bounded stage I/O, ELF admission, descendant-aware process
capture and replacement publication. Windows MSVC low/optimized policy tests exercise real
Fairy compile/link/publish code; the C bridge only runs there through POSIX substitutes, so no
Linux-native execution is claimed.

The `linux_gnu_companion.fy` entry matches the private CLI flags and
loads the fixed sibling `fairyc-linux-runtime-assets.pack` through the Linux rooted reader.
`tools/prepare-candidate-linux-cli.py --companion` selects it in an isolated main. The matching
companion and pack require explicit opt-in installation; the original Go backend remains the
default and has no automatic replacement. Option checks pass under MSVC low/optimized;
complete entry emission passes for both targets with closed Linux support and no Windows
foreign symbols. The companion preparation mode only selects the entry; frontend/cache
platform rewrites stay with the ordinary CLI preparation mode.

Fresh original-workspace `check`/`check --locked` pass with 574 sources and 7,232 MIR
functions; the isolated companion workspace passes with 574 sources and 7,230 functions.
The focused companion suite and GNU adapter boundary checks pass, as do runtime-pack pin
verification, Python syntax checks, scoped Go vet and diff checks. Native adapter and complete
companion tests are authored but skipped on Windows; WSL reports
`Wsl/CallMsi/Install/REGDB_E_CLASSNOTREG`. Earlier Debian evidence covers the Go backend. Native
Linux GNU launch, fixed-point reproduction and distribution remain unverified. Item 2 remains
**Next / in progress** and the target maturity state is unchanged. This progress introduces no
new owner approval gate and does not trigger push, CI, release or deployment.

## Item 3 progress: aggregate call transport

Code `d558eb1f` introduces the shared bidirectional probe and fixes accidental backend admission
of Fairy `bool` as a repr(C) field; exact `c::bool` remains supported. Windows MSVC 14.51.36231
passes 116 record shapes with three field/element-varied C value sets in C11/C17 low/optimized.
Each execution retains two object receipts and checks final PE/unwind structure, status 42 and
empty output. A fixture-only bridge enters actual generated Fairy reconstruction bodies from C;
the marker and independent C observer check their execution and every result leaf. Two additional
low/optimized executions prove assertion-trap delivery on the private channel inside that path.
Six unsupported aggregate forms fail explicitly. Both target emission and focused backend tests,
scoped Go vet and diff checks pass. The [receipt](../ops/windows-aggregate-crossings-20261101.md)
states the exact boundary: no public exports, aggregate callback signatures, retained pointers,
ownership transfer, foreign exception unwinding, hosted CI or target promotion.

## Item 13 progress: repr(C) record-array transport

Code `e5b03f3f` and its final probe add actual C structure arrays for bounded elements of nonempty
acyclic `@repr(C)` records. The final [Windows receipt](../ops/windows-record-array-transport-20261101.md)
records 133 portable/139 POSIX emission shapes and 428/448 generated functions. The complete
foreign-struct package passes in 19.670s; only the user-deferred Linux native test is skipped. Four
forward MSVC C11/C17 low/optimized executions run 12 record-array cases each, including the
4,096-element family. Four reverse executions carry 133 records x 3 value sets through separate
objects, and both assertion traps pass with status 134 and control byte 6. Length 1/3/4,096,
target LLP64/LP64 layout, MIR mutation, nominal interface identity and cache-context rebinding
checks pass. Seven packages have full tests; compiler coverage is limited to the record-array and
aggregate filter. Native Linux execution remains deferred.

## Item 14 progress: exact C-scalar pointers and output slots

Code `3ea7a217` and the [Windows receipt](../ops/windows-c-scalar-pointers-20261101.md) admit
exact `ptr c::T`, `ptr mut c::T` and one `ptr mut ptr mut c::T` output slot for the sixteen
existing C scalars subject to target availability. Windows has 15 because it lacks `c::ssize_t`;
`c::long` follows LLP64 there and LP64 on the POSIX target. The old exact `void **` slot
remains. Other qualifiers, depth-three pointers, nested record/array/Fairy-scalar pointers and all
raw dereference, arithmetic, retyping, integer cast, mutability strengthening and safe-reference
construction stay closed.

The implementation and core tests are local commit `3ea7a217`. The native probe and receipt are
in the commit containing `docs/ops/windows-c-scalar-pointers-20261101.md`. The fixture-only C
bridge calls actual generated Fairy identity functions for const, mutable and `T **` shapes; it
is not a public export or callback surface.

Focused compiler tests pass all 16 identities and MIR codec round trips, reject Windows
`c::ssize_t`, test the local-only POSIX pointer provider header/size assertion, reject ten
unsafe/type negatives, and reject forged raw-address/raw-borrow MIR provenance, pointee and
strengthening mutations. The full C backend package passes in 30.217s; full module-interface and
package-artifact packages pass with exact target/qualifier/depth identity and all 16 x 3 pointer
shapes surviving context-descriptor rebind. The filtered foreign-struct probe passes in 3.770s,
including 15 Windows/16 POSIX emission inventories and four MSVC 14.51.36231 C11/C17
low/optimized executions with 45 cases each. Separate caller/callee objects, PE/AMD64 unwind
inspection, exit 42 and empty streams pass.

Native Linux execution is deferred: the guarded `TestCScalarPointersLinux` Clang/GCC
cross-helper compiles and skips correctly under Windows, and its native run has not happened.
Scoped `go vet` passes with exit 0, and the broader compiler filter
`(CScalarPointer|RawPointer|RawAddress|Opaque|Borrow.*Pointer|IntegerTransportEmissionBoundaries)`
passes in 0.517s. A full compiler-package run, full `go test ./...`, hosted CI, push and
publication remain unexecuted. The language and C ABI subset extend; grammar, MIR opcodes and
codec format, runtime, SDK, serialized formats, dependencies, CLI and CI are unchanged. ADR
0488's historical proof is untouched; only its blanket new-pointer closure is superseded for
this named subset.

## Item 4 progress: generated and runtime COFF relocations

Code `07fd28a9` adds `InspectGeneratedRelocations` and the native
`TestGeneratedRelocationCoverage`. The corpus is `std_bytes_quickstart`,
`std_alloc_quickstart`, `std_text_from_buffer_quickstart`, `std_vec_quickstart` and
`process_args_quickstart`. Each is checked/lowered/emitted from Fairy source; native support is
selected from its actual foreign-symbol set. No handwritten object replaces generated output.
MSVC 14.51.36231 passes low/optimized builds, final-value assertions and exact native outcomes.

REL32 accounts for 1,510 object fields (1,322 linked, 188 discarded); ADDR32NB accounts for 1,623
(1,082 linked, 541 discarded). Source section identity, object addend, target resolution and final
image bytes are recorded for every retained field. Pdata is matched by unique begin/end pairs;
code/data section bases use owned map symbols. COMDAT omission requires the existing exact
selection/discard proof. Unknown kinds and unresolved ownership fail rather than disappearing
from the counts. The [receipt](../ops/windows-coff-relocations-20261101.md) records commands,
source/object/image hashing, negative controls and the bounded scope. Full PE/COFF unit checks,
scoped Go vet and diff checks pass. This Windows-only slice does not add a Linux job or change
the deferred Linux queue.

## Item 5 progress: weak chains and actual alternatives

Code `cbd3b26a` and the [Windows receipt](../ops/windows-weak-aliases-20261101.md) cover
class-105 ALIAS records and C linker directives in low/optimized profiles, with forward/reversed
object order. Fallback, root/middle strong replacement and strong cuts of a two-node cycle bind
both call sites to exact selected owners and final displacement bytes. An unresolved cycle
fails in the bounded inspection planner and independently in LINK, with two unresolved symbols.
The inspection helper does not add a new production driver policy.

The production corpus has no class-105 weak records. The process-argument runtime offers one
SDK data alternative, but it has no root references and neither name is in its final map.
Separate real-header C probes prove fallback value 0 and strong provider value 7 at both
profiles. Actual `libcmt.lib(initializers.obj)` metadata and startup map/library logs establish
strong UCRT/VCRuntime initialization selection in two more executions. The named suites pass
56 native executions and four expected LINK failures, plus PE/COFF unit checks, scoped Go vet
and diff checks. Archive search/absence, broader COMDAT removal, hosted CI and target promotion
remain separate. Linux work stays in the unchanged deferred queue below.

## Item 6 progress: default libraries and CRT archive failures

Code `53d7ed5f` and the [Windows receipt](../ops/windows-default-libraries-20261101.md) bind
ordinary `/MT` low/optimized object defaults to exact searched archive paths/hashes and loaded
CRT owners. Startup must be `LIBCMT:exe_main.obj` at the final PE entry; UCRT/VCRuntime
initialization must have their expected strong archive owners and executable final addresses.
All five generated/runtime workspaces pass again with source/object/image and library receipts.

The C control also links and runs from the same objects with default search disabled, absolute
library inputs and empty `LIB`/`LIBPATH`. Missing explicit CRT archives fail with LNK1181;
suppressing required defaults or removing their demanded actual members fails with LNK1120.
An isolated directory missing unused OLDNAMES still fails default lookup with LNK1104, while
suppressing that unused default succeeds. Sixteen positive executions and twenty expected
link failures pass, alongside PE/COFF unit checks, scoped Go vet and diff checks. Installed
archives remain unchanged. Broader archive/COMDAT forms are next; Linux remains deferred below.

## Item 7 progress: production archive and COMDAT forms

Code `e9ca8758`, matrix `4f300ea1` and the
[Windows receipt](../ops/windows-archive-comdat-20261101.md) cover the actual observed policies:
ordinary sections, NODUPLICATES, ANY and direct associative unwind sections. Five selected CRT
objects agree with both archive indexes, loaded-member logs and owned map symbols. Three named
CRT functions bind direct pdata/xdata fields at both profiles; two small roots without those
children are recorded separately. Duplicate selected startup objects fail with LNK2005.

The five-workspace generated corpus accounts for 226 parents (76 retained, 150 discarded) and
462 unwind children (194 retained, 268 discarded), with 402 child RVA fields, 97 final pdata
entries and 1,572 xdata bytes checked. Existing controlled archive probes additionally prove
unused-member non-extraction, competing ANY values and removal of the losing parent's distinct
unwind payload under both input orders. Twenty native executions and two expected failures,
PE/COFF unit checks, scoped Go vet and diff checks pass. Broader unobserved forms and runtime
unwind semantics remain outside this receipt; Linux stays in the unchanged deferred queue.

## Item 8 progress: bounded unwind behavior and handler data

The [Windows receipt](../ops/windows-unwind-completion-20261101.md) records sources `4ae0abbe`,
`a025545d` and `01d30ff6`. Six C/GS scope tables bind eight records and 29 relocated scope fields
to final bytes; the body tables also match live handler/data pointers. Normal return, inner catch and outer
catch prove exact filter/finally order, application payload and abnormal status; 57 payload
mutations are rejected. The optimized constant handler uses literal sentinel 1.

The CRT chain partition now accounts for 37/41 fragments, including 12/14 additional entry-only
contexts. A closed RBP/XMM6 chain fixture adds twenty contexts across both profiles, with exact
tail fields, partial prolog/epilog state and wrong-frame/vector controls. Existing v2 epilogs,
original chain tails and mixed Fairy/C walks/traps pass again. Twenty native executions and
204 controlled unwind contexts pass, as do PE/COFF and probe unit checks, scoped Go vet and diff
checks. Arbitrary handler/opcode/frame forms, production chain emission and full target
conformance remain unverified. Hosted same-commit CI stays item 9; Linux remains deferred below.

## Batched native Linux validation, 2026-11-01

The maintainer requested deferred, combined Linux validation because this laptop has no usable
WSL environment. Continue bounded Windows work without repeating WSL discovery or provisioning.
Unexecuted Linux cases remain unexecuted and do not block the next Windows work item.

| Pending slice | Native command from the repository root | Current evidence |
| --- | --- | --- |
| Fairy-authored GNU companion entry, runtime pack and publication | `go test -mod=readonly ./internal/cli -run '^TestLinuxGNUCompanionNative$' -count=1 -v` | Windows-development and entry emission checks pass; no native execution for this companion. |
| GNU POSIX adapter | `go test -mod=readonly ./internal/candidatehost -run '^TestLinuxGNUAdapterNative$' -count=1 -v` | Windows POSIX substitutes pass; native test remains unexecuted. |
| Aggregate calls and record arrays in both directions | `go test -mod=readonly ./internal/foreignstructprobe -run '^TestBidirectionalAggregateLinux$' -count=1 -v` | POSIX 139-record emission with 448 generated functions passes on Windows, including the 4,096-element record-array family. The authored native test links Clang/GCC in both compiler directions and low/optimized profiles; it has not run. |
| Exact C-scalar pointers and `T **` output slots | `go test -mod=readonly ./internal/foreignstructprobe -run '^TestCScalarPointersLinux$' -count=1 -v` | Windows emission covers 15 portable/16 POSIX targets x 3 shapes. The guarded helper compiles and skips under Windows and links Clang/GCC in both directions at low/optimized; its native run has not happened. |
| Ordinary synchronous callbacks | `go test -mod=readonly ./internal/foreignstructprobe -run '^TestOrdinaryCallbacksLinux$' -count=1 -v` | Windows C11/C17 low/optimized pass with 27 scalar families, 0-8 value arguments, void return, mixed float/integer stack arguments, typed context and pointer slots. The 28-family POSIX unit emits locally. The guarded Clang/GCC cross-helper compiles and skips on Windows; native execution remains unexecuted. |
| Zero-field resources and nested owning tags | `go test -mod=readonly ./internal/compiler -run '^Test(ZeroFieldResourceNative|NestedOwningResourceLinux)$' -count=1 -v` | Windows execution and Linux-profile production/instrumentation emission pass. The guarded nested-tag Linux helper compiles and skips locally; later native runs cover GCC/Clang low/optimized and one ASan/UBSan execution per compiler for each root. |
| Cross-package inline and cache v5 | `go test -mod=readonly ./internal/workspacecompiler -run '^(TestCrossPackageInlineLinux|TestCrossPackageInlineCacheInvalidation|TestInlineCacheRejects)' -count=1 -v` plus `go test -mod=readonly ./internal/cbackend -run '^TestPartitionedInlineBody$' -count=1 -v` | MSVC C11/C17 low/optimized and private shared-header partition execution pass on Windows. The guarded GCC/Clang low/optimized native probe compiles and skips locally; Linux execution remains unexecuted. |
| Go/Fairy candidate inline, cache v5 and packet parity | `go test -mod=readonly ./internal/cli -run '^(TestCandidateCacheKeyV5Parity|TestCandidateInlineAdmissionBoundary|TestCandidateInlineIdentity|TestCandidateInlinePacketParity)$' -count=1 -timeout 5m -v` | Windows low/optimized identities, actual admission, source-derived keys, restoration and generated C pass; see the [receipt](../ops/windows-candidate-inline-20261102.md). The named lightweight probes have unexecuted Linux paths. The opt-in complete-key and pipeline private build harnesses remain Windows-only and need a Linux companion path; a skip is not native evidence. |
| Broader scalar generic transport | `go test -mod=readonly ./internal/compiler -run '^TestBroaderGenericsScalarTransport$' -count=1 -v` and `go test -mod=readonly ./internal/workspacecompiler -run '^TestBroaderGenericsScalarCache$' -count=1 -v` | Windows C11/C17 low/optimized, seven new direct scalar families, memory/fresh-session disk and both-target C emission pass. The same native helper selects GCC/Clang low/optimized on Linux; that execution remains deferred. |
| Static operation interfaces | `go test -mod=readonly ./internal/compiler -run '^TestStaticInterface' -count=1 -v` and `go test -mod=readonly ./internal/workspacecompiler -run '^TestStaticInterface' -count=1 -v` | Explicit i32/i64 implementation selection, source diagnostics, cross-package memory/disk invalidation and Windows four-profile native execution pass. The GCC/Clang native helper is authored; Linux execution remains deferred. |
| Copied local closures | `go test -mod=readonly ./internal/compiler -run '^TestCopiedClosure' -count=1 -v` and `go test -mod=readonly ./internal/workspacecompiler -run '^TestCopiedClosure' -count=1 -v` | Windows interpreter/native snapshots, zero/four captures, eight helper arguments, classic-for/local shadowing, ordered argument observer and ordinary/generic memory/fresh-session disk pass. The shared helper selects GCC/Clang low/optimized on Linux; native Linux remains deferred. |
| Backend review C reference measurements | `go run -mod=readonly ./tools/dev/fairyc-backend-study --cc gcc --samples 3 --output .cache/backend-study-linux-gcc.json`, then repeat with `--cc clang` and `.cache/backend-study-linux-clang.json` | Windows C baseline and both-target emission pass. Each native run covers low/optimized; Linux execution remains unexecuted. LLVM/QBE adapters and new-target execution are not included. |
| Pinned HTTPS remote package preparation | `go test -mod=readonly ./internal/packagesource -count=1 -v` and `go test -mod=readonly ./internal/cli -run '^TestRemotePackageCLI' -count=1 -v` | Windows TLS/cache/publication/lock checks and two native MSVC profiles pass. The authored native CLI test selects GCC/Clang x low/optimized on Linux; that execution and Linux filesystem behavior remain unexecuted. |
| Atlas impact, read-only MCP and body edits | `go test -mod=readonly ./internal/atlasmap ./internal/atlasmcp -count=1` and `go test -mod=readonly ./internal/cli -run '^TestAtlas' -count=1 -v` | Windows graph/protocol/atomic-edit checks and two native profiles pass. The native edit helper selects GCC/Clang low/optimized on Linux; Linux rooted rename and execution remain unexecuted. |
| Local state execution and persistence | `go test -mod=readonly ./internal/contractstate ./internal/contractstatemir ./tools/dev/fairyc-state -count=1 -v` | Windows source-to-Go-MIR execution, separate-process lock/restart/forced-exit tests and six standalone executable calls pass. Linux `flock`, rooted rename and process behavior remain unexecuted; this replay does not certify generated-C persistence or a chain VM. |

Use one later source snapshot and record compiler versions, exact source commit, commands, exit
codes and omissions for these slices together. This queue neither launches Linux work now nor
relabels earlier Go-bootstrap evidence as proof of the new Fairy companion or aggregate matrix.

## Completed Linux baseline

The Linux candidate implementation is not a pending item in this queue. The maintainer-supplied
external record [Linux bootstrap acceptance after frontend memory reclamation](../ops/linux-bootstrap-revalidation-62f2a006.md)
validates source `62f2a006436e3ef054955b1681d7041fa87892aa`. It records successful Debian-native
bootstrap builds, CLI/reference comparisons, cache invalidation, tests, traps, binary streams and
generated-C execution. The four default manual images passed 576 CLI calls, 60 Go-reference
comparisons and 12 generated-C executions in total; the repeated GCC optimized image was not
double-counted.

| Completed scope | Status | Evidence | Limit attached to the evidence |
| --- | --- | --- | --- |
| Linux bootstrap and candidate command path | **Complete within this record** | [Evidence and scope](../ops/linux-bootstrap-revalidation-62f2a006.md#evidence-and-scope), [default acceptance](../ops/linux-bootstrap-revalidation-62f2a006.md#unchanged-default-acceptance), [regression results](../ops/linux-bootstrap-revalidation-62f2a006.md#regression-results) | Transcribed external results on Debian 13.3 amd64, Linux 6.18.57, glibc 2.41, Go 1.26.10, GCC 14.2.0 and Clang 18.1.8, with a four-core/4 GiB cgroup. It was not a new local execution or hosted-CI receipt, and the raw-results ZIP was not supplied. |
| Linux `check/build/run/test/new` and cache behavior | **Complete as covered by the record** | [Historical Go bootstrap pair](../architecture/candidate-linux-build-v1.md#historical-go-bootstrap-pair), cited regression results | Coverage includes ordinary/prepared checks, cache cold/hit/corruption/invalidation, stale locks, replay, stdin/EOF/timeouts, traps, binary output and argument boundaries. It does not establish every possible host, load or input. |

The record explicitly leaves a full `go test ./...`, the complete fixture/sanitizer/ABI sweep,
Ubuntu execution, native Windows execution, clean-machine distribution, self-host acceptance,
repeated load distribution and compiler throughput comparison outside its evidence. Those are
separate scopes. In particular, future Linux execution must reuse this completed baseline rather
than reopen it merely because local Windows probes skip Linux.

## Excluded from this queue

These actions and evidence remain separate and do not become complete through this queue:

- Push, pull-request creation, hosted workflow dispatch, publication and release.
- Deployment, service activation, repository visibility and supported-distribution claims.
- Clean-machine or fresh-OS transfer, installation and portability proof.

No excluded action was performed while updating this document.

## Status and evidence rules

- Execute the rows in order, with the explicit batched-Linux exception above. A bounded contract may be prepared for the next row while the current
  row is being completed, but implementation evidence does not skip ahead.
- A passing focused probe is evidence for its named slice. It does not complete a broader row,
  promote a target or turn a documented gap into a reproducer.
- A completion statement names the source revision, commands or receipt, profiles, omitted checks
  and material limits. An unexecuted or unavailable check is recorded as skipped with its reason.
- Later completion records supersede earlier pending prose. This document replaces the status
  decisions in the historical ordered-work table on `completion-plan.md`; neither preserves a
  conflicting "pending" claim after a later owner supplies completion evidence.
- Keep code, grammar, target flags, contracts, fixtures, CI and packaging changes in their own
  reviewed slices. This queue authorizes the named implementation sequence; it does not make
  unlisted syntax, API or target support available. Those surfaces change only with their owning
  contract, implementation and fixtures.
