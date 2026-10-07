# Fairy candidate module progress

The existing Go package/module contracts remain the reference. These are
candidate implementation units, not additions to the public language.

## Owned declaration and type graph

`module_catalog.fy` records declarations by source-set module index and original
declaration index. Structs, resources, and enums share lookup rules with functions;
public visibility is checked for every qualified lookup. An alias changes the
lookup path, never the declaration identity. Module/import/name collisions and
missing providers fail before a catalog is published. The owner retains exact
source bytes and rejects resolution against a different source set.

`module_types.fy` interns nominal leaves by their exact module/declaration pair.
Option/Result and shared/mutable reference descriptors refer to canonical child
IDs. Existing scalar, C-scalar, raw-pointer, text, and byte-view descriptors remain
distinct. Failed resolution rolls back all speculative nodes. Graph publication
checks nominal coordinates and rejects forward/cyclic child references.

Construction uses separate local buffers/vectors and whole-resource publication;
it does not introduce mutable access to private resource fields. Bounds remain
32 modules, 32768 source bytes per module, 4096 descriptors, and depth 64.
These IDs belong only to the current source-set graph; package-instance identity,
persistent artifacts, and imported layout publication remain subsequent work.

The native dev/release API test executes the Fairy parser, catalog, and resolver
on real source fixtures. It covers equal file-local declaration numbers in
different modules, multiple aliases to one provider, private types, duplicate and
missing imports, wrong source-set inputs, rollback, and forged graph publication.
Evidence: `evidence/candidate-module-type-graph-20261008.json`.

This unit does **not** complete nominal multi-file body typing, HIR/MIR linking,
SDK admission, or native compilation of a multi-file user program. The older
scalar-only module CLI remains closed until those consumers can own the same
type and layout graph. Linux execution is deferred to the combined phase.

## Owned callable signatures

`module_signatures.fy` resolves provider and caller annotations into the same
canonical type graph, including nominal/resource/enum leaves, Option/Result,
references, C scalars, and the exact pointer output slot. The published owner
retains result/parameter IDs, module/declaration coordinates, public visibility,
effects, unsafe, must-use, and exact C linker bytes after the parser owners leave
scope. Conflicting C alias prototypes are rejected across modules; identical
prototypes retain their shared linker name.

Invalid declaration effects/must-use, private type imports, and void parameters
fail before publication. Generic signatures, receivers, body checking, and SDK
authentication remain outside this unit. Signature success is not a whole-program
check result. Evidence: `evidence/candidate-module-signatures-20261008.json`,
with actual Fairy dev/release native API execution.
