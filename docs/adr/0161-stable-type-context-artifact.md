# ADR 0161: Establish a stable type-context artifact before body rebinding

- Status: Accepted
- Date: 2026-07-22
- Implementation source: `internal/packageartifact/context.go`
- Package artifact source: `internal/packageartifact/artifact.go`

## Context

The process-local package MIR boundary already gives functions package-stable `FunctionID` values,
but its type and foreign-declaration context still carries build-local `PackageID`, `SymbolID`,
source-span, and entity values. The process codec deliberately rejects bytes created by another
codec instance. Removing that guard before defining stable context semantics would turn incidental
Go fields into a disk compatibility contract.

Persistent reuse needs a representation that can say two type contexts are semantically identical
even when a fresh compiler process assigns different local package and symbol numbers. It does not
yet need to claim that function bodies can be rebound into that context.

## Decision

Add exact-version `fairyc.package-type-context/v1` artifacts and full-digest `ContextID` values.
The stable projection includes:

- structs, enums, resources, declaration-order fields, variants, and `repr(C)`;
- nominal and nested types rendered through exact `PackageInstanceId` values;
- resource drop behavior, canonical effects, and non-empty drop links rendered as `FunctionID`;
- foreign declaration owner, logical name, link name, C ABI, and signature.

The projection excludes source spans, entity strings, local package numbers, and local function or
foreign symbols. Its slices have deterministic ordering. Encoding uses a mandatory caller byte
limit, one schema and version, strict unknown-field rejection, one JSON value, canonical
re-encoding, and no implicit default. Decoding validates names, stable types, duplicate and ordering
rules, effects, resource/drop consistency, and foreign declarations before returning the value.

Every process-local package artifact now carries this stable projection as well as its exact MIR
context. Artifact verification recomputes the projection from current MIR and stable function
bindings and rejects disagreement. Reassembly first requires the same stable context and then keeps
the existing exact process-local context requirement. The process package codec advances to v1;
its random identity and fail-closed cross-codec behavior remain unchanged.

## Evidence

Unit tests construct equivalent contexts with different `PackageID`, drop `SymbolID`, and foreign
`SymbolID` values. They require identical descriptors, canonical bytes, and `ContextID`, then cover
strict round trip, byte bounds, unknown fields, trailing whitespace, invalid stable types,
unsorted/duplicate definitions, and contradictory drop metadata. Existing package extraction,
process codec, cache, reassembly, workspace-session, interpreter-result-42, and full compiler tests
continue through the integrated descriptor check.

## Consequences and limits

- Stable type-context meaning is no longer inferred from process-local MIR field equality.
- A changed or forged stable descriptor is rejected before reuse.
- This does not serialize or rebind function-body types, locals, spans, operations, terminators,
  foreign calls, or source provenance into a fresh compiler process.
- The descriptor still covers the complete workspace context. It can therefore change when an
  otherwise unrelated package instance changes, and it does not yet enable package-granular mixed
  generation reuse.
- `PackageInstanceId` includes the exact resolved package identity, so this contract promises
  stability across local numbering, not across source revisions.
- There is still no cache directory, lock, cleanup policy, retained-byte budget, cross-process
  coordination, remote transport, or speedup claim. The process codec remains the only cache codec.

## Next gate

Define a stable function-body artifact whose types and internal/foreign calls reference stable
context entries and function identities, plus a checked rebind operation into fresh local IDs.
Only after independently produced bytes decode, rebind, pass `mir.Verify`, and execute the same
result may the persistent-store corruption and coordination policy be reopened.

The first implementation prerequisite now exists: stable type spellings rebind recursively into a
fresh build's canonical HIR types through an exact one-to-one `PackageInstanceId` map. Built-ins,
borrows, raw pointers, fixed arrays, `Option`, `Result`, and nominal types round-trip across package
renumbering; unknown, duplicate, or malformed identities fail closed. Function-body encoding,
local-ID allocation, call rebinding, and cross-process execution evidence remain in this gate.
