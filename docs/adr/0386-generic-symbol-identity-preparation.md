# ADR 0386: Generic declaration identity preparation

- Status: Accepted source identity and compiler snapshot integration; versioned map integration pending
- Date: 2026-09-06

## Decision

Represent original generic function signatures with ordered display parameter names and positional
type-parameter references. A type parameter's identity is its zero-based binder position, not its
source spelling or one observed concrete specialization. Arity remains significant even when a
parameter is unused in the callable signature.

The existing SymbolId v1 tuple already encodes arity. FromDeclarations now derives it from the
detached declaration and validates binder references, including nested types. The existing type
encoding can encode the new `parameter` kind with the canonical decimal index in Name; other type
fields must be empty/zero. No existing non-generic identity/signature byte changes. Parameter-name,
ordinary argument-name, and visibility changes do not alter the join key; arity and binding/type
changes do.

Only ordinary module functions are admitted at this boundary. Invalid/duplicate names, unbound or
noncanonical indices, generic foreign functions, methods, and aggregate declarations fail closed.
This is an identity boundary, not independent proof that a caller-supplied declaration is a valid
source program. The source bridge must validate original headers against current compiler facts.

## Structural source-header bridge

The compiler's immutable GenericProvenance now retains original written parameter/result types
and attributes before expansion, checks original declaration header identity, and detaches all
mutable syntax nodes. GenericDeclarations converts that source-declared signature to positional
snapshot types without using observed concrete specializations. It includes unused declarations,
declared effects, scalar/builtin types, binder references, borrows, pointers, positive literal
fixed arrays, byte slices, Option, Result, grouping, and compiler-resolved nominal types. Source
spelling is not reparsed as a type.

Successful semantic analysis publishes deterministic visible nominal scope facts, including local
types and public imported types selected by the existing exact package-import resolver. They are
omitted from JSON and rejected analyses. Compiler provenance clones these facts; the Atlas adapter
replaces their build-local type qualification with PackageCoordinateId. Alias spelling and package
renumbering do not affect identity; a different provider package UID does. Private imported names
are unavailable, and no nominal lookup is inferred from source strings or concrete specializations.

This adapter does not claim that unused bodies have been checked. Unresolved nominal names,
non-literal array lengths, unsupported shapes, and malformed effects are explicit errors rather
than invented compiler facts. GenericDeclarations returns detached declarations; BuildCompiler
adds checked source-to-specialization joins and executable completeness as described below.
No stable language, SDK, runtime ABI, CLI, or map version changes in this preparation.

Borrow facts now retain a non-serialized build-local function discriminator. The executable
summary join uses both that identity and the original span, so two specializations sharing one
source borrow do not collide or receive each other's facts. Legacy caller-built facts without
the discriminator are accepted only when their span has exactly one owner.

## Compiler snapshot integration

BuildCompiler checks the original signature against every typed implementation after substituting
its exact positional bindings. It rejects missing provenance, orphan implementations, source-span
or signature contradictions, and missing executable summaries. Original declarations and their
documentation appear once; implementations retain separate summaries and concrete type arguments.
Calls retain the original declaration plus the exact specialization ID, including generic helper
calls. Accessors detach both type arguments and all summary slices.

The compiler marks semantic results requiring generic provenance without adding a serialized
field. The raw Build entry rejects that marker, even when all templates are unused. Ordinary
functions are not classified by a generated-looking name prefix. The workspace adapter now uses
BuildCompiler; ordinary snapshots and their existing SymbolIds remain unchanged.

The concrete specialization set has exact completeness. Original template bodies remain explicitly
unavailable as universal executable facts, whether or not a particular specialization was checked.

## Transport boundary

.fymap v1 has no generic parameter fields. Its planner and encoder therefore reject generic
declarations explicitly. Silently serializing them as arity zero would erase information and make
rederived IDs contradict the original identity. The snapshot now preserves generic declarations,
specializations, and completeness together; a versioned transport change is still required.
The v1 encoder also rejects specialization calls rather than omitting their implementation target.
The Atlas CLI is not enabled by this preparation, and language/SDK/runtime versions do not change.

## Evidence

Generic identity tests cover alpha-renaming, unused arity, changed binder positions, nested
references, malformed/out-of-range indices, and inadmissible declaration kinds. Existing ordinary
SymbolId vectors remain the compatibility oracle. The codec mutation corpus rejects generic data
instead of dropping it from v1 records.
