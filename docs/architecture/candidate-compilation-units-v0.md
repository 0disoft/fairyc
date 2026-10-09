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
