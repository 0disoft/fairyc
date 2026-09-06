# ADR 0386: Generic declaration identity preparation

- Status: Accepted detached declaration/identity and structural source-header bridge; full snapshot and map integration pending
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
than invented compiler facts. It returns detached declarations, not a full executable snapshot;
source-to-specialization joins, completeness, and workspace integration are still pending.
No stable language, SDK, runtime ABI, CLI, or map version changes in this preparation.

## Transport boundary

.fymap v1 has no generic parameter fields. Its planner and encoder therefore reject generic
declarations explicitly. Silently serializing them as arity zero would erase information and make
rederived IDs contradict the original identity. A subsequent source snapshot and versioned
transport change must preserve generic declarations, specializations, and completeness together.
The Atlas CLI is not enabled by this preparation, and language/SDK/runtime versions do not change.

## Evidence

Generic identity tests cover alpha-renaming, unused arity, changed binder positions, nested
references, malformed/out-of-range indices, and inadmissible declaration kinds. Existing ordinary
SymbolId vectors remain the compatibility oracle. The codec mutation corpus rejects generic data
instead of dropping it from v1 records.
