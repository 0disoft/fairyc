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

## Owned AST composition

`module_ast_join.fy` copies parser-owned arenas into one source-set arena.
Original source bytes remain exact with a single newline between files; names
are not substituted and the concatenation is not reparsed. Expression, statement,
type, field, entry and attribute links shift by their own arena bases. Token
indices are mapped through their original byte positions, including trivia that
merges at the separator. `syntax.fy` exposes token facts and module-position
queries for these consumers.

The focused native test checks every source span against the original owners,
plus call arguments, literal cooking, assignment operators, query tokens,
tagged captures, switch and if bodies, for headers, cleanup receivers and enum
payload fields. Empty, generic and over-count inputs fail before publication.
Evidence: `evidence/candidate-module-join-20261008.json`.

This arena is bounded to 32 files and 32768 combined source bytes, 512
declarations, 4096 expressions/types/attributes/arguments, 8192 statements and
1024 entries/fields. Generic expansion, source-set binding and program lowering
are separate consumers; AST composition alone does not claim their completion.

## Nominal program consumers

`module_bind.fy` resolves named annotations and qualified expression uses through
the owned declaration catalog, then translates each exact module/declaration
coordinate to a unique declaration slot in the joined arena. Its input must
match both the signature owner's retained source set and the joined source bytes.
Aliases retain the same slot; identical names from different modules retain
different slots. These slots are transient program-owned identities, not stable
package IDs or persisted interface keys.

`module_program.fy` composes signature publication, AST join, binding, ordinary
name and type checking, owned HIR lowering, and MIR lowering. Name lookup and
duplicate checks stay within the source module. Bound nominal leaves flow into
borrow, tagged and array type consumers. Cross-module resource field access and
direct resource construction reject with private candidate code 6201; factories
and provider-owned borrowed access remain available. Callable effects, unsafe and
must-use are checked against the actual provider declarations.

The focused native API test renders verified C for imported structs/resources/
enums, aliases and duplicate helper names, and rejects mismatched nominal types,
private calls/representation, missing permissions, ignored must-use, and an
invalid provider with its original file-local position. It also rejects stale
signature inputs. Evidence: `evidence/candidate-module-program-20261008.json`.

This private API currently reuses one joined HIR/MIR owner (module number zero,
unique joined declaration numbers). The catalog-to-arena mapping is owned during
compilation; persistent package identity and per-file debug maps are not claimed.
The combined source/global-name limits remain 32768 bytes/128 globals. SDK,
generic and receiver admission, imported return-origin generalization, CLI
program output and actual execution of the emitted multi-file C remain separate
work. The existing scalar-only `--check-modules-file` command is unchanged.

## Multi-file C CLI

`--c-modules-file <source-list>` uses the same file loader and explicit relative
path-list rules as `--check-modules-file`, then compiles the whole source set
through `module_program.fy`. It publishes buffered C only after every provider
and caller body, owned HIR/MIR and C entry check succeeds. Diagnostics retain the
original path and byte range, with private phase/code values. C output requires
exactly one nonforeign `fn i32 main()` across the input set. This is an explicit
source list, not the production workspace/package manifest command.

`verify_module_program.py --phase cli` compares emitted C from development and
release candidate images, then executes separately compiled observed C at low
and optimized profiles. The resource case calls a separately compiled C helper
from its provider-owned cleanup; the expected single `A` trace checks exactly
one destruction after return. Other cases exercise imported struct values,
duplicate helper names and enum aliases. Private access, provider errors and
missing entry fail with empty stdout; input bytes are unchanged.

Evidence: `evidence/candidate-module-cli-20261008.json`. The previous 120-second
validation bootstrap timed out building the enlarged release CLI. A separate
validation-only 240-second Go driver overlay preserves the original compiler
source bytes; the receipt records its exact compiler and overlay hashes. This
does not resolve the production CLI timeout task (75).

## Owned repr(C) permission

`repr_c_declarations.fy` admits an exact single `@repr(C)` marker on nonempty
inert structs with C identifiers and closed scalar/C-scalar/raw-pointer fields.
Arrays, nested records, borrows, resources and tagged fields remain outside this
lane. Marker errors use private code 5106; unsupported field lanes use 5102.

`hir_shapes::Abi` retains the nominal declaration identity plus marker/name spans
in owned source bytes. HIR publication checks the actual binding, field layout
and marker bytes; MIR publication and verification recheck the same permission.
This is a separate layout proof, not a caller-supplied repr flag. HIR and MIR
observation schemas include Q rows for these records.

The focused native API tests cover declaration admission, exact retained spans,
owner teardown, HIR-to-MIR survival, missing/wrong marker bytes and forged field
lanes. Evidence: `evidence/candidate-module-repr-20261008.json`.
By-value foreign results and C names/layout transport are the next unit; this
declaration unit does not claim them. Linux remains deferred.

## repr(C) foreign results and C transport

Extern C results may now name a provider-owned repr(C) struct. Ordinary struct
parameters remain outside the closed foreign lane. Imported result annotations
resolve to the actual module/declaration before admission; a same-spelled plain
struct cannot grant ABI permission. Owned HIR and independently checked MIR
require the matching ABI record, and C output retains the verified external tag
and field names in definitions, prototypes, local values, constructors and
borrowed field access. Internal nominal names remain private numeric identities.

The separate C provider returns 8-byte and 20-byte records, plus mixed-width,
C-double and pointer fields. Development and release candidates emit identical C;
the low and optimized native executions verify all returned field values, aliases,
borrowed reads and nested ordinary wrappers. Nine invalid single-file fixtures
and two imported repr errors fail without output. Removing a valid MIR owner's
ABI proof fails verification and leaves an existing output buffer unchanged.

The larger field predicate exposed development-candidate stack exhaustion in
recursive MIR expression lowering; it also reproduces with a plain struct.
Switch, if and logical-expression snapshots now live in dedicated helpers rather
than every recursive expression frame. Source-depth accounting and ownership
joins stay unchanged. The unchanged five-field conjunction now executes through
the module CLI. This targeted repair does not establish a global stack bound.

Evidence: `evidence/candidate-module-repr-results-20261008.json` (six programs,
nine errors, twelve native executions), `evidence/candidate-module-repr-cli-20261008.json`
(seven module programs, six errors, fourteen native executions), plus three
development-native API checks. Both candidate build profiles succeeded with the
validation-only 240-second bootstrap. SDK symbols, nested repr(C) layouts,
by-value struct parameters and Linux execution remain subsequent work.

## Owner-based raw text and byte views

The candidate recognizes `str::from_raw_parts`, `byte_slice::from_raw_parts` and
`byte_slice::from_raw_parts_mut` as compiler operations inside an unsafe boundary.
Exactly three ordered arguments provide a safe owner borrow, a raw u8/void
pointer and usize length. Shared constructors require a shared owner; mutable
bytes require an exclusive owner and mutable pointer. A by-value resource or raw
pointer cannot supply owner provenance. CStr admission is still pending.

HIR RawView nodes retain all three operands and independently check lexical
unsafe permission and exact types. Origin summaries, retained loans and mutable
generation ancestry follow only the owner operand. MIR operation 38 retains the
owner even though C emission needs only the pointer and length; its independent
origin fixed point substitutes the owner's parameter at calls. Removed unsafe
permission, owner replaced with a pointer, and missing length edges fail before
any C is appended. Existing byte indexing bounds checks remain in force; text
bracket indexing remains unavailable.

Unsafe callers remain responsible for accessible pointer extent, lifetime and
valid UTF-8 for text. Construction performs no new allocation, pointer/owner
identity test, or UTF-8 scan. The native examples use valid backing storage and
exercise static/local/resource owners, parameter-return wrappers, UTF-8 text,
mutable updates, parent resumption and null/zero-length construction. Error
examples cover unsafe permission, argument types, local escape, donor movement
and simultaneous mutable siblings. Candidate donor-movement rejection currently
uses its existing private HIR E2101 loan diagnostic, not Go's E2105 spelling.

Evidence: `evidence/candidate-raw-views-20261008.json` and
`evidence/candidate-raw-view-modules-20261008.json`. SDK package authentication,
Vec specialization, CStr representation and Linux execution remain separate.

## SDK source authentication and Vec graph admission

`sha256.fy` computes bounded SHA-256 directly in Fairy. Checked arithmetic uses
16-bit limbs for modular additions and masks discarded rotation bits before
left shifts. `sdk_authority.fy` pins the exact bundled language/20 runtime/0
sources, including the optional vector module. A source-set module claiming the
reserved std namespace must match these bytes; a name or caller-supplied digest
does not grant authority.

The owned module type graph now records authenticated Vec instantiations with
the canonical provider declaration and one canonical element edge. Aliases of
the same provider reuse the same identity; differing scalar or resource elements
remain distinct. The six storage fields are opaque data followed by five usize
fields, independent of the element identity. Bare templates, safe borrow elements
including nested tagged borrows, mutated source bytes and forged provider
coordinates fail admission. Failed resolution rolls back speculative interning.

Evidence: `evidence/candidate-sdk-authority-20261008.json`. Native API tests use
an explicit SDK source argument because isolated native tests run in temporary
directories. The type graph does not yet instantiate SDK function/resource ASTs
or publish executable Vec HIR/MIR/C layouts; those are subsequent work in 46/47.
The existing ordinary module graph contract is included as the narrow regression.
