# Self-hosting readiness v0

Cache-key v4/v0 encoders and the checked ordinary state-body identity are now
verified: [key checkpoint](../ops/candidate-generic-cache-key-20261014.md) and
[body checkpoint](../ops/candidate-cache-body-identity-20261014.md). Remaining
Complete-owner package key composition and actual artifact reuse stay open.
The current candidate-owned IR environment is verified in the
[environment checkpoint](../ops/candidate-cache-environment-20261014.md).
The actual running Windows compiler image identity is verified
in the [compiler identity checkpoint](../ops/candidate-compiler-identity-20261014.md).
Exact loaded workspace facts are verified in the
[workspace checkpoint](../ops/candidate-cache-workspace-20261014.md). Complete
original specialization membership is verified in the
[specialization checkpoint](../ops/candidate-cache-specializations-20261014.md).

Current source-identified build/run integration and its 51 process observations:
[2026-10-14 native checkpoint](../ops/candidate-identified-native-20261014.md).
Cache consumers, latest complete fixed point, product defaults/performance and
platform/CI promotion remain open; the historical entries below retain their
original scope.

## Ordinary type application diagnostics (2026-10-13)

The original-source request collector separates three states. With no templates,
an applied ordinary local or imported function keeps its exact applied node span
as private 4504 and the check renderer publishes the complete Go
unsupported-call-target stderr at exit 3. With templates present, an unknown or
ordinary target that is not a template reports E0101 over the complete applied
span, independent of aliases; with no templates a missing symbol keeps the bare
path span. The 4504 path is private and source-only: it adds no callable or
language, publishes no IR, and leaves product stack and timeouts unchanged.

Five authored generic_source_requests tests pass in all three profiles. The
native authored-Go and both candidate C images match all 21 old process
comparisons (63/63) and all 17 generic statuses (51/51). Exact stdout and first
diagnostic or complete unsupported message matches 16/17 on each image (48/51);
only the unsupported scalar u16 diagnostic stays incorrect while its reject 3 is
correct. Root build check passes 446 files / 5767 MIR functions with the lock
updated, and the final --locked recheck verifies the same 446 files / 5767 MIR
functions.

The receipt records the verified source, tool, image and copied-input hashes and
reports genericDifferentialPassed=false, stableCheckComplete=false and the
remaining UnsupportedScalar. Public API, SDK ABI, DB and CI are unchanged;
no push occurred. Receipt:
[candidate-ordinary-application-20261013.json](../ops/evidence/candidate-ordinary-application-20261013.json).

## Identified check command (2026-10-13)

`project_check` collects the exact loaded-workspace coordinates, calls
`build_complete` and derives its Stats from the checked identified IR, rejecting
the phase 6 publication as internal 4. The candidate Vec count is 178 and matches
Go 178 on all three images after five separately counted verified glue entries;
the earlier 173-against-178 probe stays as the failure record and the ordinary
core SDK count of 166 is intact.

The original generic request traversal returns exact original AST node spans and
separates arity/implicit/nominal (4502), missing (101) and private (13) requests;
the source plan diagnostic maps these to E4502/E0101/E0103 and cycles to E4503,
with the declaration E4501 summary registered. Four `generic_source_requests` and
four `generic_source_root_location` regressions pass Go native and candidate C
development/release. Root check passes 446 files / 5767 MIR functions with the lock
unchanged, and the final `--locked` check verifies 446 files / 5767 MIR functions.

The native image and both candidate C images pass all 21 original process
comparisons (63/63) and every one of the 14 generic statuses (42/42). Twelve of
fourteen generic cases match exact stdout and first diagnostic on each image;
ordinary nongeneric explicit type application (Go unsupported call target against
E4502) and unsupported scalar u16 (Go unsupported template signature positions
against E0101 at the actual type argument) remain the two diagnostics-only
mismatches.

The receipt records the two expected/actual diagnostics, the failed history and
verified inputs, tools, images and copies, and reports `connected=true` with
`stableCheckComplete=false` and `genericDifferentialPassed=false`. This is a
private-API-only change; the public API, SDK ABI, DB and CI are unchanged and no
push occurred. Private test images keep 16MiB stacks. Next, first the two
unsupported diagnostics, then the identified build/run/cache, the latest full
fixed point and the remaining platform and CI gates. Receipt:
[evidence](../ops/evidence/candidate-identified-check-20261013.json).

## Original generic planning failure locations (2026-10-13)

`generic_source_plan` keeps the exact original input on any failed attempt and
seals only a successful complete computation. The error-only original AST
request scan matches the retained canonical request key and orders equivalent
aliases by coordinate namespace, module and span. `project_source_plan` phase 5
retains the original location for cycle, budget and scalar-ability failures;
stale inputs, invalid roots and the shared budget fail closed without a partial
plan.

The actual Go PreparePackages/PrepareSDKStoragePackages oracle covers seven
cases, Direct, Aliases, Reordered, OwnerOrder, PerDeclaration, Depth and the SDK
Vec type-root, and nine authored tests pass across Go native and both candidate C
profiles: four `generic_source_root_location` tests and the five existing
`generic_source_plan` regressions, whose earlier run completed both profiles but
failed its final root-lock snapshot when an independent check refreshed the lock
concurrently. The refreshed fixture stays byte-identical to the linked receipt.

Root check and locked check both pass 446 files / 5765 MIR functions; the
unlocked run reports the lock unchanged and the locked run verifies it.

Next: identified CLI/cache consumers, the latest-source full image and fixed
point, product stack deadlines, Linux, the full suite and same-head CI. This is a
private-API-only change; the public API, SDK ABI, DB and CI are unchanged and no
push occurred. Receipt:
[evidence](../ops/evidence/candidate-source-root-location-20261013.json).

## Complete project owner publication (2026-10-13)

`project_compiler_core` `build_complete`/`build_complete_sources` runs the original
source plan, the complete body/Flow verification and the identified IR linking, then
builds every package interface and returns one `Complete` owner holding the checked
IR plus the detached interface `Records`. Each record carries the ordinary
descriptor, the source public template closures, the ordinary v5 hash and the
generic source hash. `generic_source_closure`, `dependency_json` and
`module_public_templates` keep their existing Graph API and read the plan input's
borrowed catalog and dependency rows directly. Source freshness, owners, imports and
coordinates are authenticated and the phase fails closed without a partial result.
Existing 16MiB source, 8MiB link and 1MiB phase histories are retained.

Thirteen tests pass on Go native, candidate C development and release: four
`project_compiler_complete`, four `project_compiler_identified` workspace tests (three
existing plus one real-workspace completed-owner check) and five existing
`module_public_templates` regressions, spanning real workspace Current/SDK/Generic. The
new core testcase keeps six Go graphs, 18 descriptors with ordinary and generic hashes,
and 113 templates; root and locked check pass 445 source files / 5745 MIR functions.
The initial verification-only workspace C development run timed out at 120s; after
re-checking the authored entry, Go-native copied sources and packet SHA, the existing Go
result/packet/dev image was reused to finish it at verification-only 900s (development
269.08s, release 22.19s, release build 195.94s) on private test-harness stacks; product
defaults unchanged.

Next: identified CLI/cache consumers, latest-source full image/fixed point, product
default stack/deadlines, and Linux/full suite/same-head CI. Private test images keep
16MiB stacks; public API, SDK ABI, DB and CI are unchanged; no push.
Receipt: [evidence](../ops/evidence/candidate-project-complete-20261013.json).

## Complete source-identified Vec SDK interfaces (2026-10-13)

The identified source-interface helper combines original public template closures
with the checked source-identified ordinary descriptor. Both the template graph
and specialization plan must match current source bytes, package owners,
resolved import providers and coordinate namespaces. Completed phase histories
retain their existing bounds and fold into the shared 16MiB project ledger.

Six actual Go graphs match all 113 public-template triples, 18 ordinary hashes
and 18 generic hashes, including all six Vec SDK package interfaces previously
rejected with 4503. Cases cover private edits, source/owner reorder, inert views,
multi-module ownership and no public user templates. Three authored tests and
five existing source-interface regressions pass Go native and candidate C
development/release. Exact/exhausted work, invalid initial work, changed/missing
coordinates, different plan namespaces and stale plans/graphs fail closed.
Root check and locked check pass 440 files / 5677 MIR functions.

Next: owning dispatcher/cache consumers must gate publication on completed
body/Flow checking. These interface facts alone admit no executable or cached
artifact. Latest full-image/fixed-point promotion, product timing/stack
integration, Linux/full suite and hosted CI remain open. Private test images use
16MiB stacks; public API, SDK ABI, limits, DB and CI are unchanged; no push.
Receipt: [evidence](../ops/evidence/candidate-source-interface-identified-20261013.json).

## Identified public descriptor extraction (2026-10-13)

Identified public descriptor extraction validates the complete source/type-name
table once before rendering its fields and signatures. Prepared rendering keeps
the same bounded work history without repeating whole-source authentication per
field. Original templates remain separately recorded and generated declarations
do not enter the ordinary public item stream.

All 3471 v5 encoding bytes and all three ordinary hashes match actual Go
publication for user, provider and Vec SDK packages. Three authored tests and
four existing extraction regressions pass Go native and candidate C dev/release.
Detached SDK descriptors, exact-fit/exhausted work, invalid owners and the legacy
Vec rejection path are covered. The Go observer's original vector fixture stays
byte-identical. Root check and locked check pass 440 files / 5676 MIR functions.

Next: combine source-planned public template closures and the identified ordinary
descriptor into complete Vec SDK source-interface hashes, then dispatcher/cache
consumers. Executable/cache publication still belongs to completed body/Flow
checking. Latest full-image/fixed-point promotion, product timing/stack integration,
Linux/full suite and hosted CI remain open. Public API, SDK ABI, limits, DB and CI
are unchanged; no push. Receipt: [evidence](../ops/evidence/candidate-identified-public-extract-20261013.json).

## Original-source Vec public type names (2026-10-13)

The source type-name factory derives detached Vec names from the original
specialization plan and checked typed handles. Retained type rows and complete
source/owner stamps reject a different graph, stale sources or changed owners.
The recursive renderer uses `self::std::vec::_fy_generic_g0_<digest>` or the
coordinate-qualified dependency form, including shared/mutable references and
Option/Result wrappers. Ordinary rendering retains its previous behavior.

The actual Go SDK preparation and completed-Flow public descriptor supply six
Vec names and four composite signature strings. All twelve self/dependency
spellings and the four signatures match. Three authored tests and four ordinary
regressions pass Go native and candidate C development/release. Detached owners,
stale nominal sources, changed owners and exact-fit/exhausted work are covered.
Root check and locked check pass 440 files / 5673 MIR functions. Private C API
test images use explicit 16MiB stacks; application defaults are unchanged.

Next: identified public descriptor extraction and complete SDK source-interface
hashes, then dispatcher/cache consumers. This fact table does not authorize
executable/cache publication. Latest full-image/fixed-point promotion, product
timing/stack integration, Linux/full suite and hosted CI remain open. Public
CLI/API, SDK ABI, limits, DB and CI are unchanged; no push. Receipt: [evidence](../ops/evidence/candidate-public-vector-types-20261013.json).

## Source-identified project C partitions (2026-10-13)

Source-identified full and entry-selected partition APIs derive one owned name
table from the linked IR owner. Prototypes, definitions, calls and Vec cleanup
glue retain canonical g0 symbols across all body files. Ordinary numeric APIs
remain available; no IR copy or multi-owner borrow is introduced.

Two authored cases pass through an isolated Go-built program and candidate C
development/release. All 75,802 output bytes match. The complete bundle contains
100 frames in 13 body files; each compiles separately and the linked program
returns 42 under /Od and /O2 with ordinary application stacks. Detached names,
entry-only omission, work boundaries, invalid inputs and sticky output failure
are covered. The existing five-frame numeric partition passes Go native and
actual /Od-/O2 linkage. Four malformed packet variants reject.

The first native-test capture exceeded its 64KiB report bound. The private
verifier's explicit isolated native-program mode compares complete streams
without changing that product bound. Root check and locked check pass 438 files /
5648 MIR functions. Candidate API images retain explicit 16MiB stacks.

Next: complete Vec public interfaces and dispatcher/cache consumers. Latest
full-image/fixed-point promotion, product timing/stack integration, Linux/full
suite and hosted CI remain open. Public CLI/API, SDK ABI, limits, DB and CI are
unchanged; no push. Receipt: [evidence](../ops/evidence/candidate-project-c-names-20261013.json).

## Current local candidate evidence (2026-10-13)

### Source-identified project linking (2026-10-13)

Project linking reads ordinary and identified IR owners through disjoint input
paths without copying input IR or returning a borrow with two input origins.
Ordinary numeric linking retains its API. Identified generic functions deduplicate
by canonical g0 identity, require matching rebound frame contracts and compare
complete rebound bodies. Calls use the shared global frame. Type and cleanup
maps retain owned source IDs; conflicting mapped identities reject. The linked
owner factory derives its program, ID table and call index from the same input.

Five authored cases pass Go native and candidate C development/release: generic
deduplication, detached owners and call targets; work/domain rejection; actual
SDK Vec type/cleanup rebinding and C transport; and ordinary missing/duplicate
body rejection. The 57,516-byte generated C matches across Go and both candidate
profiles and returns 42 at /Od and /O2 with separate current runtime support and
ordinary application stacks. Private compiler API tests explicitly use 16MiB.
Root check and locked check pass 437 files / 5629 MIR functions.

Next: source-identified partition symbols, complete Vec public interfaces and
dispatcher/cache consumers. The current partition renderer remains numeric.
Latest full-image/fixed-point promotion, product timing/stack integration,
Linux/full suite and hosted CI remain open. Limits, public CLI/API, SDK ABI,
DB and CI are unchanged; no push. Receipt:
[evidence](../ops/evidence/candidate-project-identified-link-20261013.json).

### Owned identified project units (2026-10-13)

The project collector retains each independently checked identified IR owner,
source identities and exact source range through the existing adaptive grouping.
One capacity retry splits 81 unique source specializations across two units;
82 specialized frames include the same shared g0 ID in both units. All 82
specialized calls retain checked targets after the source/planner owners leave
scope. The ordinary and identified coordinators share one capacity predicate.
Stage-8 attachment failures retain the shared 16MiB identity-work history and
never retry. A later-unit type error returns no collected prefix and preserves
the ordinary compiler's complete error fields.

Three authored tests pass Go native and candidate C development/release. The
candidate emits 246 sources in 97.54 seconds;
private compiler API images explicitly reserve 16MiB stacks. Root check and
locked check pass 434 files / 5597 MIR functions. Product limits/timeouts,
public CLI/API, SDK ABI, DB and CI are unchanged; no push.

Next: identity-aware project link/rebinding and specialization deduplication,
then named partitions, complete Vec interfaces and dispatcher/cache consumers.
Latest full-image/fixed-point promotion, product timing/stack integration,
Linux/full suite and hosted CI remain open. Receipt:
[evidence](../ops/evidence/candidate-project-identified-compile-20261013.json).

### Source-identified C functions and cleanup (2026-10-13)

The identified renderer derives an owned name table from the same checked IR
owner. Specialized functions use `fy_fn_g0_<digest>`; specialized cleanup bodies
use `fy_cleanup_g0_<digest>`. Prototypes, definitions, ordinary/deferred calls
and aggregate drop-glue calls consume that table. Ordinary functions and the
existing numeric project partition path retain their current spellings. The
table validates canonical lowercase IDs, rejects same-kind duplicate names,
owns frame/name bytes, and uses the existing 1MiB work/text bounds. Final C
publication remains transactional; raw name-table rendering is private.

Three authored tests pass fresh Go native and candidate C development/release.
Both candidate executions reproduce the complete 37,787-byte Go C output,
SHA256 `a48bf7c0837cd8980e332d2c270adc8a9e33ed885f2007cc258dd34624e02f98`.
The output defines five distinct identified functions and six identified Vec
cleanup bodies. Separately compiled current runtime support and the generated
program link and return 42 at `/Od` and `/O2`, with ordinary application stacks.
Tests cover names in prototypes/calls/glue, detached ownership, exact-fit and
exhausted work, sticky writer failure and unchanged output on invalid entry.

The first candidate replay hit stage-4 code 10 in the enlarged emitter. Eight
callable helpers moved unchanged into `module_unit_c_calls`; the emitter is now
24KiB and the helper 8KiB. The final replay passes in 120.18 seconds without a
capacity increase. Root check and locked check pass 433 files / 5582 functions.
The maintained private-test verifier now accepts an explicit bounded stack
reserve and checks both PE headers; compiler API images use 16MiB. Product
defaults, public CLI/API, SDK ABI, DB and CI are unchanged; no push.

Project-wide identity rebinding/deduplication and named partition output, Vec
type/interface names, precise planning diagnostics and dispatcher/cache consumers
remain. Latest full-image/fixed-point promotion, default frontend timing, Linux,
full suite and hosted CI are deferred. Receipt: [evidence](../ops/evidence/candidate-c-source-names-20261013.json).

### Checked IR identity bridge: current-source C execution (2026-10-13)

A fresh private project driver uses exact copies of 370 current Fairy modules
and a check-only C-project entry, with no diagnostic overlays. The Go bootstrap
emits its initial C; current SDK/runtime sources supply the native support.
MSVC `/O2 /Ob1` compiles the driver in 38.86 seconds. The driver processes the
277-source identified-IR test closure without a Go frontend fallback in 106.37
seconds, emitting 19,046,779 bytes in 39 C body partitions. The earlier Ast
cleanup failure is absent in this current-source replay.

All three authored identified-IR tests pass fresh Go native and generated C
development/release execution. They retain the actual SDK Vec bytes, five
concrete functions, six Vec descriptors/cleanup frames, detached identity bytes,
checked call-target selection and stages 4/5/7/8 failure behavior. Both private
compiler API test images and the seed driver explicitly reserve 16MiB stacks;
ordinary application defaults are unchanged. Source, fixture and tool hashes
are retained and checked. The preceding 430-file / 5562-function root checks
are reused after confirming unchanged compiler inputs and root lock.

This closes C execution for the identity-attachment unit. The private driver's
entry is not the full dispatcher; its 106-second replay exceeds the default
60-second frontend allowance. Latest full-image/fixed-point promotion, product
timeout/stack integration, Linux/full suite/hosted CI remain open. IR/C symbol
names, project identity rebinding, full Vec interfaces and dispatcher/cache
publication also remain. Public API, SDK ABI, DB and CI are unchanged; no push.
Receipt: [evidence](../ops/evidence/candidate-identified-ir-c-20261013.json).

### Empty cleanup arena projection (2026-10-13)

An ordinary unannotated empty resource cleanup now owns only its original block.
AST copying and type-demand collection avoid unrelated function/type arenas;
generic, annotated and nonempty cases keep the existing projection path. Source
authentication, declaration checks, owned spans and the 1MiB work limit remain.
The empty type-resolution batch already returns without inspecting arguments or
graph records; the shortcut is restricted to nongeneric declarations with no
arguments and leaves the type prefix unchanged.

Three authored tests pass Go native and candidate C development/release. They
cover 600 unrelated statements, constant work, original block spans, exact-fit
and exhausted work, and the annotated/nonempty fallback. Root check and locked
check pass 430 files / 5562 MIR functions. This proves the projection helper,
not a fresh full dispatcher or resolution of the identified-IR closure failure.
Linux/full suite/hosted CI are deferred; API, SDK ABI, DB, CI and product limits
are unchanged; no push. Receipt: [evidence](../ops/evidence/candidate-empty-cleanup-20261013.json).

### Initial checked IR source identities: native verified, C bootstrap pending (2026-10-13)

`module_project_compile::range_identified` reuses the ordinary preparation and
checking path. Type checking, moves, borrowed-result inference and owned IR
construction finish before source IDs are attached. Stage 8 returns attachment
failures without exposing an IR prefix. `module_identified_ir` owns IR, source
identity bytes, frame rows and a target index; original AST/planner/interface
owners can leave scope. It checks the canonical type prefix, rejects an
unselected late Vec, maps concrete function/cleanup frames and resolves
specialized calls through the existing backend frame index.

Three authored Go-native tests pass on exact current source copies. A closed
three-module SDK Vec fixture covers five concrete functions, six vector types,
six vector cleanup frames, ordinary/imported calls and ordinary C rendering.
Type/move/borrow failures preserve stages 4/5/7 and source spans. Exhausted work
and mismatched planning owners fail at stage 8. The single-root unit correctly
leaves imported ordinary bodies for project linking; the C rendering case uses
all three original modules. Numeric C symbols are still unchanged.

At this initial checkpoint candidate C dev/release execution was **not verified**. The pinned
2026-10-12 frontend rejects the 277-source test closure at
`module_unit_body_ast` cleanup, project phase 2/stage 3/code 10. A current-source
private C project-driver probe passes the exact Ast range with 32 shapes/232
fields, below the unchanged metadata cap. The unoptimized whole-project probe
times out at 180 seconds, and its optimized build hits the existing 60-second
MSVC host deadline. The original development probe's default 1MiB stack also
overflows; private PE stack-reserve changes only support diagnosis. Neither
source capacity nor product timeouts were raised. These probes are not a latest
dispatcher build, fixed point or successful candidate test run.

IR/C symbol naming, project-wide identity rebinding, complete Vec interfaces,
precise planning diagnostics and dispatcher/cache publication remain open.
Linux/full suite/hosted CI are not rerun. Public API, SDK ABI, DB and CI are
unchanged; no push. Receipt: [evidence](../ops/evidence/candidate-identified-ir-20261013.json).

### Compiler-handle source specialization selection (2026-10-13)

`module_source_specializations` maps the typed compiler's one-based function,
type and cleanup handles to original-source specialization nodes and owned g0_
bytes. Source template declaration indices are zero-based and explicitly
converted. Ordered concrete arguments preserve nominal provider ownership;
aliases share a node. Ordinary rows are zero. Missing or incomplete matches
publish no selection and retain the failing handle kind/index. Source stamps,
package owners, resolved imports and the retained type prefix must match first.

Three tests exercise the real SDK Vec source: five concrete functions, six vector
descriptors, vector cleanup and alias/cleanup calls. Go native and candidate C
development/release pass, including owner destruction, stale providers, wrong
owners and exact 16MiB work boundaries. Root check and check --locked pass
429 files / 5549 MIR functions. An initial successful child run was invalidated
by a concurrent root-lock refresh; the final run preserves the refreshed lock.

This private bridge selects actual typed handles. Executable IR/C naming and
call identity consumers, full SDK Vec public interfaces, precise root planning
locations and dispatcher/cache publication remain. Body/Flow gates, public
language/API, SDK ABI, DB and CI are unchanged. Latest-source full image/fixed
point, Linux, full suite and hosted CI are not rerun; no push.
Receipt: [evidence](../ops/evidence/candidate-source-selection-20261013.json).

### Dependency-closed original-source specialization plans (2026-10-13)

`generic_source_plan` consumes the detached source planning input and computes
real source-function and authenticated SDK Vec g0_ IDs. Iterative depth-first
expansion substitutes ordered parameters and concrete types, sorts resolved
request keys, deduplicates requests after substitution, and shares completed
instances. A per-template chain avoids scanning unrelated nodes. Requests enter
in the Go planner's order: depth, memo/cycle, count limits, then scalar abilities.
Dependency IDs sort before entering the identity stream. Every successful node
is published only after its dependencies and identity complete; failed planning
returns no partial plan. Source capability constraints remain absent under the
current admitted source policy; no provider or broader source feature is invented.

Four actual Go graphs match all 84 specialization IDs and their full ordered
bindings/dependencies, 48 root IDs and dependency-first specialization order.
Private helper edits, reordered original modules/owners and alternate helper
ownership are included. The observer independently crosschecks actual roots and
IDs against PrepareSDKStoragePackages. Its real SDK-owner adapter also verifies
G2005 cycle, G2003 scalar ability and G2006 total/per-declaration/depth/invalid-limit
failure categories. Five final authored tests pass Go native and candidate C
development/release. They include detached ownership, post-substitution duplicate
edges, shared instances, unused cycles, exact work limits and empty plans. After
an initial four-test pass, a zero-template invalid-initial-work gap was fixed and
all five tests rerun. Initial namespace, must-use/loan and observer fixture errors
were corrected before the final passes. Root check and check --locked pass
428 files / 5528 MIR functions with the final source lock.

Planning uses the reference defaults (4096 nodes, 256 per declaration, depth 64),
the existing 32 direct edges, at most 2048 active pending requests, 16MiB project
work and 1MiB owned-text/per-identity scopes. No executable expansion limit,
public scalar set, body/Flow admission or SDK ABI is changed by this helper.
Precise root planning locations, actual compiler declaration/call mapping, body
publication gates, complete SDK Vec public interfaces and dispatcher/cache
consumers remain. Latest-source full image/fixed point, Linux/full suite and
same-head hosted CI are not rerun. Product timeouts and CI are unchanged; no push.
Receipt: [evidence](../ops/evidence/candidate-source-plan-20261013.json).

### Bound original-source specialization inputs (2026-10-13)

`generic_source_plan_input` owns direct roots, the original template catalog,
direct dependencies and the concrete type catalog in one detached input. Root
arguments select exact concrete descriptors; dependency arguments retain either
an originating-template parameter position or a concrete descriptor. Binding
walks each edge/argument once and preserves original slots after edge sorting,
without searching all declarations for every parameter. The existing owning
dependency-graph API now delegates to the same borrowed collector.

Four actual Go package graphs match 48 roots, 91 declarations and 85 direct
edges. Parameter names/abilities, declaration owners/definitions, dependency
targets and all selected concrete type IDs/owners/definitions/abilities match.
The actual Plan.Roots and all specialization IDs are independently crosschecked
against PrepareSDKStoragePackages by the Go observer. Cases include private
helper edits, module/owner reordering and a distinct owner of the same helper
module. Four new tests and one owning-dependency regression pass Go native and
candidate C development/release. Detached ownership, unused symbolic cycles,
exact-fit/exhausted work/encoding/hash scopes, sticky failure and changed-source
rejection pass. Import/accessor namespace collisions were fixed before the final
passing check. Root check and check --locked pass 427 files / 5480 MIR functions
with the final refreshed source lock.

The existing 16MiB bookkeeping, 1GiB encoding, 96MiB streamed-hash, template,
argument and edge bounds remain unchanged. This input performs no executable
specialization, capability selection or body/Flow admission. Dependency-first
expansion, scalar requirements and cycle/growth limits, complete source g0_ IDs,
precise planning locations, SDK Vec interfaces and dispatcher/cache publication
remain next. Latest-source full image/fixed point, Linux/full suite and same-head
hosted CI are not rerun. Public API, SDK ABI, product timeouts and CI are unchanged;
no push. Receipt: [evidence](../ops/evidence/candidate-source-plan-input-20261013.json).

### Direct original-source specialization requests (2026-10-13)

`generic_source_requests` owns detached direct source roots and the original
source catalog. It reuses the dependency walk's executable AST projection for
nongeneric function signatures/bodies, aggregate fields and resource cleanup.
Unused generic bodies contribute no direct roots. Exact declaration/ordered
concrete-type keys deduplicate aliases and repeated calls across all sources;
inert written view producers remain excluded while loop bodies are traversed.
SDK ordinary signatures also request their concrete Vec types. Output sorts by
canonical request key; planner ordering and dependency expansion remain separate.

Four real Go package graphs supply twelve unique roots each, resolved from the
actual expandAll Plan.Roots. All 48 request keys, declaration IDs and ordered type
arguments match. The observer also crosschecks both roots and all specialization
IDs against PrepareSDKStoragePackages. Private helper edits, original module/owner
reordering and an alternate owner of the same helper module are included. Four
new tests and three existing dependency regressions pass Go native and candidate
C development/release. They cover detached ownership, alias deduplication,
unused roots, source stamps, private/unknown targets, arity/argument/generated-name
rejection and exact-fit/exhausted work. Initial fixture helper and twin-coordinate
errors were corrected before the final passing runs. Root check and check --locked
pass 426 files / 5458 MIR functions with the refreshed source lock.

The root set is bounded to 4096 requests under the existing 16MiB collection work,
1MiB owned text, eight arguments and bounded pending tasks. Generic-template
dependencies keep their existing 32-edge per-template limit. This collector does
not admit executable bodies or issue final planning diagnostics. Concrete catalog
binding, dependency-closed function/Vec plans and identities, precise planning
locations, complete SDK Vec interfaces and dispatcher/cache publication remain.
Latest-source full image/fixed point, Linux/full suite and same-head hosted CI are
not rerun. Public API, SDK ABI, product timeouts and CI remain unchanged; no push.
Receipt: [evidence](../ops/evidence/candidate-source-requests-20261013.json).

### Actual source planner concrete type descriptors (2026-10-13)

The Fairy concrete catalog now collects the real source planner type facts:
thirteen builtin scalar IDs with canonical copy/drop/inert/scalar/store abilities,
and nongeneric struct/resource IDs bound to original package/module/declaration
coordinates and closed nominal source definitions. Nominal planner abilities stay
empty, as in Go; value typing, layout and resource Flow remain separate. Source
aliases resolve the exact provider and reject imported private types. Typed
selection admits the candidate's six existing scalar kinds and exact nominal
coordinates; enum/composite/Vec argument forms stay explicitly unsupported.
Owned argument copying feeds these derived facts into the existing g0_ encoder.

The actual expandAll immutable catalog supplies 75 matching descriptors across
four source graphs: 52 scalar and 23 nominal records. All 23 nominal selections
come from the actual typed signature graph. Four dependency-free helper IDs match
the real source planner; all observer plan IDs are also crosschecked against
PrepareSDKStoragePackages. Private helper edits, original source/owner reordering,
and two distinct owners of the same helper module are included. Five authored
tests pass Go native and candidate C development/release, covering ownership after
source/catalog destruction, alias/private selection, stale sources and exact
collection/encoding/hash/lookup/argument-copy work boundaries. Initial observer,
static-return and fixture syntax errors were corrected before the final pass.
Root check and check --locked pass 425 files / 5452 MIR functions with the refreshed
source lock. Collection reuses the existing 16MiB bookkeeping, 1GiB encoding and
96MiB streamed-hash scopes, with 4096 records and 1MiB owned text; no caps increase.

Full source/typed root requests, dependency-closed function/Vec planning, complete
SDK Vec interface rendering and dispatcher/cache publication remain next. This
descriptor helper admits no new scalar value type or executable artifact. Latest-
source full image/fixed point, Linux/full suite and same-head hosted CI are not
rerun. Public API, SDK ABI, product timeout and CI are unchanged; no push. Receipt:
[evidence](../ops/evidence/candidate-concrete-types-20261013.json).

### Original public template projection and source interfaces (2026-10-13)

Public source templates now project canonical module/name/reachable-closure rows
from the owned dependency graph. Bounded heap sorting preserves module-prefix
ordering, including root versus root::child. A source-interface helper combines
these rows with the typed ordinary v5 descriptor. It checks original source
digests, owner-aligned coordinates and exact resolved import providers before
combining independently measured phase histories into the 16MiB project ledger.

Six actual Go PrepareSDKStoragePackages/AnalyzeSDKStoragePackages cases finish
Flow. All 113 public-template triples and twelve ordinary plus twelve generic
user-package hashes match. Cases cover private edits, source/owner reordering,
inert view edits, nested modules and a package with no public templates. All six
SDK packages intentionally reject full interface hashing with 4503 until concrete
Vec specialization names are represented. Five authored tests pass Go native and
candidate C development/release, including changed import providers, stale
sources/coordinates, missing owners and exact-fit/exhausted shared work. Root
check and check --locked pass 424 files / 5406 MIR functions with the
final refreshed source lock.

This helper supplies source facts; the owning compiler still must gate publication
on body/Flow success. Actual specialization facts, complete SDK Vec type rendering,
dispatcher and cache consumers remain next. No latest-source full image/fixed
point, Linux/full suite or hosted CI is claimed. Public API, SDK ABI, product
timeouts and CI are unchanged; no push. Receipt:
[evidence](../ops/evidence/candidate-public-templates-20261013.json).

### Region dependencies of retained template bodies (2026-10-13)

Region capability collection now skips uninstantiated generic source bodies.
Selected ordinary bodies, retained generic/imported functions and retained resource
cleanup contexts still demand the authenticated SDK Arena and AllocError types.
New cleanup contexts trigger another bounded collection pass; unrelated sources
are not scanned. The existing source, type and work limits remain unchanged.

Actual Go AnalyzeGenericPackages reaches Flow for all four valid authored cases
and rejects the active ordinary body without an allocator. Three authored tests
pass Go native and candidate C development/release with the unchanged SDK alloc
source. Their isolated import closure passes check and native locked execution.
The initial zero-based cleanup lookup trapped with integer-overflow; the final
one-based traversal passes. No latest-source full image/fixed point, Linux/full
suite or hosted CI is claimed. Public API, SDK ABI, CI and product timeout are
unchanged; no push. Receipt: [evidence](../ops/evidence/candidate-region-template-demands-20261013.json).

### Reachable original template closure hashes (2026-10-13)

The Fairy source-template graph now computes the actual Go templateClosure
identity: Root plus all reachable canonical declarations, ordered by complete
DeclarationID, then DigestText under source-public-template-closure-v0.
An iterative visited set marks each declaration once, so unused self/mutual
recursion remains finite to describe without admitting executable recursive
specializations. The retained source catalog supplies canonical ordering; no
source parser ownership or disposable module number enters this closure record.

All 100 closure digests and 100 declaration JSON records from the four actual
Go catalog graphs match. Reordered modules/owners preserve all 25 oracle hashes;
an inert written view producer body edit changes only that producer's closure.
Five authored tests pass Go native and candidate C development/release. Private
helper edits propagate through transitive dependencies while unrelated cycle
and unused roots retain their hashes. Detached graph ownership, exact-fit/shared
1MiB work exhaustion, invalid roots/ranges/targets/argument spans and text bounds
reject without publishing an identity. The dependency JSON writer now checks
these ranges before indexed access. One initial new-guard mutable-argument loan
error was fixed before the final passing run. Source check and check --locked
pass 421 files / 5389 MIR functions with the refreshed final root lock.

Public-template summary projection and source-interface integration, real
specialization facts and dispatcher/cache consumers remain next. Latest-source
complete image/fixed point, Linux/full suite and same-head hosted CI are not
rerun. Private candidate API added; public API, SDK ABI, DB, CI/runners and
product timeout defaults unchanged. No push. Receipt:
[evidence](../ops/evidence/candidate-generic-source-closure-20261013.json).

### Direct original template dependencies (2026-10-13)

The source catalog now owns detached direct dependency edges and ordered type
arguments. Iterative arena traversal follows the actual Go executable AST
projection, including function signatures, aggregate fields/cleanup and loop
bodies; inert written ByteFor view facts do not create execution edges.
Exact aliases deduplicate, imported private/unknown targets and invalid arity or
arguments reject, and generated type names reject as in the Go source adapter.
Finite recursive declaration graphs remain descriptions, not admitted executable
recursive specializations. Per-template edges sort by the canonical expression
key; exact genericplan.Declaration JSON includes scalar ability closure and the
original package-bound definition bytes.

The real expandAll catalog observer is crosschecked against
PrepareSDKStoragePackages. Four source graphs each contain 25 templates and
28/29 direct edges; all 100 canonical declarations match byte for byte. Three
authored tests pass Go native and candidate C development/release, including
ownership after parser destruction, cycles, stale/private/unknown input and
exact-fit/exhausted histories. The first fixture read failed because 121717
bytes exceeded the test manifest reader; an independent 128KiB test reader fixes
that without changing product limits. Source check and check --locked pass
420 files / 5383 MIR functions with the final root lock refreshed.

Bounds: 16MiB collection work, 1MiB owned graph text, 4096 source templates,
eight arguments and 32 unique dependencies per template, bounded iterative
pending work; canonical JSON retains the existing 1MiB work/output bound.
Reachable-template closure hashes, actual specialization facts and dispatcher/
cache consumption remain next. Latest-source complete image/fixed point,
Linux/full suite and same-head hosted CI are not rerun. Private candidate API
added; public API, SDK ABI, DB, CI/runners and product timeouts unchanged. No push.
Receipt: [evidence](../ops/evidence/candidate-generic-source-dependencies-20261013.json).

### Original source template catalog and resolved targets

`generic_source_catalog` owns original generic function and authenticated SDK Vec
metadata before expansion: canonical package/module/name ID, package-bound original
AST definition hash, public visibility, original source/name spans and ordered
parameters. User function parameters require scalar abilities; exact SDK Vec
parameters retain their admitted nominal exception. Every input source stamp is
checked before collection; duplicate/builtin parameters, namespace collisions,
generated names, unauthenticated std sources and user aggregates reject.
Detached exact local/alias lookup separates functions and aggregates, reuses
aliases to one provider, and rejects imported private templates.

Actual Go PrepareSDKStoragePackages supplies 21 source origins from two user
modules and the unchanged SDK Vec: 20 functions, one resource, 22 parameters and
16 public templates. All IDs, definitions, visibility, parameters and source/name
spans match. Four tests pass Go native and candidate C development/release,
including detached ownership, source/owner reordering, stale source, admission,
and exact-fit/exhausted collection/lookup histories. Source check passes 415 files
/ 5317 MIR functions. Final verification checks source/copy, fixture, tool and
lock hashes. The initial oracle file-path and test type/origin issues were fixed
before the final passing run.

The follow-up reserves all thirteen scalar names recognized by the Go generic
source adapter, including currently unadmitted value types such as i8 and f64.
Go rejects both authored parameter-name cases; the four tests pass again in all
three engines. The final root check refreshes the changed source lock, and
check --locked verifies 415 files / 5317 MIR functions after the test executions.
No additional scalar value type is admitted.

Bounds: 16MiB collection/lookup work, 1MiB per definition and owned text, 4096
templates and eight parameters per template. Successful definition histories fold
once into collection work. These records are not yet planner declarations:
direct dependencies, reachable public closures and actual specialization facts
remain next, followed by dispatcher/cache consumption. Latest-source full image,
fixed point, Linux/full suite and same-head hosted CI are not rerun. Private
candidate API added; public API, SDK ABI, DB, CI/runner, hygiene and product
timeouts remain unchanged. No push. Receipt:
[evidence](../ops/evidence/candidate-generic-source-catalog-20261013.json).


Closed nominal package context and streamed definitions (2026-10-13)

`generic_package_context` retains every original module AST and exact resolved
alias-to-provider namespace in the Go closed-graph identity. Modules sort by
canonical package owner then module path; import maps sort by alias and retain
empty {}. Original source stamps are checked. The shared module header writer
keeps standalone bytes unchanged while graph/module JSON streams in declaration,
header and original binding/literal fragments. A first pass measures the exact
joined byte length; a second feeds DigestChunks-compatible domain/length framing
and fragments into SHA-256. Complete graph and module JSON are not retained.
Owned contexts survive workspace/table destruction and produce
source-nominal-package-v0 definitions for nongeneric structs/resources.

Six Go context hashes and 127 nominal definitions match: four real workspaces
contain 54 modules and 124 nominal definitions, the three-module authored graph
adds three, and eight unchanged original SDK Vec ASTs form a 1260177-byte scale
graph. All real workspace contexts also exceed 1MiB. Private helper/provider
changes invalidate identity; manifest order, source order and disposable owner
numbers do not. Five authored tests pass Go native and candidate C dev/release,
including exact-fit/exhausted bookkeeping, encoding and hash histories, retained
failure, stale sources and invalid/generic selection. Three existing complete
module JSON/nominal regression tests pass all three engines. All 55 fixture files
remain unchanged. Source check passes 414 files / 5286 MIR functions; source/copy
hashes and the original lock are verified.

Bounds are explicit: 1MiB per declaration/header/sidecar writer, 16MiB graph
bookkeeping, 1GiB cumulative abstract encoding work and 96MiB hash input. This
adds a graph operation without changing the existing standalone writer limit.
The initial whole-module work exhaustion required fragmentation. The initial
combined candidate development run exceeded the verifier's default 60-second
execution allowance and is not a pass. An explicit verification-only 240-second
option now bounds that workload; its default remains 60 seconds and invalid
0/901 values reject. Unchanged Go-native results are reused after exact input
fingerprint and 135 copied source/test hash checks. Product timeout is unchanged.
No memory or compiler-speed claim is made; batch nominal extraction remains a
later consumer optimization.

Reachable-template closures and actual planner facts remain next, followed by
dispatcher/cache integration. Latest-source full image/fixed point, Linux/full
suite and same-head hosted CI are not rerun for this bounded slice. Private
candidate API and verification harness change; public API, SDK ABI, DB,
repository hygiene, CI/runner and product timeout remain unchanged. No push.
Receipt: [evidence](../ops/evidence/candidate-generic-package-context-20261013.json).

Resolved package import definitions (2026-10-13)

`generic_package_bindings` collects the exact projected source owners and import
providers together with owner-aligned package coordinates. It owns canonical
package_coordinate_v1_<hex> namespaces, normalized module paths and byte-sorted
alias=owner::module records. Original source SHA-256 stamps prevent stale AST
reuse. Duplicate coordinate namespaces, missing/contradictory providers, alias
collisions and conflicting providers across modules in one package reject.
No-import lists remain null, matching Go. Function and aggregate definitions now
combine the original AST and actual resolved bindings with source-package-function-v0
and source-package-aggregate-v0. Collection uses the existing 16MiB workspace
ledger; each JSON/definition request preserves its independent 1MiB ledger.

Four real Go-loaded workspace graphs supply 54 modules, 36 import records and 26
function/aggregate definition hashes. All match the Fairy projection. Private
provider body edits and manifest dependency reordering preserve caller identity;
changing the resolved feature-instance provider invalidates it. The unchanged
original SDK Vec supplies another 16 matching package definition hashes. Five
authored tests pass Go native and candidate C development/release, including
prefix-alias byte ordering, same-package multi-module provider conflicts, stale
source, malformed coordinates/bindings, detached ownership, retained failure and
exact-fit/exhausted work. All 49 fixture files remain unchanged. Source check
passes 413 files / 5249 MIR functions; original/copy hashes and lock are verified.
The initial test typing/origin issues and coordinate-prefix length error were
corrected before the final passing replay.

Closed nominal graph context, reachable-template closures and actual planner
facts remain next, followed by dispatcher/cache consumption. Latest-source full
image/fixed point, Linux/full suite and same-head hosted CI are not rerun for this
bounded slice. Private candidate API added; public API, SDK ABI, DB, CI/runner and
product timeout remain unchanged. No push.
Receipt: [evidence](../ops/evidence/candidate-generic-package-bindings-20261013.json).

Complete original module AST and nominal definitions (2026-10-13)

The original module writer now composes every declaration category and both
original CST sidecars in exact Go AST field order. It retains canonical module
path, root span, full source byte count and LF-derived line starts, including
BOM/CRLF and final-newline geometry. Absent declaration categories remain null;
source order is retained within each category. Function JSON exposes the same
writer through the shared ledger without changing its output. Nongeneric local
struct/resource definitions now hash the entire original module, newline and
name with source-nominal-module-v0, conservatively including private helpers.

Eighteen full modules and five standalone nominal definition hashes match actual
Go json.Marshal / genericplan.DigestChunks. The unchanged real SDK Vec module's
157400-byte AST matches exactly, including ordinary/generic function bodies,
cleanup, original bindings and literals. Three tests pass Go native and candidate
C development/release: exact modules/definitions, field/helper/line-geometry
invalidation, invalid source/selection, generic aggregate exclusion, retained
failure and exact-fit/exhausted combined work. Source check passes 412 files /
5216 MIR functions; source/copy hashes and updated lock are verified. The test
fixture's incorrectly embedded literal newlines were corrected before replay.

Resolved package import bindings and the closed nominal graph context remain
next, followed by template closures and actual planner/dispatcher/cache facts.
Latest-source full image/fixed point, Linux/full suite and same-head hosted CI
are not rerun. No public API, SDK ABI, DB, CI/runner or product timeout change;
no push.
Receipt: [evidence](../ops/evidence/candidate-generic-source-modules-20261013.json).

Original source binding and literal observations (2026-10-13)

Original module sidecars now retain the Go CST binding and literal lists.
Bindings preserve source preorder, including enclosing-before-nested bindings
and classic-for initializer spans without the semicolon. Literals include
expression values, numeric array lengths and attribute arguments, with original
Raw/Kind/Cooked values. Function and C-function-type children visit parameters
before the earlier-written result; bounded interval rotations retain that order
without changing source records. Byte-for synthetic AST bindings/literals and
implicit region/pattern names do not enter the original CST lists.

Fifteen sources, including the unchanged original SDK Vec, match the real Go
AST sidecars byte for byte: 32 bindings and 93 literals. Three tests pass Go
native and candidate C development/release, including nested declarations,
attributes, array types, source literals, nested C function types, empty lists,
retained failure state and exact-fit/exhausted work. Source check passes 411
files / 5211 MIR functions; source/copy hashes and updated lock are verified.
An initially misspelled region fixture was corrected before the final replay.

Complete module AST/nominal definitions, resolved import binding, template
closures and planner/dispatcher/cache consumers remain next. Latest-source full
image/fixed point, Linux/full suite and same-head hosted CI are not rerun.
No public API, SDK ABI, DB, CI/runner or product timeout change; no push.
Receipt: [evidence](../ops/evidence/candidate-generic-source-observations-20261013.json).

Remaining source declaration JSON (2026-10-13)

The source item writer retains exact Go AST records for imports, constants,
static assertions, state declarations and foreign functions. Imports join
significant path tokens but retain written path/alias spans. Constants and
assertions reuse the original expression serializer; ordered cell/map fields
preserve null cell keys and written map key/value types. Foreign declarations
retain ABI C, attached documentation, attributes, parameters and result type.
The existing parameter serializer is shared without changing its body. This is
syntax projection only; semantic admission and resolved import providers remain
separate.

Thirteen authored cases match actual Go syntax-only AST JSON byte for byte,
including BOM/CRLF, comments in import paths, UTF-8/HTML-escaped strings, attributed
and empty state, nullable cell/map keys and C-function parameter types. Three
tests pass Go native and candidate C development/release, including alias,
constant-expression, state-field and foreign-signature changes, invalid
selections, retained failure state and exact-fit/exhausted work. Source check
passes 410 files / 5191 MIR functions; final receipts verify source/copy hashes
and the updated lock.

Complete module AST/nominal definitions, resolved package import binding,
reachable-template closures, actual planner facts and dispatcher/cache consumers
remain next. Latest-source full image/fixed point, Linux/full suite and same-head
hosted CI are not rerun. No public API, SDK ABI, DB, CI/runner or product timeout
change; no push.
Receipt: [evidence](../ops/evidence/candidate-generic-source-items-20261013.json).

Source aggregate JSON and definitions (2026-10-13)

The retained syntax now emits exact Go AST JSON for structs, resources and
restricted enums, preserving type parameters, visibility, attributes/documentation,
ordered fields and variant payloads. Resource cleanup retains its own span,
attributes and complete original body, including written/synthetic byte-view
loops. Struct/resource wrappers produce source-aggregate-ast-v0 definition digests
through the existing shared 1MiB JSON/hash work ledger. This is syntax projection,
not new generic aggregate or type admission.

Thirteen source cases match actual Go syntax AST JSON, and eleven struct/resource
wrappers and definition hashes match json.Marshal / genericplan.DigestText. The
cases include the unchanged 8702-byte SDK Vec<T> with six fields and cleanup,
BOM/CRLF, documentation and foreign-function field types. Three tests pass Go
native and candidate C development/release: exact JSON/hash equality, field/
cleanup/type-parameter invalidation, invalid selections, sticky failure and
exact-fit/exhausted combined work. Source check passes 409 files / 5187 MIR
functions; final receipts verify source/copy hashes and the updated lock.

Module AST definitions, resolved import providers, reachable-template closures,
actual planner facts and dispatcher/cache consumers remain next. Latest-source
full image/fixed point, Linux/full suite and same-head hosted CI are not rerun.
No public API, SDK ABI, DB, CI/runner or product timeout change; no push.
Receipt: [evidence](../ops/evidence/candidate-generic-source-aggregates-20261013.json).

Source function definition digest (2026-10-13)

The source function serializer now computes the real standalone template
Definition directly from original Function JSON, using the existing
source-function-ast-v0 / fairyc.genericplan.digest.v0 framing and SHA-256 owner.
JSON projection and hash encoding/streaming share one 1MiB work ledger; errors
publish no successful digest. Definition text is exact lowercase 64-byte hex.

The Go oracle reads sixteen function-body fixtures, sixteen written/synthetic
view-loop fixtures and six definition variants. All 38 function JSON values and
Definition digests match actual Go json.Marshal and genericplan.DigestText.
Body, visibility, documentation, parameter name and type-parameter name changes
invalidate the baseline digest. Three tests pass Go native and candidate C
development/release, including invalid declaration selection and measured
exact-fit/exhausted combined JSON/hash work. Source check passes 408 files /
5179 MIR functions; final tests verify the updated lock and original/copy hashes.
The 129931-byte oracle uses the existing project reader with a test-only 1MiB
allowance.

This closes the standalone function-definition component. Aggregate and module
AST definitions, resolved package import binding, reachable-template closures,
actual planner facts and dispatcher/cache consumers remain next. Full image /
current-source fixed point, Linux/full suite and hosted CI are not rerun.
No public API, SDK ABI, DB, CI/runner or product timeout changes; no push. Receipt:
[evidence](../ops/evidence/candidate-generic-source-function-definitions-20261013.json).

Source view-loop written and synthetic JSON (2026-10-13)

ByteFor now retains exact written span/name/type/view/body JSON and the Go
syntax-only synthetic expression. Eight admitted projection shapes produce the
private NUL-named bound/index bindings, usize types, length observation, classic
loop condition/post and original body prefixed by item binding. Mutable item
borrows retain the exact name span and &mut prefix. Named/field/grouped-field
and sliced sources match Go; unsupported written element/source forms keep
Expression null rather than losing their original ByteFor. Nested loop blocks
retain original scopes, source order and repeated syntax representations.

Sixteen authored functions match Go JSON byte for byte, with eight synthetic
root loops and eight null projections. Three tests and three function regressions
pass Go native and candidate C development/release. Malformed facts, sticky
failure and exact-fit/exhausted work pass. Source check passes 408 files / 5178
MIR functions; final receipts verify original/copy hashes and the updated lock.
The initial fixture reader rejected the 79934-byte oracle at its 64KiB bound;
only this test now uses the existing project reader with an explicit 1MiB bound.
A grouped item type outside the existing lookahead was replaced by a written
array case. Product input and analysis bounds remain unchanged.

Scope correction: arbitrary classic-for expression clauses are outside the
normative for_init/for_post grammar and ADR 0213. The Go syntax parser's wider
retention is not another admission task. Next: source definition digests,
aggregate/module and resolved import extraction, template dependency closures,
planner facts and dispatcher/cache consumption. Full image/current-source fixed
point, Linux/full suite and hosted CI are not rerun. No public API, SDK ABI, DB,
CI/runner or product timeout changes; no push. Receipt:
[evidence](../ops/evidence/candidate-generic-source-view-loops-20261013.json).

Source function body JSON (2026-10-13)

The function serializer joins original header metadata to retained statement
and expression arenas. It emits exact Go AST blocks/tails, typed and inferred
let/var bindings, assignments, return/assert/discard/defer/burn, jumps, while,
classic for, region, unsafe blocks, if/else-if and switch with positional or
named payload patterns. Classic-for initializer spans exclude the semicolon;
ordinary statement spans retain it. Pattern paths and named binding Cooked text
are reconstructed from original tokens. Internal parser list/header nodes are
never published as source AST statements. Byte-view loops remain an explicit
4503 rejection until their retained and synthetic Go AST members are connected.

Sixteen authored functions match actual Go syntax-only frontend JSON, including
empty blocks, absent return operands, nested tails, empty loop clauses, comments,
BOM/CRLF, UTF-8, associated functions and attributes. Three new tests and three
expression regressions pass Go native and candidate C development/release.
Invalid node/selection, exact-fit/exhausted work, depth and sticky failure checks
pass. Source check passes 407 files / 5160 MIR functions; final tests verify the
updated lock and original/copy hashes. A reserved-name fixture was corrected
before the reference oracle passed.

Next: byte-view loop written/synthetic AST, broader classic-for clause retention,
then definition/import/aggregate extraction and template closure integration.
Full image/current-source fixed point, Linux/full suite and hosted CI are not
rerun. No public API, SDK ABI, DB, CI/runner or product timeout changes; no push.
Receipt: [evidence](../ops/evidence/candidate-generic-source-functions-20261013.json).

Source expression JSON for generic definitions (2026-10-13)

Source expression JSON now preserves exact Go AST field order, null/omitted
fields, byte spans and scanner Cooked data for literals, canonical paths,
groups, prefixes, binary operations, calls with sparse argument labels,
explicit type arguments and casts, field/index access, array lists/repeats,
record initialization (including shorthand and explicit generic arguments),
contextual variants, propagation and all four slice-bound forms. Parser list
links are validated and emitted in source order. Generic record constructors
project their type arguments and original path rather than a synthetic
specialization child. Omitted slice bounds retain Go's precise spans.

Sixteen authored sources / 110 Go expression nodes match the real syntax-only
frontend JSON byte for byte. Three tests pass Go native and candidate C
development/release, including invalid/internal IDs, unsupported control nodes,
exact-fit/exhausted work, depth admission and retained failure state. Source check
passes 405 files / 5149 MIR functions; final tests verify the updated lock and
exact original/copy hashes. An initial malformed fixture escape was corrected
before the oracle passed.

Blocks, control-flow and statement JSON remain next, then complete function and
aggregate definitions, resolved imports, reachable-template closures and actual
planner/dispatcher/cache integration. Full image/current-source fixed point,
Linux/full suite and hosted CI are not rerun. No public API, SDK ABI, DB,
CI/runner or product timeout changes; no push. Receipt:
[evidence](../ops/evidence/candidate-generic-source-expressions-20261013.json).

Source declaration metadata and documentation spans (2026-10-13)

The retained syntax now produces the exact Go Function header JSON: written
name/owner, public/unsafe flags, generic parameters, attributes and their raw and
Go-decoded string values, parameter names/types, result type and original byte
spans. The shared token helper canonicalizes significant-token paths; the
documentation helper follows exact standalone consecutive /// attachment across
LF/CRLF, indentation and BOM. Blank lines, ordinary/block comments and trailing
same-line documentation break attachment. The private attribute parser now
retains byte/C-string arguments already admitted by Go syntax; their Cooked
field remains empty. All helpers share the existing bounded JSON work ledger.

Eleven authored sources match the actual Go syntax-only frontend, with five
attached documentation spans. Four tests pass Go native and candidate-generated
C development/release, including qualified paths, UTF-8, HTML/separator escaping,
invalid selections and exact-fit/exhausted work. Source check passes 403 files /
5136 MIR functions; final tests verify the unchanged lock and original/copy hashes.

This writes only function header fields; Body is still absent. Function body and
aggregate definition JSON, resolved imports and reachable-template closures,
actual planner facts and dispatcher/cache consumers remain next. Current-source
fixed point, full differential, Linux/full suite and hosted CI are not rerun.
No public API, SDK ABI, DB, CI/runner or product timeout changes; no push. Receipt:
[evidence](../ops/evidence/candidate-generic-source-declarations-20261013.json).

`generic_source_type_json` reads retained syntax type records into the exact Go
`ast.TypeReference` JSON used by generic source definition digests. The output
retains original spelling/spans, paths, mutability, elements/arguments, C
function parameters/results, array lengths and ABI, including `null` fields and
Go string escaping. The shared writer retains the 1048576 work/output bound.
The real Go frontend fixture contains six authored sources and fifteen type
values. Four authored tests pass Go native and both candidate C profiles; source
check passes 400 files / 5121 MIR functions. Named paths normalize token trivia
without changing written type text. The verifier's opt-in isolated-native mode
hash-verifies 29 copied local source/test files and executes all four tests in
26.38 seconds; default verification and CI/runner settings remain unchanged.
Full declaration/body/aggregate JSON, resolved imports and template
dependency-closure extraction remain open, along with dispatcher/cache
consumption and current-source fixed-point/Linux/full-suite/hosted-CI evidence.
No public API, SDK ABI, DB, CI/runner configuration or product timeout change;
no push. Receipt:
[evidence](../ops/evidence/candidate-generic-source-types-20261013.json).

`generic_specialization_identity` computes the private `g0_` identity of
`internal/genericplan/specialization.go`. It frames the domain, declaration ID,
owner and definition digest, followed by ordered type arguments with their
owners, canonical abilities and definitions, resolved capability constraints
with argument definitions and selected provider digests, and sorted unique
dependency specialization IDs. Noncanonical names, digests, abilities, ranges
and ordering reject under the existing 1048576 shared work/output bound.
Distinct symbolic constraints that resolve to the same constraint remain in the
stream, matching the Go planner.

Four authored tests pass Go native and both candidate C development/release
profiles. Twelve records match real Go planner IDs, including the current SDK's
`Vec<i32>` and `Vec<i64>`, two type arguments and duplicate resolved constraints.
Definition, type, ability, capability, provider and dependency changes invalidate
the identity; malformed records and exact-fit/exhausted work histories pass.
Source check passes 398 files / 5108 MIR functions; native tests verify the lock.

Resolved planner facts are caller-supplied. Typed-source template definition and
reachable-template closure extraction, dispatcher wiring and cache consumption
remain pending. This does not admit providers or executable specializations.
No full image, corpus, current-source fixed point, Linux/full suite or hosted CI
is rerun for this bounded encoder slice. No public API, SDK ABI, DB, CI, runner or
product timeout change; no push. Receipt:
[evidence](../ops/evidence/candidate-specialization-identity-20261013.json).

`module_generic_interface` computes the private source-interface identity of
`internal/moduleinterface/generic.go`. The encoder frames the generic and ordinary
algorithm names, appends the raw ordinary v5 digest, the template count and each
module/name/nonzero closure triple in canonical order, then produces lowercase
`fygif0` text. Sorting detached indices preserves the caller's original records.
Missing modules, empty names, duplicate keys, ordinary-public name collisions,
invalid ranges and malformed/zero closure digests reject within the existing
1048576 shared work/output bound. `generic_identity::digest_text` also matches
Go's domain-separated definition digest algorithm.

Four tests pass Go native and both candidate C profiles. Five interface hashes
and four definition digests match the real Go implementation; the canonical full
stream matches the oracle fixture. Public/template changes invalidate the hash,
input ordering does not, and exact-fit/exhausted histories pass. Source check
passes 397 files / 5084 MIR functions; native tests verify the source lock.

Closures are caller-supplied. Typed-source definition and reachable-template
closure extraction and dispatcher/cache consumers remain pending. The resolved
specialization/Vec encoder is covered above. No full image, corpus or
current-source fixed point is rerun.
No public API, SDK ABI, DB, CI, runner or product timeout change; no push. Receipt:
[evidence](../ops/evidence/candidate-generic-public-interface-20261013.json).

`project_coordinate_graph::collect` walks the retained workspace graph and
produces owned `module_public_type_text::Coordinates` under the same owner
numbers used by `project_compiler_input`. Root UID/version use an empty feature
set; each local instance retains its selected incoming edge's resolved features;
the authenticated SDK uses its own UID/version with no features. Local package
and feature-edge coordinates are checked before reading them. Exact instance IDs
remain separate. Private candidate read accessors expose the owned rows/text.

The committed Go workspace-loader oracle prepares four graphs: Base, Edited and
Reordered each contain five local instances plus SDK, while NoSDK has five local
instances. The authored tests match coordinates by exact instance ID and verify
compiler-input owner alignment. Editing a private common-package body changes
both feature-instance IDs while all coordinates remain identical. Feature sets
remain separate namespaces; instance text rejects as a coordinate. Owned tables
survive workspace destruction. Exact-fit/exhausted shared-work histories pass,
and the 43 fixture files remain unchanged through every test invocation.

Four tests pass Go native and both candidate C development/release profiles.
Source check passes 395 files / 5061 MIR functions; native tests verify the lock.
Dispatcher wiring, package-cache v4 consumption and generic/Vec identities remain
pending. The 384-source dispatcher closure is unchanged; no full image, corpus or
current-source fixed point is rerun. No public API, SDK ABI, DB, CI, runner or
product timeout change; no push. Receipt:
[evidence](../ops/evidence/candidate-package-coordinate-graph-20261013.json).

`project_coordinate_identity` now computes `PackageCoordinateId` in Fairy.
The encoder writes `fairyc.package-coordinate`, 00, version byte 01, tagged u/v
strings and the tagged f feature list with unsigned 64-bit network-order lengths,
and SHA-256 hashes the complete bytes into canonical lowercase `fypc1` text.
Source identity is absent by construction. The committed Go oracle reads the
actual authored inputs and matches full bytes and hashes on five fixtures,
including Unicode, NUL, empty/two-feature sets and ambiguous unframed strings.

Four authored tests pass Go native and both candidate C profiles. Empty/invalid
UTF-8 components, unsorted/duplicate/empty/invalid UTF-8 features and malformed
text reject. Encoding, validation and hashing charge the existing 16777216
workspace ledger; measured exact-fit and one-unit-exhausted histories pass.
Source check passes 394 files / 5052 MIR functions, with the lock updated and
verified by native tests. No public API, SDK ABI, DB, CI or timeout change; no push.

Root/dependency/SDK graph collection, dispatcher wiring, generic/Vec identities
and package-cache v4 consumption remain pending. The 384-source main closure is
unchanged; no full image, corpus or current-source fixed point is rerun. Receipt:
[evidence](../ops/evidence/candidate-package-coordinate-20261013.json).

Typed ordinary declaration extraction is now closed. `module_public_interface_extract`
projects nongeneric pub struct / enum / resource / function / foreign-function
declarations from the actual typed source, signature metadata and typed
`Interfaces` into the v5 descriptor, heap-sorts modules and items, keeps enum
ordered payloads, repr(C), unsafe / must_use, effects and resource abilities with
drop effects, and validates the selected owner `fypc1` coordinate. Private
declarations are skipped; the type renderer rejects only a public type that
contains a private nominal. Stale sources reject with 12, a missing owner
coordinate with 9 and exhausted work with 10. Public generic templates are
flagged; the generic and Vec identities stay separate and unfinished. The
committed Go oracle harness `tools/verify-candidate-public-interface-extract.go`
reads the committed test fixture bytes and goldens directly and reproduces the
actual Go frontend, `semantic.Analyze` and
`moduleinterface.Extract/Encode/Compute` baseline with
`bun tools/run-go.ts run tools/verify-candidate-public-interface-extract.go`,
recording a 1246-byte two-module / seven-item result in which module and
declaration reordering with a changed private body keeps the same `fyif5` hash
while a public `Leaf` field change from `i64` to `i32` moves it. The four
authored tests pass Go native and both candidate C development / release
profiles with the lock verified, and the source check passes 393 files / 5043
MIR functions. Dispatcher wiring, coordinate collection and cache v4
consumption remain open, and the 384-source closure is unchanged with no
full-image or fixed-point claim. Evidence: [typed ordinary declaration extraction](../ops/evidence/candidate-public-interface-extract-20261013.json).

Canonical public type rendering now consumes typed interface records, retained
signature metadata and exact nominal sources. Go supplies matching expectations
for 37 own and 37 dependency parameter strings, including all C scalars, scalar/
nominal views, references, tagged types, arrays and raw pointers. Dependency
namespace text uses source-independent coordinate IDs; instance IDs, malformed
coordinates, private types and stale sources reject. Four tests pass Go native
and both candidate C profiles; source check passes 392 files / 5009 MIR functions.
Typed declaration extraction, coordinate collection, generic/vector identities
and cache consumption remain open. The helper is not connected to the dispatcher
yet; its 384-source main closure is unchanged. No fresh full-image/corpus/fixed-
point claim is added. Evidence:
[canonical public type names](../ops/evidence/candidate-public-type-text-20261013.json).

The preceding source slice provides the complete ordinary public-interface v5
descriptor encoder and canonical text hash, linked to selected signature state
metadata. All five ordinary item kinds, ordered modules/items/fields/variants,
public callable facts and resource abilities match actual Go Encode/Compute
bytes and hash text. Four tests pass Go native and both candidate C profiles;
source check passes 391 files / 4997 MIR functions. Typed source/type projection,
the separate generic-template identity and cache v4 consumption remain pending.
The current 384-source dispatcher has not been rebuilt or replayed for a fixed point.
Evidence: [complete v5 encoding](../ops/evidence/candidate-public-interface-v5-20261012.json).

The preceding source slice adds the canonical public-interface v5 header/state
prefix through package signature metadata. Selected UID and raw layout hash
exclude local package/module/declaration numbers; selected canonical facts are
revalidated independently of unrelated packages. Four tests pass Go native and
both candidate C profiles, including actual Go encoder bytes and partial-budget
exhaustion. Source check passes 388 files / 4948 MIR functions. Full module/item
extraction, interface hashing and cache consumption remain pending. The current
381-source dispatcher has not been rebuilt or replayed for a new fixed point.
Evidence: [state public-interface prefix](../ops/evidence/candidate-state-public-prefix-20261012.json).

The preceding complete image covers the 380-source state-layout snapshot: a 30,962,039-byte
C packet and a 4,398,080-byte release compiler with the explicit 16 MiB stack.
Current source hashes are verified. Fresh paired CLI comparison matches all 75
semantic/resource cases, with zero status and first-code/span differences. Only
the two previous package-state differences change; the other 73 observations and
fixture source hashes remain unchanged. Sixteen extra state cases also match
first diagnostic path/line/column/code/span. A state-declaration program returns
42 in both native profiles; live owned-project ssize_t C emission succeeds.
That snapshot's consecutive generations match complete image bytes, with no
exclusions; Stage1 and Stage2 replay all 380 sources into the identical packet.
Both pass 249 lexer cases and malformed-frame checks. The maintained bootstrap
verifier reuses these native builds, checks the explicit 16 MiB stack in both PE
headers and verifies the input seed and both generations against the Go lexer.
The release compiler also passes 15 real public build/run executions, including
argument/stream/exit handling, lock failures, failed publication and scratch cleanup.
Broader/multiple diagnostics, state interface/cache admission, product timeout/
default stack, Linux/full suite and hosted CI remain open; promotion is still false.
Evidence: [fresh state CLI integration](../ops/evidence/candidate-package-state-cli-20261012.json).
[current fixed point and native CLI](../ops/evidence/candidate-state-fixed-point-native-cli-20261012.json).

The following paragraphs retain earlier snapshot evidence for comparison.

The candidate project check path enforces transitive return-origin and final
live-loan checks before owned IR. The previous borrow-gate source snapshot's
complete private Windows bootstrap retains
identical 30136053-byte C packets and identical full 4280832-byte Stage1/Stage2
images. Stage2 processes all 370 admitted repository sources in 253.81 seconds
and emits the identical packet; both generations pass 249 lexer/framing cases.
The diagnostic-output and span revisions reduce first-code/span differences
from 27 to nine, five, two and then one over 75 fresh paired observations, retaining one
status difference after c::ssize_t admission while preserving other observations. Six additional cases match
the first diagnostic's path/line/column/span and attached provider declaration
locations; multiple semantic diagnostics remain a separate parity requirement.
The diagnostic-span snapshot's 371 self sources emit and build a complete private native release
image; six selected authored tests pass Go native and both candidate C profiles.
Current-source fixed-point replay remains open; the byte-identical generations
above describe the earlier snapshot. Evidence:
[candidate diagnostic codes](../ops/evidence/candidate-project-diagnostic-codes-20261012.json).
[first diagnostic and provider spans](../ops/evidence/candidate-project-diagnostic-spans-20261012.json).

The checked bool/u8/i32/i64/u32 evaluator now admits forward module constants,
private const_eval calls with immutable parameter binding, combined graph cycles,
selected conditional branches, pure tail blocks, value-preserving casts and
static_assert. Annotated functions stay ordinary runtime callables. Graph traversal
uses explicit DFS frames; runtime calls keep the exact 32-depth limit and owners
share the 65,536-node allowance. Four new authored tests pass Go native and both
candidate C profiles; eight declaration/constant/package-budget regression tests
pass both C profiles. Twelve extra CLI cases pass their explicit expectations and
five programs pass ten native development/release executions. Source check passes
385 files / 4886 MIR functions. All 378 current self sources emit and build one
complete 4,376,576-byte release image with the explicit 16 MiB stack. The full
current-source consecutive-generation fixed-point replay remains a later gate.

All 16 declared C scalar identities, including c::ssize_t, now survive semantic
and MIR transport. Legacy C output requires POSIX.1-2008 before system headers
and proves 8-byte signed ssize_t; Windows MSVC rejects at the target guard.
Three authored tests pass Go native and both candidate C profiles. Source check
passes 385 files / 4888 MIR functions after the owned C change. The previous
semantic snapshot's 378 self sources build one
complete 4,376,064-byte release image with an explicit 16 MiB stack. Owned project
C source now carries ssize_t through foreign calls and locals; both standalone
output and partitioned headers select POSIX before system headers. Three authored
tests pass Go native and both candidate C profiles, with identical emitted C;
Windows MSVC rejects at the explicit target guard. A fresh full
dispatcher image remains to be rebuilt; Linux positive execution is unverified.
The fresh 75-case comparison matches 73 cases, retaining one status gap (package
state) and one first-code/span gap (package-state codec). Six live CLI cases match
reference exit and first diagnostic code/span, including E1014 named arguments.
Broader/multiple diagnostics and current-source fixed-point replay remain open.
The grammar has no user scalar type-alias declaration, so no alias gate is added.
Extra probes record Go reference differences: Go accepts an ill-typed unselected
if/logical branch and reports E4302 for wrong const-eval arity while candidate
uses E1010. Full semantic parity is not claimed. Evidence:
[constant arithmetic](../ops/evidence/candidate-constant-arithmetic-20261012.json),
[connected module constants](../ops/evidence/candidate-constant-engine-20261012.json),
[const-eval declarations](../ops/evidence/candidate-const-eval-declarations-20261012.json),
[const-eval execution](../ops/evidence/candidate-const-eval-calls-20261012.json),
[named-argument diagnostic](../ops/evidence/candidate-argument-label-code-20261012.json),
[c::ssize_t semantic identity](../ops/evidence/candidate-ssize-semantic-20261012.json),
[owned project C transport](../ops/evidence/candidate-owned-ssize-transport-20261012.json).

Package state declarations now retain owned UID/name bytes, ordered cell/map codec
facts, canonical field prefixes and layout hash. Package-instance coordinates own
uniqueness; malformed authority, duplicate names and unsupported storage codecs
retain E7201/E7202/E7203 source spans. Scanner string hex transport is decoded
before UID validation. Ordinary type binding and annotation keep storage codecs
outside the executable scalar graph. Project linking copies the state plan into
owned IR; verification recomputes canonical identities and rejects alterations.
Four tests pass both candidate C profiles. Go native coverage reuses three
unaffected metadata/hash tests and freshly replays the affected linked-IR test.
The actual Go builder supplies matching canonical prefix/hash expectations.
Source check passes 387 files / 4942 MIR functions. The full dispatcher and CLI
comparison are being rebuilt; the last 73/75 result remains the older snapshot.
Public interface/cache admission, broader diagnostics and promotion remain open.
Evidence: [package state layout](../ops/evidence/candidate-package-state-layout-20261012.json).

Public borrow admission is complete, while full frontend/executable
parity, product timeout/host-stack integration, Linux/full suite and same-head
hosted CI remain promotion requirements. No shipping Go compiler or SDK ABI
change, release or push is claimed. Evidence:
[borrow-gate admission and fixed point](../ops/evidence/candidate-borrow-gate-admission-20261012.json).

## Purpose

This document is the executable ownership checklist for the roadmap self-hosting gate. It does not start a Fairy-written compiler and it does not weaken the Go compiler's role as bootstrap and reference semantics implementation.

A Fairy-written compiler-component PR may start only when every `start` row below is `ready` on the same `main` commit. Promotion of that candidate into a required bootstrap path additionally requires every `promotion` row to be `ready`. Evidence must originate from Fairy source where the row claims source or native behavior. C-only probes are supporting evidence, not substitutes for generated Fairy execution.

## Gate rule

The start gate is closed while any `start` row is `blocked`. A row becomes `ready` only when its linked source, fixture, or command is maintained by the repository and the ordinary CI lanes exercise the claimed behavior. The first bounded candidate may provide the evidence for a `promotion` row; that row is not a circular prerequisite for starting the candidate. A future regression re-closes the affected gate.

| Prerequisite | Gate | Status | Current evidence | Blocking work |
| --- | --- | --- | --- | --- |
| Modules and imports | start | ready | Package/module parsing, workspace resolution, public interfaces, locks, and package build/run/test contracts under `docs/package/`, `docs/cli/`, `internal/package*`, and compiler frontend fixtures | none |
| Owned strings and byte text | start | ready | `std::text`, `std::bytes::Buffer`, generated Buffer/String transfer fixtures, runtime contracts under `docs/runtime/runtime-stdlib-v0.md` | none |
| Containers required by a compiler port | start | ready | Windows SDK-owned move-only `std::vec::Vec<T>` in ADRs 0388–0392; token/diagnostic storage and issue #44 reconciliation below; ordinary CI run 34325894239 at `a76b09dea75f7b953048a71c1d0481469d64ba24` | none for the bounded Windows AMD64 MSVC lexer candidate; broader container and target admission remain separate |
| `Result` and explicit propagation | start | ready | Language Result contract, Result Flow, MIR verification, interpreter and native fixture coverage | none |
| Resource moves and cleanup | start | ready | Resource Flow, move-only resources, explicit abilities/burn boundaries, MIR cleanup verification, interpreter traces, and native resource fixtures | none |
| C FFI bootstrap boundary | start | ready | `unsafe extern C fn`, target-aware scalar/pointer/aggregate ABI work, constrained-C backend and bidirectional native fixture lanes | none for bootstrap use; #47 remains a broader public tagged-aggregate ABI gate |
| Package build graph | start | ready | Stable workspace build/run/test, exact-root `fairy.toml`, lock validation, package interfaces and cache identity | none |
| Source maps and durable source identity | start | ready | Lossless source ownership, source spans, Atlas durable anchors, emitted-source mapping and diagnostic source-location contracts under `docs/atlas/` and compiler fixtures | none; #48 is editor tooling, not a bootstrap prerequisite |
| Diagnostics suitable for differential comparison | start | ready | Stable diagnostic catalog/codes, structured diagnostics, source anchors and compile-fail fixtures under `docs/diagnostics/` | none |
| Test runner suitable for bootstrap comparison | start | ready | Stable package test discovery/execution plus versioned list/report JSON documented by the CLI and exercised by native test fixtures | none |
| Reproducible two-stage bootstrap evidence | promotion | blocked | The Go compiler is the only authoritative stage-0 implementation; no Fairy compiler stage has been admitted | the first bounded candidate must produce matching stage1/stage2 artifacts and differential results |

## First component after the gate opens

The public vector release removes the absence of a typed sequence as a blocker; it
does not by itself prove that the compiler port's container requirements are met.
Assess the lossless lexer first, using the shipped `examples/std_vec_quickstart`
and `TestWindowsVectorStableCLI` as integration evidence, not as a compiler-workload
substitute. Mutable element access, iteration APIs, nested vectors and non-Windows
vector execution are not part of ADR 0390. Add only capabilities required by the
demonstrated port workload; do not make every possible container feature a new
self-hosting prerequisite. A workflow's test command or local native success alone
does not establish the required same-commit CI result.

ADR 0392 adds a maintained token-storage prerequisite probe with kind/trivia tags,
source-byte bounds and an owned cooked Buffer inside each vector element. Ordinary
Windows CLI execution and separate injected-failure/allocation-accounting tests
cover that representation. The subsequent record-transport assessment below adds
source reconstruction and diagnostic storage evidence without implementing a lexer
or implementing token recognition. The same-commit hosted evidence below now closes
the container start row for the bounded Windows candidate.

### Current lexer storage assessment

The Go scanner in `internal/lexer/lexer.go` appends completed `Token` and
`diagnostic.Diagnostic` values. It does not update previously appended elements.
`internal/lexer/token.go` reads their ranges to recover source spelling and
reconstruct the original bytes. `internal/lexer/documentation.go` also reads
adjacent tokens without modifying the sequence. This access pattern maps to the
released vector subset as follows:

| Reference requirement | Admitted representation and operation | Maintained evidence |
| --- | --- | --- |
| Immutable original bytes, including malformed UTF-8 | Owned Buffer with shared byte view | Native record roundtrip compares exact reconstructed output bytes |
| Token kind, trivia kind and byte range | Scalar fields in a directly named resource; `usize` for nonnegative byte offsets | Token field output compared with Go-produced records |
| Canonical `Token.Cooked` string | Owned Buffer field, retaining the reference string bytes without decoding or reinterpretation | Cooked length and every byte compared after vector growth |
| Ordered diagnostic code, repair, token text and range | Directly named resource with three owned Buffers and scalar offsets | Diagnostic field output compared after growing beyond the initial vector capacity |
| Append completed values, read by index, then destroy | Fallible push, owner-bound shared `at`, reverse cleanup | Public CLI roundtrip plus ADRs 0388–0392 ownership, failure and cleanup probes |

`TestWindowsVectorLexerRoundTrip` in
`internal/cli/vector_lexer_roundtrip_test.go` obtains records from the Go lexer,
generates Fairy source that owns them in separate token and diagnostic vectors,
then compares exact native output under low and optimized Windows MSVC. Its
bounded cases cover empty input, BOM, CRLF, documentation/block trivia, a Unicode
escaped literal, invalid UTF-8, and unterminated string/comment diagnostics.
The source reconstruction and diagnostic comparison run through ordinary `run`;
the optimized run uses the resulting lockfile through `--locked`.

This is evidence for container representation and access, not a differential
tokenizer: the Fairy program receives precomputed records. It does not establish
recognition, recovery, budget-exhaustion behavior, full corpus coverage, diagnostic
sorting, source discovery or stage1/stage2 reproducibility. Allocation-failure
evidence remains in the independent ownership probes, not the record comparison.
The current scanner's append/read pattern supplies no requirement for mutable
element access, iterator syntax or nested vectors; add those only if an actual
port operation demonstrates the need.

### Same-commit start evidence and issue #44 reconciliation

[Frontend run 34325894239](https://github.com/0disoft/fairyc/actions/runs/34325894239)
completed successfully for `main` commit
`a76b09dea75f7b953048a71c1d0481469d64ba24` on 2026-09-09. Its ordinary Windows
lane runs the complete Go suite without cached results and the Windows native
smoke. The Linux frontend, race checks and downstream C differential, sanitizer,
layout, ABI and product-build job also passed. This is evidence for that exact
commit, not a promise that later revisions cannot regress.

Issue #44's acceptance items map to the released Windows subset as follows:

| Acceptance item | Repository evidence |
| --- | --- |
| Bounded source, HIR, MIR and backend admission | ADRs 0384/0385 and 0388–0390; ordinary stable CLI vector execution in `internal/cli/vector_support_test.go` |
| Move-only layout and allocator ownership | SDK `internal/sdkstdlib/sources/optional/std/vec.fy`; private storage layout and SystemHeap ownership in ADR 0388 |
| Inert and admitted resource elements | `internal/compiler/vector_generic_test.go`, `vector_resource_test.go`, `vector_sdk_test.go` and `nested_resource_test.go` |
| Reserve/grow, push/pop, truncate/clear and borrowed access | SDK operations above; `internal/cli/vector_truncate_test.go`, `vector_support_test.go` and the nested-resource ownership probe |
| Initialized prefix and allocation-failure atomicity | `TestPrivateVectorNestedResourceTokenOwnership` forces initial/growth failures, checks intact returned donors, pop ownership, retained capacity, reverse cleanup and zero live allocations |
| Low and optimized generated-Fairy execution | Native compiler ownership probes and stable CLI token/diagnostic record roundtrips, both exercised by the successful Windows lane |

This reconciliation does not admit arbitrary user generic aggregates, mutable
views, iteration, nested vectors or non-Windows vector execution. The lexer
storage workload needs none of those features. Issue tracking state alone does
not control the gate: these maintained sources and same-commit CI establish its
bounded start evidence. Full tokenizer corpus comparison remains the acceptance
condition for the subsequent Fairy component, not a claim made by these probes.

The first Fairy-written compiler component is the lossless lexer/token stream producer, not the parser, type checker, MIR verifier, optimizer, backend, package resolver, or runtime.

The lexer is selected because its observable contract is bounded and can be compared byte-for-byte and token-for-token against the Go reference without introducing a second semantic authority. The initial port must consume the same source bytes and language revision and emit a canonical token record containing token kind, exact source range, cooked literal data where applicable, and diagnostics.

The first port must not add syntax, recovery rules, token kinds, preprocessing, host-dependent Unicode behavior, filesystem discovery, package semantics, or compiler plugins.

## Differential acceptance

A Fairy-written component is accepted only when the Go reference and Fairy candidate produce identical canonical results for the checked-in positive, malformed, previous-revision, and mutation corpora. Comparison covers output bytes or canonical records, diagnostics and source ranges, process status, and deterministic artifact hashes where artifacts exist.

Native success alone is insufficient. The candidate must also prove failure behavior against the same corpus. Any mismatch keeps the Go implementation authoritative and blocks promotion of that component.

## Bootstrap stages

The Fairy-written candidate now executes authenticated scalar/resource/struct
Vec storage, `std::bytes::Buffer`, String/CString and UTF-8 bodies, including
allocation failures and ownership cleanup. Authenticated `std::io` console
output/input bodies also execute against the original Windows runtime, with
exact effect authority and failure results independently verified. The
[candidate progress](../ops/candidate-module-progress-20261008.md)
records the exact native evidence. Copied byte iteration now also executes
through existing owned HIR/MIR loops, including one-time slice bounds and
resource exits. Fixed byte-array field subranges also preserve original storage and
execute in both candidate profiles. Local mixed-origin payload selections now retain every possible owner
in HIR and independently recomputed MIR. Indexed reachability, finalized-origin
caching and owner-indexed exclusive access checks pass the scanner's prior HIR
work limits without increasing the budget. Development and release candidates
compile original scanner and SDK bodies whose Token/cooked/diagnostic storage,
byte-budget diagnostics and cleanup run in both native C profiles. MIR expression
helpers also remove the development scanner's default-stack overflow. The
checked-in current/previous, malformed, mutation and budget corpora now match the
Go lexer through candidate-emitted C in both native profiles (7670 cases per
profile, plus malformed framing). This proves the scanner component scope.
Authenticated file SDK extern declarations and C record authority now also
pass development/release native API tests, including 14 rejected mutations.
A separate private Result encoding now preserves enum/record and
Option-record/record pairs without changing prior valid IDs or analysis bounds.
Original File SDK open/read/write/sync/close/replace bodies now also execute
with the Windows runtime in both candidate and native profiles, including
UTF-8/NUL data, buffer preservation and failure/effect cases. Project integration
remains a separate check. The actual bounded source/manifest loader is now
a standalone Fairy leaf; both candidate/native profiles prove its byte limits,
early errors and handle/allocation cleanup, and the prior CLI import cycle is
removed.
An opt-in SDK compaction pipeline now also compiles the authored file-token path
with the original SDK source owners. Exact source/token spans and all user
bodies survive compaction; existing join and work bounds are unchanged. Both
candidate profiles emit identical C (769317 bytes for the observed program),
and both native profiles reproduce the prior CLI's four file cases, including
their exact streams and asserted statuses. Unused invalid user code, private SDK
access and missing filesystem effects remain rejected. Byte-loop induction
spans no longer incorrectly extend unrelated call loans; real later-argument
conflicts still reject. Evidence:
`docs/ops/evidence/candidate-file-tokens-fixed-dev-20261009.json`,
`docs/ops/evidence/candidate-file-tokens-fixed-release-20261009.json` and
`docs/ops/evidence/candidate-byte-loop-calls-release-20261009.json`.
The complete command dispatcher and project-driver integration are still open.
Full-dispatcher self-source admission has now been observed directly: MIR
lowering was split into six parser-admitted units while preserving all original
function bodies and byte-identical generated C for the focused regression.
The current 142-file input still rejects at parsing; HIR model, type checking
and HIR lowering source splits remain. These observations retain the existing
budgets and do not establish whole-compiler native execution or promotion.
The owned HIR model has subsequently been split into a factory/query owner and
eight proof helpers. Its nine files now parse with the same guard bodies and
publication order. Existing thirteen drop-link mutations still reject; focused
generated C remains identical and passes twelve native executions. Type checking
and HIR lowering remain the two measured self-source parser blockers. The full
150-file input remains below 1MiB (1047406 joined bytes) and still rejects.
Project-driver and platform work remain. These scanner results do not establish
the complete compiler's stage1/stage2 or promotion gates.
Type checking now also parses as eight responsibility-based files. Extracted
bodies and source-depth accounting are retained; focused C is byte-identical,
twelve native runs pass and eight type-error diagnostics match the prior driver.
The full input is now 157 files and 1054271 joined bytes, still rejected at
parsing. HIR lowering partitioning and source deduplication below the unchanged
1MiB budget remain prerequisites. Evidence:
`docs/ops/evidence/candidate-type-source-split-20261009.json` and
`docs/ops/evidence/candidate-dispatcher-admission-after-type-split-20261009.json`.
HIR lowering has subsequently been split into four parser-admitted files with
all five function bodies retained. The full dispatcher now passes parsing and
reaches phase 1 code 10 at the cumulative source cap: 160 files and 1055400
joined bytes. Source deduplication below 1MiB remains necessary before later
admission stages can be evaluated. Focused C is identical and twelve native
runs pass. Evidence:
`docs/ops/evidence/candidate-hir-lower-source-split-20261009.json` and
`docs/ops/evidence/candidate-dispatcher-admission-after-hir-lower-split-20261009.json`.
Shared CLI stages/output and typed HIR default factories subsequently reduced
the full input to 1048516 joined bytes across 160 files, below the same 1MiB cap.
The actual dispatcher now passes parsing and cumulative source admission but
rejects at module-program phase 1 code 9 on `mir_output::row`'s by-value
`usize[20]` signature. Source-set fixed-array type registration is still open.
Forty-six CLI cases retain exact streams/status, five generated C outputs match
the prior driver and twenty focused native runs pass. Only 60 source bytes of
headroom remain; this does not establish full-dispatcher compilation or
stage1/stage2. Evidence:
`docs/ops/evidence/candidate-cli-source-dedup-20261009.json` and
`docs/ops/evidence/candidate-dispatcher-admission-after-source-dedup-20261009.json`.
Source-set fixed-array signatures now retain canonical element identity and
bounded rank-one/rank-two lengths, including imported nominal/enum and Option
arrays. Three API tests pass in each native profile; five source-set programs
produce identical C across candidate profiles and pass twenty native runs.
The actual dispatcher now passes usize[20] signature registration and reaches
SDK compaction phase 8 code 11 at its reference-index cap. Current input is
160 files and 1046965 joined bytes, within 1MiB after moving AST-join notes out
of compiler source. Reference-index/work reduction and accurate limit/error
classification remain necessary. Evidence:
`docs/ops/evidence/candidate-module-array-signatures-parity-20261009.json` and
`docs/ops/evidence/candidate-dispatcher-admission-after-array-signatures-20261009.json`.
The reference cap now reports limit code 10 separately from allocation code
11. The development driver reproduces phase 8 code 10 at the same span on the
160-file, 1047016-byte input, confirming the index-cap blocker. Two existing
native SDK compaction regressions pass. Index reduction still remains before
full-dispatcher compilation can advance. Evidence:
`docs/ops/evidence/candidate-reachability-limit-classification-20261009.json`.
Per-name reachability storage has subsequently been replaced by checked
declaration expression ranges. A 17600-reference, 22-module native case retains
all user AST/source facts; SDK diagnostics and the four real file-token cases
remain exact with identical generated C. The full 160-file input (1046901
joined bytes) passes the previous reference-index blocker and now reaches the
phase-8 lookup-work cap in `numbers.fy`. Catalog lookup reduction remains open.
Evidence: `docs/ops/evidence/candidate-reachability-ranges-20261009.json` and
`docs/ops/evidence/candidate-dispatcher-admission-reach-ranges-20261009.json`.
Module-local catalog ranges and actual probe accounting now pass name
reachability on the 160-file, 1048411-byte dispatcher. Two native catalog and
three SDK compaction regressions pass in development. The next phase-8 work
blocker occurs during AST copying at `std/vec.fy:0..0`; full-dispatcher C/native
execution and stage1/stage2 comparison remain open. Evidence:
`docs/ops/evidence/candidate-catalog-ranges-20261009.json` and
`docs/ops/evidence/candidate-dispatcher-admission-catalog-ranges-20261009.json`.
Token-qualified local-name detection and cached SDK function ownership further
advance the 1048534-byte dispatcher to AST copying at `bodies.fy:0..0`.
Three development native regressions pass, including 17600 long local names
under 50000 reachability work units. Moving unchanged user AST owners is next.
Evidence: `docs/ops/evidence/candidate-ast-owner-cache-20261009.json` and
`docs/ops/evidence/candidate-dispatcher-admission-owner-cache-20261009.json`.
The opt-in CLI now moves unchanged user AST/scanner owners after all SDK
replacement copies and transfer capacity are ready. Three development native
tests retain exact observations and validation, including a 22-module move
under 50000 work units. The 160-file dispatcher (1048460 joined bytes) passes
phase 8 and reaches phase 2 code 10 at the joined scanner's storage rejection.
Cooked-storage reduction is next; joined AST/HIR/MIR and stage1/stage2 remain
open. Evidence: `docs/ops/evidence/candidate-ast-owner-transfer-20261009.json`
and `docs/ops/evidence/candidate-dispatcher-admission-in-place-20261009.json`.

The joined scanner now shares equal canonical cooked buffers, preserving token
kinds, spans, diagnostics and snapshot indices. Its original 65536-byte pool
and 16777216 work bounds remain fixed. Mutable scalar-array field views through
record parameters retain the original owner. Eight development native API
tests and fourteen array-view low/optimized native runs pass; file-token
stdout/stderr/status remain identical to the reference in two native profiles.
The 160-file dispatcher is now 1045810 joined bytes. Its 69949 logical cooked
bytes occupy 41226 owned bytes, with zero scanner diagnostics. The next observed
failure is phase 2 code 10 at module 18 (`module_cli.fy`): compacted cumulative
AST entries reach 1049 against the unchanged 1024-entry bound. Partitioning or
reducing those arenas is next; complete joined AST/HIR/MIR, full-dispatcher C,
and stage1/stage2 are still open. Evidence:
`docs/ops/evidence/candidate-cooked-storage-20261009.json`,
`docs/ops/evidence/candidate-dispatcher-admission-cooked-intern-20261009.json`,
`docs/ops/evidence/candidate-file-tokens-cooked-intern-20261009.json` and
`docs/ops/evidence/candidate-mutable-record-array-view-native-20261009.json`.

The AST partition layer now preserves the actual 160-module dispatcher in
twelve independently bounded joined owners. The 1046245-byte full input still
fits 1MiB; all 152020 expressions and 11509 entries are retained across groups.
Eight development native API regressions pass. Each group remains within the
original joined AST bounds, and range-join errors retain full-source module
coordinates. This does not yet connect groups to body binding, HIR/MIR, C output
or final linking. The current command dispatcher still uses one joined owner.
Cross-unit declaration identity is the next compiler connection; stage1/stage2
and promotion remain open. Architecture:
`docs/architecture/candidate-compilation-units-v0.md`; evidence:
`docs/ops/evidence/candidate-module-units-20261009.json`.

Cross-unit declaration binding now retains original module/declaration/span
coordinates in a separate owned target table, with alias sharing and private
lookup enforcement. Stale provider bytes outside the selected unit are rejected.
Five development native regressions pass, and the actual 160-module dispatcher
(1046447 joined bytes) binds all twelve ranges while preserving 152032 expressions
and 11513 entries. Measured work peaks at 134589 against the existing 1048576
bound. Original parser/signature owners are not retained by the Bound result.
Body name/type checking must still import provider layouts and signatures using
these coordinates; HIR/MIR, C and CLI integration remain open. Evidence:
`docs/ops/evidence/candidate-module-unit-bind-20261009.json`.

Owned provider interfaces now retain canonical field shapes and callable
signatures with their declaration flags, foreign link names, resource abilities
and declared cleanup authority. Generic function interfaces remain deferred
templates. Field queries enforce resource privacy, and fresh catalog lookups
revalidate retained bindings. Six development native regressions pass, including
queries after source/metadata removal and stale/private provider rejection.
All 160 modules (1046575 bytes) import interfaces for thirteen groups; maxima
are 131 shapes, 910 fields, 198 callables and 401 type nodes within existing
bounds. Actual unit body typing must still consume these interfaces and resolve
local annotations/templates before HIR/MIR/C/CLI integration. Evidence:
`docs/ops/evidence/candidate-module-unit-interfaces-20261009.json`.

Unit interfaces now own canonical annotation roots for local entries,
declaration results, fields, casts, explicit type arguments and byte-loop
binders. Resolution uses original module coordinates; generic declaration roots
remain explicitly deferred. Eight development native regressions and all
thirteen dispatcher groups pass (10289 concrete roots, 69 deferred, maximum 409
graph nodes). Actual expression/body typing and template specialization must
consume this mapping next; remaining CLI/HIR/MIR/C integration and stage proofs
stay open. Evidence: `docs/ops/evidence/candidate-module-unit-annotations-20261009.json`.

Owned explicit generic signature instances now preserve provider coordinates,
ordered canonical arguments, declaration flags and per-expression request
identities. Thirteen native regressions pass, including SDK resource admission
and failure rollback. Initial AST-only groups exposed specialization pressure at module 37
(E4503); the semantic refinement below resolves that pressure. Body instantiation
and all unit compiler consumers remain open. Evidence:
`docs/ops/evidence/candidate-module-unit-generics-20261009.json`.

Semantic range admission now resolves specialization pressure by splitting only
code10/E4503 ranges. Two new native planner regressions pass, and the thirteen
prior kernel regressions retain matching source hashes. All 160 dispatcher
modules (1048186 bytes) now admit fifteen ranges after two splits; at most 61
instances occur in one range under the existing 64 cap. Per-range instance
counts sum to 476 for 1281 explicit requests; fourteen generic-body requests
remain deferred. This completes current-input range/interface signature
planning. Owned template body cloning, nested requests, structural cross-unit
instance deduplication, expression/body typing, all IR/backend/CLI consumers and
stage proofs remain open. Evidence: `docs/ops/evidence/candidate-module-unit-plan-20261009.json`.

Transitive generic function request graphs now resolve forwarded arguments and
private provider targets, reject specialization recursion/depth exhaustion, and
own SDK intrinsic and Vec aggregate records at original expression coordinates.
Ten native regressions and all eighteen dispatcher units pass (160 modules,
1048277 bytes, maximum 57 function instances per unit). Per-unit counts sum to
592 function instances, with 116 nested function edges, 594 intrinsic records
and 100 aggregate records. Provider function and SDK Vec cleanup AST ownership,
resource cleanup specialization, expression/body typing and all IR/backend/CLI
consumers remain open. Evidence:
`docs/ops/evidence/candidate-module-unit-generic-dependencies-20261009.json`.

Requested generic function AST ownership now passes development native tests
(7/7) and the full 160-module source corpus in 25 bounded units. Provider source
streams, body-local node links, canonical annotation overlays, signatures and
per-instance nested request edges are retained after original owners are
released. The original plus cloned AST sizes remain inside existing admission
bounds. SDK Vec cleanup AST ownership,
resource cleanup contexts and implicit drop dependencies, local expression/body
typing, structural cross-unit links, layouts/effects, HIR/MIR/C/CLI and actual
stage1/stage2 remain open. Evidence:
`docs/ops/evidence/candidate-module-unit-bodies-20261009.json`.

Required resource and SDK Vec cleanup AST contexts now pass development native
tests (7/7) and the 160-module, 1048472-byte corpus as 26 bounded units. Declared
effects/abilities and compiler-owned dropping receiver facts are retained;
cleanup annotations and explicit generic request edges are owned separately.
The authenticated SDK take-slot leaf is folded only in cleanup contexts with
checked signature/body structure and retained unsafe provenance to preserve the
16-per-template quota. Implicit/nested drop dependencies, transitive cleanup
effects, body symbol/expression typing and inline provenance consumption,
cross-unit links/layouts, HIR/MIR/C/CLI and actual stage1/stage2 remain open.
Evidence: `docs/ops/evidence/candidate-module-unit-cleanups-20261009.json`.

Type-level nested drop requirements now use an immutable graph of owned
interfaces. Contained field/Option/Result/array/Vec effects and no-drop abilities
propagate to a fixed point; references and pointers have no drop edges.
Unprojected nominal metadata is explicitly incomplete and cannot be queried as
a proven drop requirement. Unit admission includes this bounded graph without
raising type, dependency or work limits. Cleanup-body locals, inferred types,
move-aware obligations and new generic-body local field projection still need
body typing/ownership consumers. Cross-unit links/layouts, HIR/MIR/C/CLI and
actual stage1/stage2 remain open. Evidence:
`docs/ops/evidence/candidate-module-unit-drops-20261009.json`.
Development check/probe check/build and seven native tests pass; the current
160-module, 1048472-byte corpus retains 26 bounded units, with drop work at most
6036 within the unchanged 1048576 bound. This does not establish executable
drop lowering or whole dispatcher native compilation.

Requested generic functions and cleanup contexts now collect explicit annotation
and named constructor type demands before body cloning. Newly discovered
private types request provider field shapes and cleanup contexts; cleanup
helpers feed back into the same bounded closure. Original context/type root/
expression coordinates and diagnostic spans are retained. Development check
and eight native regressions pass. Ordinary-call result inference, body symbol/
expression typing, move-aware obligations and inline provenance consumption
remain open, along with cross-unit links/layouts, HIR/MIR/C/CLI and stage1/stage2.
Evidence: `docs/ops/evidence/candidate-module-unit-body-types-20261009.json`.

Owned lexical binding for concrete bodies (2026-10-09):

`module_unit_names`, `module_unit_name_model` and `module_unit_name_walk` resolve
requested generic function and resource/SDK Vec cleanup ASTs. Current source
stamps, canonical graphs, ordered arguments and provider streams are checked
before publishing any result. Parameters, let/var declarations, payload patterns,
byte iteration, regions and the dropping receiver have separate symbol kinds.
Explicit annotation types remain canonical; inferred locals and payload types
remain unresolved. Entry/pattern/use tables and provider module/declaration
coordinates retain original expression IDs and spans.

Cleanup `this` is kind 10 with a resource type and no ordinary by-value parameter.
References record lexical unsafe function/block context; SDK storage names are
classified only in authenticated std::vec source. The safe `take_slot<T>` wrapper
has an unsafe intrinsic in its body: its source call can have `unsafe_context`
false while the separate request edge retains `inline_unsafe` provenance.
Binding records these facts without granting unsafe authority or performing
expression/flow checks. Ordinary unit bodies, expression typing, implicit drop
ownership/effect checks, inline provenance consumption, structural links,
layouts, HIR/MIR/C/CLI and full dispatcher native/stage1/stage2 remain pending.
Public language/SDK ABI, SDK sources, DB, CI and runners are unchanged. Evidence:
`docs/ops/evidence/candidate-module-unit-names-20261009.json`.

| Observation | Value |
| --- | --- |
| Development native tests | 7/7 passed |
| Dispatcher source corpus | 160 modules; 1048472 bytes |
| Admitted units | 26; 39 attempts; 13 splits |
| Per-unit function / cleanup totals | 625 / 763 |
| Per-unit symbol / reference totals | 3963 / 10161 |
| Maximum symbols / references | 260 / 648 |
| Maximum resolved-unit name work | 10639 / 1048576 |
| Maximum combined entries | 1015 / 1024 |

Lexical binding for ordinary unit bodies (2026-10-09):

module_unit_ordinary_names binds every non-generic function body within an
admitted original module range, reusing module_unit_body_ast and
module_unit_name_walk with borrowed original scanner streams. The temporary
body AST is discarded, while immutable resolved name tables retain the
original module and declaration, expression, and entry IDs and spans. Explicit
canonical annotation roots are read from the exact same bounded interface
range, and interface range mismatches and stale source stamps are rejected
before publication. Generic templates stay deferred, and requested generic and
cleanup binding stays in module_unit_names. Unit admission accounts for
combined ordinary, generic, and cleanup symbols, references, and name work
under the existing 16384, 16384, and 1048576 bounds, with the existing 128
active locals and depth 64 unchanged. Expression type checking, ordinary-call
result inference, move-aware drop obligations, unsafe and inline provenance
consumption, cross-unit structural dedup and layout, HIR, MIR, C, and CLI, and
the full dispatcher native, stage1, and stage2 remain unfinished. The public
language and SDK ABI, SDK sources, DB, CI, and runners are unchanged.

Evidence: `docs/ops/evidence/candidate-module-unit-ordinary-names-20261009.json`.


| Observation | Value |
| --- | --- |
| Development native tests | 7/7 passed |
| Dispatcher source corpus / units | 160 modules; 1048562 bytes; 26 units |
| Ordinary functions / symbols / references | 1175 / 7479 / 48136 |
| Maximum combined symbols / references | 723 / 4094 (each <= 16384) |
| Maximum combined name work | 751853 / 1048576 |
| Maximum combined entries / type nodes | 1015 / 1024; 2242 / 4096 |
| Observation execution timeout | Initial 300s run timed out after 23 units; final 600s selection passed all 26 |

Declared call results and interface closure (2026-10-09):

The fairy compiler's module_unit_calls records declared-call canonical results
and parameters, along with provider and original expression coordinates. It
also records effects, unsafe, and foreign flags after lexical binding.
module_unit_call_closure repeatedly seeds unseen ordinary callable signatures
into the existing bounded field and cleanup closure. Cleanup helper calls can
request more result types. Local name shadows and stale input are rejected.
Generic calls, closed builtins, and variant constructors stay separate.
Existing arena bounds remain unchanged, with at most 64 rounds and 1048576
units of call-projection work. Full expression and argument type matching,
inferred local types, ownership and effects, unit HIR/MIR/C/CLI, and self-
hosting all remain pending.

Public language/SDK ABI, SDK sources, DB, CI and runners are unchanged.
Evidence: `docs/ops/evidence/candidate-module-unit-call-results-20261009.json`.


| Observation | Value |
| --- | --- |
| Development native tests | 8/8 passed |
| Dispatcher source corpus / units | 160 modules; 1048562 bytes; 26 units |
| Per-unit declared / specialized call-site totals | 11189 / 2622 |
| Per-unit variant / closed-builtin totals | 610 / 4 |
| Maximum closure rounds / work | 2 / 64; 305763 / 1048576 |
| Maximum combined entries / type nodes | 1015 / 1024; 2242 / 4096 |
| Native closure-chain fixture | New call-result fields and cleanup-returned resources close after at least 3 rounds |
| Array reference regression | Kind-24 element links use right; a declared call beneath a cast retains its callee binding |
| Remaining expression consumers | Argument/operator matching, inferred-local symbol types, ownership and unsafe/inline checks |
| Deferred checks | Release, Linux, full dispatcher native and stage1/stage2; no push |

`stage0` is the current Go compiler built by the supported Go toolchain.

`stage1` is produced by `stage0` from the admitted Fairy compiler sources.

`stage2` is produced by `stage1` from the exact same Fairy compiler sources and locked inputs.

The eventual bootstrap gate requires `stage1` and `stage2` to match under the repository's declared reproducibility policy for compiler artifacts and to produce the same differential fixture results as `stage0`. Target-specific nondeterministic metadata must be either eliminated or explicitly excluded by an already-versioned artifact contract before this comparison is accepted.

## Authority and rollback

Until all rows are ready and the two-stage evidence lands, the Go compiler remains the sole bootstrap and reference semantics implementation. A Fairy port is an experimental differential implementation and may be deleted without a compatibility promise.

If a prerequisite regresses, self-hosting promotion stops. Existing Go release and bootstrap paths remain the recovery path; no release may require the Fairy-written compiler while this gate is closed.

## Non-goals

This gate does not require replacing Go, implementing the full compiler in Fairy, stage-2 trust claims, trusting-trust mitigation, a second backend, LSP/SCIP, generic containers beyond the compiler's demonstrated needs, or freezing unstable syntax and ABI decisions.

## Compilation-unit known value analysis (2026-10-09)

`module_unit_values` collects generic, cleanup and ordinary contexts. Its
immutable value, local and return tables retain provider module/declaration,
original expression/entry IDs and source spans. Declared and explicitly
specialized function results seed known values. Direct literals take local or
return context through parentheses; integer range errors are retained.
Unannotated locals infer from known initializer types, and local references and
parentheses propagate those types. Only resolved returns are matched.
Operators, casts, fields, arguments, variants, joins and builtin result typing
remain deferred, along with ownership, effects and unsafe/inline validation.
Existing canonical IDs are preserved; missing primitive/literal descriptors
append to a separate bounded graph. Stale source stamps are rejected before
publication. Existing arenas, 64 fixed-point rounds and 1048576 work remain
bounded, with code-10 admission using the existing range-split planner.

| Observation | Value |
| --- | --- |
| Development native tests | 8/8 passed |
| Dispatcher input snapshot | 160 modules; 1048562 bytes |
| Admitted units / attempts / splits | 26 / 39 / 13 |
| Per-unit context / expression totals | 2563 / 186137 |
| Known / unresolved / non-value expression totals | 58300 / 68976 / 58861 |
| Local / inferred local totals | 11442 / 539 |
| Return / checked return totals | 4783 / 2876 |
| Maximum value work / graph | 754886 / 1048576; 427 / 4096 |
| Maximum combined entries / type syntax nodes | 1015 / 1024; 2242 / 4096 |
| Preserved local / iteration bounds | 128 active locals; 64 fixed-point rounds |


The same dispatcher input snapshot is reused for comparison. The new value
modules are tested as implementation code, but are not added to the 1MiB input
snapshot. This observation is not full current-source dispatcher compilation.
Argument/operator consumers and move-aware cleanup, unsafe/inline provenance,
structural cross-unit links/layouts and unit HIR/MIR/C/CLI remain unfinished.
Public language/SDK ABI, SDK sources, DB, CI and runners are unchanged.
Release, Linux, full dispatcher native and stage1/stage2 remain deferred; no push.
Evidence: `docs/ops/evidence/candidate-module-unit-values-20261009.json`.

## Compilation-unit scalar operator typing (2026-10-09)

`module_unit_value_scalar` types builtin integer arithmetic, bitwise/shift,
comparisons, boolean logic/negation and signed unary operators within local
initializer and return roots. Pure integer literal trees remain flexible; a
typed peer anchors their scalar type while original source links retain
execution order. Direct signed i32/i64 minimum operands retain a unary minimum
marker. Unknown field/cast/builtin anchors keep literal peers unresolved, avoiding
premature i32 range errors. A primitive descriptor cache per value plan replaces
repeated graph scans. Existing canonical IDs, arenas, 64 rounds and 1MiB work
remain bounded across ordinary, requested generic and cleanup contexts.

| Observation | Result |
| --- | --- |
| Development check | 185 source files; 2298 MIR functions; passed |
| Selected Fairy native tests | 8/8 passed |
| Reference fixture acceptance / selected primary code and span | 8/8 matched; additional cascade output retained |
| Unsupported reference case | Experimental exit 2 and span, distinct from an E diagnostic |
| Integer range correction | E1013, replacing the earlier internal 1008 |

Call arguments and other expression/control/place/ownership consumers, inline
and unsafe provenance, unit HIR/MIR/C/CLI and self-hosting remain unfinished.
Full dispatcher input observations are grouped with the next argument unit;
this unit makes no new full-corpus claim. Release, Linux, full dispatcher native,
stage1/stage2 and push were not run. Public language/SDK ABI, SDK sources, DB,
CI and runners are unchanged.
Evidence: `docs/ops/evidence/candidate-module-unit-operators-20261009.json`.

## Compilation-unit argument matching (2026-10-09)

Fairy C's module_unit_argument_collect/types now adds immutable invocation and argument records across generic, cleanup, and ordinary contexts, preserving the provider module/declaration, specialization instance, ordered original expression IDs, and spans. Declared and explicit generic calls validate arity (E1010) and known argument types (E1006), along with contextual scalar literals/operators and range (E1013); positional labels are only allowed for same-module private non-generic ordinary functions per ADR0396, requiring the exact name at the same position or else E1014. Converting ref mut T to ref T records a shared reborrow, while ownership/place validation remains pending, and unknown field, cast, borrow, builtin, and variant types stay unresolved. Existing arena, work, depth, and round limits are unchanged, and code10 splits only oversized units.

| Observation | Result |
| --- | --- |
| Development check | 187 source files; 2338 MIR functions; passed |
| Selected development native tests | 9/9 passed |
| Reference acceptance/code/span | 10 matching cases; one recorded callable-value difference |
| Fixed dispatcher input snapshot | 160 modules; 1048562 bytes |
| Units / attempts / splits | 26 / 39 / 13 |
| Invocation / checked invocation totals | 12590 / 6167 |
| Argument / checked argument totals | 29822 / 20527 |
| Shared reborrow argument records | 402 |
| Maximum calls / arguments | 919 / 2687; each bounded to 16384 |
| Maximum value work / graph | 737134 / 1048576; 427 / 4096 |


The observation reuses the fixed 160-module, 1MiB input snapshot; new analysis
modules are compiled as implementation code. It does not establish full
current-source dispatcher compilation. Arguments whose field/cast/borrow/variant
or builtin types are unresolved stay pending, even when the call result is known.
The reference compiler accepts the callable-name argument fixture; the candidate
retains its existing non-value callable classification and reports E1006. That
case is recorded as a difference, not counted among the matching fixtures.
Move-aware cleanup, ownership/place and unsafe/inline validation, structural
cross-unit deduplication/layouts and unit HIR/MIR/C/CLI remain pending.
Release, Linux, full dispatcher native, stage1/stage2 and push were not run.
Public language/SDK ABI, SDK sources, DB, CI and runners are unchanged.
Evidence: `docs/ops/evidence/candidate-module-unit-arguments-20261009.json`.

## Compilation-unit scalar casts (2026-10-09)

module_unit_value_model.Expression now keeps the canonical cast target separate from the contextual expected/result type. module_unit_value_collect seeds these targets from owned generic/cleanup annotations or the original ordinary source type offsets, while retaining provider original expression/type IDs and spans and leaving stale input checks unchanged. module_unit_value_scalar applies the existing scalar_casts::allowed rules, including widening to i64 and the exact i32 / c::int bridge. Operands are checked without target context, so oversized unsuffixed literals still fail E1013. Unsupported cast pairs keep the experimental code2 and the full cast span rather than an invented E1052. Known casts propagate to inferred locals, scalar peer operators, call arguments, and returns. Unknown operands and raw-pointer conversions remain pending until the field/place/unsafe work. The existing graph/arena/depth64/round64/work1MiB bounds are unchanged. Full dispatcher observation is deferred to the next field/place unit; this does not claim a new corpus pass, stage1/stage2, or Release/Linux/push.

| Observation | Result |
| --- | --- |
| Development check | 187 source files; 2340 MIR functions; passed |
| Selected development Fairy native tests | 8/8 passed |
| Reference acceptance/code/span | 7/7 matched |
| Unsupported conversion | Experimental code 2 and full cast span; no invented E1052 |
| Retained bounds | 4096 types; 16384 expressions; depth/rounds 64; work 1048576 |


Full dispatcher observation is bundled with the next field/place unit; this unit
claims no new corpus pass. Field/place, raw-pointer casts and unsafe provenance,
move-aware cleanup, structural cross-unit links/layouts and HIR/MIR/C/CLI remain
pending. Release, Linux, full dispatcher native, stage1/stage2 and push were not
run. Public language/SDK ABI, SDK sources, DB, CI and runners are unchanged.
Evidence: `docs/ops/evidence/candidate-module-unit-casts-20261009.json`.

## Compilation-unit field access typing (2026-10-09)

module_unit_value_field now resolves known nominal fields from owned interfaces, preserving the canonical owner and type, the provider module and declaration, and the original field index alongside the original expression ID and span in an immutable value table. Shared and mutable receiver borrows are unwrapped for nominal field lookup, while resource fields outside the provider are rejected as E0103, with the internal helper 6201 normalized only by the consumer. Unknown operands remain pending, and missing, nonaggregate, or otherwise unsupported member uses fall back to the experimental code2 over the full expression span. Closed builtin fields cover ref str, byte and scalar views' len and is_empty, fixed-array len, and Result is_ok, and the work integrates with scalar peer context, casts, inferred locals, arguments, and returns. Measured shape and name comparisons are charged to the existing work budget; valid known return mismatches are now E1009 while locals and arguments stay E1006. Borrow, place, ownership, and unsafe consumers as well as pointer-cast consumers remain pending, and the existing arena, depth64, round64, and work1MiB bounds are unchanged, with code10 range splitting retained.

| Observation | Result |
| --- | --- |
| Development check | 188 source files; 2344 MIR functions; passed |
| Selected development native cases | 11 passed; 9 unchanged cases reused and 2 corrected-coordinate cases rerun |
| Reference acceptance/code/span | 9/9 matched, including imported field privacy |
| Fixed dispatcher snapshot | 160 modules; 1048562 bytes |
| Detailed observations | 24 units before the 900s process timeout; remaining 2 units passed separately |
| Units / admission attempts / splits | 26 / 39 / 13 |
| Field / checked field totals | 14031 / 6085 |
| Nominal provider field records | 6015 |
| Cast / checked scalar cast totals | 1789 / 52 |
| Argument / checked argument totals | 29822 / 25740 |
| Maximum value work / graph | 747467 / 1048576; 427 / 4096 |


The native reuse receipt retains the original 9-pass/2-failure report and proves
that the only test-file change corrected two expected declaration IDs from 1 to
2. The parser counts the module declaration; compiler/implementation/SDK/lock
identities and the other nine fixtures are unchanged. Both corrected cases were
rerun natively. No failed execution is labeled as a passing execution.
The first process admitted all 26 units, then completed detailed assertions for
24 before its 900-second deadline. The final two ranges (145..152 and 152..160)
passed with the same compiler/manifest/implementation. All ranges continuously
cover modules 0..160; the initial timeout is retained as a failed execution.
This combined observation uses the same bounded 160-module input snapshot; new unit
modules are compiled as implementation code. Full current-source dispatcher
native and stage1/stage2 remain pending. Borrow/dereference/index/slice and
place/ownership/unsafe consumers, raw-pointer casts, move-aware cleanup,
structural cross-unit links/layouts and HIR/MIR/C/CLI remain unfinished.
Release, Linux and push were not run. Public language/SDK ABI, SDK sources,
DB, CI and runners are unchanged.
Evidence: `docs/ops/evidence/candidate-module-unit-fields-20261009.json`.

## Named safe reference and place projection (2026-10-09)

The module_unit_value_place project mainly targets scalar, nominal, and resource references plus nested nominal fields. Safe reference borrow and dereference types reach inferred locals, declared and generic call arguments, scalar casts, and returns, and canonical shared/mutable types are appended to the existing 4096-type graph. Place metadata keeps the whole local owner root and scope, the expression projection ID, mutability, indirect access, and the resource-field boundary, while reference aliases propagate owner provenance across bounded fixed-point rounds. Immutable and nonplace mutable borrows and mutable resource-field borrows emit E1008 at the target span, whereas unsupported shared nonplace borrows and non-reference dereferences use experimental code2 at the whole expression span; ref *r and ref (x) stay unsupported under the reference compiler, so general reborrow support is not claimed. Field grouping retains its named root, array/view/indexed borrows and unsafe pointer dereferences remain explicitly pending, and this is type/provenance projection rather than lifetime, conflict, or move verification; the existing program/work 1MiB, entries 1024, types 4096, activeLocals 128, depth 64, and rounds 64 are unchanged.

| Observation | Result |
| --- | --- |
| Development check | 189 source files; 2350 MIR functions; passed |
| Selected development native cases | 11 passed; unchanged ten cases reused with a source/hash proof, corrected cleanup case rerun |
| Reference acceptance/code/span | 14/14 matched, including generic and cleanup provider context |
| Bounds | program/work 1048576; types 4096; entries 1024; active locals 128; depth/rounds 64 |

The unit only consumes initializer, call-argument and return expressions. Standalone
expression/discard typing, lifetime/conflict/move checking, array/view/index/slice
borrows, unsafe pointer consumers, cross-unit links/layouts and HIR/MIR/C/CLI remain
pending. The standalone implementation receipt excludes corpus telemetry; the
separate full fixed-corpus observation is documented below. Full current dispatcher native and stage1/stage2, release,
Linux and push were not run. Public language/SDK ABI, SDK sources, DB, CI and runners
are unchanged. Evidence: `docs/ops/evidence/candidate-module-unit-places-20261009.json`.


## Safe-reference corpus observation (2026-10-09)

The fixed dispatcher snapshot completed admission and all detailed assertions in
one development execution. Original expression IDs/spans, nominal provider field
coordinates, canonical borrow types and whole-owner root/scope metadata were
checked. The input snapshot hashes are unchanged from the field-unit observation;
new unit modules are compiled as implementation inputs outside this 1MiB snapshot.

| Observation | Result |
| --- | --- |
| Input / units | 160 modules; 1048562 bytes; 26 units |
| Admission attempts / splits | 39 / 13 |
| Borrow expressions / typed borrows | 2891 / 1422 |
| Dereferences / typed safe dereferences | 388 / 254 |
| Expressions retaining owner root/scope | 51532 |
| Checked arguments | 27324 / 29822; previous field-unit result 25740 |
| Checked fields / nominal provider records | 6486 / 6416; previous checked fields 6085 |
| Checked scalar casts | 52 / 1789 |
| Maximum value work / graph | 823678 / 1048576; 427 / 4096 |
| Detailed coverage | All ranges continuously cover 0..160; execution passed without timeout |

Array/view/index/slice borrows, standalone expression/discard typing, lifetime,
conflict and move checking, unsafe/pointer provenance, cross-unit links/layouts,
unit HIR/MIR/C/CLI, full current-source dispatcher native and stage1/stage2 remain
pending. Release, Linux and push were not run. Public language/SDK ABI, SDK source,
DB, CI and runner contracts are unchanged.
Evidence: `docs/ops/evidence/candidate-module-unit-places-corpus-20261009.json`.

## Indexed array and scalar-view values (2026-10-09)

module_unit_value_index resolves known scalar views and admitted fixed-array indexed values with canonical element types. Index operands require usize; E1006 preserves the operand span, an unsupported receiver becomes experimental code2 at the receiver span, and a return mismatch stays E1009. An index place keeps its original owner root/scope and expression projection, and view mutability comes from the view type rather than the binding.

Named array and view element borrows feed canonical shared/mutable references, casts, inferred locals, arguments and returns. Temporary, call and raw-pointer roots never gain borrow address identity. Nested arrays sit outside the current language contract and are rejected by the value consumer even when an older internal graph can represent them. The generic provider and owned cleanup contexts reuse it. Existing limits are unchanged, and the compiler logic is entirely Fairy. Array literal/view-construction/slice, nominal view interfaces, lifetime/conflict/move, unsafe/pointer, HIR/MIR/C/CLI and stage1/stage2 remain pending.

| Observation | Result |
| --- | --- |
| Development check | 190 source files; 2352 MIR functions; passed |
| Selected development native cases | 5 passed; four identical cases reused and the corrected provider cleanup case rerun |
| Reference acceptance / code / span | 12/12 matched |
| Retained bounds | types 4096; expressions/calls/arguments 16384; entries 1024; active locals 128; depth/rounds 64; work 1048576 |

This observation only covers known receiver types in initializer, argument and
return consumers. Declaration/shape admission remains separate: resource array
fields and nested array type annotations still need earlier interface rejection.
No new full-corpus observation is claimed; it is bundled with array/view/slice
construction. Release, Linux, full current dispatcher native, stage1/stage2 and
push were not run. Public language/SDK ABI, SDK sources, DB, CI and runners are
unchanged. Evidence: `docs/ops/evidence/candidate-module-unit-indexes-20261009.json`.

## Canonical array literal projection (2026-10-09)

A new `module_unit_value_array` module checks list and repeat array literals against canonical graph types. The first uncontextualized element anchors the type; explicit array expectations contextualize each element, while an array length mismatch is reported by the existing local/argument E1006 or return E1009 consumer. The count must be a literal in 1..4096; an unsupported count produces code2 at the count span, empty and unsupported element arrays produce code2 at the whole literal, and a mismatched element produces E1006 with scalar overflow producing E1013, both preserving the element span. List edges are collected once and evaluated in source order, with the repeated initializer evaluated once. `module_unit_value_shapes` validates nonempty acyclic nominal field graphs, including permitted inline arrays, and rejects resources, raw pointers, borrows, tags, and nested array elements. The 64-depth and 1 MiB work limit are retained; comparisons are charged and the canonical graph stays at 4096. Known arrays feed inferred locals, indexing, declared and generic call arguments, and returns, with provider expression IDs and spans unchanged. The unknown record initializer remains pending, and no general standalone expression, lifetime, or move completion is claimed. View construction, slices, nominal-view representation, earlier declaration rejection, unsafe and pointer handling, HIR/MIR/C/CLI, and stage1/stage2 remain. There is no new full corpus pass, release, Linux build, or push.

| Observation | Result |
| --- | --- |
| Development check | 192 source files; 2355 MIR functions; passed |
| Selected development native cases | 5 passed; four identical cases reused and the corrected shape-rejection case rerun |
| Reference acceptance / code / span | 15/15 matched |
| Literal order / identity | Source-order list edges, one repeated initializer, canonical element and length retained |
| Bounds | graph 4096; literal elements 1..4096; work 1048576; active shape depth 64 |

The original resource-bearing struct negative fixture was rejected earlier by
reference declaration analysis with E1002. It was replaced with a legal
pointer-bearing struct whose array literal is rejected by the value-shape gate.
General declaration rejection, record construction, standalone expression,
loan lifetime/conflict/move, view construction/slices, nominal view interfaces,
unsafe/pointers and downstream HIR/MIR/C/CLI remain incomplete. Full corpus
observation is bundled with the next view/slice unit. Release, Linux, full current
dispatcher native, stage1/stage2 and push were not run. Public language/SDK ABI,
SDK sources, DB, CI and runners are unchanged.
Evidence: `docs/ops/evidence/candidate-module-unit-arrays-20261009.json`.

## Scalar array views and slices (2026-10-09)

module_unit_value_view now covers all existing byte, bool, and integer view families. Direct ref or ref mut of a named scalar array takes the exact contextual view type; without a matching view context it keeps a canonical reference to the array. An explicit ref over a grouped named array or a plain struct array-field range yields a view slice. Ordinary slices of existing views keep their shared or exclusive type and the whole owner root and scope across aliases. Omitted bounds stay as zero source IDs; explicit bounds are usize in receiver, start, end order; half-open bounds are checked at later runtime and no static E is invented. ref view[..] is unsupported because the compiler does not borrow temporary descriptors. Mutable-to-shared shortening applies only to call arguments, not local declarations.

E1008 points at the target span for the immutable mutable-borrow and the nonplain field subview; E1006 points at the bound, argument, or local span; code2 keeps the receiver or whole-borrow span. Field owners with plain shapes are validated for array-field views. The bounds wrapper is non-value syntax and original expression IDs and spans are preserved. Limits graph4096, work1MiB, locals128, depth/round64 are unchanged. This is type and provenance work only: lifetimes, conflicts, moves, and child generations remain incomplete, and nominal view interface type representation is still pending with no claim of completion. No unobserved corpus numbers, no release, Linux, or push, and no HIR, MIR, C, CLI, stage1, or stage2.

| Observation | Result |
| --- | --- |
| Development check | 193 source files; 2366 MIR functions; passed |
| Selected development native cases | 9/9 passed |
| Reference acceptance / code / span | 17/17 matched |
| Aliases / bounds | Whole-owner root/scope retained; omitted bounds keep zero IDs; explicit usize bounds typed in order |
| Call shortening | Mutable scalar views can shorten to shared call arguments; local declarations remain exact |
| Bounds | types 4096; work 1048576; active locals 128; depth/rounds 64 |

The implementation receipt covers scalar view families and contextual consumers.
Nominal view interfaces and nominal array-field borrowing remain pending, as do
record/control/variant/builtin and standalone expressions, earlier declaration
rejection, loan generations/lifetime/conflict/move, unsafe/pointer provenance,
move-aware cleanup, cross-unit links/layouts and HIR/MIR/C/CLI. The separate full
fixed-corpus observation is documented below. Full current dispatcher native and
stage1/stage2, release, Linux and push were not run. Public language/SDK ABI,
SDK source, DB, CI and runner contracts are unchanged.
Evidence: `docs/ops/evidence/candidate-module-unit-views-20261009.json`.

## Scalar-view corpus observation (2026-10-09)

The fixed dispatcher snapshot completed admission and all detailed assertions in
one development execution. The 160-module, 1048562-byte input was admitted as 26
bounded units with continuous coverage over 0..160, and the accepted source
manifest hashes were re-verified against the receipt
(`dispatcher-admission-cooked-intern-20261009.txt`, sha256
8281ef6c4761a3d6627f81a0050b9f967e3d9ee3851e16a1c76aee73ab5dcd05). The accepted
view families report 74 array expressions (32 checked), 473 index expressions (135
checked), 171 slice expressions (166 checked) and 10 recorded view origins, with
original expression IDs, spans and whole-owner roots retained. The existing 9/9
development native selection and 17/17 reference acceptance remain the
implementation evidence.

| Observation | Result |
| --- | --- |
| Input / units | 160 modules; 1048562 bytes; 26 units |
| Array expressions / checked | 74 / 32 |
| Index expressions / checked | 473 / 135 |
| Slice expressions / checked | 171 / 166 |
| View origins | 10 |
| Maximum value work / graph | 825705 / 1048576; 427 / 4096 |
| Retained bounds | work 1048576; types 4096; entries 1024; active locals 128; depth/rounds 64 |
| Detailed coverage | All ranges continuously cover 0..160; execution passed without timeout |

Nominal view interfaces and nominal array-field borrowing remain pending, as do
lifetime/conflict/move validation and the remaining expression families, unsafe
and pointer provenance, cross-unit links/layouts and unit HIR/MIR/C/CLI. Full
current dispatcher native, stage1/stage2, release, Linux and push were not run.
Public language/SDK ABI, SDK source, DB, CI and runner contracts are unchanged.
Evidence: `docs/ops/evidence/candidate-module-unit-views-corpus-20261009.json`.

## Nominal view interfaces and array-field borrows (2026-10-09)

module_types now carries canonical shared (kind 17) and mutable (kind 18) nominal
view interfaces whose first slot is the existing nominal element type, validated
against the canonical graph. module_unit_interfaces follows each descriptor's
element into the pending shape queue. A nominal array becomes such a view only
when the contextual expected type is a matching shared or mutable view, or when
an array range is explicitly borrowed; a bare `ref p.data` without that context
is rejected at the nominal field with experimental code2, and a named fixed
array without context keeps an array reference instead of a view. The plain named
whole owner keeps its root and scope, and the element keeps its provider module
and declaration identity.

Field owners are re-checked through field_owner and element selection runs the
element_borrow gate. An immutable storage with a mutable borrow is E1008 at the
target span. A nonplain owner with an indexed or leaf mutable borrow is E1008 at
the selected span, while the shared borrow stays experimental code2 at the whole
expression. A nonplain field range is E1008 at the array-field owner span. The
index and field builtins extend element resolution and len/is_empty to the shared
and mutable descriptors. Bounds stay at types 4096, work 1 MiB, entries 1024,
active locals 128 and depth/rounds 64.

This unit is type and provenance projection only: lifetime, conflict, move and
generation checks remain unfinished, so no borrow-conflict or ownership claim is
made. The selected development native cases pass 12/12, the reference acceptance
matches 19/19 code/span fixtures, and check reports 193 source files and 2370 MIR
functions. The two fixtures that first failed were a test-authoring error: the
expected declaration number did not account for module declaration 1, which the
receipt records. The new full corpus observation is bundled with the next
consumer. Control/variant/builtin/standalone and discard expressions, earlier
declaration rejection, unsafe/pointer provenance, cross-unit links/layouts and
unit HIR/MIR/C/CLI remain. Full current dispatcher native, stage1/stage2,
release, Linux and push were not run. Public
language/SDK ABI, SDK sources, DB, CI and runners are unchanged.
Evidence: `docs/ops/evidence/candidate-module-unit-nominal-views-20261009.json`.

## Canonical record construction (2026-10-09)

module_unit_value_record now admits record-aggregate constructor expressions. A
named constructor resolves its target through canonical module/declaration
identity, so `S{...}` reuses the canonical struct or resource instead of a fresh
type. A constructor whose raw name resolves to a local or function is not an
aggregate, so it stays experimental code2 over the whole expression, and a
cross-module resource or owned-aggregate constructor is E0103 at the declared
name span. A missing interface shape is the internal invariant code12, and
contextual enum records remain pending at code0 rather than supported.

Field rows are collected by walking the record's right-linked chain once and are
then checked in source order, independent of declaration order. Unknown,
malformed and duplicate rows fail code2 at the row span, a missing field fails
code2 at the whole constructor span, and a value mismatch is E1006 with scalar
overflow E1013, both at the value span. Field wrappers keep state2/type0 and
store the field origin (provider module/declaration and source index) plus the
expected type; a complete record becomes state1 with its canonical target and an
empty place, and zero-field aggregates are accepted.

Known records feed arrays, fields, indexes, calls, returns, cleanup and generic
consumers, and the provided field coordinates survive into imported and generic
provider units. The owned SDK `Vec` aggregate records its specialization as a
kind13 owned-aggregate seed, and the array-literal plain-shape gate is retained.

An earlier run left unannotated ordinary constructors such as `B{...}` and
`Empty{}` out of the frozen interface graph: 8 of the 10 selected native cases
passed and 2 failed. module_unit_body_types::ordinary_records now registers
those constructors before the graph freezes: it walks non-generic ordinary
functions for constructors without filtering the function's public or private
kind (only its ordinary_scan helper is private), with transaction rollback,
three-context constructor demand marks, no templates, and preserved original
declaration IDs and spans. The failed-then-fixed run is kept in the receipt.

| Verification | Result |
| --- | --- |
| check | 194 source files; 2376 MIR functions; pass |
| native | 11/11 selected development cases (10 previous + 1 body-types regression) |
| reference | 15/15 code/span fixtures (9 negative, 3 positive, 3 multi-file) |

The fixed 160-module corpus was restored with matching prior manifest hashes
and re-measured under a new relative-path manifest; that observation passed and
is recorded below. Limits types 4096, work
1 MiB, entries 1024, active locals 128 and depth/rounds 64 are unchanged, and
the compiler logic is entirely Fairy. Control/variant/builtin/standalone and
discard expressions, earlier declaration/shape rejection, loan
generations/lifetime/conflict/move, unsafe/pointer provenance and move-aware
cleanup, cross-unit links/layouts and unit HIR/MIR/C/CLI remain, as do full
current dispatcher native and stage1/stage2. Release, Linux and push were not
run. Public language/SDK ABI, SDK sources, DB, CI and runner contracts are
unchanged.
Evidence: `docs/ops/evidence/candidate-module-unit-records-20261009.json`.

## Record construction corpus observation (2026-10-09)

The fixed dispatcher snapshot completed admission and all detailed assertions in
one development execution. The 160-module, 1048562-byte input was admitted as 26
bounded units through 39 admissions and 13 splits, with continuous coverage over
0..160. The accepted records report 450 checked of 570 record constructions,
2420 provider fields, 457 ordinary constructor-only demands, and 7370 total type
demands (3289 function, 3624 cleanup, 457 ordinary). The maximum value work was
845545 of 1048576 with graph 427 of 4096, and the maximum interface size was
295576 (previously 283813).

The original manifest was not reproducible because the mutable module_types grew
by 454 bytes to 1049016; the fixed snapshot was restored from the current other
159 files plus the earlier bf28c16b module_types bytes, and every prior receipt
160-file hash matched. This deleted no input bytes and relaxed no limit. It is the
existing fixed snapshot only, not a full current dispatcher pass.

| Observation | Result |
| --- | --- |
| Input / units | 160 modules; 1048562 bytes; 26 units |
| Admission / splits | 39 admissions; 13 splits |
| Records checked / total | 450 / 570 |
| Provider fields | 2420 |
| Ordinary constructor-only demands | 457 |
| Total type demands | 7370 (function 3289; cleanup 3624; ordinary 457) |
| Maximum value work / graph | 845545 / 1048576; 427 / 4096 |
| Maximum interface | 295576 (previous 283813) |
| Detailed coverage | All ranges continuously cover 0..160 |
Evidence: `docs/ops/evidence/candidate-module-unit-records-corpus-20261009.json`.

## Conditional blocks and statement consumers (2026-10-09)

`module_unit_value_control` projects ordinary and unsafe block tails and `if`
results into canonical value types. Both branches receive the outer expected
context; unequal results report E1006 at the alternative expression. A missing
`else` yields void after its condition and consequent are known. Return and
literal-final assert statements do not change ordinary if-branch joins. Derived
values carry no storage place; borrow-origin merging remains pending.

Owned statement consumers cover standalone expressions, discard, assert, while
conditions and scalar for conditions in ordinary, generic and cleanup contexts.
They preserve original expression IDs and spans. Bool contexts report E1007,
and the structural ADR0213 for-header restriction is charged against the same
1 MiB work budget. Assignment/post clauses, defer/burn, missing-return checks,
function-body tails, switch/variant/builtin consumers and declaration/shape
rejection remain pending. The existing inferred-void-local rejection also still
differs from reference semantic admission.

Verification: development check (195 sources, 2390 MIR functions); 19 reference
acceptance/code/span fixtures; development native selection 7/7, followed by
2/2 affected tests after the no-else deferred-tail gate correction, with the five
unaffected tests reused. The initial 6/7 native attempt assumed an assignment
post-clause was already typed; the fixture now exercises the admitted condition
consumer. These runs and their exact source identities are retained in
`docs/ops/evidence/candidate-module-unit-control-20261009.json`.

Types 4096, work 1048576, entries 1024, active locals 128, depth/rounds 64 remain
unchanged. No new full corpus, full current dispatcher, stage1/stage2, release,
Linux or push was run. Corpus observations are bundled with the next tagged
consumer unit. Public language/SDK ABI, SDK sources, DB, CI and runners are
unchanged. Next: switch/variant payload typing and assignment consumers.

## Contextual variant construction (2026-10-09)

`module_unit_value_variant` now checks contextual `.none`, `.some(value)`,
`.ok(value)` and `.err(value)` against canonical Option/Result payload types.
Nested tags and nominal/resource payload identities are retained. Enum values
and named payload records retain canonical module/declaration identity and the
original provider variant/field coordinates. Payload rows are checked in source
order; wrappers and contextual callees remain nonvalues. Unsupported context,
arity, member/field, duplicate/missing fields and payload mismatches retain their
original spans; scalar overflow remains E1013 and argument labels remain E1014.

Development check passed (196 sources, 2399 MIR functions), selected native
cases passed 7/7 including SDK Vec and conditional regressions, and reference
acceptance/code/span fixtures matched 19/19. Types 4096, work 1048576, entries
1024, active locals 128 and depth/rounds 64 are unchanged. Evidence:
`docs/ops/evidence/candidate-module-unit-variants-20261009.json`.

Switch/payload binding, propagation/builtin, assignment/post, defer/burn,
function-tail/missing-return validation, earlier declaration/shape rejection,
inferred-void-local admission, ownership/unsafe provenance and unit HIR/MIR/C/CLI
remain. No new corpus, full current dispatcher, stage1/stage2, release, Linux or
push was run; corpus work is bundled with switch consumers. Public language/SDK
ABI, SDK sources, DB, CI and runner contracts are unchanged.

## Switch values and scoped payload bindings (2026-10-09)

`module_unit_value_match` now projects exhaustive canonical Result/Option/enum
switches in source order, with duplicate/missing variant and payload checks.
Pattern nodes remain nonvalues; their payload type and existing scoped symbol
root annotate every matching path and its local metadata. Unrelated same-name
symbols remain separate. Enum pattern fields retain original provider module,
declaration and field coordinates. Typed payloads feed inferred locals, fields,
call arguments and returns in ordinary, generic and cleanup contexts.

Result excludes no-tail final direct-return and literal-final assert(false)
arms from value joins; Option excludes only literal-final assert(false), and
user enums exclude neither. All-excluded built-in switches have void type.
Other arm mismatches report E1006 at the arm block, overflow remains E1013,
and unsupported scrutinee/arity/pattern/exhaustiveness retains original spans.
A standalone non-void function with only switch-contained returns is still
rejected by the reference's checked-return gate; the all-terminal native fixture
uses a void function and does not claim that missing-return analysis is done.

Development check passed (197 sources, 2404 MIR functions), selected native
cases passed 7/7 including SDK Vec, and reference acceptance/code/span fixtures
matched 19/19. Types 4096, work 1048576, entries 1024, active locals 128 and
depth/rounds 64 remain unchanged. Evidence:
`docs/ops/evidence/candidate-module-unit-matches-20261009.json`.

The fixed 160-module corpus observation completed and passed with added
switch/variant/payload metadata assertions: the 1048562-byte input was admitted
as 26 bounded units covering 0..160 continuously through 39 admissions and 13
splits, with verified source, compiler, test, lock and input hashes. Match totals
are 109/109 checked, 811 variant values, 222 bound payloads and 38 pattern
fields; corpus value totals are 186137 value expressions, 117985 known values,
11442 value locals, 4783 value returns and 8500 unresolved values, with maximum
value work 937515 of 1048576 and graph 427 of 4096. This is the existing fixed
input, not a full current dispatcher pass.
Evidence: `docs/ops/evidence/candidate-module-unit-matches-corpus-20261009.json`.
Propagation/builtin, assignment/post, defer/burn, function tails/missing returns,
earlier declaration/shape rejection, inferred-void-local admission, ownership,
unsafe provenance, cross-unit HIR/MIR/C/CLI, full current dispatcher and
stage1/stage2 remain. Release, Linux and push were not run. Public language/SDK
ABI, SDK sources, DB, CI and runner contracts are unchanged.

## Recursive value dispatcher stack repair (2026-10-09)

The fixed 160-module corpus aborted on Windows with STATUS_STACK_OVERFLOW
(3221225725). A diagnostic generated-C trace placed the failure in module17
type_cli declaration11, near the depth-31 tail of the long else-if ladder.

`module_unit_value_scalar`'s evaluate function now routes its group, field,
index, safe-borrow and cast branches through separate Fairy functions while
preserving dispatch results and depth increments. In the same debug workspace
the MSVC `/Od` diagnostic scalar-evaluate frame measured 19384 bytes before and
4808 bytes after. Types 4096, work 1048576, entries 1024, active locals 128 and
depth/rounds 64 stay unchanged, as does the stack reserve.

The new 56-branch else-if regression fixture covers normal i64 values plus the
final bool type error E1006 with its original span. Development native cases
passed 13/13 (6 conditional, 6 switch, 1 SDK Vec); the new reference
acceptance/span fixtures matched 2/2, and stage0 error/normal exit codes stayed
3 and 0. Evidence:
`docs/ops/evidence/candidate-module-unit-value-stack-20261009.json`.

The full 160-module corpus re-run completed and passed, recorded in the fixed
corpus observation above. The full dispatcher, stage1/stage2, release, Linux and
push paths were not run. Public language/SDK ABI, SDK sources, DB, CI and runner
contracts are unchanged.

## Result propagation value consumer (2026-10-09)

A new `module_unit_value_result` module projects the canonical Result operand's
success type. The operand is evaluated with zero context, so the enclosing
success type never manufactures a Result for it; the operand and the enclosing
kinds must both be 6 with equal second (error) canonical IDs, while an enclosing
success type may differ. A non-Result operand reports E4102, a non-Result or
error-mismatched enclosing type reports E4103, and both point at a 1-byte span
at the operand's original end. An unsupported construction whose fallback is
code2 is promoted to E4102 to match the reference diagnostic priority, and other
diagnostics and budget errors pass through unchanged.

Ordinary and generic providers, nested Results and nominal or resource success
payloads feed inferred locals, fields, call arguments and returns, and no new
place root is created.

Development check passed (198 sources, 2410 MIR functions), selected native cases
passed 6/6 (4 new propagation cases, deep else-if, SDK Vec), and reference
acceptance/code/span fixtures matched 13/13 (10 errors, 2 single-file positives,
1 provider generic). Limits types 4096, work 1048576, entries 1024, active locals
128 and depth/rounds 64 remain unchanged. Evidence:
`docs/ops/evidence/candidate-module-unit-propagation-20261009.json`.

Result must-use, flow, move and loan/cleanup ownership remain separate follow-up
work. Builtin, assignment/post, defer/burn, function tails and missing returns,
earlier declarations, provenance, unit HIR/MIR/C/CLI, full current dispatcher and
stage1/stage2 remain. The full fixed corpus re-run is bundled with the next
builtin unit, so no new corpus pass is claimed here. Release, Linux and push were
not run. Public language/SDK ABI, SDK sources, DB, CI and runner contracts are
unchanged.

## Raw-view and source-location builtin value consumers (2026-10-09)

`module_unit_value_builtin` preserves the closed builtin kind from name binding
and the exact lexical unsafe flag in the Values Expression builtin metadata.

Raw view builtins 9..12 and CStr kind 14 project their result types from three
arguments in source order: the owner must be a safe shared or exclusive borrow
whose mutability exactly matches the builtin, the data carries a byte or opaque
raw pointer with matching mutability, and the length is usize. Arity reports
E1010, a non-usize length reports E1006, a missing unsafe context reports E5101,
and an unsupported owner or data keeps code2 at the original span.

The source_location builtin 200 takes exactly zero arguments, yields refstr and
leaves the root at 0; E4401 covers reserved declarations, parameters, locals and
non-call use, and E1014 covers labels. Only the borrowed tail origin inside a
single normal or unsafe block is preserved; scalar and owned tails create no
place, and multi-branch origin joins and lifetimes remain pending. An inferred
raw text alias root updates to the original parameter root by fixed point, and
generic providers and cleanup keep source_location's original coordinates.

Development check passed (199 sources, 2420 MIR functions), and selected native
cases passed 7/7 (5 new builtin cases, deep else-if, SDK Vec) after two initial
failures: one fixture bug that expected raw builtin 12 where the canonical CStr
kind is 14, and one real bug where an unsafe block dropped the borrow origin.
Reference acceptance/code/span fixtures matched 19/19 (15 errors, 3 single-file
positives, 1 provider-generic-cleanup). Limits types 4096, work 1048576, entries
1024, active locals 128 and depth/rounds 64 are intact. Evidence:
`docs/ops/evidence/candidate-module-unit-builtins-20261009.json`.

The SDK storage consumer is a separate follow-up implementation and is not
claimed complete, and the full corpus re-run is bundled after the next SDK
builtin. Source_location string and MIR emission stay under unit values and
follow the full unit HIR/MIR/C/CLI, so only its type projection is complete and
no whole source_location CLI completion is claimed. Unsafe pointer allocation,
generation, provenance and loan/move remain. Release, Linux, push, full current
dispatcher and stage1/stage2 were not run. Public language/SDK ABI, SDK sources,
DB, CI and runner contracts are unchanged.

## SDK storage value consumer (2026-10-09)

`module_unit_value_storage` projects builtins 101..107: storage_size and
storage_alignment yield usize with no arguments, storage_take yields the exact
canonical element from one argument, storage_put yields void,
storage_element_address yields ptr mut void, and
storage_borrow/storage_borrow_mut yield ref kinds 7/8 or the matching scalar
and nominal slice descriptors for T[]. Arguments are checked in source order;
arity and storage operand type failures keep code2 at the whole original call
span, and a missing unsafe context reports E5101. Child expression diagnostics
retain their own source spans.

Only authenticated std::vec storage borrow slice arguments peel the element
for annotation demands; unsized by-value types stay rejected. Canonical
element IDs, slice flags and lexical unsafe facts survive generic dependencies
and owned bodies. The cleanup inline of take_slot<T> retains inline_unsafe.
Borrowed owner roots propagate through block tails and aliases; take, put and
element addresses introduce no safe owner root. Nominal slice elements use the
existing scalar-leaf struct graph including bounded inline arrays.

Development check passed (200 sources, 2433 MIR functions), selected native
tests passed 7/7 (5 storage tests plus raw alias and deep else-if
regressions), and reference fixtures matched 5/5. Native storage tests cover
scalar operations, nominal array-field views, moved resources and cleanup,
five malformed-metadata/unsafe mutations, and user/SDK-spoof rejection.
Candidate lookup code5 maps to reference E0101; candidate E6801 at the spoofed
module name corresponds to the stable reference invalid-standard-library
namespace failure. Evidence:
docs/ops/evidence/candidate-module-unit-storage-20261009.json.

The fixed 160-module observation planned 26 units and completed 0..60 across
12 units before the 600-second observer timeout; this is not a complete corpus
pass. The probe now accepts from=<module-index> at a planned unit boundary,
and the 60..160 suffix is being observed with the prior 900-second allowance.
Imported ordinary SDK view_u8/view_mut_u8 still fail closed because
module_signatures::build excludes those SDK function signatures; generic
view<T> storage value projection is covered.

Limits 4096 types, 1048576 work, 1024 entries, 128 active locals and 64
depth/rounds are unchanged. Assignment/post, defer/burn, function tails and
missing returns, earlier declaration and shape rejection,
loans/lifetimes/conflicts/moves, unsafe pointer provenance, cross-unit
HIR/MIR/C/CLI, the full current dispatcher and stage1/stage2 remain. Release,
Linux and push were not run. Public language/SDK ABI, SDK sources, DB, CI and
runner contracts are unchanged.

## Fixed corpus resumed observation (2026-10-09)

The earlier 600-second observation completed 0..60 across 12 units before
timing out. Source, input and compiler identities were re-verified, and
from=60 resumed the 60..160 suffix with a 900-second allowance. The two
observations cover the unchanged 160-module, 1048562-byte input continuously
across 26 bounded units; the plan admitted 39 ranges with 13 splits. Evidence:
docs/ops/evidence/candidate-module-unit-storage-corpus-20261009.json
(implementation commit 3080ead6).

All 1221 storage value calls were checked. Total builtin calls were 1225, with
1223 checked; the two remaining raw-view values remain unresolved rather than
accepted. Maximum value work was 974364/1048576 and maximum graph size was
427/4096. The fixed input contains no storage slice calls; selected native
tests separately cover scalar and nominal T[] storage borrows. This is an
observation of the fixed input archive, not a full current dispatcher or
stage1/stage2 promotion. Release, Linux and push were not run.

## Cached SDK annotation authority (2026-10-09)

`module_signatures::Program` carries a per-module vector_sources bool cache,
populated from the existing sdk::authentic result at metadata construction.
After source_matches or validated source stamps, body type, owned-body and
interface annotation consumers reuse that fact by O(1) lookup. Per-body and
per-cleanup SHA rehashes in those annotation paths are removed, and each
authority read is charged as one unit of analysis work. This does not change
the existing generic-expansion or name-binding authentication paths.

Development check passed (200 sources, 2435 MIR functions) and the five
storage native tests passed 5/5 again. The fixture also checks ordinary
caller=false, authentic SDK=true and out-of-range module=false;
namespace/source spoof rejection is retained. Evidence:
docs/ops/evidence/candidate-sdk-annotation-authority-cache-20261009.json. The
complete fixed-corpus observation remains pinned to implementation commit
3080ead6; it was not re-measured for this cache-only change. Type projection
behavior is unchanged while some analysis work counters increase. Release,
Linux and push were not run; public language/SDK ABI, SDK sources, DB, CI and
runner contracts are unchanged.

## Ordinary SDK view signatures and unit storage bodies (2026-10-09)

`module_signatures::build` now collects authenticated std::vec non-generic
functions through the same canonical signature path as other ordinary
functions. The ten shared/mutable byte, i64, i32, u32 and usize view functions
retain their declaration coordinates, public visibility, parameter and result
types, effects, must-use, unsafe and foreign flags. Five scalar Vec
instantiations are retained in the metadata graph; ref/ref-mut parameters and
matching slice results stay canonical.

Development check passed (200 sources, 2435 MIR functions), selected native
tests passed 4/4 and reference fixtures matched 4/4. Caller tests cover exact
parameter/result IDs and E1006/E1010 caller spans, SDK unit tests check all
ten ordinary storage-slice bodies and owner roots against original SDK
coordinates, and the existing nominal array-field storage view remains
covered. Evidence:
docs/ops/evidence/candidate-module-unit-sdk-calls-20261009.json. Ordinary-call
origin summaries, loans/lifetimes/moves, assignment/post, defer/burn, function
tails/missing returns and cross-unit HIR/MIR/C/CLI remain. No new fixed-corpus
observation, full dispatcher, stage1/stage2, release, Linux or push was run
for this slice; the next corpus observation is bundled after the statement
consumers. Public language/SDK ABI, SDK sources, DB, CI and runners are
unchanged.

## Assignment and post-clause value consumer (2026-10-09)

`module_unit_value_assignment` consumes assignment statements and post clauses
from the unit value walk. It resolves the canonical target expression with
zero context, then evaluates the right-hand side with the target's canonical
type as its expected context. A non-value or type-mismatched right-hand side
reports E1006 at the value span. The target must project a writable mutable
local, field, checked index or named admitted reference dereference; immutable
and non-place targets keep E1008 at the original span. Indexed struct
array-field replacement and empty-struct dereference are rejected with E1008.
Plain = accepts matching types, while the ten compound operators require one
of the five integer scalar types. Other compound targets retain code2 over the
statement.

Header init/post markers 1/2 require a direct scalar local target and
structurally scalar right-hand sides. Declaration initializers have the same
scalar header restriction. Violations retain code2 over the original clause
without its separator. Explicit allowed casts are admitted by both the unit
header consumer and the existing type_control predicate. Checks record
expected/actual canonical IDs, target and operator token coordinates, header
flags and the checked flag.

Development check passed (201 sources, 2443 MIR functions), selected native
tests passed 6/6 and reference acceptance/code/span fixtures matched 18/18.
Native cases check 31 assignments, ten compound operators across five integer
scalar types, plain bool assignment, mutable field/array/view/named-reference
paths, original provider generic and cleanup coordinates, and error and header
restrictions. Evidence:
docs/ops/evidence/candidate-module-unit-assignments-20261009.json.

Defer/burn, function completion and missing returns, inferred-void-local
admission, earlier declaration/shape rejection,
loans/lifetimes/conflicts/moves, unsafe pointer provenance and cross-unit
HIR/MIR/C/CLI remain. All consumer logic is Fairy. No new fixed corpus, full
dispatcher, stage1/stage2, release, Linux or push was run; the corpus
observation remains bundled after statement consumers. Public language/SDK
ABI, SDK sources, DB, CI and runner contracts are unchanged.

## Function and cleanup completion gate (2026-10-09)

`module_unit_value_completion` applies the reference type checker's syntactic
completion gate to each function and cleanup body. It resolves the original
declaration body, requires a statement block without a top-level value tail,
and admits void results. Non-void bodies must contain a direct return, literal
assert(false), a completing unsafe/region body, or an if/else chain whose
branches complete. The parser's reverse statement links are walked directly
with charged work rather than repeatedly scanning indexed statements. Missing
completion retains code2 at the original whole-body span; Values publishes
checked_completion.

This is the reference's syntactic gate, not a CFG proof. A loop, plain nested
block or switch returning from every arm does not complete a non-void body. An
if without an else also remains insufficient. A branch with a value tail
follows the reference's semantic admission; actual return and termination
validation still belongs to the subsequent HIR/MIR path. Function and cleanup
bodies remain statement blocks, and top-level function value tails remain
parser errors. No syntax was added.

The walk now admits inferred canonical void locals and their aliases, removing
the earlier unconditional void-local rejection. Declared-type mismatches
retain E1006. Development check passed (202 sources, 2447 MIR functions),
selected native tests passed 6/6 (four new completion tests plus header and
provider/cleanup assignment regressions), and reference acceptance/code/span
fixtures matched 12/12. The stable Go check also admitted the void-inference
fixture. Native tests cover the syntactic boundaries, original error spans,
canonical alias identity, parser rejection and fail-closed exhaustion of the
unchanged 1048576 work budget. Evidence:
docs/ops/evidence/candidate-module-unit-completion-20261009.json.

Defer/burn, earlier declaration/shape rejection,
loans/lifetimes/conflicts/moves, unsafe pointer provenance and cross-unit
HIR/MIR/C/CLI remain. Type, work, entry, active-local and depth/round bounds
are unchanged. The completion logic is entirely Fairy. No new fixed corpus,
full current dispatcher, stage1/stage2, release, Linux or push was run. Public
language/SDK ABI, SDK sources, DB, CI and runner contracts are unchanged.

## Defer, burn and validated resource abilities (2026-10-09)

`module_unit_value_terminal` consumes defer and burn statements from the unit
value walk. A defer is a direct checked invocation with a void result and
deferrable scalar arguments (bool and the five integers). Ordinary callables
must be non-foreign/non-template; a generic specialization must resolve a
checked concrete instance with matching owner, result and arity. The consumer
reuses invocation/argument tables and stays pending while a call or argument
is unchecked. Invalid targets keep code2 at the original statement, and
argument mismatches keep E1006 at the value span.

A burn requires a bare named local/parameter place with a canonical resource
declared by the current module, no drop ability and no resource-bearing
fields. Mutability is not required. Droppable resources, projected/non-place
operands, integers and foreign-module resources keep E1109 at the original
operand. Non-value expressions are rejected; unknown types remain pending.
Resource ability metadata is now validated declaration-locally by
resource_abilities::read before shape/cleanup publication, retaining E1107
spans for empty/repeated/malformed attributes, unknown/duplicate abilities and
key without store. Attribute names and argument texts are charged to the
existing work budget; other resource attributes remain with declaration
admission.

Development check passed (203 sources, 2453 MIR functions), reference fixtures
matched 18/18 (16 single-file and two module inputs), and native coverage
passed 5/5 (four terminal tests and one storage regression). The initial
native selection passed four tests; the imported-burn fixture used an illegal
direct provider constructor and was corrected to p::make, then passed alone.
Exact implementation/SDK/compiler/lock hashes and unchanged
test-file/preamble/function bytes were verified before reusing the four
earlier individual passes. Evidence:
docs/ops/evidence/candidate-module-unit-terminal-20261009.json.

This slice checks types and admission, not deferred-call MIR scheduling or
lifetime/loan/conflict/move validation. Earlier declaration/shape rejection,
ownership, unsafe pointer provenance and cross-unit HIR/MIR/C/CLI remain.
Existing type, work, entry, active-local and depth/round bounds are unchanged.
No new fixed corpus, full dispatcher, stage1/stage2, release, Linux or push
was run. Public language/SDK ABI, SDK sources, DB, CI and runner contracts are
unchanged.

## Unit nominal field contracts and bounded defer lookup (2026-10-10)

module_unit_shape_contracts checks duplicate nominal fields, enum variants and
payload names (E0102), safe-borrow fields (E1003), resource-bearing struct
fields (E1002), invalid inline array field shapes (code2) and resource
abilities (E1107). Selected unused nongeneric nominal declarations are
retained and checked before body typing. The authenticated SDK Vec generic
resource keeps its existing exception. Canonical field validation preserves
original module coordinates and type spans. Other declaration attributes and
general generic nominal admission remain separate.

Defer invocation lookup uses binary search over the expression-ordered
invocation table, and checked defer/burn consumers are not re-evaluated. A
regression with 900 ordinary zero-argument calls and 900 defers stays within
the original work budget. Development check passed (204 sources, 2459 MIR
functions), reference diagnostic/token cases matched 14/14 and selected native
development tests passed 6/6: three shape/budget tests plus SDK nominal array
views, missing returns and imported generic/cleanup defer/burn regressions.
The initial budget fixture exceeded the unchanged 4160 parser-expression
bound. After correcting only that test body, it passed alone; byte-identical
preamble, other test functions, implementation, compiler, SDK, inputs and lock
justified reusing the five earlier passes. Evidence:
docs/ops/evidence/candidate-module-unit-shape-contracts-20261010.json.

Type 4096, work 1048576, entries 1024, active locals 128 and depth/rounds 64
remain unchanged. Loans/lifetimes/conflicts/moves, ordinary-call origins,
unsafe pointer provenance and cross-unit HIR/MIR/C/CLI remain. This slice does
not claim a new fixed-corpus observation, full dispatcher, stage1/stage2,
release or Linux validation. Public language/SDK ABI, SDK sources, DB, CI and
runner contracts are unchanged; nothing was pushed.

## Fixed corpus after statement and field consumers (2026-10-10)

The unchanged fixed 160-module corpus was freshly observed after SDK ordinary
view signatures, assignment/post consumers, syntactic completion, defer/burn
and nominal field contracts. The archived manifest, all source inputs and
compiler hashes were verified. Development probe check/build and every unit
observation passed across 26 continuous ranges covering 0..160, with 1048562
joined bytes. The current native build gate is the six individually verified
shape/budget, SDK view, completion and terminal regressions from commit
93920dd4.

The observation reports 2566 value contexts, 186158 expressions, 123386 known
values and 3113 unresolved values; 4781/4783 returns are checked. Maximum
value work is 987087 and maximum value graph is 427, within unchanged limits.
Unresolved values remain unresolved; these counts do not establish full source
compilation or ownership safety. Evidence:
docs/ops/evidence/candidate-module-unit-statements-corpus-20261010.json. Other
declaration attributes, ordinary-call origins,
loans/lifetimes/conflicts/moves, unsafe pointer provenance and cross-unit
HIR/MIR/C/CLI remain. Full current dispatcher, stage1/stage2, release and
Linux validation remain separate. Public language/SDK ABI, SDK sources, DB, CI
and runner contracts are unchanged; nothing was pushed.

## Restricted enum payload field rejection (2026-10-10)

Follow-up review exposed four enum payload admission mismatches against the
reference: safe borrow, resource, fixed array and an inert struct containing a
fixed array. The preserved 93920dd4 development probe accepted all four; the
reference rejected them. Nominal field contracts now reject direct enum
payload arrays with code2 and recursively require inert-copyable payloads,
including Option/Result and nested structs/enums. Resource, borrow, void,
non-inert aggregate and cyclic payloads retain E1004 at the original field
type span; scalars, C scalars and raw pointers follow the reference admission.
Recursive traversal retains the shared work and depth bounds.

Current development check passed (204 sources, 2460 MIR functions), reference
acceptance/code/span cases matched 21/21, and two native development tests
passed: the enum payload regression and the earlier 14 malformed-field cases.
The positive payload case checks plain struct, Option<struct> and
Result<struct,i32> together to exercise sibling traversal state. Evidence:
docs/ops/evidence/candidate-module-unit-enum-payload-shapes-20261010.json. The
preceding 160-module corpus observation remains evidence for commit 93920dd4;
it was not rerun after this focused rejection fix. Ownership, unsafe
provenance and unit execution integration remain. No language/SDK ABI, SDK
source, DB, CI or runner contract changed; no release, Linux or stage1/stage2
validation or push was performed.

## Checked provider bodies for borrow-origin analysis (2026-10-10)

module_unit_origin_inputs collects the checked value bodies required by
borrowed-result calls. The primary plan stays caller-owned; an additional
provider module is checked and owned once, including its private ordinary
forwarding calls. It uses the existing bind, call-closure and value phases
instead of guessing a return origin from the signature. Analysis frame/owner
IDs preserve original declaration coordinates and local lexical scopes.
Generic instance IDs are scoped to their owning unit, and canonical type IDs
remain in each unit graph.

Development check passed (205 sources, 2499 MIR functions), native tests
passed 4/4 and reference cases matched 4/4. Coverage includes repeated
provider-call caching and private forwarding, distinct generic parameter
owners, E1009 at the provider return-expression span, and matching generic
instance numbers that refer to different concrete types in two units. The
fourth test was appended after three passes; byte-identical preamble/functions
and compiler, SDK, implementation, input and lock hashes justified reusing
those passes. Existing work, type, context, expression, owner, invocation and
active-parameter bounds are retained. Evidence:
docs/ops/evidence/candidate-module-unit-origin-inputs-20261010.json.

This is the input prerequisite for return-origin inference; it does not
publish guessed origins or validate lifetime/conflict/move safety.
Static/unknown origins, branch joins and call substitution remain next,
followed by ownership, unsafe pointer provenance and cross-unit execution. No
new fixed corpus, full dispatcher, stage1/stage2, release or Linux validation
was run. Public language/SDK ABI, SDK sources, DB, CI and runner contracts are
unchanged; nothing was pushed.

## Bounded syntactic paths for origin consumers (2026-10-10)

module_unit_origin_paths now provides bounded syntactic outcomes (fallthrough,
return/trap, break and continue) for a checked provider frame. It removes
definite dead statement/expression suffixes, joins if/switch arms, consumes
nested loop jumps and excludes a for post when its body can only break or
return. Loops retain a possible zero-iteration exit; no predicate correlation,
value-based branch elimination or at-least-once loop proof is inferred. Cast
and generic type-arena IDs are not traversed as expressions.

Development check passed (206 sources, 2517 MIR functions). Three native
development tests passed, covering return/trap and branch suffixes, loop/post
reachability and work-exhaustion failures. Reference grammar acceptance passed
for all 11 distinct marker fixtures; this is a syntax check, not semantic or
return-origin equivalence. The first native run passed two tests and failed
the loop test because its byte-view for fixture lacked let. Only that failed
fixture was corrected and rerun; byte-identical passing functions, preamble,
other tests, compiler, SDK, sources, inputs and lock hashes justify reusing
the other two passes. Evidence:
docs/ops/evidence/candidate-module-unit-origin-paths-20261010.json.

Dead intervals are inserted in ascending start order. Each maximum stores the
largest end in its sorted prefix, so binary search tests containment even when
intervals nest or share a start. All traversal, insertion, prefix construction
and lookup use the shared 1048576-work budget; depth, list and storage bounds
fail with original source coordinates rather than declaring a path dead.
Return-origin inference and ownership integration remain subsequent consumers.
No new corpus, full dispatcher, stage1/stage2, release, Linux or push was
performed; public language/SDK ABI, SDK sources, DB, CI and runner contracts
are unchanged.

## Lexical binding identity for origin consumers (2026-10-10)

module_unit_value_model.Expression now retains the original local binding ID
separately from Place.root. The value walk may forward a reference alias to
its initial owner, but that forwarded root cannot identify the alias being
reassigned. module_unit_value_collect preserves the resolver-owned binding
before projection, and module_unit_origin_inputs.binding_owner converts it to
a frame-scoped owner with bounds and kind checks. Nonlocal syntax returns
owner zero; invalid frame/binding IDs fail with code 12.

Development check passed (206 sources, 2518 MIR functions). Three native
development tests passed: reference reassignment retains the alias binding
while its value-place root is forwarded; shadowed aliases retain distinct
bindings and lexical scopes; and the earlier provider-unit generic-instance
collision regression still passes. Reference observations matched 2/2: the
reassigned return is rejected with E2103 at the original return r span, while
the shadowed nonescaping aliases are accepted. The candidate value-only plan
still retains the reassigned example for the subsequent ownership consumer;
this is not semantic-admission equivalence. Evidence:
docs/ops/evidence/candidate-module-unit-origin-bindings-20261010.json.

This change supplies lexical identity for return-origin inference without
guessing origin sets from current value-place roots. Static/unknown
distinctions, branch and assignment joins, call substitution,
loan/lifetime/conflict/move checks and unsafe provenance remain. Original
module/declaration coordinates, type IDs and bounds are retained. No new
corpus, full dispatcher, stage1/stage2, release, Linux or push was performed.
The internal candidate value model/accessor changed; public language/SDK ABI,
SDK sources, DB, CI and runner contracts are unchanged.

## Distinct bounded origin sets (2026-10-10)

module_unit_origin_sets provides canonical analysis-only origin sets.
Unreachable/no-result, Unknown and Static have separate handles; Unknown
absorbs a join, while Static remains a member when joined with a lexical
owner. Owner leaves preserve the global frame/local identity supplied by
origin inputs. Sorted merges deduplicate members and canonical interning makes
equivalent sets share a handle. These handles are not published HIR/MIR type
or origin IDs.

Development check passed (207 sources, 2534 MIR functions), and three native
development tests passed. They verify static/unknown/unreachable behavior,
order-independent canonical unions, duplicates, membership lookup, boundary
owner IDs, malformed handles and exhaustion even on a no-op join. Existing
1048576-work, 16384-owner, 4096-set and 32768-member bounds fail closed.
Evidence: docs/ops/evidence/candidate-module-unit-origin-sets-20261010.json.
This abstract set algebra has no Go semantic-reference equivalence claim.

Return-flow environments, branch/loop joins and argument substitution are the
next consumers. Loan/lifetime/conflict/move, unsafe provenance, cross-unit
execution and self-hosting remain. No new corpus, full dispatcher,
stage1/stage2, release, Linux or push was performed; public language/SDK ABI,
SDK sources, DB, CI and runner contracts are unchanged.

## Bounded return-origin flow and call substitution (2026-10-10)

module_unit_origin_returns now infers direct safe-reference/view return
summaries from checked original and provider bodies. Its flow interpreter uses
stable lexical bindings, evaluates assignments in source order, excludes
terminated branch environments, preserves Static in mixed sets and substitutes
unit-scoped callee parameter origins through argument values captured once in
evaluation order. Loop header environments converge through
fallthrough/continue backedges; exit environments include the zero-iteration
path and break edges. Probe iterations suppress return facts, and nested loops
retain independent edge depths. Scalar-for post execution follows live
backedges; byte-view iteration evaluates its input once.

Development check passed (211 sources, 2575 MIR functions), native development
tests passed 3/3, and reference acceptance/diagnostic/span observations
matched 17/17. Coverage includes static versus unknown, dead return suffixes,
overwrite versus branch joins, terminated branches, argument side effects,
recursive unknown summaries, retained local origins, five loop cases and
colliding generic instance numbers across provider units. Exhausted prior work
fails with code 10 and stale source metadata with code 12. Binary value lookup
charges actual probes and source/node/binding coordinates are rechecked before
analysis. Existing work/depth/round/storage bounds remain fail-closed.
Evidence:
docs/ops/evidence/candidate-module-unit-origin-returns-20261010.json.

A function-wide summary is not a diagnostic for each return: distinct
individually valid returns may join multiple parameter sources for a
subsequent call. Individual return-expression records and diagnostic/lifetime
consumers remain next. Invalid local origins remain identifiable in the
summary; forwarding a nonparameter/local origin yields Unknown rather than
inventing a caller owner. Tagged borrowed return payloads, region origins,
loan/subview facts, lifetime/conflict/move and unsafe provenance are still
pending. No new fixed corpus, full dispatcher, stage1/stage2, release, Linux
or push was performed. Public language/SDK ABI, SDK sources, DB, CI and runner
contracts are unchanged.

## Original return sites and stable origin records (2026-10-10)

Return-origin plans now own individual return records in addition to
function-wide summaries. Each record retains its analysis frame, inferred set
and original return-expression byte span. Per-frame ranges provide bounded
access without scanning unrelated returns. Summary-round records are discarded
before the next round; loop probes never publish records. The final plan
therefore contains only the stable round, live return sites and the original
expression coordinates required by the next diagnostic consumer.

Development check passed (211 sources, 2595 MIR functions), and four native
development tests passed, including all three preceding inference tests rerun
against the changed implementation. The new test verifies two individually
valid alternative returns, a mixed returned call at its complete expression
span, loop-return records without probe duplicates, and exclusion of a dead
return suffix. Reference observations matched 20/20: the three new cases prove
that alternative returns and loop returns are accepted while forwarding the
mixed summary through a single returned call reports E2103. The preceding 17
reference cases were reused only after compiler identity, original fixture
hashes and byte-identical original test functions were verified. Evidence:
docs/ops/evidence/candidate-module-unit-origin-return-sites-20261010.json.

This distinction prevents a future validator from rejecting a whole function
merely because its valid return paths use different parameters. Actual
E2102/E2103/unsupported diagnostics, related owner locations, tagged/region
origins, loan/subview/lifetime/conflict/move checks and unsafe provenance
remain next. The 8192-site cap, 1048576 work and existing frame/depth/round
bounds fail closed. No new fixed corpus, full dispatcher, stage1/stage2,
release, Linux or push was performed; public language/SDK ABI, SDK sources,
DB, CI and runner contracts are unchanged.

## Located static sources through return summaries (2026-10-10)

Reference probes showed that two distinct static literals must retain separate
origin locations: returning their conditional join reports E2103, and
forwarding a function summary preserves the callee literal locations.
Collapsing both literals to the abstract Static handle would erase that
distinction. Origin inputs now append kind-11 static owners with original
expression spans and a bounded mapping for each frame expression. A charged
open-addressed cache shares the same module/source-expression identity across
generic instances without scanning every owner. Static owners have no lexical
local/scope; frame-local ranges and parameter owner IDs remain unchanged.

Return flow now uses these located static owners for borrowed literals and
source_location calls. Call substitution preserves a static source unchanged
instead of replacing it with the caller parameter or call location.
Development check passed (211 sources, 2604 MIR functions), and all five
native development tests passed, including the earlier four tests rerun
against the implementation. The new test covers distinct literal coordinates,
forwarding and shared static identity across two generic instances. Reference
observations matched 24/24, including exact candidate-origin literal spans in
the four new cases. The previous 20 cases were reused after compiler and
embedded source-fixture identity checks. One initial native invocation refused
a stale lock after a source change; the current check refreshed the lock
before the successful run. Evidence:
docs/ops/evidence/candidate-module-unit-origin-static-locations-20261010.json.

The abstract set-algebra Static handle remains distinct from Unknown, but
actual inferred static leaves now retain input owner coordinates. Combined
lexical/static owners, expression mappings, hash slots, work, return sites and
depth/round caps remain bounded. This is still origin inference and input
preparation; E2102/E2103/unsupported diagnostic publication, tagged/region
origins, loan/subview/lifetime/conflict/move, unsafe provenance and cross-unit
execution remain. No new fixed corpus, full dispatcher, stage1/stage2,
release, Linux or push was performed; public language/SDK ABI, SDK sources,
DB, CI and runner contracts are unchanged.

## Direct return-origin diagnostic reports (2026-10-10)

An internal Fairy report collector now consumes the stable individual return
records. Returning a local/by-value parameter owner reports E2102, multiple
origins report E2103, and an unknown origin records unsupported analysis with
exit status 2. Valid alternative returns are checked separately instead of
rejecting their function-wide summary. Diagnostic records retain original
module indices and return-expression spans; related records retain origin or
candidate-origin roles and original owner spans, including provider-module
static literals.

Related locations are sorted and identical source diagnostics are deduplicated
across generic instances using a bounded open-addressed cache. The collector
carries forward the inference work count even when a caller passes a smaller
prior-work value. Exhausted work, allocation failures and inconsistent records
return a Problem rather than a successful report. Bounds cover 8192 return
sites/diagnostics, 16384 related rows per diagnostic, 32768 total related rows
and 16384 cache slots within the existing 1048576-work budget.

Development check passed (212 sources, 2631 MIR functions). Four selected
native development tests passed, covering local escapes, by-value parameters,
branch unions, distinct static literals, recursion, live/dead alternative
returns, provider coordinates, generic duplicate diagnostics and work
exhaustion/no-reset. Eight fresh reference comparisons matched exit status,
diagnostic code, module index, primary byte span and every related role/span
against the Go Stage0 compiler. Compiler, SDK, source, test, lock and embedded
fixture hashes were checked before accepting the result. Evidence:
docs/ops/evidence/candidate-module-unit-origin-diagnostics-20261010.json.

This completes direct return-origin diagnostic collection, not CLI publication
or the full dispatcher. Tagged/region origins,
loan/subview/lifetime/conflict/move, unsafe provenance and cross-unit
HIR/MIR/C/CLI integration remain. No new fixed corpus, stage1/stage2, release,
Linux or push was performed; public language/SDK ABI, SDK sources, DB, CI and
runner contracts are unchanged.

## Manifest return-origin checking entry point (2026-10-10)

The private candidate CLI now exposes --check-return-origins-file <manifest>.
It reuses the existing bounded manifest loader and parses every file once,
then runs signatures, binding, call closure, value typing, return-origin
inference and report collection against those exact source owners. Errors and
related origins print the original manifest filename and byte coordinates.
Direct semantic origin errors return 3; unknown origins, unsupported forms,
phase failures and loader/analysis limits return 2; a successful direct
return-origin report prints return origins ok.

An initial CLI probe reproduced a false success for a local borrow returned
inside Option. The coordinator now scans nested result types and imported
aggregate/payload shapes with a depth/work bound, refusing borrowed payload
returns until their origin flow is implemented. Valid borrowed Option and
nested Result payloads are also deliberately unsupported. Inert Option results
remain admitted. This boundary is recorded separately from semantic
equivalence; it does not count as matching the reference compiler for tagged
borrowed returns.

Development check passed (213 sources, 2635 MIR functions) and a fresh native
candidate build passed. Eleven CLI/reference comparisons matched exit status,
diagnostic codes, original files, primary spans and related role/spans,
including CRLF manifests, provider-first source order and a local aggregate
borrow. Two existing checker modes, four malformed-input refusals, four tagged
boundary cases and command help passed. Current compiler, candidate, SDK,
source, test, lock and fixture identities were verified. Evidence:
docs/ops/evidence/candidate-module-unit-origin-cli-20261010.json.

The command currently uses one bounded primary unit and checks direct return
origins. Tagged/region origin inference, loans, lifetimes, conflicts, moves,
unsafe provenance, split-unit execution and the complete project dispatcher
remain. Release, Linux and stage1/stage2 were deferred as planned. No push,
public Stage0 CLI/language/SDK ABI change, DB change or CI/runner change was
made.

## Single-leaf tagged return origins and wrapper storage (2026-10-10)

The Fairy origin pipeline now recognizes a single safe-borrow leaf through
nested Option/Result types, matching the current Stage0 admission rule.
Constructors preserve the selected payload origin, including success and
failure payloads; .none and inert selected payloads use a known-empty handle
distinct from Unknown and an out-of-scope binding. Empty payloads remain live
through variables, assignments, branch/loop joins and call substitution,
without inventing an owner or emitting an unknown-origin diagnostic. Multiple
borrowed leaves remain explicitly unsupported.

Origin inputs now distinguish a tagged wrapper storage owner from the owner of
its contained borrow. Shared forwarding of an Option parameter retains its
payload origin, while taking a reference to a local or by-value wrapper
records kind-12 storage provenance and reports E2102 at the original
expression, with the wrapper declaration as the related origin. All storage
aliases share the existing owner/work bounds. The unit type resolver and
borrow-place eligibility also admit refs to Option/Result so those storage
cases reach origin analysis. Generic and provider calls retain original
declaration and module coordinates.

The manifest command now admits these single-leaf wrapped returns. Development
check passed (214 sources, 2640 MIR functions); five selected native tests
passed, including known-empty algebra, invalid handles/work exhaustion,
direct-return regression and explicit refusal of multiple borrowed leaves and
pending ? flow. Twenty-four API/reference comparisons matched exit status,
diagnostic codes, module indices, primary spans and related role/spans. A
fresh native candidate build then matched 35 CLI/reference cases and passed
existing checker modes, malformed input, unsupported boundaries, inert tagged
returns and help. Current compiler, candidate, SDK, source, test, lock and
fixture hashes were checked. A source check refreshed the lock after the
native observer snapshot; the native report itself passed 5/5 with lock
verified. Only that snapshot differed, so unchanged compiler/SDK/source/test
identities were verified and the successful native output was re-observed
against the final lock without rebuilding those tests. The recovery and both
lock digests are retained in the receipt. Evidence:
docs/ops/evidence/candidate-module-unit-origin-tagged-20261010.json.

Initial native failures are retained as failures: one test helper missed its
statement terminator, a loop fixture included an unsupported empty statement,
and container-ref cases initially reached the old unit type admission refusal.
The corrected fixtures and the implemented type path were rerun; these earlier
attempts are not counted as passes. Match payload bindings, implicit Result ?
return origins, region origins, loans, lifetimes, conflicts, moves and unsafe
provenance remain. Split-unit execution, the full project dispatcher,
stage1/stage2, release and Linux remain deferred. No push, public Stage0
language/SDK ABI change, DB change or CI/runner change was made.

## Pattern payload origins and complete unsupported reports (2026-10-10)

Option/Result switch arms now seed borrowed payload bindings by lexical symbol
ID from the subject origin, and clear the binding at arm exit. Parameter,
static, local, nested and forwarded origins retain their original owner
coordinates. Taking the address of a by-value scalar pattern payload reports
E2102 with the pattern declaration as its related origin. Inert and
known-empty payloads remain distinct from Unknown.

The origin flow now retains every bounded unsupported return, binding,
assignment and match fact, ordered by frame and preceding return-site ordinal.
Reports preserve their feature and source span, deduplicate equivalent source
facts across instances, and publish semantic E2102/E2103 diagnostics when
present; otherwise the CLI prints all unsupported facts. Facts remain limited
to 8192 and the existing work budget. The Go reference was corrected
separately so assert(false) ends a borrow-flow path and scalar match bindings
have Local provenance (70dfc3c5); a stale bool-view premise was aligned with
ADR 0465 (d3a1eaae). The semantic package passed.

Development source checking passed (215 files, 2654 MIR functions), three
selected native tests passed, and 21 exact API/reference comparisons matched
codes, module indices, primary and related spans. A fresh locked candidate
build matched all 56 CLI/reference cases, plus two existing checker modes,
four malformed inputs, three tagged boundary checks and help. Compiler, SDK,
sources, tests, fixtures, candidate and lock hashes were verified. The earlier
incomplete unsupported projection is retained as a failed observation.
Evidence:
docs/ops/evidence/candidate-module-unit-origin-patterns-20261010.json.

Result ? success and implicit error-return origins remain next.
Region/loan/subview/lifetime/conflict/move and unsafe provenance, split-unit
execution, full project command integration, stage1/stage2, release and Linux
remain pending. No push, public CLI/SDK ABI, DB or CI/runner change was made
in this candidate slice.

## Result propagation return origins (2026-10-10)

The Fairy origin pipeline now admits a single safe-borrow leaf returned through
Result `?`: the success payload keeps its owner provenance, and the implicit
error return is recorded at the full `?` expression only when the failure
carries a safe borrow. Empty payloads, Unknown, parameter/local/static owners,
nested tagged wrappers, generic and provider calls, and loop bodies keep the
existing bounded rules. Multiple borrowed leaves remain explicitly
unsupported.

Development source checking passed (215 files, 2654 MIR functions) and three
selected native tests passed. Twenty exact API/reference comparisons matched
diagnostic codes, module indices, primary spans and related role/spans; a
fresh locked candidate build matched all 76 CLI/reference cases with the same
code/module/primary/related spans, and the legacy pair, four malformed inputs,
two tagged boundary checks and help also passed. Evidence:
docs/ops/evidence/candidate-module-unit-origin-propagation-20261010.json.

Region/loan/subview/lifetime/conflict/move and unsafe provenance remain
pending, together with split-unit execution, the full project dispatcher,
stage1/stage2, release and Linux. No push, public CLI/SDK ABI, DB or CI/runner
change was made in this candidate slice.

## Region capability type groundwork (2026-10-10)

`Region` is now a canonical compiler-owned capability type: `module_types.fy`
resolves the bare name to a single kind-19 node shared across modules with no
scalar, first or second metadata. The graph validator rejects forged
kind-19 records, and `module_bind.fy` recognizes the builtin type name.

Admission covers ordinary, direct, provider and scalar-generic fixed `Region`
input parameters, whose capability identity forwards across modules and calls.
Struct, resource and enum payload Region fields, Region results in ordinary
and specialized value contexts, and `let`/`var` copies including an inferred
`let` are rejected with E3103. A lexical region construct is explicitly refused
too: the unit value walker rejects a lexical region symbol (kind 6) with
unsupported code 2 pending Arena/source/allocation/reset validation, so no
incomplete local capability is published. Wrapper, reference and array Region
shapes stay outside candidate admission; the Go reference admits them, which
is an admission boundary rather than a proven contract bug.

Reference discrepancy: the Go reference still accepts a direct Region return
and `let`/`var` Region copies contrary to the Region language contract and the
E3103 entry. No Go change was made in this unit; the discrepancy is kept for
later alignment without a broad semantic-parity claim.

Final validation passed on the lexical Region refusal gate: root source
checking covered 215 files and 2654 MIR functions, four selected native tests
passed (three new tests plus the tagged return-origin regression), and a
fresh locked candidate build passed all 14 CLI observations. Current compiler,
product source, test, candidate and lock hashes were verified. The CLI keeps
phase failures at status 2; the Region refusals are not yet a complete
structured semantic diagnostic interface.
Earlier failed attempts and the fixed builtin Region binding are recorded in
the receipt. Evidence:
docs/ops/evidence/candidate-module-region-types-20261010.json.

Region.place and lexical region enter/reset/origins remain pending, as do
loans, lifetimes, conflicts, moves, unsafe provenance, cross-unit runtime, the
dispatcher, stage1/stage2, release and Linux. No push, public CLI/SDK ABI, DB
or CI/runner change was made in this candidate slice.

## Region.place and Region input origins (2026-10-10)

`Region.place` resolves against an exact authenticated `std::alloc` public
`AllocError` and a source-set match; `module_unit_interfaces.fy` owns the
implicit error shape. Kind 300 typechecks one operand on an exact Region receiver
and returns `Result<ref T, AllocError>`, rejecting `NeedsDrop`/resource
transitively with code 3102; borrow-containing placements remain unsupported
with code 2.

Fully inert scalar, raw pointer, struct, enum, tagged and array placement is
supported, and enum reference admission joined the owned type and borrow-place
path; a regression placing the same nominal struct through direct and Option
field traversal is fixed. `sdk_scopes` admits only the exact authenticated Arena
initialize/drop declarations (rules 70/71, `u32(ptr mut void)`), with SDK
bytes/ABI unchanged and unauthenticated or forged `fy_` symbols refused.

Region parameters carry a kind-13 origin owner distinct from ordinary borrowed
parameters and wrapper storage, and allocation derives Region provenance rather
than the placed value's origin. Direct, generic and provider forwarding,
argument reordering, `?`, match, nested wrappers and mixed-input E2103 keep their
original coordinates.

Validation: source check passed (217 files, 2658
MIR functions), native 3/3 passed (tagged regression and exhaustion), 17/17
API/reference comparisons matched codes, module indices, spans and related roles
on the authentic Alloc SDK, and a fresh locked candidate build passed CLI/reference 17/17 with refusals 8/8;
native and API hashes match the current sources and tests. A forged
`fy_std_arena_drop_v0` alias is rejected with code 5106/exit 2 where the reference
exits 0, holding the reserved SDK symbol authentication boundary; the retained
E5106 initial failure was a missing SDK Arena foreign-symbol admission. Evidence:
docs/ops/evidence/candidate-module-region-place-20261010.json.

Go discrepancy: the reference rejects a direct resource placement (negative 3,
exit 3) but accepts nested Option-resource (negative 4) and borrowed Option
(negative 5) placement at exit 0 that the candidate rejects as normative
non-inert inputs, so no broad Region parity is claimed. Runtime Region execution
and the lexical Region construct with E3101 region escape remain pending, as do
release, Linux and stage1/stage2; no push or public Stage 0 CLI/SDK ABI, DB or CI
change was made.

## Lexical Region source typing (2026-10-10)

The resolved lexical Region binding receives the canonical kind-19 capability
type. Its source must be an exact `ref mut std::alloc::Arena` expression, with
the Arena declaration owned by the authenticated SDK. The source expression is evaluated before
entering the nested block, and invalid sources report code 2 with the primary span
on the source expression rather than the binding name. The gate also refuses
unchecked or unresolved source consumers after a bounded fixed point.

Development source checking passed (218 files, 2664 MIR functions). Native 5/5
selected tests and 14/14 exact API source comparisons passed, matching Go codes,
module indices and source spans. A fresh locked CLI build passed all 14 exact comparisons on status and
span, and a local region-escape boundary case reports candidate code 2/status 2
for the unsupported borrow return against the reference E3101/status 3 with an
allocation related span. This unit covers typing and block analysis only; E3101
region escape, lifetime, loan and reset handling and runtime Region execution are
excluded, and a local Region borrow-return stays explicitly unsupported pending
the next unit. Current compiler, SDK, source, test, lock and candidate hashes were verified.
Public Stage 0 CLI/SDK ABI, DB and CI are unchanged; no push was made. Linux,
release and stage1/stage2 remain deferred. Evidence:
docs/ops/evidence/candidate-module-region-sources-20261010.json.

## Region escape and allocation provenance (2026-10-10)

Region local origins are now kind-14 facts seeded only inside the statement-17
block, with the capability identity kept distinct from a per-origin allocation
map. `g::Value` transports the origin and allocation map through variables,
assignments, match, `?`, arguments, branches and loops, and remaps generic and
provider summaries; a mixed summary keeps each leaf's allocation because caller
Region arguments may alias. A `RegionParam` uses its callee's allocation, while a
borrowed parameter forwards the actual argument allocation, and local origin
declarations for E2103 use the entire lexical Region statement. Provider
cross-module origin analysis is implemented.

E3101 reports the primary span at the return expression or `?` with the
allocation as related role 3 (module and span), without an E2102 cascade, and the
existing private `--check-return-origins-file` prints E3101 and the allocation.
Allocation sites are capped at 16384 immutable entries, and the provenance ledger
holds 4096 maps and 36864 rows inside the existing 1 MiB work budget.

Development source checking passed (219 files, 2700 MIR functions), five selected
native tests and all 21 exact API cases passed against the Go reference on code,
module and primary and related roles and spans, including alias Region arguments
(callee E2103 with the caller E3101 at the first callee site), direct and assigned
second sites, a provider-index-2 related allocation, implicit borrow-failure `?`
full-span reporting, a loop return and accepted known-empty Result/Option values,
with recursion's unknown-2 summary preserved. A fresh locked dev build passed all
128 CLI comparisons on status, code, file and primary and related roles and spans,
and the current native and API inputs match their hashes. Split-unit HIR/MIR and
runtime execution and the full dispatcher remain pending, together with runtime
region enter/reset, loans, lifetimes, conflicts, moves, unsafe provenance,
stage1/stage2, release and Linux. The public Stage 0 CLI and SDK ABI are
unchanged. DB and CI are unchanged; no push was made. Evidence:
docs/ops/evidence/candidate-module-region-escape-20261010.json.

## Borrowed-array read fusion (2026-10-11)

The candidate C emitter now imports `module_unit_c_borrowed_arrays` and fuses one
narrow shape: a single-use array field read on a reference receiver whose index is
a literal or a local, with no intervening effect or control node, is emitted
against the field itself instead of first copying the whole aggregate into a value
temporary. An effectful index remains outside the fused shape and retains the
array snapshot, so mutating the source still observes 1 before 99, and a bounds
failure still traps with 106. Outside the proven shape the previous copy behavior
is unchanged.

Checking passes 293 files and 3681 MIR functions with the lock verified. Three
native emission tests pass in 110.7170 seconds, and six generated-C executions
pass under `/Od` and `/O2`, including 33 recursive frames with 24 `usize[1024]`
reads.

A fresh Go-emitted cache `/O2 /Ob1` bootstrap joins the full 286 sources
(2028270 bytes) and emits a 25485402-byte packet with 35 units, exit 0 in 37.5842
seconds. The generated Stage1 development compiler builds in 61.9567 seconds
(9652224 bytes), yet the full exact-source replay still exits 0xC00000FD after
33.3301 seconds with no output. The reproducible two-stage bootstrap
`promotion` row stays `blocked`: stage1/stage2 artifacts and differential
results do not match yet, and this slice does not claim the whole compiler is
fixed. The stack attribution is deliberately left to the next slice. The 60-second
frontend cap, the public Stage 0 CLI, the SDK ABI, DB and CI are unchanged; Linux,
the full suite and the Stage1/Stage2 gate were not run, and no push was made.
Evidence: docs/ops/evidence/candidate-borrowed-array-read-20261011.json.

## Move-analysis stack isolation (2026-10-11)

A native trace of the earlier failing packet attributed the move-analysis stack
pressure to two functions: `moves::build` (`fy_fn_2857`, about 401472 bytes of
frame delta) and `move_walk::expression` (`fy_fn_2849`, about 35728 bytes per
recursive frame). The bounded 1024-by-3-array Context constructor now lives in
`g::context`, and `move_walk::expression` is split into binding, logical and
projected helpers over a borrowed `Expression`. The change is limited to
emission and layout: traversal order, the depth 64 and 1 MiB work bounds, and the
pending-count restore are unchanged.

Checking passes 293 files and 3685 MIR functions with the lock updated, and the
three existing native tests (reachable branches and loop edges, exact move facts,
retained and call-held owner loans) pass in 112.7483 seconds.

The full 286-source 2029931-byte join emits a 25490785-byte packet with 35 units,
exit 0 in 35.7388 seconds from the Go-emitted baseline, and the full development
native build exits 0 in 56.2073 seconds (9651200 bytes). The exact self replay
still exits 0xC00000FD after 41.3244 seconds, so the reproducible two-stage
bootstrap `promotion` row stays `blocked`. The cache-only trace now repeats
`fn_3516`/`fn_3522` (`evaluate`/`binary`) in scalar value typing on the
former move path, and localization continues. The source C root runtime
`e4b91c553b5f6db9e65cd031569763a9ddf73e77e58db4ce9fad4c19830be6b7` verifies
against the current Go runtime owners. The optimized native compiler builds
from the retained packet in 200.6866 seconds, but its exact-source replay reaches
the unchanged 60-second frontend timeout without a packet. This slice claims
only reduced move-frame pressure.

The 60-second frontend cap, the caps, the public Stage 0 CLI, the SDK ABI, DB and
CI are unchanged; Linux, the full suite, Stage 2 and push were not run. Evidence:
docs/ops/evidence/candidate-move-analysis-stack-20261011.json.

## Scalar typing snapshots (2026-10-11)

The scalar `evaluate` path now borrows `E` for dispatch, and `binary` keeps only
its five coordinate fields in a 40-byte `BinarySource` with borrowed operands,
freezing the scalar result before the mutable record is touched. Inference order
and source spans, the depth 64 bound and the work budget are unchanged.

Checking passes 293 files and 3686 MIR functions with the lock updated. Four
targeted native outcomes pass after a focused rerun: generic, cleanup and ordinary
call results, direct literal contexts, type and overflow spans, and scalar casts.
The first four-run pass was 3 of 4 in 133.7185 seconds, and the single failure
reproduced identically against the old-source baseline (31.6533 seconds) with the
previous source reporting 4 frames, 5 inferred, 4 checked returns and 537 work
against stale 3/3 expectations. The updated test expects 4/4 and the focused
rerun passes in 46.1721 seconds, reusing the three unchanged results. The nine CLI
cases and the Vec development and release runs pass.

The full 286-source 2030206-byte join emits a 25490067-byte, 35-unit packet from
the Go-emitted baseline in 39.226 seconds
(SHA256 `98d7ccd80786a7611e8f39bc0b7392d39bf70a31a643a00f4dea6ed61a35f7c3`). The
Stage1 development native build takes 61.117 seconds (9650688 bytes), and its full
same-source replay now reaches the 60.0419-second frontend timeout with no output
and no observed stack failure. The release raw packet build takes 196.7524
seconds over 37 objects and its full self replay stops at the 60.0415-second
timeout.

The earlier observed stack failures are gone inside the unchanged 60-second
frontend cap, but neither a full finish nor the reproducible two-stage bootstrap
`promotion` row is proven, so that row stays `blocked` and bottleneck profiling
comes next. No cap, timeout, public Stage 0 CLI, SDK ABI, DB, CI or stack reserve
change, and no push; Linux, the full suite and Stage 2 were skipped. Evidence:
docs/ops/evidence/candidate-scalar-stack-20261011.json.

## Inert struct field projections (2026-10-11)

The CFG fusion now also covers a single-use inert struct value node that is
immediately followed by a field read. It accepts only a direct or original place
with local or reference field-safe ancestry, no intervening effect, store or
control node, and it ignores call-result owners and by-value or resource use. The
owner temporary is skipped, while the selected field still goes through the
existing opaque `fy_unit_copy`, so the MSVC view aliasing protection stays in
place.

Checking passes 293 files and 3689 MIR functions with a comment-only lock
refresh. Six native emission tests pass in 198.1855 seconds, and twelve `/Od` and
`/O2` generated-C executions pass, including a new 33-depth, 24-frame-read
`usize[512]` array reproducer where the previous old-low binary's 0xC00000FD is
0 in the new low build, plus the effectful index snapshots, the bounds trap and
the original root and projected snapshot with view mutation regression. The nine
own CLI cases and the Vec development and release runs pass.

The full 286-source 2032081-byte join emits a 24476397-byte, 35-unit packet from
the Go-emitted baseline in 37.4733 seconds
(SHA256 `62bc25db9f06cebfe647c77c4072d66f7954af80bb7955e5ae50a82a8acaa02b`),
1013670 bytes below the previous 25490067-byte packet. The full development
native build takes 56.3886 seconds (9346560 bytes) and its same-source self replay
stops at the 60.0522-second frontend timeout. The optimized raw packet build emits
the same 35 units over 37 objects and exits 0 in 184.8278 seconds, and its exact
same-source re-emission stops at the 60.0590-second frontend timeout.

The four locally verified slices leave the whole self-hosting gates pending: the
reproducible two-stage bootstrap `promotion` row stays `blocked` because no
stage finishes the exact same-source re-emission inside the unchanged 60-second
frontend cap. No public API, SDK ABI, DB, CI, cap, timeout or stack reserve
change, and no push; Linux, the full suite and Stage 2 were deferred. Evidence:
docs/ops/evidence/candidate-inert-field-projection-20261011.json.

## Canonical name normalization (2026-10-11)

`scanner::canonical_path` reuses the existing letter, digit and keyword(false)
tables so that only ASCII `identifier(::identifier)*` paths with no whitespace
or comments are admitted, and the `source_bytes` cap is unchanged.
`module_registry::normalize` appends the accepted bytes once and leaves every
other input to the existing scanner fallback, so reserved keywords and `as`,
`module`, `import` and `defer`, trivia and CRLF and comments, malformed UTF-8,
partial-failure output, append semantics and over-limit handling behave as
before. The addition is a private scanner helper; the public surface is
unchanged.

Source checking passes 293 files and 3690 MIR functions. The two authored
normalization native cases and the existing
`nominal_module_consumers_keep_identity_and_privacy` case pass 3 of 3 in 108.8601
seconds, with the real native result in
`.cache/candidate-tagged/sdk-c-execution/native-normalize.json`, and the new
Stage1 CLI nine-case matrix plus the Vec development and release runs pass.

The fresh baseline joins the exact 286 sources (2032971 bytes) and emits 24485052
bytes over 35 units in 33.4711 seconds
(SHA256 `d74de2f7b43b8ba25288da0514f8ae9d1b52e18f5c00d9676777a01a3f80edee`). The
development native build takes 53.8747 seconds (9350144 bytes) and the full
exact-source self replay stops at the 60.0351-second frontend timeout. The
optimized release build exits 0 in 188.2586 seconds over the same 35 units and 37
objects
(`.cache/candidate-tagged/sdk-c-execution/stage1-normalize-release-build-observation.json`),
and its exact same-source self replay stops at the 60.0460-second frontend timeout
(`.cache/candidate-tagged/sdk-c-execution/stage1-normalize-release-replay.json`),
so the reproducible two-stage bootstrap `promotion` row stays `blocked`.

The sparse cache-only development phase profiler shows the signature phase at
about 14.66 seconds, then repeated closure processing until timeout, while a
separate counter probe records many normalize-driven scanner invocations; all of
these are diagnostic only and none of them measures a whole-frontend speedup.
Public semantics, SDK ABI, DB,
CI, caps, stack reserve and the timeout are unchanged, and no push, Linux, full
suite or Stage 2 run has happened. Evidence:
docs/ops/evidence/candidate-canonical-name-normalization-20261011.json.

## Borrowed canonical type comparisons (2026-10-11)

The private `same` helper in `module_types` now reads two `ref Type` instead of
two `Type` values, `validate` keeps its outer `t` as a shared reference to a
`Vec` element, and `intern` compares the candidate value against the existing key
borrowed from `Vec::at` before the loan ends and the push happens. The comparison
keeps its six fields and their order, the first-match ID, duplicate rejection, and
graph rollback, admission and the 4096 cap; the index and hash algorithm are
unchanged.

Source checking passes 293 files and 3690 MIR functions, and the three existing
native cases `module_type_graph_native_contract`,
`reordered_wrappers_rebind_edges_and_preserve_array_lengths` and
`retained_plan_rejects_changed_provider_bytes_and_graph_prefix` pass 3 of 3 in
97.6437 seconds.

The real generated-C canonical key probe interns 256 keys for 32 passes and
measures only private interner key identity rather than catalog admission. Old
and new builds both return 0 in native development and `/O2` across three runs
each, with median seconds moving from 0.1983919 to 0.1056381 in the low profile
and from 0.0538169 to 0.0189355 optimized
(`.cache/candidate-tagged/sdk-c-execution/type-comparison-probe-before.json` and
`type-comparison-probe-after.json`). This is a call-specific measurement and does
not establish a whole-frontend speedup.

A fresh bootstrap joins the same 286 sources (2033072 bytes) and emits a
24487241-byte packet over 35 units in 33.1035 seconds
(SHA256 `0bc3d3db452ee5a9b3868aaa078485d003350da361ce7ac4e2071eddba6c626f`). The
development native build takes 52.9479 seconds (9350656 bytes) and the
exact-source full self replay stops at the 60.0371-second frontend timeout. The
new Stage1 CLI nine-case matrix and the Vec development and optimized runs pass.
The optimized build exits 0 in 185.0607 seconds over the same 35 units and 37
objects
(`.cache/candidate-tagged/sdk-c-execution/stage1-type-borrow-release-build-observation.json`),
and its exact same-source self replay stops at the 60.0479-second frontend
timeout
(`.cache/candidate-tagged/sdk-c-execution/stage1-type-borrow-release-replay.json`),
so the reproducible two-stage bootstrap `promotion` row stays `blocked` while
the by-reference signature stays private to the `same` helper. The public source
and API surface, the SDK ABI, DB, CI, caps, stack reserve and the timeout are
unchanged, and no push, Linux, full suite or Stage 2 run has happened. Evidence:
docs/ops/evidence/candidate-borrowed-type-comparisons-20261011.json.

## Candidate check CLI (2026-10-12)

The private Fairy `check [--locked]` composes workspace/SDK/import loading,
lock policy and package compilation with retained-source diagnostic rendering.
Source checking passes 357 files / 4665 MIR functions. The Go-frontend native
dispatcher and candidate-generated C development/release probes match 21 cases
each (63 observations): 15 complete stderr comparisons, three invocation first
lines and three package prefixes/statuses per image. Locked inputs retain bytes
and modification times; three unlocked update cases produce Go-identical locks.

The candidate probes forward real OS arguments through a private check-only
entry; the full dispatcher is verified through the Go frontend. The current
350-file closure is 2,483,126 source bytes and exceeds the 2 MiB catalog cap;
this inventory is not a full-source candidate replay. General diagnostic/help
parity, package entry/build/run/native driver and timeout wiring, Linux/full
suite and Stage1/Stage2 remain. Promotion is not completed. Public API, SDK
ABI/version, DB and CI are unchanged; no push occurred. Receipt:
[candidate-check-cli-20261012.json](../ops/evidence/candidate-check-cli-20261012.json).

## Source-set cap (2026-10-12)

Aggregate source admission rises from 2 MiB to 4 MiB; bootstrap `sources()` uses the same cap.
Per-file 64 KiB, 512 files, joined-source 1 MiB, all analysis caps and product timeouts stay unchanged.
The stale 256-file/2-MiB `source_limits` test now follows the actual 512-file/4-MiB limits:
512 files, raw 4 MiB and separator-inclusive exact 4 MiB pass; an extra file, raw-source overflow and separator-inclusive one-byte overflow fail.
The existing joined-source 1-MiB and expression-arena 16k bounds remain covered.
Three native tests and all three tests in both candidate-generated C profiles pass; native results were reused without a rerun.
Anchored regex fixes inline `@test` stripping after the earlier phase-1 code-3 failure at line 27.
Source check passes 357 files / 4665 MIR functions. A fresh Go-native full-dispatcher seed builds in 37.70 seconds.
The full 350-file candidate input (2,483,350 bytes) runs emission for 237.47 seconds, exits 2 and emits zero C bytes:
`module_unit_generics.fy`, project phase 2, stage 4, code 10, bytes 0..0. Byte admission is resolved; analysis work still stops execution.
Stage2 is not achieved. Next: analysis partitioning, then package entry/C bundle/build/run/driver and timeouts, Linux/full suite, Stage1/Stage2.
API, SDK ABI/version, DB and CI are unchanged; no push. Receipt:
[candidate-source-set-cap-20261012.json](../ops/evidence/candidate-source-set-cap-20261012.json).

## Generic plan copies (2026-10-12)

Seven concrete helpers preserve copy order and work/error semantics without
raising analysis limits. Source check passes 358 files / 4672 MIR functions;
nine generic regressions pass in Go-native and candidate C development/release.
Full emission over 351 files / 2484337 bytes now passes the previous generic
module failure and stops at `module_unit_interfaces.fy`, phase 2, stage 4,
code 10, after 258.23 seconds. No C packet or Stage2 is produced. Interface
analysis partitioning remains the immediate blocker; entry/driver/build/run,
timeouts, Linux/full-suite and Stage1/Stage2 remain. Public API, SDK ABI/version,
DB and CI are unchanged; no push. Receipt:
[candidate-generic-copy-20261012.json](../ops/evidence/candidate-generic-copy-20261012.json).

## Interface analysis partition (2026-10-12)

Eight helpers move to three modules with unchanged traversal and work/error
semantics. Source check passes 361 files / 4672 MIR functions; five interface
regressions pass in Go-native and candidate C development/release. Full
emission over 354 files / 2486396 bytes passes the interface failure and stops
at `syntax.fy`, phase 2, stage 4, code 10, after 469.94 seconds. No C packet or
Stage2 is produced. Parser analysis, entry/driver/build/run, timeouts,
Linux/full-suite and Stage1/Stage2 remain. Limits, public API, SDK ABI/version,
DB and CI are unchanged; no push. Receipt:
[candidate-interface-analysis-20261012.json](../ops/evidence/candidate-interface-analysis-20261012.json).

## Parser driver partition (2026-10-12)

Declaration/recovery routines move to a helper; parser entry signatures and
the original `Parsed` owner remain. Source check passes 362 files / 4673 MIR
functions. Three parser regressions pass in Go-native and candidate C
development/release, covering metadata, rollback/recovery, lexical diagnostics
and declaration bounds. Full emission over 355 files / 2486853 bytes exits
`0xC00000FD` after 566.51 seconds without stderr or C output. The preceding
structured syntax analysis error is not reported, but the crash phase remains
unattributed. Stack attribution/reduction is next; full C emission and Stage2
are not achieved. Limits, stack reserve, public API, SDK ABI/version, DB and CI
are unchanged; no push. Receipt:
[candidate-syntax-driver-20261012.json](../ops/evidence/candidate-syntax-driver-20261012.json).

## CFG recursive stack reduction (2026-10-12)

Six CFG expression helpers reduce recursive stack use without changing depth,
work or graph semantics. The formerly crashing 50-term logical chain now emits
C and executes in both profiles. Five CFG regressions pass with the Go frontend
and candidate-generated C in development/release. Source check passes 362 files /
4679 MIR functions. The full 355-file replay returns phase 5 code 1002 after
645.61 seconds rather than crashing. Diagnostic instrumentation identifies
reachability exhausting the shared preparation counter after SDK and indexes
pass. Independent bounded preparation histories, all charged to the existing
8 Mi work ledger, are next. Full C emission and Stage2 remain unachieved.
The verification harness adds opt-in test subsets and frontend allowance;
product timeout, stack reserve, public API, SDK ABI, DB and CI are unchanged.
No push. Receipt:
[candidate-cfg-stack-20261012.json](../ops/evidence/candidate-cfg-stack-20261012.json).

## Independent C reachability history (2026-10-12)

Index/SDK preparation (545600 work) and reachability (625962) now pass separate
1 Mi work bounds while both count toward the unchanged 8 Mi project ledger.
The full 355-file replay still fails phase 5 code 1002: aggregate declarations
use 343438 work, then literal collection exhausts their shared 1 Mi meter.
Literal-pool isolation is next; no complete C packet or Stage2 is claimed.
Source check passes 362 files / 4679 MIR functions. Two Release native cases
pass with verification-only build allowance 240 seconds after the product's
60-second allowance timed out. Exact packets permit four prior C executions
to be reused. Public API, SDK ABI, DB and CI are unchanged; no push. Receipt:
[candidate-c-reachability-scopes-20261012.json](../ops/evidence/candidate-c-reachability-scopes-20261012.json).

## Complete dispatcher C emission (2026-10-12)

Separate bounded aggregate/literal/common-declaration histories resolve the
preparation failure without raising the project ledger or output bounds.
The complete 355-file dispatcher emits a 29372117-byte C packet in 67.27 seconds:
5200 selected frames, 45 body files and 1226 shared literals. Source check passes
362 files / 4679 MIR functions. The Release native literal-pointer case passes;
unchanged packet bytes retain its two prior C executions. This is complete C
emission from a Go-produced optimized seed. Full C-part compiler linkage,
execution, consecutive generations and promotion remain separate gates.
Product timeout, public API, SDK ABI, DB and CI are unchanged; no push. Receipt:
[candidate-c-literal-scopes-20261012.json](../ops/evidence/candidate-c-literal-scopes-20261012.json).

## Complete C-part compiler Stage1 (2026-10-12)

The complete 45-unit compiler links in 249.85 seconds to a 4177920-byte optimized
image. It processes all 355 self sources without Go frontend fallback in 85.27
seconds and emits exactly the original 29372117 packet bytes. Stage1 C-part
construction and replay pass; the second image and promotion remain pending.
The bootstrap verifier permits opt-in frontend allowance 1..900 seconds while
retaining the default 60 and unchanged product timeout. Help/boundary option
checks pass. API, SDK ABI, DB and CI are unchanged; no push. Receipt:
[candidate-complete-stage1-20261012.json](../ops/evidence/candidate-complete-stage1-20261012.json).

## Complete C-part compiler Stage2 fixed point (2026-10-12)

The second C-part compiler builds in 271.92 seconds from Stage1's identical
packet. Complete Stage1/Stage2 PE bytes match exactly (4177920 bytes, no excluded
bytes). Both generations process all 355 self sources without Go frontend
fallback: Stage1 takes 85.27 seconds, Stage2 takes 95.15, and both emit the same
29372117 packet bytes. The seed and both images match 249 fresh Go lexer cases
and the private framing contract. The bootstrap verifier passes by reusing the
measured emissions/builds with explicit verification allowance 900 seconds.
An initial lexer-oracle invocation used the general CLI and was corrected to
the dedicated current-source oracle before successful candidate comparisons.

This complete private Windows C-part fixed point satisfies consecutive
generation/image comparison. Promotion remains false: product timeout and
package entry/native driver/build/run integration, full parser/typing/executable
differential, Linux/full suite and hosted CI remain. Product defaults, public
API, SDK ABI, DB and CI are unchanged; no push. Receipt:
[candidate-complete-stage2-20261012.json](../ops/evidence/candidate-complete-stage2-20261012.json).

## Direct frontend-to-native build deadline (2026-10-12)

The private bridge now forwards explicit frontend deadline `(0,10m]`, default
60 seconds, and terminates the Windows process tree on expiry. With 2 minutes,
the complete 355-source Stage2 frontend and 45-unit C compilation publish an image
in 365.75 seconds, identical to the full Stage1/Stage2 image. No cached packet or
Go frontend fallback is used. Actual child termination, previous-image retention,
invalid options and existing build failures pass their checks.
Workspace entry/build/run integration, stable deadline mapping and the remaining
differential/platform/promotion gates are still open. Private command/internal Go
API changed; public CLI, SDK ABI, DB and CI are unchanged; no push. Receipt:
[candidate-frontend-timeout-20261012.json](../ops/evidence/candidate-frontend-timeout-20261012.json).

## Root-owned workspace entry (2026-10-12)

The Fairy selector resolves the root instance's exact `main::main` rather than
the first manifest source. It enforces fn() -> i32 and ignores helper/dependency
entries. Six locked workspace cases agree with the Go oracle; two authored tests
pass in Go-native and both candidate C profiles, including bounded work and
source provenance. Source check passes 363 files / 4681 MIR functions.
Workspace emission/build/run integration and the remaining promotion gates are
still open. Public API, SDK ABI, DB and CI are unchanged; no push. Receipt:
[candidate-entry-selection-20261012.json](../ops/evidence/candidate-entry-selection-20261012.json).

## Fairy workspace C command (2026-10-12)

`--c-workspace <root> [--locked]` connects Fairy-owned manifest/dependency/lock,
source and semantic/IR processing to canonical entry selection and C partitioning.
The 359-source dispatcher builds through the Stage2 seed without Go frontend
fallback. Its workspace packet executes with result 42 in development and release.
Invalid entries, missing/stale locks, syntax/type errors and invalid arguments
reject without a packet; locked fixtures remain unchanged. The initial type
fixture was corrected to pass lock admission before checking E1009/code 3.
Source check passes 366 files / 4701 MIR functions. Stable build/run, native bridge
workspace selection and the broader differential/platform/promotion gates remain.
Public API, SDK ABI, DB and CI are unchanged; no push. Receipt:
[candidate-workspace-emission-20261012.json](../ops/evidence/candidate-workspace-emission-20261012.json).

## Direct workspace native bridge (2026-10-12)

The private bridge accepts `--workspace` with optional `--locked`, forwards the
arguments to Fairy and compiles its packet through the Windows MSVC driver.
Both direct workspace profiles return 42 without a cached packet or Go frontend
fallback. Nine frontend rejection cases preserve the previous image; selector,
argv and existing deadline/build-failure checks pass. Source check passes 366
files / 4701 MIR functions; runtime reuses the `43c45458` dispatcher before its
help-only update. Stable build/run, program arguments, status/deadline mapping and
the broader differential/platform/promotion gates remain. Private bridge API
changed; public API, SDK ABI, DB and CI are unchanged; no push. Receipt:
[candidate-workspace-native-build-20261012.json](../ops/evidence/candidate-workspace-native-build-20261012.json).

## Fairy Windows argv encoder (2026-10-12)

The private Fairy encoder owns complete UTF-16 command construction and CRT
quoting, including empty/Unicode arguments, quotes and trailing backslashes.
The exact 32767-word limit includes NUL; argument count is bounded at 1024
including the executable. Invalid UTF-8/NUL, executable quotes and overflow
reject without a successful prefix. Two Go-native tests and both candidate C
profiles pass; source check passes 367 files / 4712 MIR functions. Process launch
and stable build/run remain pending. Public API, SDK ABI, DB and CI are unchanged;
no push. Receipt:
[candidate-windows-command-20261012.json](../ops/evidence/candidate-windows-command-20261012.json).

## Fairy Windows process host (2026-10-12)

Fairy owns hidden suspended launch, job assignment/resume, deadlines and handle
resources. C adapters only marshal Win32 calls. Two actual candidate C cases pass
in both profiles on ordinary and Korean/space paths: argv/full exit bits,
repeated warmed handle counts and native confirmation that the observed live
parent/child terminate at the deadline. The initial one-time handle increase is
not attributed. An ordinary development image retains its exact prior bytes.
Go source check passes 368 files / 4716 MIR functions and host/bridge checks pass.
Go-native is explicitly skipped for private adapter linkage. Stream byte tests,
stable argument admission, build/run/status/trap integration and broader platform
promotion gates remain. Private host ABI added; public API, SDK ABI, DB and CI are
unchanged; no push. Contract:
[candidate-windows-host-v1.md](candidate-windows-host-v1.md). Receipt:
[candidate-windows-process-20261012.json](../ops/evidence/candidate-windows-process-20261012.json).

## Native stream forwarding (2026-10-12)

All three process tests pass in both candidate C profiles. The new case forwards
binary stdin including NUL, exact stdout/stderr bytes and child exit 41 through
the existing Fairy process API. The unchanged library's prior source check is
reused; Go-native remains explicitly skipped for private adapter linkage.
Stable argument admission and build/run/status/trap integration remain open,
alongside the broader platform/promotion gates. Public API, SDK ABI, DB and CI
are unchanged; no push. Receipt:
[candidate-windows-streams-20261012.json](../ops/evidence/candidate-windows-streams-20261012.json).

## Windows argument admission parity (2026-10-12)

The private encoder removes its provisional 1024-argument cap and enforces the
actual 32767 UTF-16 words including NUL. Seventy Go `syscall.EscapeArg` command
lines match byte-for-byte, including quoted executable trailing backslashes.
Three authored tests pass in Go-native and both candidate C profiles; source
check passes 368 files / 4717 MIR functions. Private trap control, stable
build/run, Linux/full suite and promotion remain open. Public API, SDK ABI,
DB and CI are unchanged; no push. Receipt:
[candidate-windows-argv-20261012.json](../ops/evidence/candidate-windows-argv-20261012.json).

## Private Windows run control (2026-10-12)

Four control-channel tests and three existing process tests pass in both
candidate C profiles. Fairy preserves ordinary u32 exits, normalizes the private
C trap tags only with matching receipts, preserves parent/inherited environment
bytes and forwards binary streams. The child uses a separate inherited pipe;
receipt inspection never waits for EOF. The bridge explicitly opts into the
existing receipt runtime at base 128; its default stays at base 100.
Source check passes 369 files / 4730 MIR functions and selected bridge checks
pass. Go-native is skipped for private adapter linkage. One transient assertion
failure remains unattributed; its exact binary passed unchanged on replay, and
the final labelled tests pass both profiles. Stable build/run wiring, broader
differential, Linux/full suite and promotion remain open. Private ABI, bridge
and verifier changed; public API, SDK ABI, DB and CI are unchanged; no push.
Receipt: [candidate-windows-run-control-20261012.json](../ops/evidence/candidate-windows-run-control-20261012.json).

## Bounded native program output (2026-10-12)

Five capture tests pass in both candidate C profiles: the shared 64KiB boundary,
continued draining after overflow, malformed control flooding, real traps/raw
exits, binary streams and live parent/child timeout cancellation. Ten standalone
executions retain five raw Windows statuses through the signed main ABI in both
profiles. Source check passes 370 files / 4748 MIR functions and host selection
passes. Go-native is skipped for private adapter linkage. Public build/run wiring,
broader differential, Linux/full suite and promotion remain open. Private API and
clock adapter changed; public API, SDK ABI, DB and CI are unchanged; no push.
Receipt: [candidate-windows-run-capture-20261012.json](../ops/evidence/candidate-windows-run-capture-20261012.json).
