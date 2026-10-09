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

## Bounded declared-call argument matching (2026-10-09)

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

## Canonical scalar cast targets (2026-10-09)

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

## Owned nominal and builtin field types (2026-10-09)

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
