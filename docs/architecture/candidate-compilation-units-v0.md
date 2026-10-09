# Bounded candidate compilation units

Owners: `examples/lexer_candidate/src/module_units.fy` and
`examples/lexer_candidate/src/module_ast_join.fy`.

The original single joined owner cannot hold the actual compiler source set:
the observed compacted dispatcher has 152020 expressions and 11509 entries.
Raising its numeric caps would also expand every downstream binding, HIR and
MIR arena. Compilation therefore needs several independently bounded owners.

The first implemented layer groups whole input modules in manifest order.
`module_units::build` measures the actual parser-owned arena counts and emits
maximal contiguous half-open module ranges. Every original module appears in
exactly one range, including unused user functions. No declaration, body,
source byte or token is removed to make a group fit. A module that cannot fit
by itself is rejected; function-level subdivision is not implemented here.

Every group retains the existing joined-owner caps: 512 declarations, 16384
expressions, 8192 statements, 1024 entries, 4096 types, 1024 fields, and 4096
attributes and attribute arguments. The entire input still has at most 256
files, at most 65536 bytes per file and at most 1048576 bytes including its
original separator newlines. Partitioning does not admit larger source sets.

`module_ast_join::join_range` borrows the full source vector and joins only one
nonempty range. It applies all existing join checks and remaps each selected
file's source spans, token IDs and arena references exactly as the full join
does. The returned owner starts at local source offset zero. Failures retain
the original full-vector module index, not the group's relative index. Invalid
ranges return code 9 before subtraction or indexing. The original `join` API
delegates to the full range and keeps its original limits and rejection behavior.

The plan is a grouping aid, not semantic or SDK authority. `joined` rechecks
the live sources through the range join; cached usage does not grant permission
to exceed a bound. Source-set signature validation and SDK authentication remain
with their existing owners. Separate groups can be joined sequentially without
copying a second vector of all parser owners or retaining all joined results.

The development probe authenticated and compacted the actual 160-file dispatcher
and joined all twelve groups. Each group's observed arena counts equal its
measured plan, and all original modules are covered once. The eight native tests
include a 1206-entry source set that the full join rejects but two bounded joins
retain exactly, source-byte preservation, original error coordinates, invalid
ranges, and the existing join/cooked-storage regressions. Evidence:
`docs/ops/evidence/candidate-module-units-20261009.json`.

This layer does not yet compile the groups. Cross-group calls, imported nominal
types, generic specialization, owned HIR/MIR identity, C symbols, final linking
and diagnostics must consume full-source coordinates while using local arenas.
The current CLI continues to use the single joined owner until those connections
are implemented; successful AST partitioning is not a full-dispatcher C result
or stage1/stage2 proof.

## Owned cross-unit declaration binding

`module_unit_bind::bind` now authenticates the full live source set against the
owned source-set signatures before joining one range. An unchanged caller with
a changed provider outside the range is rejected as stale metadata (code 12).
The binder creates the range AST itself, avoiding a caller-supplied combined
AST and a second scanner/source copy.

Every local declaration retains a coordinate containing its kind, original
module index, original one-based declaration index and original byte span.
Named type leaves and qualified value references index a separate owned target
table with the same coordinate form. Repeated aliases of one catalog entry
share one target; equal file-local declaration indexes in different modules
remain distinct. These target indexes are not nominal scalar type IDs or local
joined declaration indexes. Coordinates and the range AST remain readable after
the original parser and signature owners leave scope.

Builtin type leaves, generic parameters and compiler-owned raw intrinsics retain
their existing handling. Local variable and unqualified value resolution remain
with the body name resolver. This binding layer does not claim that a mapped
call has passed argument, effect, unsafe, ownership or return-type checks.
Imported type leaves must name a struct, resource or enum. Catalog lookup
enforces public/private visibility for cross-module references; a private name
does not become usable merely because its provider is in the same group.

The existing 1048576 binding work bound charges arena visits, catalog-cache setup
and measured lookup probes. The cache has one slot per already-admitted catalog
entry and performs constant-time reuse after a lookup. Source matches and byte
comparisons retain the existing metadata validation behavior. Type declaration
ownership advances through parser declaration order rather than rescanning all
declarations to identify every generic parameter. Failures publish no Bound
owner and retain the original full-source module index and module-local span.

The development probe rebuilds signatures after SDK compaction and binds all
twelve ranges of the actual 160-module dispatcher. It independently compares
every retained definition and target coordinate against the corresponding
original Parsed declaration. It retains 152032 expressions and 11513 entries;
the sum of per-group unique targets is 2146, with at most 252 in one group.
Measured binding work peaks at 134589, below the unchanged bound. Five native
tests cover alias sharing, distinct provider identities, owner removal, private
types/functions, stale external providers, generic parameter handling, original
locations, invalid ranges and AST partition regressions. Evidence:
`docs/ops/evidence/candidate-module-unit-bind-20261009.json`.

The next layer must consume these coordinates in local body name/type checking
and import exact provider layouts/signatures without flattening all bodies.
HIR/MIR and C/link identities must subsequently retain the same coordinates.
The command dispatcher still uses the original single-owner compiler path.

## Provider field shapes and callable interfaces

`module_unit_interfaces` now owns the provider data required by a unit checker:
the canonical type graph, selected struct/resource/enum/SDK-Vec field shapes,
enum payload ranges, field names, callable argument/result types, effects,
must-use and unsafe flags, foreign link names, repr(C) admission, resource
abilities, and declared cleanup authority. Original parser, binding and signature
owners can leave scope without invalidating the retained queries. Target byte
offsets/alignment and required transitive drop effects remain checker/backend work.

The global signature graph is copied as a stable prefix. Additional provider
field types use the existing canonical resolver and its 4096-node/depth-64
limits. A deduplicated queue follows only the selected unit's target types,
callable signature types and their field/composite dependencies. It does not
materialize a field table for every nominal node in the global prefix. Each
result retains the existing 512-shape, 1024-field and 512-callable bounds;
parameter storage is capped at 32768 and owned spelling storage at 65536 bytes.

Generic function targets retain their coordinates and declaration flags with
`template=true`, result zero and no concrete parameter list. They must be
instantiated before call typing. An unspecialized SDK Vec declaration is not
treated as an ordinary resource; concrete Vec graph nodes retain their exact
element and provider identity and share the original authenticated physical fields.

Binding validation compares the retained unit bytes to the current source range
and repeats target lookup against the current catalog. This prevents an old
binding from bypassing a newly private provider even when supplied with freshly
rebuilt metadata. All full-source bytes must also match the supplied metadata.
The implementation uses borrowed local builder containers and transfers them
once into the result, keeping the existing resource field mutation restrictions.

The projection work counter charges graph copying, queue visits, fields,
parameters and owned spelling bytes within 1048576 units. Canonical resolution
continues to use its existing node/depth bounds. `field` returns the canonical
field type through a one-based field ID, unwraps safe record references, and
enforces resource/Vec module privacy. Enum payload access still requires the
dedicated active-variant path and is rejected by ordinary field lookup.

Six development native regressions pass, including nested array/nominal fields,
resource metadata, enum payload ranges, foreign signatures/link names, effects,
must-use, generic-template deferral, detached-owner queries, field privacy,
stale caller/provider rejection and original binding regressions. The actual
160-module dispatcher occupies 1046575 bytes and imports interfaces for all
thirteen AST groups. Group maxima are 131 shapes, 910 fields, 198 callables and
401 canonical type nodes. Projection work peaks at 10629; name binding peaks
at 146397. The probe compares retained shape/field coordinates and names to
original provider syntax. Evidence:
`docs/ops/evidence/candidate-module-unit-interfaces-20261009.json`.

The unit body checker must now consume these field and callable interfaces,
resolve local annotation roots and specialize templates. Existing single-owner
type-check/HIR/MIR code still uses local numeric declaration tags; adapting that
consumer is the next step. The command dispatcher, full C output and stage1/stage2
are not yet connected to the unit pipeline.

## Canonical local annotation roots

Fairy C's module_unit_interfaces now maps declaration results, parameter and
local entries, field types, cast targets, explicit specialization arguments, and
byte-loop binder types from joined AST root IDs to canonical graph IDs.
Resolution runs against the original Parsed owners and original module
coordinates while preserving original error spans, and local annotation types
seed provider field projection. Generic declaration roots carry required=true,
deferred=true, and type=0 until specialization, whereas all unrequested child
and list nodes carry required=false. Interfaces owns this mapping, so no
borrowed owner survives.

Eight development native regressions pass. The actual dispatcher processes 160
modules (1046575 bytes) in thirteen units, with 10289 concrete annotation roots
and 69 deferred template roots. Group maxima are 409 canonical graph types, 131
shapes, 910 fields and 198 callables; projection work peaks at 27474. Existing
numeric limits remain fixed. Evidence: `docs/ops/evidence/candidate-module-unit-annotations-20261009.json`.

Next: specialize deferred templates and connect local symbol/expression/body
typing to the owned annotations, provider fields and callable signatures.
Annotation collection does not validate initializer, argument or return
expression compatibility. CLI/HIR/MIR/C unit integration, target layouts,
transitive cleanup effects, stage1/stage2, release rebuilding and Linux remain
open.

## Retained type and signature implementation notes

### module_types.fy

IDs belong to one owned graph. Composite identity refers to canonical child
IDs; nominal leaves retain module+declaration, never file-local numeric tags.
1 scalar, 2 struct, 3 resource, 4 enum, 5 Option, 6 Result,
7 shared borrow, 8 mutable borrow, 9 text, 10 bytes, 11 mutable bytes, 12 pointer.
13 authenticated SDK Vec, whose first edge is its exact element identity.
16 fixed array: first is the canonical element, second is the bounded length.
Roll back all speculative interning after a failed resolution. A rejected
branch cannot publish a valid prefix or consume the graph's type budget.

### module_signatures.fy

Owned source-set signatures; table indices are never the single-file scalar
type IDs used by the older module checker. That checker remains closed until
body typing and nominal layout ownership can consume this exact graph.

### sdk_vector_plan.fy

A concrete SDK template instance is identified by its canonical graph node,
exact provider declaration and element node. Aliases never create instances.
This owner retains the catalog's original bytes and the type graph after all
parser owners leave scope. It does not yet instantiate SDK executable bodies.
Candidate Windows AMD64 storage transport: opaque data + five usize fields.
These are descriptor facts, not element sizeof/alignment or a public C ABI.

## Owned generic signature instances (2026-10-09)

`module_unit_generics` builds an owned per-unit plan of explicit generic-call
signature instances. Requests are keyed by the original provider module and
declaration together with the ordered canonical type arguments, so alias and
repeated requests share one entry while distinct providers and argument order
stay separate. Each instance owns its result and parameter graph IDs, effects,
must-use and unsafe flags, and per-expression request identities. Canonical type
resolution substitutes generic names without touching the original AST or source
bytes, which keeps provider error spans intact, and admits a defined set of
generic wrappers: safe borrows, scalar slices, pointers, fixed arrays,
Option/Result, and the authenticated SDK Vec.

Admission stays narrow. Ordinary user functions accept only the six existing
builtin scalars, and exact authenticated `std::vec` sources additionally accept
named non-generic structs and resources; unrequested generic-body calls remain
explicitly deferred, and exact SDK storage intrinsics are flagged separately.
The existing caps are retained: 8 type args, 16 instances per template, 64 per
unit, canonical graph 4096, parameters 32768, and work 1048576. Failed plan
collection rolls graph appends back, and the interface builder follows the
resulting signature types for provider fields.

Development check and probe build pass, along with thirteen native regressions
covering alias reuse, provider/argument-order identity, owned signature queries,
pointer/slice/array substitution, rejected argument/contract/shadow cases,
atomic graph rollback, the 64-instance cap and exact SDK resource admission. The
actual 160-module dispatcher remains within 1MiB at 1048186 bytes. The initial
thirteen AST groups pass interface specialization for the first two groups (12
and 61 instances), then reject module 37 at 4953..4970 with E4503. AST-only
packing must now refine its ranges under the existing specialization limits.
Full dispatcher interface planning is incomplete. Evidence:
`docs/ops/evidence/candidate-module-unit-generics-20261009.json`.

After range refinement, the owned instance keys and canonical signatures must
drive template body cloning, nested specialization and recursion checks, local
expression/body typing, and HIR/MIR/C/CLI consumers. Target layout, transitive
cleanup effects, stage1/stage2, release rebuilding and Linux remain open. Public
language/SDK ABI, DB, CI and runner implementations are unchanged.

## Semantic pressure range refinement (2026-10-09)

Fairy C's module_unit_plan starts from the existing AST arena grouping and, for
each range, checks binding together with owned interfaces and signature
specialization. It bisects only pressure failures code10 or E4503 while
preserving ordered source module/declaration coordinates and every input
declaration and body; ordinary semantic, privacy, and stale errors return
immediately, and an irreducible single-module error stays rejected. Planning is
bounded by 1048576 work and depth 10, and each admission retains the existing
arena/type/field/shape, 64-total-instance, and 16-per-template bounds. The
planner returns owned ranges and observed admission maxima, and the current
observation independently reconstructs each admitted interface afterward.

Development check, probe build and two new native planner regressions pass. The
earlier thirteen generic/interface/type regressions remain valid after
rechecking their original compiler, source, SDK fixture and test hashes. The
complete 160-module dispatcher (1048186 bytes) now admits fifteen ranges after
seventeen attempts and two splits. Planning work is 197; observed binding and
admitted-interface work maxima are 146311 and 81469. All fifteen interfaces are
reconstructed and checked against input syntax. Per-range maxima are 61
specialization instances, 433 graph types, 131 shapes, 907 fields and 164
callables. Per-range instance counts sum to 476, linking 1281 explicit requests;
fourteen generic-body requests remain deferred. Annotation counts are 10327
concrete and 69 deferred. Evidence: `docs/ops/evidence/candidate-module-unit-plan-20261009.json`.

Next: clone owned template bodies using provider coordinates and the ordered
type arguments, then process forwarded nested requests with recursion/depth
checks. Graph IDs remain owner-local; cross-unit instance/link deduplication
must compare structural type identities. Local expression/body typing, target
layouts, transitive cleanup effects, HIR/MIR/C/CLI integration, stage1/stage2,
release rebuilding and Linux remain open. Public language/SDK ABI, DB, CI and
runner implementations are unchanged.

## Transitive generic function request graphs (2026-10-09)

This stage expands the generic function request graph from each owned signature
instance, reading the original provider AST without modifying source or copying
bodies yet. Forwarded parameter arguments are substituted in provider
coordinates, and private same-provider calls plus imported visibility are
resolved against fresh metadata. Function instances are deduplicated by provider
module/declaration together with ordered graph IDs. DFS active/completed states
reject direct, indirect, and changing-argument specialization recursion, with
the existing depth-32 bound also applying to a previously completed leaf at that
call depth. Owned dependency records are keyed by caller instance and original
provider expression ID. The design differentiates function edges, authenticated
SDK storage intrinsic element/slice facts, and authenticated Vec<T> aggregate
construction; the aggregate branch fixed an initial assertion failure in the SDK
resource-element native case. Constructor typing uses canonical graph interning
with final catalog graph verification, while interface projection follows
dependency type seeds and keeps materialized Vec shapes capped at the existing
64. Other existing bounds hold: 64 function instances, 16 per template, 8 args,
graph 4096, dependency records 16384, and work 1048576.

Development check, probe build and ten native regressions pass. They cover
direct/indirect/changing-argument recursion, a depth-32 request to an already
completed leaf, ordered argument forwarding through a private provider helper,
alias reuse, rollback, quota and semantic-error preservation, and exact SDK
resource admission. The complete dispatcher is 160 modules and 1048277 bytes,
admitted as eighteen units after five splits and twenty-three attempts. Per-unit
maxima are 57 function instances, 433 canonical graph types, 124 shapes, 888
fields and 164 callables; binding/interface work peaks at 146700/90658. Per-unit
function instance counts sum to 592 for 1281 initial calls, with 116 nested
function edges, 594 private storage intrinsic records and 100 Vec aggregate
records. Original template syntax still carries fourteen deferred request slots;
instance-specific dependency records resolve those sites separately. Evidence:
`docs/ops/evidence/candidate-module-unit-generic-dependencies-20261009.json`.

Next: own the provider function AST bodies and authenticated SDK Vec cleanup
bodies, specialize resource cleanup contexts and their nested/implicit drop
dependencies, and connect local symbol/expression/body typing. Canonical graph
IDs stay owner-local; structural cross-unit instance/link deduplication, target
layouts, transitive cleanup effects, HIR/MIR/C/CLI integration and stage1/stage2
remain open. Release rebuilding and Linux are deferred. Public language/SDK ABI,
DB, CI and runner implementations are unchanged.

## Source-stamped generic plans (2026-10-09)

`module_unit_generics::Plan` now owns a source byte length and SHA256 per
original module together with an exact canonical type graph prefix. Its
`validate` step rejects stale module sets, provider bytes that changed even
after metadata was rebuilt, and graphs that were shortened or had prefix nodes
altered, each reported with code 12. Appending new graph types is permitted,
with downstream catalog graph validation still required. Existing capacity
limits are unchanged. Provider body cloning, cleanup specialization, and unit
HIR/MIR/C/CLI integration remain the next stages. Evidence:
`docs/ops/evidence/candidate-module-unit-source-stamps-20261009.json`.

## Owned generic function bodies (2026-10-09)

`module_unit_bodies` owns requested concrete generic function AST bodies,
using body-local remapped expression, statement, entry and type IDs. Each
provider module's scanner stream is copied once, preserving original source
spans, raw bytes and the existing hex cooked-literal representation. Ordered
canonical type arguments, signatures and annotation overlays are owned; roots
cover locals, casts, explicit type arguments and byte-loop annotations. Nested
generic function, Vec aggregate and SDK storage intrinsic requests connect to
remapped body expressions through per-instance edges. Stale plan sources and
graph prefixes are validated first, and the final canonical graph is
independently catalog-validated.

Admission accounts for original plus cloned AST sizes and bisects capacity
failures with existing bounds unchanged. Development native tests pass 7/7,
including source-owner removal, ordered private forwarding, preserved literals,
local type errors with original spans, stale sources, and body entry pressure
requiring range refinement. The 160-module, 1048277-byte dispatcher corpus
passes as 25 units after 37 attempts and 12 splits. Per-unit body counts sum to
615 with 3249 resolved annotation roots and 838 request edges. Maxima are 57
bodies and 93777 body work; combined original/body expressions reach 16293 and
entries 951 within the unchanged 16384/1024 bounds. Evidence:
`docs/ops/evidence/candidate-module-unit-bodies-20261009.json`.

SDK Vec cleanup body ownership, resource cleanup contexts and implicit drop
dependencies remain next. Body expression typing, field shape expansion for new
local types, cross-unit structural instance/link deduplication, target layouts,
cleanup effects and HIR/MIR/C/CLI consumers remain pending. Release rebuilding,
Linux, whole dispatcher native compilation and stage1/stage2 were not run in
this unit. Public language/SDK ABI, DB, CI and runner implementations are
unchanged.

## Owned resource cleanup contexts (2026-10-09)

Required canonical resource and SDK Vec shapes now request cleanup contexts.
Contexts retain the resource type, declared effects, abilities, compiler-owned
dropping receiver kind 10, original cleanup spans and ordered Vec element
argument. Cleanup generic requests have a separate owner/dependency table;
private helpers expand under the existing scalar/SDK restrictions, recursion,
depth and function quotas. Requested cleanup ASTs and attributes are copied with
body-local IDs and owned provider streams. Local annotation roots resolve in the
cleanup context. Original-plus-clone admission includes cleanup nodes. Immutable
plan builders use local vectors to preserve the existing resource-field borrow
contract; failed expansion rolls graph additions back.

The first full-corpus attempt reached the 16-per-template `take_slot<T>` quota in
a single module. Cleanup calls to the exact authenticated SDK leaf are folded
to `storage_take<T>` after structurally verifying the pointer parameter, sole
return of one unsafe intrinsic call and unchanged forwarded argument. Edges
retain `inline_declaration` and `inline_unsafe` provenance. Ordinary helper calls
remain function requests; quotas and SDK source bytes are unchanged.

Development check/probe build and seven native tests pass. They cover scalar and
named-resource Vec contexts, deduplication, declared authority/abilities,
private helper calls, shadowing/recursion rejection, source-owner removal,
original positions and local annotation types. The 160-module, 1048472-byte
dispatcher corpus passes as 26 units after 39 attempts and 13 splits. Per-unit
counts sum to 625 function bodies and 763 cleanup contexts, with 3624 cleanup
annotation roots and 595 cleanup request edges. Maxima are 57 function bodies,
59 cleanup contexts and 495691 body work; combined entries reach 1015 within
the unchanged 1024 bound. Evidence:
`docs/ops/evidence/candidate-module-unit-cleanups-20261009.json`.

Implicit/nested drop dependencies and transitive cleanup effects remain next.
Body symbol/expression typing must consume the verified inline provenance;
new local type field projection, structural cross-unit links, target layouts,
HIR/MIR/C/CLI, whole dispatcher native compilation and stage1/stage2 remain
pending. Release rebuilding, Linux and push were not run. Public language/SDK
ABI, SDK source, DB, CI and runner implementations are unchanged.

## Type-level nested drop requirements (2026-10-09)

`module_unit_drops` derives an immutable drop graph from each unit's owned
interfaces. Canonical type IDs remain owner-local. Field edges retain the
original module, declaration, field index and span; Vec element edges retain the
resource declaration span. Option, Result and fixed-array edges are structural
and use no synthetic source positions. References and raw pointers have no
outgoing drop edges. Nodes combine their declared cleanup authority with the
transitive effects of contained values and propagate the absence of the drop
ability through the same graph. Cleanup context IDs refer to the existing
specialization plan.

Recursive Vec containment uses a monotone fixed point under the unchanged
4096-type, 16384-edge and 1048576-work bounds. Capacity failures return code 10
and participate in existing range refinement. Unprojected nominal metadata is
marked `complete=false`; `required()` returns code 9 for an incomplete node,
rather than treating it as a proven zero-effect type. Required shapes must be
complete before admission succeeds. These are type-level requirements;
cleanup body locals, inferred expression types and obligations after moves
still require body typing and ownership analysis. This unit does not validate
call-site authority or emit executable drops. Generic-body local field
expansion, cross-unit deduplication, layouts, HIR/MIR/C/CLI and stage1/stage2
remain pending. Public language/SDK ABI, SDK sources, DB, CI and runners are
unchanged. Evidence: `docs/ops/evidence/candidate-module-unit-drops-20261009.json`.

Development check, probe check/build and seven native tests pass (five drop
regressions and two existing range/error regressions). The exact-current
160-module, 1048472-byte dispatcher corpus retains 26 units after 39 attempts
and 13 splits. Per-unit drop edges sum to 9878; maxima are 425 graph nodes,
660 edges and 6036 drop work. Combined original/body entries remain 1015 within
the unchanged 1024 bound. This is metadata/admission evidence; full dispatcher
native compilation, release rebuilding and Linux were not run.

## Generic-body and cleanup type-demand closure (2026-10-09)

`module_unit_body_types` scans the original provider AST for requested generic
function instances and resource/SDK Vec cleanup contexts. It collects explicit
local, parameter and return types, casts, explicit generic arguments, byte-loop
types and named non-generic record constructor types. Each request records the
context kind/ID, provider module/declaration, original type root or constructor
expression ID, canonical type and source span.

Interfaces add these types to the existing pending queue and collect their
fields and cleanup contexts. Newly requested cleanup helpers are then scanned
in the same closure, until no further types or cleanups remain. Each function
instance and cleanup context is scanned once. Unrequested generic templates
remain deferred. Failed collection rolls graph additions back. The existing
source/hash/privacy and scalar-only user generic boundaries remain unchanged.
New local field/cleanup discovery is implemented; expression type matching,
name binding, inference from ordinary function-call results, move-aware cleanup
obligations and inline provenance consumption remain open. Public language/SDK
ABI, SDK sources, DB, CI and runners are unchanged. Evidence:
`docs/ops/evidence/candidate-module-unit-body-types-20261009.json`.

Development check/probe check/build and eight native regressions pass. The
current 160-module, 1048472-byte dispatcher input retains 26 units after 39
attempts and 13 splits. Per-unit demand counts sum to 6913 (3289 function and
3624 cleanup annotations), with at most 443 demands per unit. Interface work
peaks at 265344 within 1048576; combined original/body entries remain 1015/1024.
Every admitted unit verifies unchanged canonical graph size after cloning.
The private-constructor discovery path is exercised by native regressions;
the dispatcher corpus has no such additional constructor requests. Release
rebuilding, Linux, whole dispatcher native compilation and stage1/stage2 were
not run in this unit.

## Owned lexical binding for concrete bodies (2026-10-09)

`module_unit_names`, `module_unit_name_model` and `module_unit_name_walk` resolve
requested generic function and resource/SDK Vec cleanup ASTs. Current source
stamps, canonical graphs, ordered arguments and provider streams are checked
before publishing any result. Parameters, let/var declarations, payload patterns,
byte iteration, regions and the dropping receiver have separate symbol kinds.
Explicit annotation types remain canonical; inferred locals and payload types
remain unresolved. Entry/pattern/use tables and provider module/declaration
coordinates retain original expression IDs and spans.

Initializers resolve before local registration. Nested scopes restore the active
name stack; switch bindings stay in their arm, classic for post expressions are
resolved before body locals, and byte iterable expressions precede iteration
bindings. Statement lists are reversed once for linear source-order traversal.
The existing 128-active-local, depth-64 and 1048576-work bounds are maintained;
new aggregate symbol/reference tables use the existing 16384 arena bound.
Capacity failures use existing unit refinement. Duplicate, missing, private,
out-of-scope and stale references return their original provider locations.
Resource fields remain private and immutable table accessors preserve the
existing resource-field borrowing contract.

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

## Lexical binding for ordinary unit bodies (2026-10-09)

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

## Declared call results and interface closure (2026-10-09)

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

## Known values, inferred locals and matched returns (2026-10-09)

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

## Context-driven scalar operators (2026-10-09)

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
